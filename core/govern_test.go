//   Copyright (C) 2020 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
	"testing"
)

type governManager4Test struct {
	*governManager
	guardKeys []common.PrivateKey
	adminKey  common.PrivateKey
}

func generateKey() common.PrivateKey {
	key, err := common.GenerateKey("")
	if err != nil {
		panic(err)
	}
	return key
}

func newGovenManager4Test(n int) *governManager4Test {
	guards := make([]common.PrivateKey, 0)
	for i := 0; i < n; i++ {
		guards = append(guards, generateKey())
	}
	return &governManager4Test{
		governManager: newGovernManager(),
		guardKeys:     guards,
		adminKey:      generateKey(),
	}
}

func (gm *governManager4Test) getAllGuardNodes(db types.AccountDB) ([]common.Address, error) {
	addrs := make([]common.Address, 0)
	for _, g := range gm.guardKeys {
		addrs = append(addrs, g.GetPubKey().GetAddress())
	}
	return addrs, nil
}

func signData(key common.PrivateKey, data []byte) []byte {
	sig, err := key.Sign(data)
	if err != nil {
		panic("sign data error")
	}
	return sig.Bytes()
}

func (gm *governManager4Test) generateBlackUpdateTx(nonce uint64, addrs []common.Address, remove bool) *types.Transaction {
	opType := byte(0)
	if remove {
		opType = 1
	}
	signBytes := types.GenBlackOperateSignData(gm.adminKey.GetPubKey().GetAddress(), nonce, opType, addrs)
	signs := make([][]byte, 0)
	for _, guardKey := range gm.guardKeys {
		signs = append(signs, signData(guardKey, signBytes))
	}

	b := &types.BlackOperator{
		Addrs:  addrs,
		OpType: opType,
		Signs:  signs,
	}

	data, err := types.EncodeBlackOperator(b)
	if err != nil {
		panic("encode error")
	}

	source := gm.adminKey.GetPubKey().GetAddress()

	tx := &types.Transaction{
		RawTransaction: &types.RawTransaction{
			Data:     data,
			Nonce:    nonce,
			Type:     types.TransactionTypeBlacklistUpdate,
			GasLimit: types.NewBigInt(10000),
			GasPrice: types.NewBigInt(1000),
			Source:   &source,
		},
	}
	tx.Hash = tx.GenHash()
	tx.Sign = signData(gm.adminKey, tx.Hash.Bytes())
	return tx
}

func TestBlackUpdate(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	defer db.Close()
	triedb := account.NewDatabase(db, false)
	state, _ := account.NewAccountDB(common.Hash{}, triedb)

	mm := &MinerManager{}

	blacks := []common.Address{normal1, normal2, normal3}

	gm := newGovenManager4Test(100)

	governInstance = gm

	adminAddr := gm.adminKey.GetPubKey().GetAddress()

	// add
	nonce := state.GetNonce(adminAddr)

	tx := gm.generateBlackUpdateTx(nonce+1, blacks, false)

	ok, err := mm.ExecuteOperation(state, tx, 1)
	if !ok || err != nil {
		t.Fatal(err)
	}
	t.Log("add success")

	root, err := state.Commit(false)
	triedb.TrieDB().Commit(1, root, false)
	t.Log("root after add", root.Hex())

	state, _ = account.NewAccountDB(root, triedb)
	for _, addr := range blacks {
		if !gm.isBlack(state, addr) {
			t.Fatalf("should be black %v", addr)
		}
	}

	// remove
	nonce = state.GetNonce(adminAddr)

	tx = gm.generateBlackUpdateTx(nonce+1, blacks, true)

	ok, err = mm.ExecuteOperation(state, tx, 1)
	if !ok || err != nil {
		t.Fatal(err)
	}
	t.Log("remove success")

	root, err = state.Commit(false)
	triedb.TrieDB().Commit(1, root, false)
	t.Log("root after remove ", root.Hex())
	state, _ = account.NewAccountDB(root, triedb)
	for _, addr := range blacks {
		if gm.isBlack(state, addr) {
			t.Fatalf("should not be black %v", addr)
		}
	}

}

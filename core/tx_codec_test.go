//   Copyright (C) 2018 ZVChain
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
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
)

func genTx(source string, target string) *types.RawTransaction {
	var sourceAddr, targetAddr *common.Address

	sourcebyte := common.StringToAddress(source)
	sourceAddr = &sourcebyte
	if target == "" {
		targetAddr = nil
	} else {
		targetbyte := common.StringToAddress(target)
		targetAddr = &targetbyte
	}

	tx := &types.RawTransaction{
		Data:      []byte{13, 23},
		GasPrice:  types.NewBigInt(1),
		Source:    sourceAddr,
		Target:    targetAddr,
		Nonce:     rand.Uint64(),
		Value:     types.NewBigInt(rand.Uint64()),
		ExtraData: []byte{2, 3, 4},
		GasLimit:  types.NewBigInt(10000000),
		Type:      1,
	}
	return tx
}

func genBlockHeader() *types.BlockHeader {
	castor := groupsig.ID{}
	castor.SetBigInt(big.NewInt(1000))
	bh := &types.BlockHeader{
		CurTime:    time2.TimeToTimeStamp(time.Now()),
		Height:     rand.Uint64(),
		ProveValue: []byte{},
		Castor:     castor.Serialize(),
		Group:      common.Hash{},
		TotalQN:    rand.Uint64(),
		StateTree:  common.Hash{},
	}
	return bh
}

func genBlock(txNum int) *types.Block {
	bh := genBlockHeader()
	txs := make([]*types.RawTransaction, 0)
	for i := 0; i < txNum; i++ {
		tx := genTx("0x123", "0x234")
		txs = append(txs, tx)
	}
	return &types.Block{
		Header:       bh,
		Transactions: txs,
	}
}

func TestEncodeTransaction(t *testing.T) {
	b := genBlock(10)
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(bs)
}
func TestDecodeBlockTransactionWithNoTransaction(t *testing.T) {
	b := genBlock(0)
	t.Logf("block is %+v", b.Header)
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("txs %v", txs)

}

func TestDecodeBlockTransactionWithTransactions(t *testing.T) {
	b := genBlock(10)
	t.Logf("block is %+v", b.Header)
	for i, tx := range b.Transactions {
		t.Logf("before %v: %+v", i, tx)
	}
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	_, err = decodeBlockTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}

}

func TestDecodeTransactionByHash(t *testing.T) {
	b := genBlock(11)
	t.Logf("block is %+v", b.Header)
	var testHash common.Hash
	var testIndex int
	r := rand.Intn(len(b.Transactions))

	for i, tx := range b.Transactions {
		if i == r {
			testHash = tx.GenHash()
			testIndex = i
			t.Log("test hash", i, testHash.Hex())
		}
	}
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := decodeTransaction(testIndex, bs)
	if err != nil {
		t.Fatal(err)
	}
	if testHash != tx.GenHash() {
		t.Fatal("gen hash diff")
	}
	t.Log("success")
}

func TestMarshalSign(t *testing.T) {
	s := common.HexToSign("0x220ee8a9b1f85445ef27e1ae82f985087fe40854ccc3f8a6c6a5d47116420dc6000000000000000000000000000000000000000000000000000000000000000000")
	bs, err := msgpack.Marshal(s)
	t.Log(bs, err)

	var sign *common.Sign
	err = msgpack.Unmarshal(bs, &sign)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sign.Hex())
}

func TestMarshalTx(t *testing.T) {
	tx := genTx("0x123", "0x2343")
	bs, err := marshalTx(tx)
	t.Logf("%+v, %v", tx, tx.Source)
	if err != nil {
		t.Fatal(err)
	}

	tx1, err := unmarshalTx(bs)
	if err != nil {
		panic(err)
		t.Fatal(err)
	}
	t.Log(tx1, tx1.Source)
}

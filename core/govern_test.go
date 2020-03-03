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
	"testing"
)

func signData(key string, data []byte) []byte {
	kBytes := common.FromHex(key)
	privateKey := new(common.PrivateKey)
	if !privateKey.ImportKey(kBytes) {
		panic("import key fail")
	}
	sig, err := privateKey.Sign(data)
	if err != nil {
		panic("sign data error")
	}
	return sig.Bytes()
}

func generateBlackUpdateTx(nonce uint64, adminPrivateKey string, guardNodePrivateKeys []string, addrs []common.Address, remove bool) *types.Transaction {
	opType := byte(0)
	if remove {
		opType = 1
	}
	signBytes := genSignDataBytes(opType, addrs)
	signs := make([][]byte, 0)
	for _, guardKey := range guardNodePrivateKeys {
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

	kBytes := common.FromHex(adminPrivateKey)
	privateKey := new(common.PrivateKey)
	if !privateKey.ImportKey(kBytes) {
		panic("import key fail")
	}
	source := privateKey.GetPubKey().GetAddress()

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
	sign, err := privateKey.Sign(tx.Hash.Bytes())
	if err != nil {
		panic(err)
	}
	tx.Sign = sign.Bytes()
	return tx
}

func TestSendBlackUpdateTx(t *testing.T) {

}

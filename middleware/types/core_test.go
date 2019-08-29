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

package types

import (
	"fmt"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/serialize"
)

func TestTransaction(t *testing.T) {
	raw := &RawTransaction{Value: NewBigInt(5000), Nonce: 2, GasLimit: NewBigInt(1000000000), GasPrice: NewBigInt(0)}
	transaction := &Transaction{raw, common.Hash{}}
	addr := common.StringToAddress("zvff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	transaction.Source = &addr
	fmt.Println(&addr)
	addr = common.StringToAddress("zvff5a3f5747ada4eaa22f1d49c01e52ddb7875b4c")
	transaction.Target = &addr
	fmt.Println(&addr)
	b, _ := serialize.EncodeToBytes(transaction)
	fmt.Println(b)
	addr2 := common.StringToAddress("zvff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	transaction.Source = &addr2
	fmt.Println(&addr2)
	addr2 = common.StringToAddress("zvff5a3f5747ada4eaa22f1d49c01e52ddb7875b4c")
	transaction.Target = &addr2
	fmt.Println(&addr2)
	c, _ := serialize.EncodeToBytes(transaction)
	fmt.Println(c)
}

func TestTransactionsMarshalAndUnmarshal(t *testing.T) {
	src := common.StringToAddress("zv123")
	sign := common.HexToSign("0xa08da536660b93703b979a65e7059f8ef22d1c3c78c82d0ef09ecdaa587612e131800fb69b141db55a6a16bb6686904ea94e50a20603e6d7b84da15c4a77f73900")
	tx := &RawTransaction{
		Value:  NewBigInt(1),
		Nonce:  11,
		Source: &src,
		Type:   1,
		Sign:   sign.Bytes(),
	}

	txs := make([]*RawTransaction, 0)
	txs = append(txs, tx)
	bs, err := MarshalTransactions(txs)
	if err != nil {
		t.Fatal(err)
	}

	txs1, err := UnMarshalTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}
	tx1 := txs1[0]
	if tx1.GenHash() != tx.GenHash() {
		t.Error("TransactionsMarshalAndUnmarshal failed ")
	}
}

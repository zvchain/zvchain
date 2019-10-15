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
	"encoding/base64"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"math/big"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/serialize"
)

func TestTransaction(t *testing.T) {
	transaction := &RawTransaction{Value: NewBigInt(5000), Nonce: 2, GasLimit: NewBigInt(1000000000), GasPrice: NewBigInt(0)}
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
	t.Log("raw", tx, common.Bytes2Hex(tx.Sign))
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
	t.Log("after", tx1, common.Bytes2Hex(tx1.Sign))

	hashByte := tx.GenHash().Bytes()
	sign1 := common.BytesToSign(tx.Sign)
	pk, err := sign1.RecoverPubkey(hashByte)
	if err != nil {
		t.Fatal(err)
	}
	if !pk.Verify(hashByte, sign1) {
	}
	t.Log(common.Bytes2Hex(tx.Sign))
}

func TestMsgpackMarshalRawTransaction(t *testing.T) {
	src := common.BytesToAddress([]byte("4"))
	tx := RawTransaction{
		Data:      []byte("123"),
		Value:     NewBigInt(100),
		GasLimit:  NewBigInt(200),
		GasPrice:  NewBigInt(200),
		ExtraData: []byte("23323"),
		Source:    &src,
	}
	bs, err := msgpack.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}

	tx2 := RawTransaction{}
	err = msgpack.Unmarshal(bs, &tx2)
	if err != nil {
		t.Fatal(err)
	}

	if tx.GenHash() != tx2.GenHash() {
		t.Fatalf("gen hash diff:tx1=%+v, tx2=%+v", tx, tx2)
	}
}

func TestBigEndianBigBytes(t *testing.T) {
	i := uint64(1345699999)
	b := new(big.Int).SetUint64(i)
	t.Log(b.Bytes())

	t.Log(common.Uint64ToByte(i))
}

func TestGenHash(t *testing.T) {
	b1 := NewBigInt(1).SetBytesWithSign([]byte{2, 10})
	b2 := NewBigInt(1).SetBytesWithSign([]byte{2, 2, 10})

	src := common.BytesToAddress([]byte("0x123"))
	target := common.BytesToAddress([]byte("0x234"))
	tx1 := &RawTransaction{
		Source:   &src,
		Target:   &target,
		Value:    NewBigInt(100),
		GasLimit: b1,
		GasPrice: b2,
		Nonce:    10,
	}
	t.Logf("txhash %v, gaslimit %v, gasprice %v", tx1.GenHash().Hex(), tx1.GasLimit, tx1.GasPrice)

	b1.SetBytesWithSign([]byte{2, 10, 2})
	b2.SetBytesWithSign([]byte{2, 10})

	tx2 := &RawTransaction{
		Source:   &src,
		Target:   &target,
		Value:    NewBigInt(100),
		GasLimit: b1,
		GasPrice: b2,
		Nonce:    10,
	}
	t.Logf("txhash %v, gaslimit %v, gasprice %v", tx2.GenHash().Hex(), tx2.GasLimit, tx2.GasPrice)
}

func TestGenHash2(t *testing.T) {
	src := common.StringToAddress("zvd4d108ca92871ab1115439db841553a4ec3c8eddd955ea6df299467fbfd0415e")
	target := common.StringToAddress("zvd4d108ca92871ab1115439db841553a4ec3c8eddd955ea6df299467fbfd0415e")
	data, _ :=base64.StdEncoding.DecodeString("JXU4RkQ5JXU5MUNDJXU2NjJGJXU4OTgxJXU1MkEwJXU1QkM2JXU3Njg0JXU1MTg1JXU1QkI5MTIzNDU2")
	extra, _ := base64.StdEncoding.DecodeString("JXU4RkQ5JXU5MUNDJXU2NjJGJXU4OTgxJXU1MkEwJXU1QkM2JXU3Njg0JXU1MTg1JXU1QkI5MTIzNDU2")
//	sign := common.HexToSign("0x32f4b12bfba23fbe64043becc239184f7aeccbc815f4771058907ab01379062f51f580ea494d5d70e3ae3326fc5dc90946e9629689dddab6ced86deaf3b911ea02")
	tx1 := &RawTransaction{
		Source:   &src,
		Target:   &target,
		Value:    NewBigInt(1000000000),
		GasLimit: NewBigInt(1000),
		GasPrice: NewBigInt(500),
		Nonce:    10,
		Type:     0,
//		Sign:     sign.Bytes(),
		Data:     data,
		ExtraData: extra,
	}
	t.Logf("value %v",tx1.GasLimit.Bytes())
	t.Logf("txhash %v, gaslimit %v, gasprice %v", tx1.GenHash().Hex(), tx1.GasLimit, tx1.GasPrice)
}
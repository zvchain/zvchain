//   Copyright (C) 2019 ZVChain
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
	"bytes"
	"github.com/vmihailenco/msgpack"
	"testing"
)

func TestBigInt_MustGetBytesWithSign(t *testing.T) {
	bi := NewBigInt(100000)
	bs := bi.GetBytesWithSign()
	t.Log(bs, len(bs))

	if bi.IsNegative() {
		t.Fatal("non negative bigint")
	}

	expect := []byte{2, 1, 134, 160}
	if !bytes.Equal(expect, bs) {
		t.Errorf("bytes not equal to expect")
	}
}

func TestBigInt_MustSetBytesWithSign(t *testing.T) {
	expect := []byte{2, 1, 134, 160}
	bi := new(BigInt).SetBytesWithSign(expect)

	if bi.Int64() != 100000 {
		t.Errorf("set bytes error")
	}
}

func TestBigInt_MarshalMsgpack(t *testing.T) {
	bi := NewBigInt(234)
	bs, err := bi.MarshalMsgpack()
	if err != nil {
		t.Error(err)
	}
	t.Log(bs, len(bs))

	bi2 := NewBigInt(0)
	err = bi2.UnmarshalMsgpack(bs)
	if err != nil {
		t.Error(err)
	}
	if bi2.Uint64() != bi.Uint64() {
		t.Errorf("not equal")
	}
}

func TestBigInt_Get_SetBytes_Nil(t *testing.T) {
	var bi *BigInt
	bs := bi.GetBytesWithSign()
	t.Log(bs, len(bs))

	bi2 := new(BigInt).SetBytesWithSign(bs)
	t.Log(bi2)

	if bi2 != nil {
		t.Error("set bytes error")
	}
}

type bigIntWrapper struct {
	V    *BigInt `msgpack:"v"`
	Name string  `msgpack:"name"`
}

func TestMarshal_UnmarshalWrapper(t *testing.T) {
	wp := &bigIntWrapper{
		V:    NewBigInt(402),
		Name: "fuck msgpack",
	}

	bs, err := msgpack.Marshal(wp)
	if err != nil {
		t.Error(err)
	}
	t.Log(bs, len(bs))

	wp2 := &bigIntWrapper{}
	err = msgpack.Unmarshal(bs, wp2)
	if err != nil {
		t.Error(err)
	}
	if wp.V.Uint64() != wp2.V.Uint64() {
		t.Error("v not equal", wp.V, wp2.V)
	}
	if wp.Name != wp2.Name {
		t.Error("name not equal")
	}
}

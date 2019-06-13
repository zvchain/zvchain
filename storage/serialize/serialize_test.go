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

package serialize

import (
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/zvchain/zvchain/common"
)

type Account struct {
	Nonce    uint64
	Balance  big.Int
	Root     common.Hash
	CodeHash []byte
	Amount   *int
}

type accountWrapper struct {
	data Account
}

func (enc *accountWrapper) Encode(w io.Writer) error {
	return Encode(w, enc.data)
}

func TestSerialize(t *testing.T) {
	v := 1000
	a := Account{Nonce: 100, Root: common.BytesToHash([]byte{1, 2, 3}), CodeHash: []byte{4, 5, 6}, Balance: *new(big.Int).SetInt64(133), Amount: &v}
	//aw := &accountWrapper{data: a}
	accountDump(a)
	byte, err := EncodeToBytes(a)
	if err != nil {
		t.Errorf("encoding error")
	}

	var b = Account{}
	decodeErr := DecodeBytes(byte, &b)
	if decodeErr != nil {
		t.Errorf("decode error")
	}
	accountDump(b)
}

func accountDump(a Account) {
	fmt.Printf("Account nounce:%d,Root:%s,CodeHash:%v,Balance:%v,Amount:%v\n", a.Nonce, a.Root.Hex(), a.CodeHash, a.Balance.String(), *a.Amount)
}

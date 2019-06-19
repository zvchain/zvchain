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
	"github.com/vmihailenco/msgpack"
	"math/big"
)

// BigUint used as big.Int. Inheritance is for the implementation of Marshaler/Unmarshaler interface in msgpack framework
type BigInt struct {
	big.Int
}

func NewBigInt(v uint64) *BigInt {
	return &BigInt{Int: *new(big.Int).SetUint64(v)}
}

// ZeroBigInt indicate zero value
var ZeroBigInt = new(BigInt).SetInt64(0)

func (b *BigInt) Value() *big.Int {
	return &b.Int
}

// UnmarshalMsgpack implements interface Unmarshaler
func (b *BigInt) UnmarshalMsgpack(bs []byte) error {
	data := make([]byte, 0)

	// Use msgpack to decode the bs first.
	// You cannot directly decode the byte array using GobDecode function
	err := msgpack.Unmarshal(bs, &data)
	if err != nil {
		return err
	}
	return b.GobDecode(data)
}

// MarshalMsgpack implements interface Marshaler
func (b *BigInt) MarshalMsgpack() ([]byte, error) {
	bs, err := b.GobEncode()
	if err != nil {
		return nil, err
	}

	// Use msgpack to encode the byte array
	// You cannot return the byte array directly returned from GobEncode function
	return msgpack.Marshal(bs)
}

// IsNegative check if the number is negative
func (b *BigInt) IsNegative() bool {
	return b.Sign() == -1
}

// GetBytesWithSign returns a byte array of the number with the first byte representing its sign.
// It must be success
func (b *BigInt) GetBytesWithSign() []byte {
	if b == nil {
		return []byte{}
	}
	bs, err := b.GobEncode()
	if err != nil {
		return []byte{}
	}
	return bs
}

// SetBytesWithSign set the given bytes with the first byte representing its sign to the BigInt.
// It must be success
func (b *BigInt) SetBytesWithSign(bs []byte) *BigInt {
	if b == nil || len(bs) == 0 {
		return nil
	}
	if err := b.GobDecode(bs); err != nil {
		return nil
	}
	return b
}

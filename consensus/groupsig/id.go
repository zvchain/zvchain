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

// Package groupsig defines the main structures and functions for the bls algorithm implementation
package groupsig

import (
	"fmt"
	"log"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"golang.org/x/crypto/sha3"
)

// Idlength is ID byte length (256 bits, same as private key length)
const Idlength = 32

// ID is id for secret sharing, represented by big.Int
// Secret shared ID, 64 bit int, a total of 256 bits
type ID struct {
	value BnInt
}

// IsEqual check whether id is equal to rhs
func (id ID) IsEqual(rhs ID) bool {
	return id.value.IsEqual(&rhs.value)
}

// SetBigInt construct a ID with the specified big integer
func (id *ID) SetBigInt(b *big.Int) error {
	id.value.SetBigInt(b)
	return nil
}

// SetDecimalString construct a ID with the specified decimal string
func (id *ID) SetDecimalString(s string) error {
	return id.value.SetDecString(s)
}

// SetHexString construct a ID with the input hex string
func (id *ID) SetHexString(s string) error {
	return id.value.SetHexString(s)
}

func (id *ID) GetLittleEndian() []byte {
	return id.Serialize()
}

func (id *ID) SetLittleEndian(buf []byte) error {
	return id.Deserialize(buf)
}

// Deserialize construct a ID with the input byte array
func (id *ID) Deserialize(b []byte) error {
	return id.value.Deserialize(b)
}

// GetBigInt export ID into a big integer
func (id ID) GetBigInt() *big.Int {
	return new(big.Int).Set(id.value.GetBigInt())
}

// IsValid check id is valid
func (id ID) IsValid() bool {
	bi := id.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0

}

// GetHexString export ID into a hex string
func (id ID) GetHexString() string {
	return common.ToHex(id.Serialize())
}

// Serialize convert ID to byte slice (LittleEndian)
func (id ID) Serialize() []byte {
	idBytes := id.value.Serialize()
	if len(idBytes) == Idlength {
		return idBytes
	}
	if len(idBytes) > Idlength {
		panic("ID Serialize error: ID bytes is more than Idlength")
	}
	buff := make([]byte, Idlength)
	copy(buff[Idlength-len(idBytes):Idlength], idBytes)
	return buff
}

func (id ID) MarshalJSON() ([]byte, error) {
	str := "\"" + id.GetHexString() + "\""
	return []byte(str), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min")
	}
	str = str[1 : len(str)-1]
	return id.SetHexString(str)
}

func (id ID) ShortS() string {
	return common.ShortHex12(id.GetHexString())
}

// NewIDFromBigInt create ID by big.int
func NewIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.SetBigInt(b)
	if err != nil {
		log.Printf("NewIDFromBigInt %s\n", err)
		return nil
	}
	return id
}

// NewIDFromInt64 create ID by int64
func NewIDFromInt64(i int64) *ID {
	return NewIDFromBigInt(big.NewInt(i))
}

// NewIDFromInt Create ID by int32
func NewIDFromInt(i int) *ID {
	return NewIDFromBigInt(big.NewInt(int64(i)))
}

// NewIDFromAddress create ID from TAS 160-bit address (FP254 curve 256 bit or
// FP382 curve 384 bit)
//
// Bncurve.ID and common.Address do not support two-way back and forth conversions
// to each other, because their codomain is different (384 bits and 160 bits),
// and the interchange generates different values.
func NewIDFromAddress(addr common.Address) *ID {
	return NewIDFromBigInt(addr.BigInteger())
}

// NewIDFromPubkey construct ID by public key
//
// Public key -> (reduced to 160 bits) address -> (zoom in to 256/384 bit) ID
func NewIDFromPubkey(pk Pubkey) *ID {
	// Get the SHA3 256-bit hash of the public key
	h := sha3.Sum256(pk.Serialize())
	bi := new(big.Int).SetBytes(h[:])
	return NewIDFromBigInt(bi)
}

// NewIDFromString  generate ID by string, incoming string must guarantee discreteness
func NewIDFromString(s string) *ID {
	bi := new(big.Int).SetBytes(common.FromHex(s))
	return NewIDFromBigInt(bi)
}

// DeserializeID construct ID with the input byte array
func DeserializeID(bs []byte) ID {
	var id ID
	if err := id.Deserialize(bs); err != nil {
		return ID{}
	}
	return id
}

// ToAddress convert ID to address
func (id ID) ToAddress() common.Address {
	return common.BytesToAddress(id.Serialize())
}

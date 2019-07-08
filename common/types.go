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

// Package common provides common data structures and common utility functions.
package common

import (
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"

	"github.com/zvchain/zvchain/common/secp256k1"
	"github.com/zvchain/zvchain/taslog"
)

const PREFIX = "0x"

// getDefaultCurve returns the default elliptic curve
func getDefaultCurve() elliptic.Curve {
	return secp256k1.S256()
}

const (
	//Elliptic curve parameters：
	PubKeyLength = 65 //Length of public key，1 bytes curve, 64 bytes x,y。
	SecKeyLength = 97 //length of private key，65 bytes pub, 32 bytes D。
	SignLength   = 65 //length of signature，32 bytes r & 32 bytes s & 1 byte recid.

	AddressLength = 32 //Length of Address( golang.SHA3，256-bit)
	HashLength    = 32 //Length of Hash (golang.SHA3, 256-bit)。
	GroupIDLength = 32 //Length of Group
)

var DefaultLogger taslog.Logger

var (
	hashT               = reflect.TypeOf(Hash{})
	addressT            = reflect.TypeOf(Address{})
	BonusStorageAddress = BigToAddress(big.NewInt(0))

	GroupActiveAddress  = BigToAddress(big.NewInt(3)) //save the current active groups
	GroupWaitingAddress = BigToAddress(big.NewInt(4)) //save the groups waiting active
	GroupDismissAddress = BigToAddress(big.NewInt(5)) //save the groups dismissed
)

// Address data struct
type Address [AddressLength]byte

// MarshalJSON encodes the address as byte array with json format
func (a Address) MarshalJSON() ([]byte, error) {
	return []byte("\"" + a.Hex() + "\""), nil
}

// BytesToAddress returns the Address imported from the input byte array
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BigToAddress returns the address of the input big integer assignment
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress returns the address of the input string assignment
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

// SetBytes returns the address of the input byte array assignment
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[:], b[:])
}

// SetString returns the address of the input hex string assignment
func (a *Address) SetString(s string) {
	a.SetBytes(FromHex(s))
}

// Set sets other to a
func (a *Address) Set(other Address) {
	copy(a[:], other[:])
}

// MarshalText returns the hex representation of a.
func (a Address) MarshalText() ([]byte, error) {
	return Bytes(a[:]).MarshalText()
}

// UnmarshalText parses an address in hex syntax.
func (a *Address) UnmarshalText(input []byte) error {
	return UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses an address in hex syntax with json format.
func (a *Address) UnmarshalJSON(input []byte) error {
	return UnmarshalFixedJSON(addressT, input, a[:])
}

// Hex returns the hex string representation of a
func (a Address) Hex() string { return ToHex(a[:]) }

// Bytes returns the byte array representation of a
func (a Address) Bytes() []byte { return a[:] }

// BigInteger returns the big integer representation of a
func (a Address) BigInteger() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hash converts a to hash
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

// isValid checks the validity of a
func (a Address) IsValid() bool {
	return len(a.Bytes()) > 0
}

func (a *Address) String() string {
	if a == nil {
		return "nil"
	}
	return a.Hex()
}

///////////////////////////////////////////////////////////////////////////////
// Hash data struct (256-bits)
type Hash [HashLength]byte

var EmptyHash = Hash{}

// BytesToHash
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

func BigToHash(b *big.Int) Hash    { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash      { return BytesToHash(FromHex(s)) }
func HashToAddress(h Hash) Address { return BytesToAddress(h[:]) }

// Get the string representation of the underlying hash
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return ToHex(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

func (h Hash) IsValid() bool {
	return len(h.Bytes()) > 0
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax with json format.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return Bytes(h[:]).MarshalText()
}

// SetBytes sets the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// SetString sets string `s` to h. If s is larger than len(h) s will be cropped (from left) to fit.
func (h *Hash) SetString(s string) { h.SetBytes(FromHex(s)) }

// Set sets h from other
func (h *Hash) Set(other Hash) {
	copy(h[:], other[:])
}

// Generate generates implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

func (h Hash) String() string {
	return h.Hex()
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

type Hash256 Hash
type StorageSize float64

var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(0xff)
	Big257 = big.NewInt(257)

	ErrSelectGroupNil     = errors.New("selectGroupId is nil")
	ErrSelectGroupInequal = errors.New("selectGroupId not equal")
	ErrCreateBlockNil     = errors.New("createBlock is nil")
	ErrGroupNil           = errors.New("group is nil")
)

const (
	// Integer limit values.
	MaxInt8   = 1<<7 - 1
	MinInt8   = -1 << 7
	MaxInt16  = 1<<15 - 1
	MinInt16  = -1 << 15
	MaxInt32  = 1<<31 - 1
	MinInt32  = -1 << 31
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint8  = 1<<8 - 1
	MaxUint16 = 1<<16 - 1
	MaxUint32 = 1<<32 - 1
	MaxUint64 = 1<<64 - 1
)

var InstanceIndex int

type AccountData struct {
	sk   []byte //secure key
	pk   []byte //public key
	addr []byte //address
}

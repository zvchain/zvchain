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

package base

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

// RandLength random number length = 32 * 8 = 256 bits, which is related
// to the hash function used
const RandLength = 32

type Rand [RandLength]byte

// RandFromBytes use a multidimensional byte array as a seed to perform a
// SHA3 hash to generate a random number (the underlying function)
func RandFromBytes(b ...[]byte) (r Rand) {
	HashBytes(b...).Sum(r[:0])
	return
}

// NewRand SHA3 hash generation random number for system strong random seed
// (different implementation of Unix and windows system)
func NewRand() (r Rand) {
	b := make([]byte, RandLength)
	rand.Read(b)
	return RandFromBytes(b)
}

// RandFromString generate a pseudo-random number after hashing a string
func RandFromString(s string) (r Rand) {
	return RandFromBytes([]byte(s))
}

// Bytes convert a random number to a byte array
func (r Rand) Bytes() []byte {
	return r[:]
}

// GetHexString convert a random number to a hexadecimal string (without
// the 0x prefix)
func (r Rand) GetHexString() string {
	return hex.EncodeToString(r[:])
}

// r.DerivedRand(x) := Rand(r,x) := H(r || x) converted to Rand
// r.DerivedRand(x1,x2) := Rand(Rand(r,x1),x2)

// Hash overlay function. Based on the random number r, SHA3 processing is
// performed with the multidimensional byte array x as a variable to generate
// a derived random number
//

// Hard computing, no optimization, good anti-quantum attack
func (r Rand) DerivedRand(x ...[]byte) Rand {
	ri := r
	for _, xi := range x { // Traversing a multidimensional byte array
		HashBytes(ri.Bytes(), xi).Sum(ri[:0]) // Hash overlay calculation
	}
	return ri
}

// Ders Hash overlay with multidimensional strings
func (r Rand) Ders(s ...string) Rand {
	return r.DerivedRand(MapStringToBytes(s)...)
}

// Deri Hash overlay with multidimensional integers
func (r Rand) Deri(vi ...int) Rand {
	return r.Ders(MapItoa(vi)...)
}

// Module Random number modulo operation, returning a value between 0 and n-1
func (r Rand) Modulo(n int) int {
	b := big.NewInt(0)
	b.SetBytes(r.Bytes())          // Convert random numbers to big.Int
	b.Mod(b, big.NewInt(int64(n))) // Modeling n
	return int(b.Int64())
}

func (r Rand) ModuloUint64(n uint64) uint64 {
	b := big.NewInt(0)
	b.SetBytes(r.Bytes())                // Convert random numbers to big.Int
	b.Mod(b, big.NewInt(0).SetUint64(n)) // Modeling n
	return b.Uint64()
}

// RandomPerm Randomly take k numbers (with r as a random basis) from the
// interval 0 to n-1, and output this random sequence
func (r Rand) RandomPerm(n int, k int) []int {
	l := make([]int, n)
	for i := range l {
		l[i] = i
	}
	for i := 0; i < k; i++ {
		j := r.Deri(i).Modulo(n-i) + i
		l[i], l[j] = l[j], l[i]
	}
	return l[:k]
}

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
	"hash"

	"github.com/zvchain/zvchain/common"

	"golang.org/x/crypto/sha3"
)

// HashBytes generate a SHA3_256 bit hash of a multidimensional byte array
func HashBytes(b ...[]byte) hash.Hash {
	d := sha3.New256()
	for _, bi := range b {
		d.Write(bi)
	}
	return d
}

// Data2CommonHash Generate 256-bit common.Hash of data
func Data2CommonHash(data []byte) common.Hash {
	var h common.Hash
	sha3Hash := sha3.Sum256(data)
	if len(sha3Hash) == common.HashLength {
		copy(h[:], sha3Hash[:])
	} else {
		panic("Data2Hash failed, size error.")
	}
	return h
}

func String2CommonHash(s string) common.Hash {
	return Data2CommonHash([]byte(s))
}

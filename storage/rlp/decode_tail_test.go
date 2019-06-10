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

package rlp

import (
	"bytes"
	"testing"
)

type structWithTail struct {
	A, B uint
	C    []uint `rlp:"tail"`
}

func TestExampleDecode(t *testing.T) {
	// In this example, the "tail" struct tag is used to decode lists of
	// differing length into a struct.
	var val structWithTail

	err := Decode(bytes.NewReader([]byte{0xC4, 0x01, 0x02, 0x03, 0x04}), &val)

	if err != nil {
		t.Errorf("decode error")
	}

	if val.A != 1 && val.B != 2 && val.C[0] != 3 && val.C[1] != 4 {
		t.Errorf("decode not except value")
	}

}

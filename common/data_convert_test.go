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

package common

import (
	"math/big"
	"testing"
)

func TestByteToInt32(t *testing.T) {
	i := int32(2222)
	bs := Int32ToByte(i)
	t.Log(bs)

	i2 := ByteToInt32(bs)
	t.Log(i2)
	if i2 != i {
		t.Errorf("ByteToInt32 error")
	}
}

func TestInt32ToByte(t *testing.T) {
	var i int32 = 100
	bs := Int32ToByte(i)
	if len(bs) == 0 {
		t.Errorf("IntToByte error %v", bs)
	}
}

func TestMarshalBigInt(t *testing.T) {
	bi := new(big.Int).SetInt64(1000000000)
	bs, err := bi.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("bs len ", len(bs), bi, len(bi.Bytes()))
}

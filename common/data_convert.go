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
	"bytes"
	"encoding/binary"
)

func numberToByte(number interface{}) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, number)
	return buf.Bytes()
}

func bytesToNumber(ptr interface{}, b []byte) {
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, ptr)
}

func UInt32ToByte(i uint32) []byte {
	return numberToByte(i)
}

func ByteToUInt32(b []byte) uint32 {
	var x uint32
	bytesToNumber(&x, b)
	return x
}

func UInt64ToByte(i uint64) []byte {
	return numberToByte(i)
}

func ByteToUInt64(b []byte) uint64 {
	var x uint64
	bytesToNumber(&x, b)
	return x
}

func UInt16ToByte(i uint16) []byte {
	return numberToByte(i)
}

func ByteToUInt16(b []byte) uint16 {
	var x uint16
	bytesToNumber(&x, b)
	return x
}

func Int64ToByte(i int64) []byte {
	return numberToByte(i)
}

func ByteToInt64(b []byte) int64 {
	var x int64
	bytesToNumber(&x, b)
	return x
}

func Int32ToByte(i int32) []byte {
	return numberToByte(i)
}

func ByteToInt32(b []byte) int32 {
	var x int32
	bytesToNumber(&x, b)
	return x
}

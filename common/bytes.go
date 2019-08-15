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
	"encoding/hex"
)

// ToHex converts the input byte array to a hex string
func ToHex(b []byte) string {
	hex := Bytes2Hex(b)
	// Prefer output of "0x0" instead of "0x"
	if len(hex) == 0 {
		hex = "0"
	}
	return HexPrefix + hex
}

// ToZvHex converts the input byte array to a hex string
func ToAddrHex(b []byte) string {
	hex := Bytes2Hex(b)
	// Prefer output of "0x0" instead of "0x"
	if len(hex) == 0 {
		hex = "0"
	}
	return AddrPrefix + hex
}

// FromHex converts the hex string to a byte array
func FromHex(s string) []byte {
	if len(s) > len(HexPrefix) {
		if HexPrefix == s[0:len(HexPrefix)] {
			s = s[len(HexPrefix):]
		}
		if len(s)%2 == 1 {
			s = "0" + s
		}
		return Hex2Bytes(s)
	}
	return nil
}

// Copybytes returns an exact copy of the provided bytes
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

// IsHex checks the input string is a hex string
func IsHex(str string) bool {
	l := len(str)
	return l >= 4 && l%2 == 0 && str[0:2] == "0x"
}

// Bytes2Hex converts the input byte array to a hex string
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

// Hex2Bytes converts the hex string to a byte array
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)

	return h
}

// Uint64ToByte converts 64-bits unsigned integer to byte array
func Uint64ToByte(i uint64) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes()
}

// ByteToUint64 converts the byte array to a 64-bits unsigned integer
func ByteToUint64(bs []byte) uint64 {
	return binary.BigEndian.Uint64(bs)
}

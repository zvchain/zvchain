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
	"testing"
)

func TestRandFromString(t *testing.T) {
	r := RandFromString("123")
	t.Log(r)
}

func TestNewRand(t *testing.T) {
	r := NewRand()
	t.Log(r.Bytes())
}

func TestRand_ModuloUint64(t *testing.T) {
	r := RandFromBytes([]byte("3456789"))

	for i := 0; i < 10000; i++ {
		v := r.ModuloUint64(100)
		if v >= 100 {
			t.Fatalf("modulo uint64 error")
		}
	}
}

func BenchmarkRand_ModuloUint64(b *testing.B) {
	r := RandFromBytes([]byte("3456789"))
	for i := 0; i < b.N; i++ {
		r.ModuloUint64(100)
	}
}

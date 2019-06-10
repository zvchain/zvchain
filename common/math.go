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

const (
	maxInt = int(^uint(0) >> 1)
	minInt = int(-maxInt - 1)
)

// AbsInt returns the absolute value of the x integer.
//
// Special cases are:
//   AbsInt(minInt) results in a panic (not representable)
func AbsInt(x int) int {
	if x >= 0 {
		return x
	}
	if x == minInt {
		panic("absolute overflows int")
	}
	return -x
}

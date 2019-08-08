//   Copyright (C) 2019 ZVChain
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

package types

type Epoch uint64

const (
	EpochLength = 8000
)

// Returns the epoch at the given height
func EpochAt(h uint64) Epoch {
	return Epoch(h / EpochLength)
}

func (e Epoch) HeightRange() (start, end uint64) {
	start = uint64(e) * EpochLength
	end = start + EpochLength
	return
}

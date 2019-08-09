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

type Epoch interface {
	Start() uint64
	End() uint64
	Next() Epoch
}

type EpochAlg interface {
	EpochAt(h uint64) Epoch
	// createEpochByHeight returns the creating epoch ranges of the groups at the given block height
	CreateEpochByHeight(h uint64) (start, end Epoch)
}

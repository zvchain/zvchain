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

package group

import "github.com/zvchain/zvchain/middleware/types"

const firstActivateGroupEpoch epoch = groupActivateEpochGap + 1

type epoch uint64

func (e epoch) Start() uint64 {
	return uint64(e) * epochLength
}

func (e epoch) End() uint64 {
	return e.Start() + epochLength
}
func (e epoch) Next() types.Epoch {
	return epochAt(e.End() + 1)
}

type zeroEpoch struct{}

func (ge zeroEpoch) Start() uint64 {
	return 0
}

func (ge zeroEpoch) End() uint64 {
	return 0
}
func (ge zeroEpoch) Next() types.Epoch {
	return epochAt(1)
}

func epochAt(h uint64) epoch {
	return epoch(h / epochLength)
}

func activeEpoch(h uint64) epoch {
	ep := epochAt(h)
	return ep + groupActivateEpochGap + 1
}

func dismissEpoch(h uint64) epoch {
	activeEp := activeEpoch(h)
	return activeEp + groupLiveEpochs - 1
}

type epochAlg struct {
}

func (alg *epochAlg) EpochAt(h uint64) types.Epoch {
	return epochAt(h)
}

func (alg *epochAlg) CreateEpochByHeight(h uint64) (start, end types.Epoch) {
	if h < firstActivateGroupEpoch.Start() {
		return zeroEpoch{}, zeroEpoch{}
	}
	ep := epochAt(h)

	last := ep - epoch(groupActivateEpochGap) - 1
	end = last
	if last+1 < epoch(groupLiveEpochs) {
		start = end
	} else {
		start = last + 1 - epoch(groupLiveEpochs)
	}
	return
}

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

func activeEpoch(h uint64) types.Epoch {
	ep := types.EpochAt(h)
	return ep.Add(groupActivateEpochGap + 1)
}

func dismissEpoch(h uint64) types.Epoch {
	activeEp := activeEpoch(h)
	return activeEp.Add(groupLiveEpochs)
}

type groupEpochAlg struct {
}

func (alg *groupEpochAlg) CreateEpochByHeight(h uint64) (start, end types.Epoch) {
	ep := types.EpochAt(h)

	end = ep.Add(-(groupActivateEpochGap + 1))
	start = end.Add(-(groupLiveEpochs - 1))
	return
}

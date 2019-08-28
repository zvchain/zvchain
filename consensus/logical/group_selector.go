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

package logical

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

type groupSelector struct {
	gr groupReader
}

var selector *groupSelector

func (gs *groupSelector) doSelect(preBH *types.BlockHeader, height uint64) common.Hash {
	var hash = calcRandomHash(preBH, height)

	groupIS := gs.gr.getActivatedGroupsByHeight(height)
	// Must not happen
	if len(groupIS) == 0 {
		panic("no available groupIS")
	}
	seeds := make([]string, len(groupIS))
	for _, g := range groupIS {
		seeds = append(seeds, common.ShortHex(g.header.Seed().Hex()))
	}

	value := hash.Big()
	index := value.Mod(value, big.NewInt(int64(len(groupIS))))

	selectedGroup := groupIS[index.Int64()]

	stdLogger.Debugf("verify groups size %v at %v: %v, selected %v", len(groupIS), height, seeds, selectedGroup.header.Seed())
	return selectedGroup.header.Seed()
}

// groupSkipCountsBetween calculates the group skip counts between the given block and the corresponding pre block
func (gs *groupSelector) groupSkipCountsBetween(preBH, bh *types.BlockHeader) map[common.Hash]uint16 {
	skipMap := make(map[common.Hash]uint16)
	for h := preBH.Height + 1; h < bh.Height; h++ {
		s := gs.doSelect(preBH, h)
		if c, ok := skipMap[s]; ok {
			skipMap[s] = c + 1
		} else {
			skipMap[s] = 1
		}
	}
	return skipMap
}

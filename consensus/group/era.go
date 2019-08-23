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

import (
	"fmt"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

const (
	maxGroupPerEpoch        = 25 // max group num can be created during one epoch
	steadyStateBackTrackGap = 20 // The gap from the present to the steady state

	// era window consist of following graph:
	/* ^seed |---gap1---|---round1:encrypted share piece---|--gap2--|---round2:mpk share---|---gap3---|---round3:origin share piece---|---gap4---|end$ */
	eraWindow = types.EpochLength / maxGroupPerEpoch // The window length of group-create GroupRoutine

	roundWindow = (eraWindow - 4*steadyStateBackTrackGap) / 3 // The window length of each round

)

type rRange struct {
	begin, end uint64
}

func (r *rRange) inRange(h uint64) bool {
	return r.begin <= h && r.end >= h
}

func (r rRange) String() string {
	return fmt.Sprintf("%v-%v", r.begin, r.end)
}

func newRange(b uint64) *rRange {
	return &rRange{begin: b, end: b + roundWindow}
}

type era struct {
	seedHeight                                       uint64
	seedBlock                                        *types.BlockHeader
	encPieceRange, mpkRange, oriPieceRange, endRange *rRange
}

func (e *era) Seed() common.Hash {
	if e.seedExist() {
		return e.seedBlock.Hash
	}
	return common.Hash{}
}

func newEra(seedHeight uint64, seedBH *types.BlockHeader) *era {
	e := &era{
		seedHeight: seedHeight,
		seedBlock:  seedBH,
	}
	e.encPieceRange = newRange(e.seedHeight + steadyStateBackTrackGap)
	e.mpkRange = newRange(e.encPieceRange.end + steadyStateBackTrackGap)
	e.oriPieceRange = newRange(e.mpkRange.end + steadyStateBackTrackGap)
	e.endRange = newRange(e.oriPieceRange.end + steadyStateBackTrackGap)
	return e
}
func (e *era) seedExist() bool {
	return e.seedBlock != nil
}

func seedHeight(h uint64) uint64 {
	return h / eraWindow * eraWindow
}

func (e *era) sameEra(h uint64, seedBH *types.BlockHeader) bool {
	if seedHeight(h) == e.seedHeight {
		if !e.seedExist() {
			return seedBH == nil
		} else {
			return seedBH != nil && seedBH.Hash == e.Seed()
		}
	}
	return false
}

func (e *era) String() string {
	hash := "nil"
	if e.seedExist() {
		hash = e.seedBlock.Hash.Hex()
	}
	return fmt.Sprintf("%v-%v", e.seedHeight, hash)
}

func (e *era) end() uint64 {
	return e.endRange.begin
}

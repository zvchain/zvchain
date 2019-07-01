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

package group

import "github.com/zvchain/zvchain/common"

func getSeedHeight(currentHeight uint64) uint64 {
	passed := currentHeight % GroupCreateLoopTime
	return currentHeight - passed
}

// isHeightInRound check if given height is in the round with  roundNum
func isInRound(height uint64, roundNum uint64) bool {
	if roundNum > 3 {
		roundNum = roundNum % 3
	}
	r := height % GroupCreateLoopTime
	rs := r / GroupRoundTime + 1
	return rs == roundNum
}

func getRoundFirstBlockHeight(currentHeight uint64, roundNum uint64) uint64 {
	seed := getSeedHeight(currentHeight)
	return seed + (roundNum -1 ) * GroupRoundTime
}


func maskAsSent(seed common.Hash, roundNum uint64){
	//TODO: implement it
}

func hasSent(seed common.Hash, roundNum uint64) bool{
	//TODO: implement it
	return false
}

func isSelfMiner() bool{
	// TODO: implement it
	return true
}

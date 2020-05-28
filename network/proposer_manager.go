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

package network

import (
	"math"
	"sort"
	"sync"
)

const FastGroupSize = 100
const MaxFastSize = 500
const FastGroupName = "FastProposerGroup"
const NormalGroupSize = 500
const NormalGroupName = "NormalProposerGroup"

type ProposerManager struct {
	fastBucket   *ProposerBucket
	normalBucket *ProposerBucket

	fastStakeThreshold uint64

	proposers []*Proposer

	mutex sync.RWMutex
}

func (pm *ProposerManager) Len() int {
	return len(pm.proposers)
}

func (pm *ProposerManager) Less(i, j int) bool {
	return pm.proposers[i].Stake > pm.proposers[j].Stake
}

func (pm *ProposerManager) Swap(i, j int) {
	pm.proposers[i], pm.proposers[j] = pm.proposers[j], pm.proposers[i]
}

func newProposerManager() *ProposerManager {
	pm := &ProposerManager{
		proposers:    make([]*Proposer, 0),
		fastBucket:   newProposerBucket(FastGroupName, FastGroupSize),
		normalBucket: newProposerBucket(NormalGroupName, NormalGroupSize),
	}

	return pm
}

func (pm *ProposerManager) Build(proposers []*Proposer) {
	Logger.Debugf("[proposer manager] Build size:%v proposers:%v", len(proposers), proposers)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.proposers = proposers
	sort.Sort(pm)

	totalStake := uint64(0)
	for i := 0; i < len(pm.proposers); i++ {
		totalStake += pm.proposers[i].Stake
	}

	//Up to 80% of nodes put in fast bucket
	maxFastSize := int(math.Ceil(float64(len(pm.proposers)) * 0.8))
	if maxFastSize > MaxFastSize {
		maxFastSize = MaxFastSize
	}
	fastBucketSize := maxFastSize

	//top 90% of total stake
	fastBucketMaxStake := uint64(float64(totalStake) * 0.9)
	totalFastStake := uint64(0)
	for i := 0; i < maxFastSize; i++ {
		totalFastStake += pm.proposers[i].Stake
		if totalFastStake > fastBucketMaxStake {
			fastBucketSize = i + 1
			pm.fastStakeThreshold = pm.proposers[i].Stake
			break
		}
	}

	fastProposers := pm.proposers[0:fastBucketSize]
	normalProposers := pm.proposers[fastBucketSize:]

	pm.fastBucket.Build(fastProposers)
	pm.normalBucket.Build(normalProposers)

}

func (pm *ProposerManager) AddProposers(proposers []*Proposer) {
	Logger.Debugf("[proposer manager] AddProposers size:%v", len(proposers))

	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	normalProposers := make([]*Proposer, 0)

	for i := 0; i < len(proposers); i++ {
		proposer := proposers[i]

		if !pm.fastBucket.IsContained(proposer) && !pm.normalBucket.IsContained(proposer) {
			normalProposers = append(normalProposers, proposer)
		}
	}

	if len(normalProposers) > 0 {
		pm.normalBucket.AddProposers(normalProposers)
	}
}

func (pm *ProposerManager) Broadcast(msg *MsgData, code uint32) {
	if msg == nil {
		Logger.Errorf("[proposer manager] broadcast,msg is nil,code:%v", code)
		return
	}
	Logger.Debugf("[proposer manager] broadcast, code:%v", code)
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pm.fastBucket.Broadcast(msg, code)
	pm.normalBucket.Broadcast(msg, code)
}

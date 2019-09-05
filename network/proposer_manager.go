package network

import (
	"math"
	"sort"
	"sync"
)

const FAST_GROUP_SIZE = 100
const MAX_FAST_SIZE = 500
const FAST_GROUP_NAME = "FastProposerGroup"
const NORMAL_GROUP_SIZE = 500
const NORMAL_GROUP_NAME = "NormalProposerGroup"

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
	return pm.proposers[i].ID.GetHexString() < pm.proposers[j].ID.GetHexString()
}

func (pm *ProposerManager) Swap(i, j int) {
	pm.proposers[i], pm.proposers[j] = pm.proposers[j], pm.proposers[i]
}

func newProposerManager() *ProposerManager {
	pm := &ProposerManager{
		proposers:    make([]*Proposer, 0),
		fastBucket:   newProposerBucket(FAST_GROUP_NAME, FAST_GROUP_SIZE),
		normalBucket: newProposerBucket(NORMAL_GROUP_NAME, NORMAL_GROUP_SIZE),
	}

	return pm
}

func (pm *ProposerManager) Build(proposers []*Proposer) {
	Logger.Infof("[ProposerManager] Build size:%v proposers:%v", len(proposers), proposers)
	for i := 0; i < len(proposers); i++ {
		Logger.Infof("[ProposerManager] Build members ID: %v stake:%v", proposers[i].ID.GetHexString(), proposers[i].Stake)
	}
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.proposers = proposers
	sort.Sort(pm)

	totalStake := uint64(0)
	for i := 0; i < len(pm.proposers); i++ {
		totalStake += pm.proposers[i].Stake
	}

	maxFastSize := int(math.Ceil(float64(len(pm.proposers)) * 0.3))
	if maxFastSize > MAX_FAST_SIZE {
		maxFastSize = MAX_FAST_SIZE
	}
	fastBucketThresholdIndex := maxFastSize
	fastBucketMaxStake := uint64(float64(totalStake) * 0.8)
	totalFastStake := uint64(0)
	for i := 0; i < maxFastSize; i++ {
		totalFastStake += pm.proposers[i].Stake
		if totalFastStake > fastBucketMaxStake {
			fastBucketThresholdIndex = i
			pm.fastStakeThreshold = pm.proposers[i].Stake
		}
	}

	fastProposers := pm.proposers[0:fastBucketThresholdIndex]
	normalProposers := pm.proposers[fastBucketThresholdIndex:]

	pm.fastBucket.Build(fastProposers)
	pm.normalBucket.Build(normalProposers)

}

func (pm *ProposerManager) AddProposers(proposers []*Proposer) {
	Logger.Infof("[ProposerManager] AddProposers size:%v", len(proposers))
	for i := 0; i < len(proposers); i++ {
		Logger.Infof("[ProposerManager] AddProposers members ID: %v stake:%v", proposers[i].ID.GetHexString(), proposers[i].Stake)
	}
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	fastProposers := make([]*Proposer, 0)
	normalProposers := make([]*Proposer, 0)

	for i := 0; i < len(proposers); i++ {
		proposer := proposers[i]
		if proposer.Stake >= pm.fastStakeThreshold {
			if !pm.fastBucket.IsContained(proposer) {
				fastProposers = append(fastProposers, proposer)
			}
		} else {
			if !pm.normalBucket.IsContained(proposer) {
				normalProposers = append(normalProposers, proposer)
			}
		}
	}
	if len(fastProposers) > 0 {
		pm.fastBucket.AddProposers(fastProposers)
	}
	if len(normalProposers) > 0 {
		pm.normalBucket.AddProposers(normalProposers)
	}
}

func (pm *ProposerManager) Broadcast(msg *MsgData, code uint32) {
	if msg == nil {
		Logger.Errorf("[ProposerManager] broadcast,msg is nil,code:%v", code)
		return
	}
	Logger.Infof("[ProposerManager] broadcast, ID:%v code:%v", code)
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.fastBucket.Broadcast(msg, code)
	pm.normalBucket.Broadcast(msg, code)
}

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

package logical

import (
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
)

// GroupInitPool is data receiving pool
type GroupInitPool struct {
	pool model.SharePieceMap
}

func newGroupInitPool() *GroupInitPool {
	return &GroupInitPool{
		pool: make(model.SharePieceMap),
	}
}

// ReceiveData receive data
func (gmd *GroupInitPool) ReceiveData(id groupsig.ID, piece model.SharePiece) int {
	stdLogger.Debugf("GroupInitPool::ReceiveData, sender=%v, share=%v, pub=%v...\n", id.ShortS(), piece.Share.ShortS(), piece.Pub.ShortS())

	// Did not receive this member message
	if _, ok := gmd.pool[id.GetHexString()]; !ok {
		gmd.pool[id.GetHexString()] = piece
		return 0
	}
	// Received this member message
	return -1
}

func (gmd *GroupInitPool) GetSize() int {
	return len(gmd.pool)
}

// GenMemberPubKeys generate a list of group member signature public keys
// (for the check of ingot related messages)
func (gmd GroupInitPool) GenMemberPubKeys() groupsig.PubkeyMap {
	pubs := make(groupsig.PubkeyMap, 0)
	for k, v := range gmd.pool {
		pubs[k] = v.Pub
	}
	return pubs
}

// GenMinerSignSecKey generate miner signature private key
func (gmd GroupInitPool) GenMinerSignSecKey() *groupsig.Seckey {
	shares := make([]groupsig.Seckey, 0)
	for _, v := range gmd.pool {
		shares = append(shares, v.Share)
	}
	sk := groupsig.AggregateSeckeys(shares)
	return sk
}

// GenGroupPubKey generate group public key
func (gmd GroupInitPool) GenGroupPubKey() *groupsig.Pubkey {
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range gmd.pool {
		pubs = append(pubs, v.Pub)
	}
	gpk := groupsig.AggregatePubkeys(pubs)
	return gpk
}

// MinerGroupSecret is group related secrets
type MinerGroupSecret struct {
	secretSeed base.Rand // A miner targets a group of private seeds (the miner's personal private seed fixation and group information is fixed, the value is fixed)
}

func NewMinerGroupSecret(secret base.Rand) MinerGroupSecret {
	var mgs MinerGroupSecret
	mgs.secretSeed = secret
	return mgs
}

// GenSecKey generate a private private key for a group
func (mgs MinerGroupSecret) GenSecKey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(mgs.secretSeed.Deri(0))
}

// GenSecKey generate a private private key list for a group
// n : threshold number
func (mgs MinerGroupSecret) GenSecKeyList(n int) []groupsig.Seckey {
	secs := make([]groupsig.Seckey, n)
	for i := 0; i < n; i++ {
		secs[i] = *groupsig.NewSeckeyFromRand(mgs.secretSeed.Deri(i))
	}
	return secs
}

// GroupNode is group node (a miner joins multiple groups, there are multiple group nodes)
type GroupNode struct {
	minerInfo        *model.SelfMinerDO // Group-independent miner information (essentially shared across multiple GroupNodes)
	minerGroupSecret MinerGroupSecret   // Miner information related to the group
	memberNum        int                // Number of group members

	groupInitPool     GroupInitPool   // Group initialization message pool
	minerSignedSeckey groupsig.Seckey // Output: miner signature private key (aggregated by secret shared receiving pool)
	groupPubKey       groupsig.Pubkey // Output: group public key (aggregated by the miner's signature public key receiving pool)

	lock sync.RWMutex
}

func (n GroupNode) threshold() int {
	return model.Param.GetGroupK(n.memberNum)
}

func (n GroupNode) GenInnerGroup(ghash common.Hash) *JoinedGroup {
	return newJoindGroup(&n, ghash)
}

// InitForMiner initialize miner(not related to the group)
func (n *GroupNode) InitForMiner(mi *model.SelfMinerDO) {
	n.minerInfo = mi
	return
}

// InitForGroup join a group initialization
func (n *GroupNode) InitForGroup(h common.Hash) {
	n.minerGroupSecret = NewMinerGroupSecret(n.minerInfo.GenSecretForGroup(h)) // Generate a private seed for the group
	n.groupInitPool = *newGroupInitPool()                                      // Initialize the secret receiving pool
	n.minerSignedSeckey = groupsig.Seckey{}                                    // initialization
	n.groupPubKey = groupsig.Pubkey{}
	return
}

// GenSharePiece generate secret sharing for all members of the group
func (n *GroupNode) GenSharePiece(mems []groupsig.ID) groupsig.SeckeyMapID {
	shares := make(groupsig.SeckeyMapID)
	// How many thresholds are there, how many keys are generated
	secs := n.minerGroupSecret.GenSecKeyList(n.threshold())
	// How many members, how many shared secrets are generated
	for _, id := range mems {
		shares[id.GetHexString()] = *groupsig.ShareSeckey(secs, id)
	}
	return shares
}

func (n *GroupNode) getAllPiece() bool {
	return n.groupInitPool.GetSize() == n.memberNum
}

// Receiving secret sharing,
// Returns:
// 			0: normal reception
// 			-1: exception
// 			1: complete signature private key aggregation and group public key aggregation
func (n *GroupNode) SetInitPiece(id groupsig.ID, share *model.SharePiece) int {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.groupInitPool.ReceiveData(id, *share) == -1 {
		return -1
	}
	// Has received secret sharing from all members of the group
	if n.getAllPiece() {
		if n.beingValidMiner() {
			return 1
		}
		return -1
	}
	return 0
}

// beingValidMiner become an effective miner
func (n *GroupNode) beingValidMiner() bool {
	if !n.groupPubKey.IsValid() || !n.minerSignedSeckey.IsValid() {
		// Generate group public key
		n.groupPubKey = *n.groupInitPool.GenGroupPubKey()
		// Generate miner signature private key
		n.minerSignedSeckey = *n.groupInitPool.GenMinerSignSecKey()
	}
	return n.groupPubKey.IsValid() && n.minerSignedSeckey.IsValid()
}

// getSeedSecKey obtain a private key (related to the group)
// (this function is not available in the official version)
func (n GroupNode) getSeedSecKey() groupsig.Seckey {
	return n.minerGroupSecret.GenSecKey()
}

// getSignSecKey get the signature private key (this function
// is not available in the official version)
func (n GroupNode) getSignSecKey() groupsig.Seckey {
	return n.minerSignedSeckey
}

// GetSeedPubKey obtain a private key (related to the group)
func (n GroupNode) GetSeedPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(n.getSeedSecKey())
}

// GetGroupPubKey get group public key (valid after secret exchange)
func (n GroupNode) GetGroupPubKey() groupsig.Pubkey {
	return n.groupPubKey
}

func (n *GroupNode) hasPiece(id groupsig.ID) bool {
	n.lock.RLock()
	defer n.lock.RUnlock()
	_, ok := n.groupInitPool.pool[id.GetHexString()]
	return ok
}

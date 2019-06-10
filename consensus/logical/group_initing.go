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
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
)

const (
	InitNotfound = -2
	InitFail     = -1
	Initing      = 0
	// InitSuccess initialization successful, group public key generation
	InitSuccess = 1
)

// InitedGroup is miner node processor
type InitedGroup struct {
	gInfo        *model.ConsensusGroupInitInfo
	receivedGPKs map[string]groupsig.Pubkey
	lock         sync.RWMutex

	threshold int

	status int32 // -1, Group initialization failed (timeout or unable to reach consensus, irreversible)
	// 0,Group is initializing
	// 1,Group initialization succeeded
	gpk groupsig.Pubkey // output generated group public key
}

// createInitedGroup create a group in initialization
func createInitedGroup(gInfo *model.ConsensusGroupInitInfo) *InitedGroup {
	threshold := model.Param.GetGroupK(len(gInfo.Mems))
	return &InitedGroup{
		receivedGPKs: make(map[string]groupsig.Pubkey),
		status:       Initing,
		threshold:    threshold,
		gInfo:        gInfo,
	}
}

func (ig *InitedGroup) receive(id groupsig.ID, pk groupsig.Pubkey) int32 {
	status := atomic.LoadInt32(&ig.status)
	if status != Initing {
		return status
	}

	ig.lock.Lock()
	defer ig.lock.Unlock()

	ig.receivedGPKs[id.GetHexString()] = pk
	ig.convergence()
	return ig.status
}

func (ig *InitedGroup) receiveSize() int {
	ig.lock.RLock()
	defer ig.lock.RUnlock()

	return len(ig.receivedGPKs)
}

func (ig *InitedGroup) hasReceived(id groupsig.ID) bool {
	ig.lock.RLock()
	defer ig.lock.RUnlock()

	_, ok := ig.receivedGPKs[id.GetHexString()]
	return ok
}

// convergence find out the most received values
func (ig *InitedGroup) convergence() bool {
	stdLogger.Debug("begin Convergence, K=%v\n", ig.threshold)

	type countData struct {
		count int
		pk    groupsig.Pubkey
	}
	countMap := make(map[string]*countData, 0)

	// Statistical occurrences
	for _, v := range ig.receivedGPKs {
		ps := v.GetHexString()
		if k, ok := countMap[ps]; ok {
			k.count++
			countMap[ps] = k
		} else {
			item := &countData{
				count: 1,
				pk:    v,
			}
			countMap[ps] = item
		}
	}

	// Find the most elements
	var gpk groupsig.Pubkey
	var maxCnt = common.MinInt64
	for _, v := range countMap {
		if v.count > maxCnt {
			maxCnt = v.count
			gpk = v.pk
		}
	}

	if maxCnt >= ig.threshold && atomic.CompareAndSwapInt32(&ig.status, Initing, InitSuccess) {
		stdLogger.Debug("found max maxCnt gpk=%v, maxCnt=%v.\n", gpk.ShortS(), maxCnt)
		ig.gpk = gpk
		return true
	}
	return false
}

// NewGroupGenerator is group generator, parent group node or whole network node
// group external processor (non-group initialization consensus)
type NewGroupGenerator struct {
	groups sync.Map // Group ID(dummyID)-> Group creation consensus string -> *InitedGroup
}

func CreateNewGroupGenerator() *NewGroupGenerator {
	return &NewGroupGenerator{
		groups: sync.Map{},
	}
}

func (ngg *NewGroupGenerator) getInitedGroup(gHash common.Hash) *InitedGroup {
	if v, ok := ngg.groups.Load(gHash.Hex()); ok {
		return v.(*InitedGroup)
	}
	return nil
}

func (ngg *NewGroupGenerator) addInitedGroup(g *InitedGroup) *InitedGroup {
	v, _ := ngg.groups.LoadOrStore(g.gInfo.GroupHash().Hex(), g)
	return v.(*InitedGroup)
}

func (ngg *NewGroupGenerator) removeInitedGroup(gHash common.Hash) {
	ngg.groups.Delete(gHash.Hex())
}

func (ngg *NewGroupGenerator) forEach(f func(ig *InitedGroup) bool) {
	ngg.groups.Range(func(key, value interface{}) bool {
		g := value.(*InitedGroup)
		return f(g)
	})
}

const (
	// GisInit means the group is in its original state (knowing who is a group,
	// but the group public key and group ID have not yet been generated)
	GisInit int32 = iota

	// GisSendSharePiece Sent sharepiece
	GisSendSharePiece

	// GisSendSignPk sent my own signature public key
	GisSendSignPk

	// GisSendInited means group public key and ID have been generated, will casting
	GisSendInited

	// GisGroupInitDone means the group has been initialized and has been add on chain
	GisGroupInitDone
)

// GroupContext is the group consensus context, and the verification determines
// whether a message comes from within the group.
//
// Determine if a message is legal and verify in the outer layer
type GroupContext struct {
	createTime    time.Time
	is            int32                         // Group initialization state
	node          *GroupNode                    // Group node information (for initializing groups of public and signed private keys)
	gInfo         *model.ConsensusGroupInitInfo // Group initialization information (specified by the parent group)
	candidates    []groupsig.ID
	sharePieceMap model.SharePieceMap
	sendLog       bool
}

func (gc *GroupContext) GetNode() *GroupNode {
	return gc.node
}

func (gc *GroupContext) GetGroupStatus() int32 {
	return atomic.LoadInt32(&gc.is)
}

func (gc GroupContext) getMembers() []groupsig.ID {
	return gc.gInfo.Mems
}

func (gc *GroupContext) MemExist(id groupsig.ID) bool {
	return gc.gInfo.MemberExists(id)
}

func (gc *GroupContext) StatusTransfrom(from, to int32) bool {
	return atomic.CompareAndSwapInt32(&gc.is, from, to)
}

func (gc *GroupContext) generateMemberMask() (mask []byte) {
	mask = make([]byte, (len(gc.candidates)+7)/8)

	for i, id := range gc.candidates {
		b := mask[i/8]
		if gc.MemExist(id) {
			b |= 1 << byte(i%8)
			mask[i/8] = b
		}
	}
	return
}

// CreateGroupContextWithRawMessage creates a GroupContext structure from
// a group initialization message
func CreateGroupContextWithRawMessage(grm *model.ConsensusGroupRawMessage, candidates []groupsig.ID, mi *model.SelfMinerDO) *GroupContext {
	for k, v := range grm.GInfo.Mems {
		if !v.IsValid() {
			stdLogger.Debug("i=%v, ID failed=%v.\n", k, v.GetHexString())
			return nil
		}
	}
	gc := new(GroupContext)
	gc.createTime = time.Now()
	gc.is = GisInit
	gc.candidates = candidates
	gc.gInfo = &grm.GInfo
	gc.node = &GroupNode{}
	gc.node.memberNum = grm.GInfo.MemberSize()
	gc.node.InitForMiner(mi)
	gc.node.InitForGroup(grm.GInfo.GroupHash())
	return gc
}

// PieceMessage Received a secret sharing message
//
// Return -1 is abnormal, return 0 is normal, return 1 is the private key
// of the aggregated group member (used for signing)
func (gc *GroupContext) PieceMessage(id groupsig.ID, share *model.SharePiece) int {
	result := gc.node.SetInitPiece(id, share)
	return result
}

// GenSharePieces generate secret sharing sent to members of the group: si = F(IDi)
func (gc *GroupContext) GenSharePieces() model.SharePieceMap {
	shares := make(model.SharePieceMap, 0)
	secs := gc.node.GenSharePiece(gc.getMembers())
	var piece model.SharePiece
	piece.Pub = gc.node.GetSeedPubKey()
	for k, v := range secs {
		piece.Share = v
		shares[k] = piece
	}
	gc.sharePieceMap = shares
	return shares
}

// GetGroupInfo get group information(After receiving secret sharing of all members in the group)
func (gc *GroupContext) GetGroupInfo() *JoinedGroup {
	return gc.node.GenInnerGroup(gc.gInfo.GroupHash())
}

// JoiningGroups is a joined group that has not been initialized
type JoiningGroups struct {
	//groups sync.Map
	groups *lru.Cache
}

func NewJoiningGroups() *JoiningGroups {
	return &JoiningGroups{
		groups: common.MustNewLRUCache(50),
	}
}

func (jgs *JoiningGroups) ConfirmGroupFromRaw(grm *model.ConsensusGroupRawMessage, candidates []groupsig.ID, mi *model.SelfMinerDO) *GroupContext {
	gHash := grm.GInfo.GroupHash()
	v := jgs.GetGroup(gHash)
	if v != nil {
		gs := v.GetGroupStatus()
		stdLogger.Debug("found Initing group info BY RAW, status=%v...\n", gs)
		return v
	}
	stdLogger.Debug("create new Initing group info by RAW...\n")
	v = CreateGroupContextWithRawMessage(grm, candidates, mi)
	if v != nil {
		jgs.groups.Add(gHash.Hex(), v)
	}
	return v
}

func (jgs *JoiningGroups) GetGroup(gHash common.Hash) *GroupContext {
	if v, ok := jgs.groups.Get(gHash.Hex()); ok {
		return v.(*GroupContext)
	}
	return nil
}

func (jgs *JoiningGroups) Clean(gHash common.Hash) {
	gc := jgs.GetGroup(gHash)
	if gc != nil && gc.StatusTransfrom(GisSendInited, GisGroupInitDone) {
	}
}

func (jgs *JoiningGroups) RemoveGroup(gHash common.Hash) {
	jgs.groups.Remove(gHash.Hex())
}

func (jgs *JoiningGroups) forEach(f func(gc *GroupContext) bool) {
	for _, key := range jgs.groups.Keys() {
		v, ok := jgs.groups.Get(key)
		if !ok {
			continue
		}
		gc := v.(*GroupContext)
		if !f(gc) {
			break
		}
	}
}

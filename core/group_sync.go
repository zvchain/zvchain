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

package core

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
)

const (
	sendGroupLocalTopInterval   = 10
	syncGroupNeightborsInterval = 10
	syncGroupNeightborTimeout   = 5
	GroupSyncCandidatePoolSize  = 100
)

const (
	tickerGroupSendLocalTop = "send_group_local_top"
	tickerGroupSyncNeighbor = "sync_group_neightbors"
	tickerGroupSyncTimeout  = "sync_group_timeout"
)

var groupSync *groupSyncer

type groupInfo struct {
	Groups []*types.Group
}

type groupsCache struct {
	cache []*types.Group
	lock  sync.Mutex
}

func (c *groupsCache) size() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.cache == nil {
		return 0
	}
	return len(c.cache)
}

func (c *groupsCache) setData(g []*types.Group) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache = g
}

func (c *groupsCache) getData() []*types.Group {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.cache == nil {
		return nil
	}
	g := make([]*types.Group, len(c.cache))
	copy(g, c.cache)
	return g
}

func (c *groupsCache) removeGroup(h uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	size := len(c.cache)
	for idx, g := range c.cache {
		if g.GroupHeight == h {
			if idx >= size-1 {
				c.cache = nil
			} else {
				c.cache = c.cache[idx+1:]
			}
		}
	}
}

func (c *groupsCache) firstGroup() *types.Group {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.cache == nil || len(c.cache) == 0 {
		return nil
	}
	return c.cache[0]
}

type groupSyncer struct {
	gchain *GroupChain
	bchain *FullBlockChain

	syncingPeer   string
	candidatePool map[string]uint64

	cache *groupsCache

	ticker *ticker.GlobalTicker

	lock   sync.RWMutex
	logger taslog.Logger
}

// InitGroupSyncer initialize the groupSyncer. Register the ticker for sending and requesting groups to neighbors timely
// and also subscribe these events to handle requests from neighbors
func InitGroupSyncer(gChain *GroupChain, bChain *FullBlockChain) {
	gs := &groupSyncer{
		gchain:        gChain,
		bchain:        bChain,
		syncingPeer:   "",
		candidatePool: make(map[string]uint64),
		ticker:        bChain.ticker,
		cache:         &groupsCache{},
	}
	gs.logger = taslog.GetLoggerByIndex(taslog.GroupSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	gs.ticker.RegisterPeriodicRoutine(tickerGroupSendLocalTop, gs.notifyNeighbor, sendGroupLocalTopInterval)
	gs.ticker.StartTickerRoutine(tickerGroupSendLocalTop, false)

	gs.ticker.RegisterPeriodicRoutine(tickerGroupSyncNeighbor, gs.trySyncRoutine, syncGroupNeightborsInterval)
	gs.ticker.StartTickerRoutine(tickerGroupSyncNeighbor, true)

	notify.BUS.Subscribe(notify.GroupHeight, gs.groupHeightHandler)
	notify.BUS.Subscribe(notify.GroupReq, gs.groupReqHandler)
	notify.BUS.Subscribe(notify.Group, gs.groupHandler)
	notify.BUS.Subscribe(notify.BlockAddSucc, gs.onBlockAddSuccess)

	groupSync = gs
}

func (gs *groupSyncer) onBlockAddSuccess(msg notify.Message) {
	b := msg.GetData().(*types.Block)
	first := gs.cache.firstGroup()
	if first != nil && b.Header.Height == first.Header.CreateHeight {
		gs.logger.Infof("group dependOn block on chain success: blockHeight %v, start add group, size %v, height=%v", b.Header.Height, gs.cache.size(), first.GroupHeight)
		allSuccess := gs.batchAddGroup(gs.cache.getData())
		if allSuccess {
			gs.cache.setData(nil)
		}
	}
}

func (gs *groupSyncer) notifyNeighbor() bool {
	gs.sendGroupHeightToNeighbor(gs.gchain.Height())
	return true
}

func (gs *groupSyncer) sendGroupHeightToNeighbor(gheight uint64) {
	if gheight == 0 {
		return
	}
	gs.logger.Debugf("Send local group height %d to neighbor!", gheight)
	body := common.UInt64ToByte(gheight)
	message := network.Message{Code: network.GroupChainCountMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

func (gs *groupSyncer) groupHeightHandler(msg notify.Message) {
	groupHeightMsg := notify.AsDefault(msg)

	source := groupHeightMsg.Source()
	height := common.ByteToUInt64(groupHeightMsg.Body())
	peerManagerImpl.heardFromPeer(source)

	localGroupHeight := gs.gchain.Height()
	if height > localGroupHeight+1 {
		gs.logger.Debugf("Rcv groupHeight from %v, height %v, local %v", source, height, localGroupHeight)
	}

	gs.addCandidatePool(source, height)
}

func (gs *groupSyncer) addCandidatePool(source string, groupHeight uint64) {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	if len(gs.candidatePool) < GroupSyncCandidatePoolSize {
		gs.candidatePool[source] = groupHeight
		return
	}

	for id, height := range gs.candidatePool {
		if height < groupHeight {
			delete(gs.candidatePool, id)
			gs.candidatePool[source] = groupHeight
			break
		}
	}
}

func (gs *groupSyncer) candidatePoolDump() {
	gs.logger.Debugf("Candidate Pool Dump:")
	for id, groupHeight := range gs.candidatePool {
		gs.logger.Debugf("Candidate id:%s,group height:%d", id, groupHeight)
	}
}

func (gs *groupSyncer) syncTimeoutRoutineName(id string) string {
	return tickerGroupSyncTimeout + id
}

func (gs *groupSyncer) getCandidateForSync() (string, uint64) {
	localGroupHeight := gs.gchain.Height()
	gs.logger.Debugf("Local group height:%d", localGroupHeight)

	for id := range gs.candidatePool {
		if peerManagerImpl.isEvil(id) {
			gs.logger.Debugf("peer meter evil id:%+v", peerManagerImpl.getOrAddPeer(id))
			delete(gs.candidatePool, id)
		}
	}

	candidateID := ""
	var candidateMaxHeight uint64
	for id, height := range gs.candidatePool {
		if height > candidateMaxHeight {
			candidateID = id
			candidateMaxHeight = height
		}
	}

	return candidateID, candidateMaxHeight
}

func (gs *groupSyncer) trySyncRoutine() bool {
	if gs.cache.size() > 0 {
		first := gs.cache.firstGroup()
		gs.logger.Warnf("waiting for creatingBlock, groupHeight=%v, createHeight=%v", first.GroupHeight, first.Header.CreateHeight)
		return false
	}
	gs.lock.Lock()
	defer gs.lock.Unlock()

	id, candidateHeight := gs.getCandidateForSync()
	if id == "" {
		gs.logger.Debugf("Get no candidate for sync!")
		return false
	}
	if gs.syncingPeer == id {
		gs.logger.Debugf("already syncing with %v", id)
		return false
	}
	local := gs.gchain.Height()
	if local >= candidateHeight {
		gs.logger.Debugf("local heigher than candidate: %v >= %v", local, candidateHeight)
		return false
	}
	candInfo := &SyncCandidateInfo{
		Candidate:       id,
		CandidateHeight: candidateHeight,
		ReqHeight:       gs.gchain.Height() + 1,
	}

	notify.BUS.Publish(notify.GroupSync, &syncMessage{CandidateInfo: candInfo})

	gs.requestGroups(candInfo)
	return true
}

func (gs *groupSyncer) syncComplete(id string, timeout bool) bool {
	if timeout {
		peerManagerImpl.timeoutPeer(id)
		gs.logger.Warnf("sync group from %v timeout", id)
	} else {
		peerManagerImpl.heardFromPeer(id)
	}
	gs.ticker.RemoveRoutine(gs.syncTimeoutRoutineName(id))
	peerManagerImpl.updateReqBlockCnt(id, !timeout)

	gs.lock.Lock()
	defer gs.lock.Unlock()
	if gs.syncingPeer == id {
		gs.syncingPeer = ""
	}
	return true
}

func (gs *groupSyncer) requestGroups(ci *SyncCandidateInfo) {
	id := ci.Candidate
	if gs.syncingPeer == id {
		return
	}
	gs.syncingPeer = id
	gs.ticker.RegisterOneTimeRoutine(gs.syncTimeoutRoutineName(id), func() bool {
		return gs.syncComplete(id, true)
	}, syncGroupNeightborTimeout)

	gr := &syncRequest{
		ReqSize:   int32(peerManagerImpl.getPeerReqBlockCount(id)),
		ReqHeight: ci.ReqHeight,
	}
	body, err := marshalSyncRequest(gr)
	if err != nil {
		gs.logger.Errorf("marshalSyncRequest error %v", err)
		return
	}
	gs.logger.Debugf("Req group from %s,height:%v, reqSize:%v", id, gr.ReqHeight, gr.ReqSize)
	message := network.Message{Code: network.ReqGroupMsg, Body: body}
	network.GetNetInstance().Send(id, message)
}

func (gs *groupSyncer) groupReqHandler(msg notify.Message) {
	groupReqMsg := notify.AsDefault(msg)

	gr, err := unmarshalSyncRequest(groupReqMsg.Body())
	if err != nil {
		gs.logger.Errorf("unmarshalSyncRequest error %v", err)
		return
	}

	sourceID := groupReqMsg.Source()
	reqHeight := gr.ReqHeight
	gs.logger.Debugf("Rcv group req from:%s,height:%v, reqSize:%v\n", sourceID, reqHeight, gr.ReqSize)
	groups := gs.gchain.GetGroupsAfterHeight(reqHeight, int(gr.ReqSize))

	gs.sendGroups(sourceID, groups)
}

func (gs *groupSyncer) sendGroups(targetID string, groups []*types.Group) {
	if len(groups) == 0 {
		gs.logger.Debugf("Send nil group to:%s", targetID)
	} else {
		gs.logger.Debugf("Send group to %s,size %v, groups:%d-%d", targetID, len(groups), groups[0].GroupHeight, groups[len(groups)-1].GroupHeight)
	}
	body, e := marshalGroupInfo(groups)
	if e != nil {
		gs.logger.Errorf("sendGroup marshal group error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GroupMsg, Body: body}
	network.GetNetInstance().Send(targetID, message)
}

func (gs *groupSyncer) getPeerHeight(id string) uint64 {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	if v, ok := gs.candidatePool[id]; ok {
		return v
	}
	return 0
}

func (gs *groupSyncer) groupHandler(msg notify.Message) {
	groupInfoMsg := notify.AsDefault(msg)

	var complete = false
	defer func() {
		if !complete {
			gs.syncComplete(groupInfoMsg.Source(), false)
		}
	}()

	groupInfo, e := gs.unMarshalGroupInfo(groupInfoMsg.Body())
	if e != nil {
		gs.logger.Errorf("Discard GROUP_MSG because of unmarshal error:%s", e.Error())
		return
	}
	sourceID := groupInfoMsg.Source()

	groups := groupInfo.Groups
	rg := ""
	if len(groups) > 0 {
		rg = fmt.Sprintf("[%v-%v]", groups[0].GroupHeight, groups[len(groups)-1].GroupHeight)
	}
	gs.logger.Debugf("Rcv groups ,from:%s,groups len %d, %v", sourceID, len(groups), rg)
	allSuccess := gs.batchAddGroup(groups)

	peerHeight := gs.getPeerHeight(sourceID)
	if allSuccess && gs.gchain.Height() < peerHeight {
		gs.syncComplete(groupInfoMsg.Source(), false)
		complete = true
		go gs.trySyncRoutine()
	}
}

func (gs *groupSyncer) batchAddGroup(groups []*types.Group) bool {
	allSuccess := true
	for idx, group := range groups {
		e := gs.gchain.AddGroup(group)
		if e != nil && e != errGroupExist {
			gs.logger.Errorf("[groupSync]add group on chain error:%s", e.Error())

			if e == common.ErrCreateBlockNil {
				gs.cache.setData(groups[idx:])
			}
			allSuccess = false
			break
		} else {
			gs.cache.removeGroup(group.GroupHeight)
		}
	}
	return allSuccess
}

func marshalGroupInfo(e []*types.Group) ([]byte, error) {
	var groups []*tas_middleware_pb.Group
	for _, g := range e {
		groups = append(groups, types.GroupToPb(g))
	}

	groupInfo := tas_middleware_pb.GroupInfo{Groups: groups}
	return proto.Marshal(&groupInfo)
}

func (gs *groupSyncer) unMarshalGroupInfo(b []byte) (*groupInfo, error) {
	message := new(tas_middleware_pb.GroupInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		gs.logger.Errorf("unMarshalGroupInfo error:%s", e.Error())
		return nil, e
	}
	groups := make([]*types.Group, len(message.Groups))
	for i, g := range message.Groups {
		groups[i] = types.PbToGroup(g)
	}
	groupInfo := groupInfo{Groups: groups}
	return &groupInfo, nil
}

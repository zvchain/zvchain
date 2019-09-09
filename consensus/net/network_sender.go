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

package net

import (
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
)

// NetworkServerImpl implements a network transmission interface for various types of data.
type NetworkServerImpl struct {
	net network.Network
}

func NewNetworkServer() NetworkServer {
	return &NetworkServerImpl{
		net: network.GetNetInstance(),
	}
}

func id2String(ids []groupsig.ID) []string {
	idStrs := make([]string, len(ids))
	for idx, id := range ids {
		idStrs[idx] = id.GetAddrString()
	}
	return idStrs
}

func id2NetworkProposers(ids []groupsig.ID, stakes []uint64) []*network.Proposer {
	groupMembers := id2String(ids)

	networkProposers := make([]*network.Proposer, 0)
	for i := 0; i < len(groupMembers); i++ {
		ID := network.NewNodeID(groupMembers[i])
		if ID != nil {
			networkProposers = append(networkProposers, &network.Proposer{ID: *ID, Stake: stakes[i]})
		}
	}
	return networkProposers
}

/*
Group network management
*/

// BuildGroupNet builds the group net in local for inter-group communication
func (ns *NetworkServerImpl) BuildGroupNet(gid string, mems []groupsig.ID) {
	memStrs := id2String(mems)
	ns.net.BuildGroupNet(gid, memStrs)
}

// ReleaseGroupNet releases the group net in local
func (ns *NetworkServerImpl) ReleaseGroupNet(gid string) {
	ns.net.DissolveGroupNet(gid)
}

func (ns *NetworkServerImpl) FullBuildProposerGroupNet(proposers []groupsig.ID, stakes []uint64) {
	networkProposers := id2NetworkProposers(proposers, stakes)
	if len(networkProposers) > 0 {
		ns.net.BuildProposerGroupNet(networkProposers)
	}
}

func (ns *NetworkServerImpl) IncrementBuildProposerGroupNet(proposers []groupsig.ID, stakes []uint64) {
	networkProposers := id2NetworkProposers(proposers, stakes)
	if len(networkProposers) > 0 {
		ns.net.AddProposers(networkProposers)
	}
}

func (ns *NetworkServerImpl) send2Self(self groupsig.ID, m network.Message) {
	go MessageHandler.Handle(self.GetAddrString(), m)
}

// SendCastVerify happens at the proposal role.
func (ns *NetworkServerImpl) SendCastVerify(ccm *model.ConsensusCastMessage, gb *GroupBrief) {
	bh := types.BlockHeaderToPb(&ccm.BH)
	si := signDataToPb(&ccm.SI)
	message := &tas_middleware_pb.ConsensusCastMessage{Bh: bh, Sign: si}
	body, err := proto.Marshal(message)
	if err != nil {
		logger.Errorf("marshalConsensusCastMessage error:%v", err)
		return
	}

	m := network.Message{Code: network.CastVerifyMsg, Body: body}

	ns.net.SpreadToGroup(gb.GSeed.Hex(), id2String(gb.MemIds), m, nil)
}

// SendVerifiedCast broadcast the signed message for specified block proposal among group members
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage, gSeed common.Hash) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg, Body: body}

	// The verification message needs to be sent to itself, otherwise
	// it will not contain its own signature in its own fragment,
	// resulting in no rewards.
	ns.send2Self(cvm.SI.GetID(), m)

	ns.net.SpreadAmongGroup(gSeed.Hex(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,hash:%s", cvm.BlockHash.Hex())
}

// BroadcastNewBlock means network-wide broadcast for the generated block.
// Based on bandwidth and performance considerations, it only transits the block to all of the proposers and
// the next verify-group
func (ns *NetworkServerImpl) BroadcastNewBlock(block *types.Block, group *GroupBrief) {
	body, e := types.MarshalBlock(block)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	blockMsg := network.Message{Code: network.NewBlockMsg, Body: body}

	nextVerifyGroupID := group.GSeed.Hex()
	groupMembers := id2String(group.MemIds)

	msgID := []byte(blockMsg.Hash())

	ns.net.SpreadToGroup(network.FullNodeVirtualGroupID, nil, blockMsg, msgID)

	// Broadcast to the next group of light nodes
	//
	// Prevent duplicate broadcasts
	msgID[0] += 1
	ns.net.SpreadToGroup(nextVerifyGroupID, groupMembers, blockMsg, msgID)

}

// SendCastRewardSignReq sends reward transaction sign request to other members of the group
func (ns *NetworkServerImpl) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	body, e := marshalCastRewardTransSignReqMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send CastRewardTransSignReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignReq, Body: body}

	gSeed := msg.Reward.Group

	logger.Debugf("send SendCastRewardSignReq to %v", gSeed.Hex())

	ns.send2Self(msg.SI.GetID(), m)

	ns.net.SpreadAmongGroup(gSeed.Hex(), m)
}

// SendCastRewardSign sends signed message of the reward transaction to the requester by group relaying
func (ns *NetworkServerImpl) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	body, e := marshalCastRewardTransSignMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send CastRewardTransSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignGot, Body: body}
	ns.net.Send(msg.Launcher.GetAddrString(), m)
}

// ReqProposalBlock request block body from the target
func (ns *NetworkServerImpl) ReqProposalBlock(msg *model.ReqProposalBlock, target string) {
	body, e := marshalReqProposalBlockMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send marshalReqProposalBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqProposalBlock, Body: body}

	ns.net.Send(target, m)
}

// ResponseProposalBlock sends block body to the requester
func (ns *NetworkServerImpl) ResponseProposalBlock(msg *model.ResponseProposalBlock, target string) {

	body, e := marshalResponseProposalBlockMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send marshalResponseProposalBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseProposalBlock, Body: body}

	ns.net.Send(target, m)
	logger.Debugf("send response block %v %v to %v", msg.Hash, len(msg.Transactions), target)
}

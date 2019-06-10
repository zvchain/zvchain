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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
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
		idStrs[idx] = id.GetHexString()
	}
	return idStrs
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

func (ns *NetworkServerImpl) send2Self(self groupsig.ID, m network.Message) {
	go MessageHandler.Handle(self.GetHexString(), m)
}

/*
Group initialization
*/

// SendGroupInitMessage send group initialization message to the corresponding members
// Note that the group net is unavailable currently, so p2p transmission is required
func (ns *NetworkServerImpl) SendGroupInitMessage(grm *model.ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(grm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusGroupRawMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitMsg, Body: body}
	// The target group has not been built and needs to be sent point to point.
	for _, mem := range grm.GInfo.Mems {
		logger.Debugf("%v SendGroupInitMessage gHash %v to %v", grm.SI.GetID().GetHexString(), grm.GInfo.GroupHash().Hex(), mem.GetHexString())
		ns.net.Send(mem.GetHexString(), m)
	}
}

// SendKeySharePiece transit the share piece to each other of the group members
func (ns *NetworkServerImpl) SendKeySharePiece(spm *model.ConsensusSharePieceMessage) {

	body, e := marshalConsensusSharePieceMessage(spm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusSharePieceMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.KeyPieceMsg, Body: body}
	if spm.SI.SignMember.IsEqual(spm.Dest) {
		go ns.send2Self(spm.SI.GetID(), m)
		return
	}

	begin := time.Now()
	go ns.net.SendWithGroupRelay(spm.Dest.GetHexString(), spm.GHash.Hex(), m)
	logger.Debugf("SendKeySharePiece to id:%s,hash:%s, gHash:%v, cost time:%v", spm.Dest.GetHexString(), m.Hash(), spm.GHash.Hex(), time.Since(begin))
}

// SendSignPubKey broadcast the message among the group members
func (ns *NetworkServerImpl) SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage) {
	body, e := marshalConsensusSignPubKeyMessage(spkm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.SignPubkeyMsg, Body: body}
	// Send to yourself
	ns.send2Self(spkm.SI.GetID(), m)

	begin := time.Now()
	go ns.net.SpreadAmongGroup(spkm.GHash.Hex(), m)
	logger.Debugf("SendSignPubKey hash:%s, dummyId:%v, cost time:%v", m.Hash(), spkm.GHash.Hex(), time.Since(begin))
}

// BroadcastGroupInfo means group initialization completed and then issue the network-wide broadcast
// It is slow and expensive
func (ns *NetworkServerImpl) BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage) {
	body, e := marshalConsensusGroupInitedMessage(cgm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusGroupInitedMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitDoneMsg, Body: body}
	// Send to yourself
	ns.send2Self(cgm.SI.GetID(), m)

	go ns.net.Broadcast(m)
	logger.Debugf("Broadcast GROUP_INIT_DONE_MSG, hash:%s, gHash:%v", m.Hash(), cgm.GHash.Hex())

}

/*
Group coinage
*/

// SendCastVerify happens at the proposal role.
// It send the message contains the proposed-block to all of the members of the verify-group for the verification consensus
func (ns *NetworkServerImpl) SendCastVerify(ccm *model.ConsensusCastMessage, gb *GroupBrief, proveHashs []common.Hash) {
	bh := types.BlockHeaderToPb(&ccm.BH)
	si := signDataToPb(&ccm.SI)

	for idx, mem := range gb.MemIds {
		message := &tas_middleware_pb.ConsensusCastMessage{Bh: bh, Sign: si, ProveHash: proveHashs[idx].Bytes()}
		body, err := proto.Marshal(message)
		if err != nil {
			logger.Errorf("marshalConsensusCastMessage error:%v %v", err, mem.GetHexString())
			continue
		}
		m := network.Message{Code: network.CastVerifyMsg, Body: body}
		go ns.net.Send(mem.GetHexString(), m)
	}
}

// SendVerifiedCast broadcast the signed message for specified block proposal among group members
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg, Body: body}

	// The verification message needs to be sent to itself, otherwise
	// it will not contain its own signature in its own fragment,
	// resulting in no rewards.
	go ns.send2Self(cvm.SI.GetID(), m)

	go ns.net.SpreadAmongGroup(receiver.GetHexString(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,hash:%s", cvm.BlockHash.Hex())
}

// BroadcastNewBlock means network-wide broadcast for the generated block.
// Based on bandwidth and performance considerations, it only transits the block to all of the proposers and
// the next verify-group
func (ns *NetworkServerImpl) BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief) {
	body, e := types.MarshalBlock(&cbm.Block)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	blockMsg := network.Message{Code: network.NewBlockMsg, Body: body}

	nextVerifyGroupID := group.Gid.GetHexString()
	groupMembers := id2String(group.MemIds)

	// Broadcast to a virtual group of heavy nodes
	heavyMinerMembers := core.MinerManagerImpl.GetHeavyMiners()

	validGroupMembers := make([]string, 0)
	for _, mid := range groupMembers {
		find := false
		for _, hid := range heavyMinerMembers {
			if hid == mid {
				find = true
				break
			}
		}
		if !find {
			validGroupMembers = append(validGroupMembers, mid)
		}
	}

	go ns.net.SpreadToGroup(network.FullNodeVirtualGroupID, heavyMinerMembers, blockMsg, []byte(blockMsg.Hash()))

	// Broadcast to the next group of light nodes
	//
	// Prevent duplicate broadcasts
	if len(validGroupMembers) > 0 {
		go ns.net.SpreadToGroup(nextVerifyGroupID, validGroupMembers, blockMsg, []byte(blockMsg.Hash()))
	}

}

// AnswerSignPkMessage sends the group-related public key request to requester
func (ns *NetworkServerImpl) AnswerSignPkMessage(msg *model.ConsensusSignPubKeyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AnswerSignPkMsg, Body: body}

	begin := time.Now()
	go ns.net.Send(receiver.GetHexString(), m)
	logger.Debugf("AnswerSignPkMessage %v, hash:%s, dummyId:%v, cost time:%v", receiver.GetHexString(), m.Hash(), msg.GHash.Hex(), time.Since(begin))
}

// AskSignPkMessage sends a request for group-related public key to the given receiver
func (ns *NetworkServerImpl) AskSignPkMessage(msg *model.ConsensusSignPubkeyReqMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubkeyReqMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AskSignPkMsg, Body: body}

	begin := time.Now()
	go ns.net.Send(receiver.GetHexString(), m)
	logger.Debugf("AskSignPkMessage %v, hash:%s, cost time:%v", receiver.GetHexString(), m.Hash(), time.Since(begin))
}

/*
Pre-establishment consensus
*/

// SendCreateGroupRawMessage sends the group-create raw message to other members of the group
func (ns *NetworkServerImpl) SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) {
	body, e := marshalConsensusCreateGroupRawMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupaRaw, Body: body}

	var groupID = msg.GInfo.GI.ParentID()
	go ns.net.SpreadAmongGroup(groupID.GetHexString(), m)
}

// SendCreateGroupSignMessage sends signed message for the group-create raw message to the requester
func (ns *NetworkServerImpl) SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage, parentGid groupsig.ID) {
	body, e := marshalConsensusCreateGroupSignMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupSign, Body: body}

	go ns.net.SendWithGroupRelay(msg.Launcher.GetHexString(), parentGid.GetHexString(), m)
}

// SendCastRewardSignReq sends bonus transaction sign request to other members of the group
func (ns *NetworkServerImpl) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	body, e := marshalCastRewardTransSignReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignReq, Body: body}

	gid := groupsig.DeserializeID(msg.Reward.GroupID)

	network.Logger.Debugf("send SendCastRewardSignReq to %v", gid.GetHexString())

	ns.net.SpreadAmongGroup(gid.GetHexString(), m)
}

// SendCastRewardSign sends signed message of the bonus transaction to the requester by group relaying
func (ns *NetworkServerImpl) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	body, e := marshalCastRewardTransSignMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignGot, Body: body}

	ns.net.SendWithGroupRelay(msg.Launcher.GetHexString(), msg.GroupID.GetHexString(), m)
}

// SendGroupPingMessage sends ping message to the given receiver
func (ns *NetworkServerImpl) SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID) {
	body, e := marshalCreateGroupPingMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPingMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPing, Body: body}

	ns.net.Send(receiver.GetHexString(), m)
}

// SendGroupPongMessage sends pong response to the group which the requester belongs to
func (ns *NetworkServerImpl) SendGroupPongMessage(msg *model.CreateGroupPongMessage, group *GroupBrief) {
	body, e := marshalCreateGroupPongMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPongMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPong, Body: body}

	mems := id2String(group.MemIds)

	ns.net.SpreadToGroup(group.Gid.GetHexString(), mems, m, msg.SI.DataHash.Bytes())
}

// ReqSharePiece requests share piece from the given id
func (ns *NetworkServerImpl) ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqSharePiece, Body: body}

	ns.net.Send(receiver.GetHexString(), m)
}

// ResponseSharePiece sends share piece to the requester
func (ns *NetworkServerImpl) ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceResponseMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceResponseMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseSharePiece, Body: body}

	ns.net.Send(receiver.GetHexString(), m)
}

// ReqProposalBlock request block body from the target
func (ns *NetworkServerImpl) ReqProposalBlock(msg *model.ReqProposalBlock, target string) {
	body, e := marshalReqProposalBlockMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalReqProposalBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqProposalBlock, Body: body}

	ns.net.Send(target, m)
}

// ResponseProposalBlock sends block body to the requester
func (ns *NetworkServerImpl) ResponseProposalBlock(msg *model.ResponseProposalBlock, target string) {

	body, e := marshalResponseProposalBlockMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalResponseProposalBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseProposalBlock, Body: body}

	ns.net.Send(target, m)
}

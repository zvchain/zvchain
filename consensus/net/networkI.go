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

// Package net implements the network message handling and transmission functions
package net

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
)

// MessageProcessor interface defines the process functions of all consensus messages
type MessageProcessor interface {
	// Whether the processor is ready for handling messages
	Ready() bool

	// Returns the miner id of the processor
	GetMinerID() groupsig.ID

	// OnMessageGroupInit receives new-group-info messages from parent nodes and starts the group formation process
	// That indicates the current node is chosen to be a member of the new group
	OnMessageGroupInit(msg *model.ConsensusGroupRawMessage)

	// OnMessageSharePiece handles sharepiece message received from other members
	// during the group formation process.
	OnMessageSharePiece(msg *model.ConsensusSharePieceMessage)

	// OnMessageSignPK handles group-related public key messages received from other members.
	// It simply stores the public key for future use
	OnMessageSignPK(msg *model.ConsensusSignPubKeyMessage)

	// OnMessageSignPKReq receives group-related public key request from other members and
	// responses own public key
	OnMessageSignPKReq(msg *model.ConsensusSignPubkeyReqMessage)

	// OnMessageGroupInited is a network-wide node processing function.
	// The entire network node receives a group of initialized completion messages from all of the members in the group
	// and when 51% of the same message received from the group members, the group will be added on chain
	OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage)

	// OnMessageCast handles the message from the proposer
	// Note that, if the pre-block of the block present int the message isn't on the blockchain, it will caches the message
	// and trigger it after the pre-block added on chain
	OnMessageCast(msg *model.ConsensusCastMessage)

	// OnMessageVerify handles the verification messages from other members in the group for a specified block proposal
	// Note that, it will cache the messages if the corresponding proposal message doesn't come yet and trigger them
	// as long as the condition met
	OnMessageVerify(msg *model.ConsensusVerifyMessage)

	// OnMessageCreateGroupRaw triggered when receives raw group-create message from other nodes of the parent group
	// It check and sign the group-create message for the requester
	//
	// Before the formation of the new group, the parent group needs to reach a consensus on the information of the new group
	// which transited by ConsensusCreateGroupRawMessage.
	OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage)

	// OnMessageCreateGroupSign receives sign message from other members after ConsensusCreateGroupRawMessage was sent
	// during the new-group-info consensus process
	OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage)

	// OnMessageCastRewardSignReq handles bonus transaction signature requests
	// It signs the message if and only if the block of the transaction already added on chain,
	// otherwise the message will be cached util the condition met
	OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage)

	// OnMessageCastRewardSign receives signed messages for the bonus transaction from group members
	// If threshold signature received and the group signature recovered successfully,
	// the node will submit the bonus transaction to the pool
	OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage)

	// OnMessageCreateGroupPing handles Ping request from parent nodes
	// It only happens when current node is chosen to join a new group
	OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage)

	// OnMessageCreateGroupPong handles Pong response from new group candidates
	// It only happens among the parent group nodes
	OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage)

	// OnMessageSharePieceReq receives share piece request from other members
	// It happens in the case that the current node didn't heard from the other part
	// during the piece-sharing with each other process.
	OnMessageSharePieceReq(msg *model.ReqSharePieceMessage)

	// OnMessageSharePieceResponse receives share piece message from other member after requesting
	OnMessageSharePieceResponse(msg *model.ResponseSharePieceMessage)

	// OnMessageReqProposalBlock handles block body request from the verify group members
	// It only happens in the proposal role and when the group signature generated by the verify-group
	OnMessageReqProposalBlock(msg *model.ReqProposalBlock, sourceID string)

	// OnMessageResponseProposalBlock handles block body response from proposal node
	// It only happens in the verify roles and after block body request to the proposal node
	// It will add the block on chain and then broadcast
	OnMessageResponseProposalBlock(msg *model.ResponseProposalBlock)
}

// GroupBrief represents the brief info of one group including group id and member ids
type GroupBrief struct {
	Gid    groupsig.ID
	MemIds []groupsig.ID
}

// NetworkServer defines some network transmission functions for various types of data.
type NetworkServer interface {

	// SendGroupInitMessage send group initialization message to the corresponding members
	// Note that the group net is unavailable currently, so p2p transmission is required
	SendGroupInitMessage(grm *model.ConsensusGroupRawMessage)

	// SendKeySharePiece transit the share piece to each other of the group members
	SendKeySharePiece(spm *model.ConsensusSharePieceMessage)

	// SendSignPubKey broadcast the message among the group members
	SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage)

	// BroadcastGroupInfo means group initialization completed and then issue the network-wide broadcast
	// It is slow and expensive
	BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage)

	// SendCastVerify happens at the proposal role.
	// It send the message contains the proposed-block to all of the members of the verify-group for the verification consensus
	SendCastVerify(ccm *model.ConsensusCastMessage, gb *GroupBrief, proveHashs []common.Hash)

	// SendVerifiedCast broadcast the signed message for specified block proposal among group members
	SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID)

	// BroadcastNewBlock means network-wide broadcast for the generated block.
	// Based on bandwidth and performance considerations, it only transits the block to all of the proposers and
	// the next verify-group
	BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief)

	// SendCreateGroupRawMessage sends the group-create raw message to other members of the group
	SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage)

	// SendCreateGroupSignMessage sends signed message for the group-create raw message to the requester
	SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage, parentGid groupsig.ID)

	// BuildGroupNet builds the group net in local for inter-group communication
	BuildGroupNet(groupIdentifier string, mems []groupsig.ID)

	// ReleaseGroupNet releases the group net in local
	ReleaseGroupNet(groupIdentifier string)

	// SendCastRewardSignReq sends bonus transaction sign request to other members of the group
	SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage)

	// SendCastRewardSign sends signed message of the bonus transaction to the requester by group relaying
	SendCastRewardSign(msg *model.CastRewardTransSignMessage)

	// AnswerSignPkMessage sends the group-related public key request to requester
	AnswerSignPkMessage(msg *model.ConsensusSignPubKeyMessage, receiver groupsig.ID)

	// AskSignPkMessage sends a request for group-related public key to the given receiver
	AskSignPkMessage(msg *model.ConsensusSignPubkeyReqMessage, receiver groupsig.ID)

	// SendGroupPingMessage sends ping message to the given receiver
	SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID)

	// SendGroupPongMessage sends pong response to the group which the requester belongs to
	SendGroupPongMessage(msg *model.CreateGroupPongMessage, group *GroupBrief)

	// ReqSharePiece requests share piece from the given id
	ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID)

	// ResponseSharePiece sends share piece to the requester
	ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID)

	// ReqProposalBlock request block body from the target
	ReqProposalBlock(msg *model.ReqProposalBlock, target string)

	// ResponseProposalBlock sends block body to the requester
	ResponseProposalBlock(msg *model.ResponseProposalBlock, target string)
}

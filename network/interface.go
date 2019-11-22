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

package network

const FullNodeVirtualGroupID = "full_node_virtual_group_id"

const (
	/********************** consensus message code ***********************
	*********************** range from 1 to 9999 ***********************
	**/

	//the following six messages are used for casting block
	CastVerifyMsg         uint32 = 1 // The proposal sends the proposal msg to the verifier
	VerifiedCastMsg       uint32 = 2 // The verifier sends the verified msg to the verifier group.
	CastRewardSignReq     uint32 = 3 // Verifier reward: the verifier sends the piece request msg to the other verifiers.
	CastRewardSignGot     uint32 = 4 // Verifier reward: the verifies sends the piece response msg to the other verifiers.
	ReqProposalBlock      uint32 = 5 // The verifies sends the request to the proposal to get the block
	ResponseProposalBlock uint32 = 6 // The proposal sends the response to the verifies to deliver the block

	/*********************** chain message code ***********************
	************************* range from 10000 to 19999 **************
	 */
	//The following four messages are used for block sync
	BlockInfoNotifyMsg uint32 = 10001
	ReqBlock           uint32 = 10002
	BlockResponseMsg   uint32 = 10003
	NewBlockMsg        uint32 = 10004

	//The following two messages are used for block fork processing
	ForkFindAncestorResponse uint32 = 10008
	ForkFindAncestorReq      uint32 = 10009
	ForkChainSliceReq        uint32 = 10013
	ForkChainSliceResponse   uint32 = 10014

	//The following three message are used for tx sync
	TxSyncNotify   uint32 = 10010
	TxSyncReq      uint32 = 10011
	TxSyncResponse uint32 = 10012
)

type Message struct {
	ChainID uint16

	ProtocolVersion uint16

	Code uint32

	Body []byte
}

type Conn struct {
	ID   string
	IP   string
	Port string
}

type MsgDigest []byte

type MsgHandler interface {
	Handle(sourceID string, msg Message) error
}

type Network interface {
	//Send message to the node which id represents.If self doesn't connect to the node,
	// resolve the kad net to find the node and then send the message
	Send(id string, msg Message) error

	//Broadcast the message among the group which self belongs to
	SpreadAmongGroup(groupID string, msg Message) error

	//SpreadToGroup Broadcast the message to the group which self do not belong to
	SpreadToGroup(groupID string, groupMembers []string, msg Message, digest MsgDigest) error

	//TransmitToNeighbor Send message to neighbor nodes
	TransmitToNeighbor(msg Message, blacklist []string) error

	//Broadcast Send the message to all nodes it connects to and the node which receive the message also broadcast the message to their neighbor once
	Broadcast(msg Message) error

	//ConnInfo Return all connections self has
	ConnInfo() []Conn

	//BuildGroupNet build group network
	BuildGroupNet(groupID string, members []string)

	//DissolveGroupNet dissolve group network
	DissolveGroupNet(groupID string)

	//BuildProposerGroupNet build proposer group network
	BuildProposerGroupNet(proposers []*Proposer)

	//AddProposers add proposers to proposer group network
	AddProposers(proposers []*Proposer)
}

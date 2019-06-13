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

	GroupPing       uint32 = 1
	GroupPong       uint32 = 2
	CreateGroupaRaw uint32 = 3
	CreateGroupSign uint32 = 4

	GroupInitMsg uint32 = 5

	KeyPieceMsg uint32 = 6

	SignPubkeyMsg uint32 = 7

	GroupInitDoneMsg uint32 = 8

	AskSignPkMsg    uint32 = 9
	AnswerSignPkMsg uint32 = 10

	ReqSharePiece      uint32 = 11
	ResponseSharePiece uint32 = 12

	CastVerifyMsg         uint32 = 14 // The proposal sends the proposal msg to the verifier
	VerifiedCastMsg       uint32 = 15 // The verifier sends the verified msg to the verifier group.
	CastRewardSignReq     uint32 = 16
	CastRewardSignGot     uint32 = 17
	ReqProposalBlock      uint32 = 19 // The verifies sends the request to the proposal to get the block
	ResponseProposalBlock uint32 = 20 // The proposal sends the response to the verifies to deliver the block

	/*********************** chain message code ***********************
	************************* range from 10000 to 19999 **************
	 */
	//The following four messages are used for block sync
	BlockInfoNotifyMsg uint32 = 10001
	ReqBlock           uint32 = 10002
	BlockResponseMsg   uint32 = 10003
	NewBlockMsg        uint32 = 10004

	//The following three messages are used for group sync
	GroupChainCountMsg uint32 = 10005
	ReqGroupMsg        uint32 = 10006
	GroupMsg           uint32 = 10007

	//The following two messages are used for block fork processing
	ReqChainPieceBlock uint32 = 10008
	ChainPieceBlock    uint32 = 10009

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

	//Send message to the node which id represents. If self doesn't connect to the node,
	// send message to the guys which belongs to the same group with the node and they will rely the message to the node
	SendWithGroupRelay(id string, groupID string, msg Message) error

	//Random broadcast the message to parts nodes in the group which self belongs to
	RandomSpreadInGroup(groupID string, msg Message) error

	//Broadcast the message among the group which self belongs to
	SpreadAmongGroup(groupID string, msg Message) error

	//SpreadToRandomGroupMember send message to random memebers which in special group
	SpreadToRandomGroupMember(groupID string, groupMembers []string, msg Message) error

	//SpreadToGroup Broadcast the message to the group which self do not belong to
	SpreadToGroup(groupID string, groupMembers []string, msg Message, digest MsgDigest) error

	//TransmitToNeighbor Send message to neighbor nodes
	TransmitToNeighbor(msg Message) error

	//Relay Send the message to part nodes it connects to and they will also broadcast the message to part of their neighbor util relayCount
	Relay(msg Message, relayCount int32) error

	//Broadcast Send the message to all nodes it connects to and the node which receive the message also broadcast the message to their neighbor once
	Broadcast(msg Message) error

	//ConnInfo Return all connections self has
	ConnInfo() []Conn

	//BuildGroupNet build group network
	BuildGroupNet(groupID string, members []string)

	//DissolveGroupNet dissolve group network
	DissolveGroupNet(groupID string)
}

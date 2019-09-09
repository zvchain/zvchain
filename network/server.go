//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you cas redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either versios 3 of the License, or
//   (at your option) any later versios.
//
//   This program is distributed is the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without eves the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"github.com/golang/protobuf/proto"

	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/statistics"

	"golang.org/x/crypto/sha3"
)

type Server struct {
	Self *Node

	netCore *NetCore
	config  *NetworkConfig

	consensusHandler MsgHandler
}

func (s *Server) Send(id string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		return err
	}
	if id == s.Self.ID.GetHexString() {
		s.sendSelf(bytes)
		return nil
	}
	nID := NewNodeID(id)
	if nID != nil {
		go s.netCore.sendToNode(*nID, nil, bytes, msg.Code)
	}

	return nil
}

func (s *Server) SpreadAmongGroup(groupID string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	Logger.Infof("SpreadAmongGroup :%s,code:%d,msg size:%d", groupID, msg.Code, len(msg.Body)+4)
	s.netCore.groupBroadcast(groupID, bytes, msg.Code, true, -1)

	return nil
}

func (s *Server) SpreadToGroup(groupID string, groupMembers []string, msg Message, digest MsgDigest) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	Logger.Infof("SpreadToGroup :%s,code:%d,groupMembers:%v msg size:%d", groupID, msg.Code, len(groupMembers), len(msg.Body)+4)
	s.netCore.groupBroadcastWithMembers(groupID, bytes, msg.Code, digest, groupMembers, -1)

	return nil
}

func (s *Server) TransmitToNeighbor(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.broadcast(bytes, msg.Code, false, nil, -1)

	return nil
}

func (s *Server) Broadcast(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}
	s.netCore.broadcast(bytes, msg.Code, true, nil, -1)

	return nil
}

func (s *Server) ConnInfo() []Conn {
	return s.netCore.peerManager.ConnInfo()
}

func (s *Server) BuildGroupNet(groupID string, members []string) {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nID := NewNodeID(id)
		if nID != nil {
			nodes = append(nodes, *nID)
		}

	}
	s.netCore.groupManager.buildGroup(groupID, nodes)
}

func (s *Server) DissolveGroupNet(groupID string) {
	s.netCore.groupManager.removeGroup(groupID)
}

func (s *Server) AddGroup(groupID string, members []string) *Group {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nID := NewNodeID(id)
		if nID != nil {
			nodes = append(nodes, *nID)
		}
	}
	return s.netCore.groupManager.buildGroup(groupID, nodes)
}

func (s *Server) BuildProposerGroupNet(proposers []*Proposer) {
	s.netCore.proposerManager.Build(proposers)
}

func (s *Server) AddProposers(proposers []*Proposer) {
	s.netCore.proposerManager.AddProposers(proposers)
}

func (s *Server) RemoveGroup(ID string) {
	s.netCore.groupManager.removeGroup(ID)
}

func (s *Server) sendSelf(b []byte) {
	s.handleMessage(b, s.Self.ID.GetHexString(), s.netCore.chainID, s.netCore.protocolVersion)
}

func (s *Server) handleMessage(b []byte, from string, chainID uint16, protocolVersion uint16) {

	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}
	message.ChainID = chainID
	message.ProtocolVersion = protocolVersion
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s, chainID:%v,protocolVersion:%v", from, message.Code, len(b), message.Hash(), chainID, protocolVersion)
	statistics.AddCount("Server.handleMessage", message.Code, uint64(len(b)))
	s.netCore.flowMeter.recv(int64(message.Code), int64(len(b)))

	go s.handleMessageInner(message, from)
}

func newNotifyMessage(message *Message, from string) *notify.DefaultMessage {
	return notify.NewDefaultMessage(message.Body, from, message.ChainID, message.ProtocolVersion)
}

func (s *Server) handleMessageInner(message *Message, from string) {

	s.netCore.onHandleDataMessageStart()
	defer s.netCore.onHandleDataMessageDone()

	begin := time.Now()
	code := message.Code

	if code < 10000 {
		err := s.consensusHandler.Handle(from, *message)
		if err != nil {
			Logger.Errorf("consensusHandler handle error:%s", err.Error())
		}
	} else {
		topicID := ""
		switch code {
		case TxSyncNotify:
			topicID = notify.TxSyncNotify
		case TxSyncReq:
			topicID = notify.TxSyncReq
		case TxSyncResponse:
			topicID = notify.TxSyncResponse
		case BlockInfoNotifyMsg:
			topicID = notify.BlockInfoNotify
		case ReqBlock:
			topicID = notify.BlockReq
		case BlockResponseMsg:
			topicID = notify.BlockResponse
		case NewBlockMsg:
			topicID = notify.NewBlock
		case ForkFindAncestorResponse:
			topicID = notify.ForkFindAncestorResponse
		case ForkFindAncestorReq:
			topicID = notify.ForkFindAncestorReq
		case ForkChainSliceReq:
			topicID = notify.ForkChainSliceReq
		case ForkChainSliceResponse:
			topicID = notify.ForkChainSliceResponse
		}
		if topicID != "" {
			msg := newNotifyMessage(message, from)
			notify.BUS.PublishWithRecover(topicID, msg)
		}
	}

	if time.Since(begin) > 300*time.Millisecond {
		Logger.Infof("handle message cost time:%v,hash:%s,code:%d", time.Since(begin), message.Hash(), code)
	}
}

func marshalMessage(m Message) ([]byte, error) {
	message := tas_middleware_pb.Message{Code: &m.Code, Body: m.Body}
	return proto.Marshal(&message)
}

func unMarshalMessage(b []byte) (*Message, error) {
	message := new(tas_middleware_pb.Message)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	m := Message{Code: *message.Code, Body: message.Body}
	return &m, nil
}

func (m Message) Hash() string {
	bytes, err := marshalMessage(m)
	if err != nil {
		return ""
	}

	var h common.Hash
	sha3Hash := sha3.Sum256(bytes)
	copy(h[:], sha3Hash[:])

	return h.Hex()
}

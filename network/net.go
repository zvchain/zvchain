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

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	nnet "net"
	"time"

	"github.com/zvchain/zvchain/middleware/statistics"

	"github.com/gogo/protobuf/proto"
)

// Version is p2p proto version
const Version = 1

const (
	PacketTypeSize           = 4
	PacketLenSize            = 4
	PacketHeadSize           = PacketTypeSize + PacketLenSize
	MaxUnhandledMessageCount = 10000
	P2PMessageCodeBase       = 10000
)

// Errors
var (
	errPacketTooSmall   = errors.New("too small")
	errBadPacket        = errors.New("bad Packet")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errGroupEmpty       = errors.New("group empty")
	errTimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
)

const DefaultNatPort = 3200
const DefaultNatIP = "120.78.127.246"

// Timeouts
const (
	respTimeout              = 500 * time.Millisecond
	clearMessageCacheTimeout = time.Minute
	expiration               = 60 * time.Second
	connectTimeout           = 3 * time.Second
	groupRefreshInterval     = 5 * time.Second
	flowMeterInterval        = 1 * time.Minute
)

// NetCore p2p network
type NetCore struct {
	ourEndPoint      RpcEndPoint
	id               NodeID
	nid              uint64
	natType          uint32
	addpending       chan *pending
	gotreply         chan reply
	unhandled        chan *Peer
	unhandledDataMsg int
	closing          chan struct{}

	kad            *Kad
	peerManager    *PeerManager
	groupManager   *GroupManager
	messageManager *MessageManager
	flowMeter      *FlowMeter
	bufferPool     *BufferPool

	chainID         uint16 // Chain id
	protocolVersion uint16 // Protocol id
}

type pending struct {
	from  NodeID
	ptype MessageType

	deadline time.Time

	callback func(resp interface{}) (done bool)

	errc chan<- error
}

type reply struct {
	from    NodeID
	ptype   MessageType
	data    interface{}
	matched chan<- bool
}

func genNetID(id NodeID) uint64 {
	h := fnv.New64a()
	n, err := h.Write(id[:])
	if n == 0 || err != nil {
		return 0
	}
	return uint64(h.Sum64())
}

//nodeFromRPC is RPC node transformation
func (nc *NetCore) nodeFromRPC(sender *nnet.UDPAddr, rn RpcNode) (*Node, error) {
	if rn.Port <= 1024 {
		return nil, errors.New("low port")
	}

	n := NewNode(NewNodeID(rn.Id), nnet.ParseIP(rn.Ip), int(rn.Port))

	err := n.validateComplete()
	return n, err
}

var netCore *NetCore

//NetCoreConfig net core configure
type NetCoreConfig struct {
	ListenAddr         *nnet.UDPAddr
	ID                 NodeID
	Seeds              []*Node
	NatTraversalEnable bool

	NatPort         uint16
	NatIP           string
	ChainID         uint16
	ProtocolVersion uint16
}

// MakeEndPoint create the node description object
func MakeEndPoint(addr *nnet.UDPAddr, tcpPort int32) RpcEndPoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return RpcEndPoint{Ip: ip.String(), Port: int32(addr.Port)}
}

func nodeToRPC(n *Node) RpcNode {
	return RpcNode{Id: n.ID.GetHexString(), Ip: n.IP.String(), Port: int32(n.Port)}
}

// InitNetCore For initialization
func (nc *NetCore) InitNetCore(cfg NetCoreConfig) (*NetCore, error) {

	nc.id = cfg.ID
	nc.closing = make(chan struct{})
	nc.gotreply = make(chan reply)
	nc.addpending = make(chan *pending)
	nc.unhandled = make(chan *Peer, 64)
	nc.nid = genNetID(cfg.ID)
	nc.chainID = cfg.ChainID
	nc.protocolVersion = cfg.ProtocolVersion
	nc.peerManager = newPeerManager()
	nc.peerManager.natTraversalEnable = cfg.NatTraversalEnable
	nc.peerManager.natIP = cfg.NatIP
	nc.peerManager.natPort = cfg.NatPort
	if len(nc.peerManager.natIP) == 0 {
		nc.peerManager.natIP = DefaultNatIP
	}
	if nc.peerManager.natPort == 0 {
		nc.peerManager.natPort = DefaultNatPort
	}
	nc.groupManager = newGroupManager()
	nc.messageManager = newMessageManager(nc.id)
	nc.flowMeter = newFlowMeter("p2p")
	nc.bufferPool = newBufferPool()
	realaddr := cfg.ListenAddr

	Logger.Infof("kad id: %v ", nc.id.GetHexString())
	Logger.Infof("chain id: %v ", nc.chainID)
	Logger.Infof("protocol version : %v ", nc.protocolVersion)
	Logger.Infof("P2PConfig: %v ", nc.nid)
	P2PConfig(nc.nid)

	if cfg.NatTraversalEnable {
		Logger.Infof("P2PProxy: %v %v", nc.peerManager.natIP, uint16(nc.peerManager.natPort))
		P2PProxy(nc.peerManager.natIP, uint16(nc.peerManager.natPort))
	} else {
		Logger.Infof("P2PListen: %v %v", realaddr.IP.String(), uint16(realaddr.Port))
		P2PListen(realaddr.IP.String(), uint16(realaddr.Port))
	}

	nc.ourEndPoint = MakeEndPoint(realaddr, int32(realaddr.Port))
	kad, err := newKad(nc, cfg.ID, realaddr, cfg.Seeds)
	if err != nil {
		return nil, err
	}
	nc.kad = kad
	netCore = nc
	go nc.loop()
	go nc.decodeLoop()

	return nc, nil
}

func (nc *NetCore) close() {
	P2PClose()
	close(nc.closing)
}

func (nc *NetCore) buildGroup(id string, members []NodeID) *Group {
	return nc.groupManager.buildGroup(id, members)
}

func (nc *NetCore) ping(toid NodeID, toaddr *nnet.UDPAddr) {

	to := MakeEndPoint(&nnet.UDPAddr{}, 0)
	if toaddr != nil {
		to = MakeEndPoint(toaddr, 0)
	}
	req := &MsgPing{
		Version:    Version,
		From:       &nc.ourEndPoint,
		To:         &to,
		NodeId:     nc.id[:],
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	Logger.Infof("[send ping ] id : %v  ip:%v port:%v", toid.GetHexString(), nc.ourEndPoint.Ip, nc.ourEndPoint.Port)

	packet, _, err := nc.encodePacket(MessageType_MessagePing, req)
	if err != nil {
		return
	}

	nc.peerManager.write(toid, toaddr, packet, P2PMessageCodeBase+uint32(MessageType_MessagePing), false)
}

func (nc *NetCore) RelayTest(toid NodeID) {

	req := &MsgRelay{
		NodeId: toid[:],
	}
	Logger.Infof("[Relay] test node id : %v", toid.GetHexString())

	packet, _, err := nc.encodePacket(MessageType_MessageRelayTest, req)
	if err != nil {
		return
	}

	nc.peerManager.broadcast(packet, P2PMessageCodeBase+uint32(MessageType_MessagePing))
}

func (nc *NetCore) findNode(toid NodeID, toaddr *nnet.UDPAddr, target NodeID) ([]*Node, error) {
	nodes := make([]*Node, 0, bucketSize)
	errc := nc.pending(toid, MessageType_MessageNeighbors, func(r interface{}) bool {
		nreceived := 0
		reply := r.(*MsgNeighbors)
		for _, rn := range reply.Nodes {
			n, err := nc.nodeFromRPC(toaddr, *rn)
			if err != nil {
				continue
			}
			nreceived++

			nodes = append(nodes, n)
		}

		return nreceived >= bucketSize
	})
	nc.sendMessageToNode(toid, toaddr, MessageType_MessageFindnode, &MsgFindNode{
		Target:     target[:],
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}, P2PMessageCodeBase+uint32(MessageType_MessageFindnode))
	err := <-errc

	return nodes, err
}

func (nc *NetCore) pending(id NodeID, ptype MessageType, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	select {
	case nc.addpending <- p:
	case <-nc.closing:
		ch <- errClosed
	}
	return ch
}

func (nc *NetCore) handleReply(from NodeID, ptype MessageType, req interface{}) bool {
	matched := make(chan bool, 1)
	select {
	case nc.gotreply <- reply{from, ptype, req, matched}:
		// loop will handle it
		return <-matched
	case <-nc.closing:
		return false
	}
}

func (nc *NetCore) decodeLoop() {

	for {
		select {
		case peer := <-nc.unhandled:
			for {
				err := nc.handleMessage(peer)
				if err != nil || peer.isEmpty() {
					break
				}
			}
		}
	}
}

func (nc *NetCore) loop() {
	var (
		plist             = list.New()
		clearMessageCache = time.NewTicker(clearMessageCacheTimeout)
		flowMeter         = time.NewTicker(flowMeterInterval)
		groupRefresh      = time.NewTicker(groupRefreshInterval)
		timeout           = time.NewTimer(0)
		nextTimeout       *pending
		contTimeouts      = 0
	)
	defer clearMessageCache.Stop()
	defer groupRefresh.Stop()
	defer timeout.Stop()
	defer flowMeter.Stop()

	// ignore first timeout
	<-timeout.C

	resetTimeout := func() {
		if plist.Front() == nil || nextTimeout == plist.Front().Value {
			return
		}
		now := time.Now()
		for el := plist.Front(); el != nil; el = el.Next() {
			nextTimeout = el.Value.(*pending)
			if dist := nextTimeout.deadline.Sub(now); dist < 2*respTimeout {
				timeout.Reset(dist)
				return
			}

			nextTimeout.errc <- errClockWarp
			plist.Remove(el)
		}
		nextTimeout = nil
		timeout.Stop()
	}

	for {
		resetTimeout()

		select {
		case <-nc.closing:
			for el := plist.Front(); el != nil; el = el.Next() {
				el.Value.(*pending).errc <- errClosed
			}

			return
		case p := <-nc.addpending:
			p.deadline = time.Now().Add(respTimeout)
			plist.PushBack(p)
		case now := <-timeout.C:
			nextTimeout = nil
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if now.After(p.deadline) || now.Equal(p.deadline) {
					p.errc <- errTimeout
					plist.Remove(el)
					contTimeouts++
				}
			}

		case r := <-nc.gotreply:
			var matched bool
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if p.from == r.from && p.ptype == r.ptype {
					matched = true
					if p.callback(r.data) {
						p.errc <- nil
						plist.Remove(el)
					}
					contTimeouts = 0
				}
			}
			r.matched <- matched
		case <-clearMessageCache.C:
			nc.messageManager.clear()
		case <-flowMeter.C:
			nc.flowMeter.print()
			nc.flowMeter.reset()
			nc.peerManager.checkPeers()

		case <-groupRefresh.C:
			go nc.groupManager.doRefresh()
		}
	}
}

func init() {

}

func (nc *NetCore) sendToNode(toid NodeID, toaddr *nnet.UDPAddr, data []byte, code uint32) {
	packet, _, err := nc.encodeDataPacket(data, DataType_DataNormal, code, "", &toid, nil, -1)
	if err != nil {
		Logger.Debugf("Send encodeDataPacket err :%v ", toid.GetHexString())
		return
	}
	nc.peerManager.write(toid, toaddr, packet, code, true)
	nc.bufferPool.freeBuffer(packet)
}

func (nc *NetCore) sendMessageToNode(toid NodeID, toaddr *nnet.UDPAddr, ptype MessageType, req proto.Message, code uint32) {
	packet, _, err := nc.encodePacket(ptype, req)
	if err != nil {
		return
	}
	nc.peerManager.write(toid, toaddr, packet, code, false)
	nc.bufferPool.freeBuffer(packet)
}

func (nc *NetCore) broadcast(data []byte, code uint32, broadcast bool, msgDigest MsgDigest, relayCount int32) {

	dataType := DataType_DataNormal
	if broadcast {
		dataType = DataType_DataGlobal
	}
	packet, _, err := nc.encodeDataPacket(data, dataType, code, "", nil, msgDigest, relayCount)
	if err != nil {
		return
	}
	nc.peerManager.broadcast(packet, code)
	nc.bufferPool.freeBuffer(packet)
	return
}

func (nc *NetCore) broadcastRandom(data []byte, code uint32, relayCount int32) {
	dataType := DataType_DataGlobalRandom

	packet, _, err := nc.encodeDataPacket(data, dataType, code, "", nil, nil, relayCount)
	if err != nil {
		return
	}
	nc.peerManager.broadcastRandom(packet, code)
	nc.bufferPool.freeBuffer(packet)
	return
}

func (nc *NetCore) groupBroadcast(id string, data []byte, code uint32, broadcast bool, relayCount int32) {
	dataType := DataType_DataNormal
	if broadcast {
		dataType = DataType_DataGroup
	}
	packet, _, err := nc.encodeDataPacket(data, dataType, code, id, nil, nil, relayCount)
	if err != nil {
		return
	}
	nc.groupManager.groupBroadcast(id, packet, code)
	nc.bufferPool.freeBuffer(packet)
	return
}

func (nc *NetCore) groupBroadcastWithMembers(id string, data []byte, code uint32, msgDigest MsgDigest, groupMembers []string, relayCount int32) {
	dataType := DataType_DataGroup

	packet, _, err := nc.encodeDataPacket(data, dataType, code, id, nil, msgDigest, relayCount)
	if err != nil {
		return
	}
	const MaxSendCount = 1
	nodesHasSend := make(map[NodeID]bool)
	count := 0
	// Find the one that's already connected
	for i := 0; i < len(groupMembers) && count < MaxSendCount; i++ {
		id := NewNodeID(groupMembers[i])
		p := nc.peerManager.peerByID(id)
		if p != nil && p.sessionID > 0 {
			count++
			nodesHasSend[id] = true
			nc.peerManager.write(id, nil, packet, code, false)
		}
	}

	// The number of connections has not been enough, through the penetration server connection
	for i := 0; i < len(groupMembers) && count < MaxSendCount && count < len(groupMembers); i++ {
		id := NewNodeID(groupMembers[i])
		if nodesHasSend[id] != true && id != nc.id {
			count++
			nc.peerManager.write(id, nil, packet, code, false)
		}
	}

	nc.bufferPool.freeBuffer(packet)
	return
}

func (nc *NetCore) sendToGroupMember(id string, data []byte, code uint32, memberID NodeID) {

	p := nc.peerManager.peerByID(memberID)
	if (p != nil && p.sessionID > 0) || nc.peerManager.natTraversalEnable {
		go nc.sendToNode(memberID, nil, data, code)
	} else {
		node := net.netCore.kad.find(memberID)
		if node != nil && node.IP != nil && node.Port > 0 {
			go nc.sendToNode(memberID, &nnet.UDPAddr{IP: node.IP, Port: int(node.Port)}, data, code)
		} else {

			packet, _, err := nc.encodeDataPacket(data, DataType_DataGroup, code, id, &memberID, nil, -1)
			if err != nil {
				return
			}

			nc.groupManager.groupBroadcast(id, packet, code)
			nc.bufferPool.freeBuffer(packet)
		}
	}
	return
}

// OnConnected callback when a peer is connected
func (nc *NetCore) onConnected(id uint64, session uint32, p2pType uint32) {
	nc.peerManager.newConnection(id, session, p2pType, false)
}

// OnAccepted callback when Accepted a  peer
func (nc *NetCore) onAccepted(id uint64, session uint32, p2pType uint32) {
	nc.peerManager.newConnection(id, session, p2pType, true)
}

// OnDisconnected callback when peer is disconnected
func (nc *NetCore) onDisconnected(id uint64, session uint32, p2pCode uint32) {
	nc.peerManager.onDisconnected(id, session, p2pCode)
}

// OnSendWaited callback when send list is clear
func (nc *NetCore) onSendWaited(id uint64, session uint32) {
	nc.peerManager.onSendWaited(id, session)
}

// OnChecked callback when nat type is checked
func (nc *NetCore) onChecked(p2pType uint32, privateIP string, publicIP string) {
	nc.ourEndPoint = MakeEndPoint(&nnet.UDPAddr{IP: nnet.ParseIP(publicIP), Port: 0}, 0)
	nc.natType = p2pType
	nc.peerManager.onChecked(p2pType, privateIP, publicIP)
	Logger.Debugf("OnChecked, nat type :%v public ip: %v private ip :%v", p2pType, publicIP, privateIP)
}

// OnRecved callback when data is received
func (nc *NetCore) onRecved(netID uint64, session uint32, data []byte) {
	nc.recvData(netID, session, data)
}

func (nc *NetCore) recvData(netID uint64, session uint32, data []byte) {

	p := nc.peerManager.peerByNetID(netID)
	if p == nil {
		p = newPeer(NodeID{}, session)
		nc.peerManager.addPeer(netID, p)
	}

	p.addRecvData(data)
	nc.unhandled <- p
}

func (nc *NetCore) encodeDataPacket(data []byte, dataType DataType, code uint32, groupID string, nodeID *NodeID, msgDigest MsgDigest, relayCount int32) (msg *bytes.Buffer, hash []byte, err error) {
	nodeIDBytes := make([]byte, 0)
	if nodeID != nil {
		nodeIDBytes = nodeID.Bytes()
	}
	bizMessageIDBytes := make([]byte, 0)
	if msgDigest != nil {
		bizMessageIDBytes = msgDigest[:]
	}
	msgData := &MsgData{
		Data:         data,
		DataType:     dataType,
		GroupId:      groupID,
		MessageId:    nc.messageManager.genMessageID(),
		MessageCode:  uint32(code),
		DestNodeId:   nodeIDBytes,
		SrcNodeId:    nc.id.Bytes(),
		BizMessageId: bizMessageIDBytes,
		RelayCount:   relayCount,
		MessageInfo:  encodeMessageInfo(nc.chainID, nc.protocolVersion),
		Expiration:   uint64(time.Now().Add(expiration).Unix())}
	Logger.Debugf("encodeDataPacket  DataType:%v messageId:%X ,BizMessageID:%v ,RelayCount:%v code:%v", msgData.DataType, msgData.MessageId, msgData.BizMessageId, msgData.RelayCount, code)

	return nc.encodePacket(MessageType_MessageData, msgData)
}

func (nc *NetCore) encodePacket(ptype MessageType, req proto.Message) (msg *bytes.Buffer, hash []byte, err error) {

	pdata, err := proto.Marshal(req)
	if err != nil {
		return nil, nil, err
	}
	length := len(pdata)
	b := nc.bufferPool.getBuffer(length + PacketHeadSize)

	err = binary.Write(b, binary.BigEndian, uint32(ptype))
	if err != nil {
		return nil, nil, err
	}
	err = binary.Write(b, binary.BigEndian, uint32(length))
	if err != nil {
		return nil, nil, err
	}

	b.Write(pdata)
	return b, nil, nil
}

func (nc *NetCore) handleMessage(p *Peer) error {
	if p == nil || p.isEmpty() {
		return nil
	}
	msgType, packetSize, msg, buf, err := nc.decodePacket(p)

	if err != nil {
		return err
	}
	fromID := p.ID

	switch msgType {
	case MessageType_MessagePing:
		fromID.SetBytes(msg.(*MsgPing).NodeId)
		if fromID != p.ID {
			p.ID = fromID
		}
		err = nc.handlePing(msg.(*MsgPing), fromID)
	case MessageType_MessageFindnode:
		err = nc.handleFindNode(msg.(*MsgFindNode), fromID)
	case MessageType_MessageNeighbors:
		err = nc.handleNeighbors(msg.(*MsgNeighbors), fromID)
	case MessageType_MessageRelayTest:
		err = nc.handleRelayTest(msg.(*MsgRelay), fromID)
	case MessageType_MessageRelayNode:
		err = nc.handleRelayNode(msg.(*MsgRelay), fromID)
	case MessageType_MessageData:
		nc.handleData(msg.(*MsgData), buf.Bytes()[0:packetSize], fromID)
	default:
		return Logger.Errorf("unknown type: %d", msgType)
	}
	if buf != nil {
		nc.bufferPool.freeBuffer(buf)
	}
	return err
}

func (nc *NetCore) decodePacket(p *Peer) (MessageType, int, proto.Message, *bytes.Buffer, error) {

	header := p.popData()
	if header == nil {
		return MessageType_MessageNone, 0, nil, nil, errPacketTooSmall
	}

	for header.Len() < PacketHeadSize && !p.isEmpty() {
		b := p.popData()
		if b != nil && b.Len() > 0 {
			netCore.bufferPool.freeBuffer(b)
		}
	}
	if header.Len() < PacketHeadSize {
		p.addRecvDataToHead(header)
		return MessageType_MessageNone, 0, nil, nil, errPacketTooSmall
	}

	headerBytes := header.Bytes()
	msgType := MessageType(binary.BigEndian.Uint32(headerBytes[:PacketTypeSize]))
	msgLen := binary.BigEndian.Uint32(headerBytes[PacketTypeSize:PacketHeadSize])
	packetSize := int(msgLen + PacketHeadSize)

	Logger.Debugf("[ decodePacket ] session : %v packetSize: %v  msgType: %v  msgLen:%v   bufSize:%v buffer address:%p ", p.sessionID, packetSize, msgType, msgLen, header.Len(), header)

	const MaxPacketSize = 16 * 1024 * 1024

	if packetSize > MaxPacketSize || packetSize <= 0 {
		Logger.Infof("[ decodePacket ] session : %v bad packet reset data!", p.sessionID)
		p.resetData()
		return MessageType_MessageNone, 0, nil, nil, errBadPacket
	}

	msgBuffer := header

	if msgBuffer.Cap() < packetSize {
		msgBuffer = nc.bufferPool.getBuffer(packetSize)
		msgBuffer.Write(headerBytes)

	}
	for msgBuffer.Len() < packetSize && !p.isEmpty() {
		b := p.popData()
		if b != nil && b.Len() > 0 {
			msgBuffer.Write(b.Bytes())
			netCore.bufferPool.freeBuffer(b)
		}
	}
	if msgBuffer.Len() < packetSize {
		p.addRecvDataToHead(msgBuffer)
		return MessageType_MessageNone, 0, nil, nil, errPacketTooSmall
	}
	msgBytes := msgBuffer.Bytes()

	data := msgBytes[PacketHeadSize : PacketHeadSize+msgLen]

	if msgBuffer.Len() > packetSize {
		buf := nc.bufferPool.getBuffer(len(msgBytes) - packetSize)
		buf.Write(msgBytes[packetSize:])
		p.addRecvDataToHead(buf)
	}

	var req proto.Message
	switch msgType {
	case MessageType_MessagePing:
		req = new(MsgPing)
	case MessageType_MessageFindnode:
		req = new(MsgFindNode)
	case MessageType_MessageNeighbors:
		req = new(MsgNeighbors)
	case MessageType_MessageData:
		req = new(MsgData)
	case MessageType_MessageRelayTest:
		req = new(MsgRelay)
	case MessageType_MessageRelayNode:
		req = new(MsgRelay)
	default:
		return msgType, packetSize, nil, msgBuffer, fmt.Errorf("unknown type: %d", msgType)
	}

	err := proto.Unmarshal(data, req)

	if msgType != MessageType_MessageData {
		nc.flowMeter.recv(P2PMessageCodeBase+int64(msgType), int64(packetSize))
	}

	return msgType, packetSize, req, msgBuffer, err
}

func (nc *NetCore) handlePing(req *MsgPing, fromID NodeID) error {

	if expired(req.Expiration) {
		return errExpired
	}

	p := nc.peerManager.peerByID(fromID)
	ip := nnet.ParseIP(req.From.Ip)
	port := int(req.From.Port)
	if p != nil {
		if ip != nil && port > 0 {
			p.IP = ip
			p.Port = port
		}
	}
	from := nnet.UDPAddr{IP: nnet.ParseIP(req.From.Ip), Port: int(req.From.Port)}

	Logger.Debugf("ping from:%v id:%v port:%d", fromID.GetHexString(), ip, port)

	if !nc.handleReply(fromID, MessageType_MessagePing, req) {
		_, err := nc.kad.onPingNode(fromID, &from)
		if err != nil {
			return err
		}
	}

	if p != nil && !p.isPinged {
		netCore.ping(fromID, nil)
		p.isPinged = true
	}

	return nil
}

func (nc *NetCore) handleFindNode(req *MsgFindNode, fromID NodeID) error {

	if expired(req.Expiration) {
		return errExpired
	}

	target := req.Target
	nc.kad.mutex.Lock()
	closest := nc.kad.closest(target, bucketSize).entries
	nc.kad.mutex.Unlock()

	p := MsgNeighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}

	for _, n := range closest {
		node := nodeToRPC(n)
		if len(node.Ip) > 0 && node.Port > 0 {
			p.Nodes = append(p.Nodes, &node)
		}
	}
	if len(p.Nodes) > 0 {
		nc.sendMessageToNode(fromID, nil, MessageType_MessageNeighbors, &p, P2PMessageCodeBase+uint32(MessageType_MessageNeighbors))
	}

	return nil
}
func (nc *NetCore) handleRelayTest(req *MsgRelay, fromID NodeID) error {
	TestNodeID := NodeID{}
	TestNodeID.SetBytes(req.NodeId)
	TestPeer := nc.peerManager.peerByID(TestNodeID)

	if TestPeer != nil && TestPeer.sessionID > 0 {
		nc.sendMessageToNode(fromID, nil, MessageType_MessageRelayNode, req, P2PMessageCodeBase+uint32(MessageType_MessageRelayNode))
		Logger.Infof("[Relay] handle relay test YES, test node id : %v", TestNodeID.GetHexString())
	} else {
		Logger.Infof("[Relay] handle relay test NO, test node id : %v", TestNodeID.GetHexString())

	}

	return nil
}

func (nc *NetCore) handleRelayNode(req *MsgRelay, fromID NodeID) error {
	TestNodeID := NodeID{}
	TestNodeID.SetBytes(req.NodeId)
	TestPeer := nc.peerManager.peerByID(TestNodeID)

	if TestPeer != nil {
		TestPeer.relayID = fromID
	}

	Logger.Infof("[Relay] handle relay node, test  node id : %v", TestNodeID.GetHexString())
	return nil
}

func (nc *NetCore) handleNeighbors(req *MsgNeighbors, fromID NodeID) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.handleReply(fromID, MessageType_MessageNeighbors, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (nc *NetCore) handleData(req *MsgData, packet []byte, fromID NodeID) {
	srcNodeID := NodeID{}
	srcNodeID.SetBytes(req.SrcNodeId)
	dstNodeID := NodeID{}
	dstNodeID.SetBytes(req.DestNodeId)

	Logger.Debugf("data from:%v  len:%v DataType:%v messageId:%X ,BizMessageID:%v ,RelayCount:%v  unhandleDataMsg:%v code:%v messageInfo:%v", srcNodeID, len(req.Data), req.DataType, req.MessageId, req.BizMessageId, req.RelayCount, nc.unhandledDataMsg, req.MessageCode, req.MessageInfo)

	statistics.AddCount("net.handleData", uint32(req.DataType), uint64(len(req.Data)))
	if req.DataType == DataType_DataNormal {
		if dstNodeID.IsValid() && dstNodeID != nc.id {
			var dataBuffer = nc.bufferPool.getBuffer(len(packet))
			dataBuffer.Write(packet)

			Logger.Debugf("[Relay]Relay message DataType:%v messageId:%X DestNodeId：%v SrcNodeId：%v RelayCount:%v", req.DataType, req.MessageId, dstNodeID.GetHexString(), srcNodeID.GetHexString(), req.RelayCount)

			nc.peerManager.write(dstNodeID, nil, dataBuffer, uint32(req.MessageCode), false)

		} else {
			nc.onHandleDataMessage(req, srcNodeID)
		}
		return
	}

	if expired(req.Expiration) {
		Logger.Infof("message expired!")
		return
	}

	forwarded := false

	if req.BizMessageId != nil {
		bizID := nc.messageManager.byteToBizID(req.BizMessageId)
		forwarded = nc.messageManager.isForwardedBiz(bizID)

	} else {
		forwarded = nc.messageManager.isForwarded(req.MessageId)
	}

	if forwarded {
		return
	}

	nc.messageManager.forward(req.MessageId)
	if req.BizMessageId != nil {
		bizID := nc.messageManager.byteToBizID(req.BizMessageId)
		nc.messageManager.forwardBiz(bizID)
	}
	// Need to deal with
	if len(req.DestNodeId) == 0 || dstNodeID == nc.id {
		nc.onHandleDataMessage(req, srcNodeID)
	}
	broadcast := false
	// Need to be broadcast
	if len(req.DestNodeId) == 0 || dstNodeID != nc.id {
		broadcast = true
	}

	if req.RelayCount == 0 {
		broadcast = false
	}
	if broadcast {
		var dataBuffer *bytes.Buffer
		if req.RelayCount > 0 {
			req.RelayCount = req.RelayCount - 1
			dataBuffer, _, _ = nc.encodePacket(MessageType_MessageData, req)
		} else {
			dataBuffer = nc.bufferPool.getBuffer(len(packet))
			dataBuffer.Write(packet)
		}
		Logger.Debugf("forwarded message DataType:%v messageId:%X DestNodeId：%v SrcNodeId：%v RelayCount:%v", req.DataType, req.MessageId, dstNodeID.GetHexString(), srcNodeID.GetHexString(), req.RelayCount)

		if req.DataType == DataType_DataGroup {
			nc.groupManager.groupBroadcast(req.GroupId, dataBuffer, uint32(req.MessageCode))
		} else if req.DataType == DataType_DataGlobal {
			nc.peerManager.broadcast(dataBuffer, uint32(req.MessageCode))
		} else if req.DataType == DataType_DataGlobalRandom {
			nc.peerManager.broadcastRandom(dataBuffer, uint32(req.MessageCode))
		}
	}
}

func (nc *NetCore) onHandleDataMessage(data *MsgData, fromID NodeID) {
	if nc.unhandledDataMsg > MaxUnhandledMessageCount {
		Logger.Info("unhandled message too much , drop this message !")
		return
	}

	nc.unhandledDataMsg++
	chainID, protocolVersion := decodeMessageInfo(data.MessageInfo)
	p := nc.peerManager.peerByID(fromID)
	if p != nil {
		p.chainID = chainID
		if !p.IsCompatible() {
			Logger.Info("node chain id not compatible, drop this message !")
			return
		}
	}

	if net != nil {
		net.handleMessage(data.Data, fromID.GetHexString(), chainID, protocolVersion)
	}

}

func (nc *NetCore) onHandleDataMessageDone(id string) {
	nc.unhandledDataMsg--
}

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}

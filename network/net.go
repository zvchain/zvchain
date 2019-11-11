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
	"net"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/middleware/statistics"
	zvTime "github.com/zvchain/zvchain/middleware/time"
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
	errUnknownMsg       = errors.New("unknown msg")
	errBadPeer          = errors.New("bad Peer")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	//errGroupEmpty       = errors.New("group empty")
	errTimeout   = errors.New("RPC timeout")
	errClockWarp = errors.New("reply deadline too far in the future")
	errClosed    = errors.New("socket closed")
)

// Timeouts
const (
	respTimeout              = 500 * time.Millisecond
	clearMessageCacheTimeout = time.Minute
	expiration               = 60 * time.Second
	connectTimeout           = 3 * time.Second
	peerCheckInterval        = 1 * time.Second
	groupRefreshInterval     = 5 * time.Second
	flowMeterInterval        = 1 * time.Minute
)

// NetCore p2p network
type NetCore struct {
	ourEndPoint      RpcEndPoint
	ID               NodeID
	netID            uint64
	natType          uint32
	addPending       chan *pending
	gotReply         chan reply
	unhandled        chan *Peer
	unhandledDataMsg int32
	closing          chan struct{}

	kad             *Kad
	peerManager     *PeerManager
	groupManager    *GroupManager
	messageManager  *MessageManager
	flowMeter       *FlowMeter
	bufferPool      *BufferPool
	proposerManager *ProposerManager
	chainID         uint16 // Chain ID
	protocolVersion uint16 // Protocol ID
}

type pending struct {
	from  NodeID
	pType MessageType

	deadline time.Time

	callback func(resp interface{}) (done bool)

	errc chan<- error
}

type reply struct {
	from    NodeID
	pType   MessageType
	data    interface{}
	matched chan<- bool
}

func genNetID(ID NodeID) uint64 {
	h := fnv.New64a()
	n, err := h.Write(ID[:])
	if n == 0 || err != nil {
		return 0
	}
	return uint64(h.Sum64())
}

//nodeFromRPC is RPC node transformation
func (nc *NetCore) nodeFromRPC(sender *net.UDPAddr, rn RpcNode) (*Node, error) {
	if rn.Port <= 1024 {
		return nil, errors.New("low port")
	}

	n := NewNode(*NewNodeID(rn.ID), net.ParseIP(rn.IP), int(rn.Port))

	err := n.validateComplete()
	return n, err
}

var netCore *NetCore

//NetCoreConfig net core configure
type NetCoreConfig struct {
	ListenAddr         *net.UDPAddr
	ID                 NodeID
	Seeds              []*Node
	NatTraversalEnable bool

	NatPort         uint16
	NatIP           string
	ChainID         uint16
	ProtocolVersion uint16
}

// MakeEndPoint create the node description object
func MakeEndPoint(addr *net.UDPAddr, tcpPort int32) RpcEndPoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return RpcEndPoint{IP: ip.String(), Port: int32(addr.Port)}
}

func nodeToRPC(n *Node) RpcNode {
	return RpcNode{ID: n.ID.GetHexString(), IP: n.IP.String(), Port: int32(n.Port)}
}

// InitNetCore For initialization
func (nc *NetCore) InitNetCore(cfg NetCoreConfig) (*NetCore, error) {

	nc.ID = cfg.ID
	nc.closing = make(chan struct{})
	nc.gotReply = make(chan reply)
	nc.addPending = make(chan *pending)
	nc.unhandled = make(chan *Peer, 64)
	nc.netID = genNetID(cfg.ID)
	nc.chainID = cfg.ChainID
	nc.protocolVersion = cfg.ProtocolVersion
	nc.peerManager = newPeerManager()
	nc.peerManager.natTraversalEnable = cfg.NatTraversalEnable
	nc.peerManager.natIP = cfg.NatIP
	nc.peerManager.natPort = cfg.NatPort
	nc.groupManager = newGroupManager()
	nc.messageManager = newMessageManager(nc.ID)
	nc.proposerManager = newProposerManager()
	nc.flowMeter = newFlowMeter("p2p")
	nc.bufferPool = newBufferPool()
	realAddr := cfg.ListenAddr

	Logger.Infof("kad ID: %v ", nc.ID.GetHexString())
	Logger.Infof("chain ID: %v ", nc.chainID)
	Logger.Infof("protocol version : %v ", nc.protocolVersion)
	Logger.Infof("P2PConfig: %v ", nc.netID)
	Logger.Infof("local addr: %v %v", realAddr.IP.String(), uint16(realAddr.Port))
	nc.ourEndPoint = MakeEndPoint(realAddr, int32(realAddr.Port))
	P2PConfig(nc.netID)

	if cfg.NatTraversalEnable {
		Logger.Infof("P2PProxy: %v %v", nc.peerManager.natIP, uint16(nc.peerManager.natPort))
		P2PProxy(nc.peerManager.natIP, uint16(nc.peerManager.natPort))
	} else {
		Logger.Infof("P2PListen: %v %v", realAddr.IP.String(), uint16(realAddr.Port))
		P2PListen(realAddr.IP.String(), uint16(realAddr.Port))
	}

	kad, err := newKad(nc, cfg.ID, realAddr, cfg.Seeds)
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

func (nc *NetCore) ping(toID NodeID, toAddr *net.UDPAddr) {

	if !toID.IsValid() {
		Logger.Infof("[send ping]node ID : %v  is invalid", toID.GetHexString())
		return
	}
	p := nc.peerManager.peerByID(toID)

	if p != nil {
		if p.isAvailable() || time.Since(p.lastPingTime) < 3*time.Second {
			return
		}
		p.pingCount += 1
	}

	to := MakeEndPoint(&net.UDPAddr{}, 0)
	if toAddr != nil {
		to = MakeEndPoint(toAddr, 0)
	}
	req := &MsgPing{
		Version:    Version,
		From:       &nc.ourEndPoint,
		To:         &to,
		ChainID:    uint32(nc.chainID),
		Expiration: nc.expirationTime(),
	}
	if p != nil && !p.isAuthSucceed {
		authContext := p.AuthContext()
		if authContext == nil {
			Logger.Infof("[send ping] authContext is nil, ID : %v", toID.GetHexString())
			return
		}
		req.PK = authContext.PK
		req.CurTime = authContext.CurTime
		req.Sign = authContext.Sign
	}
	Logger.Infof("[send ping] ID : %v  ip:%v port:%v", toID.GetHexString(), nc.ourEndPoint.IP, nc.ourEndPoint.Port)

	packet, _, err := nc.encodePacket(MessageType_MessagePing, req)
	if err != nil {
		return
	}

	nc.peerManager.write(toID, toAddr, packet, P2PMessageCodeBase+uint32(MessageType_MessagePing))
}

func (nc *NetCore) findNode(toID NodeID, toAddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	nodes := make([]*Node, 0, bucketSize)
	errc := nc.pending(toID, MessageType_MessageNeighbors, func(r interface{}) bool {
		nreceived := 0
		reply := r.(*MsgNeighbors)
		for _, rn := range reply.Nodes {
			n, err := nc.nodeFromRPC(toAddr, *rn)
			if err != nil {
				continue
			}
			nreceived++

			nodes = append(nodes, n)
		}

		return nreceived >= bucketSize
	})
	nc.sendMessageToNode(toID, toAddr, MessageType_MessageFindnode, &MsgFindNode{
		Target:     target[:],
		Expiration: nc.expirationTime(),
	}, P2PMessageCodeBase+uint32(MessageType_MessageFindnode))
	err := <-errc

	return nodes, err
}

func (nc *NetCore) pending(ID NodeID, pType MessageType, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: ID, pType: pType, callback: callback, errc: ch}
	select {
	case nc.addPending <- p:
	case <-nc.closing:
		ch <- errClosed
	}
	return ch
}

func (nc *NetCore) handleReply(from NodeID, ptype MessageType, req interface{}) bool {
	matched := make(chan bool, 1)
	select {
	case nc.gotReply <- reply{from, ptype, req, matched}:
		// loop will handle it
		return <-matched
	case <-nc.closing:
		return false
	}
}

func (nc *NetCore) decodeLoop() {

	for peer := range nc.unhandled {
		for {

			err := nc.handleMessage(peer)
			if err != nil || peer.isEmpty() {
				break
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
		peerCheck         = time.NewTicker(peerCheckInterval)
		timeout           = time.NewTimer(0)
		nextTimeout       *pending
		contTimeouts      = 0
	)
	defer clearMessageCache.Stop()
	defer groupRefresh.Stop()
	defer timeout.Stop()
	defer flowMeter.Stop()
	defer peerCheck.Stop()

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
		case p := <-nc.addPending:
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

		case r := <-nc.gotReply:
			var matched bool
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if p.from == r.from && p.pType == r.pType {
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
		case <-peerCheck.C:
			nc.peerManager.checkPeers()
		case <-flowMeter.C:
			nc.flowMeter.print()
			nc.flowMeter.reset()

		case <-groupRefresh.C:
			go nc.groupManager.doRefresh()
		}
	}
}

func init() {

}

func (nc *NetCore) sendToNode(toID NodeID, toAddr *net.UDPAddr, data []byte, code uint32) {
	packet, _, err := nc.encodeDataPacket(data, DataType_DataNormal, code, "", nil, -1)
	if err != nil {
		Logger.Debugf("Send encodeDataPacket err :%v ", toID.GetHexString())
		return
	}
	nc.peerManager.write(toID, toAddr, packet, code)
	nc.bufferPool.freeBuffer(packet)
}

func (nc *NetCore) sendMessageToNode(toID NodeID, toAddr *net.UDPAddr, ptype MessageType, req proto.Message, code uint32) {
	packet, _, err := nc.encodePacket(ptype, req)
	if err != nil {
		return
	}
	nc.peerManager.write(toID, toAddr, packet, code)
	nc.bufferPool.freeBuffer(packet)
}

func (nc *NetCore) broadcast(data []byte, code uint32, broadcast bool, msgDigest MsgDigest, relayCount int32) {

	dataType := DataType_DataNormal
	if broadcast {
		dataType = DataType_DataGlobal
	}
	packet, _, err := nc.encodeDataPacket(data, dataType, code, "", msgDigest, relayCount)
	if err != nil {
		return
	}
	nc.peerManager.broadcast(packet, code)
	nc.bufferPool.freeBuffer(packet)

}

func (nc *NetCore) broadcastRandom(data []byte, code uint32, relayCount int32, maxCount int) {
	dataType := DataType_DataGlobalRandom

	packet, _, err := nc.encodeDataPacket(data, dataType, code, "", nil, relayCount)
	if err != nil {
		return
	}
	nc.peerManager.broadcastRandom(packet, code, maxCount)
	nc.bufferPool.freeBuffer(packet)
	return
}

func (nc *NetCore) groupBroadcast(ID string, data []byte, code uint32, broadcast bool, relayCount int32) {
	dataType := DataType_DataNormal
	if broadcast {
		dataType = DataType_DataGroup
	}
	msg := nc.genDataMessage(data, dataType, code, ID, nil, relayCount)
	if msg == nil {
		return
	}
	nc.groupManager.Broadcast(ID, msg, nil, code)

}

func (nc *NetCore) groupBroadcastWithMembers(ID string, data []byte, code uint32, msgDigest MsgDigest, groupMembers []string, relayCount int32) {
	msg := nc.genDataMessage(data, DataType_DataGroup, code, ID, msgDigest, relayCount)
	if msg == nil {
		return
	}
	nc.groupManager.Broadcast(ID, msg, groupMembers, code)
}

// OnConnected callback when a peer is connected
func (nc *NetCore) onConnected(ID uint64, session uint32, p2pType uint32) {
	nc.peerManager.newConnection(ID, session, p2pType, false, "", 0)
}

// OnAccepted callback when Accepted a  peer
func (nc *NetCore) onAccepted(ID uint64, session uint32, p2pType uint32, ip string, port uint16) {
	nc.peerManager.newConnection(ID, session, p2pType, true, ip, port)
}

// OnDisconnected callback when peer is disconnected
func (nc *NetCore) onDisconnected(ID uint64, session uint32, p2pCode uint32) {
	nc.peerManager.onDisconnected(ID, session, p2pCode)
}

// OnSendWaited callback when send list is clear
func (nc *NetCore) onSendWaited(ID uint64, session uint32) {
	nc.peerManager.onSendWaited(ID, session)
}

// OnChecked callback when nat type is checked
func (nc *NetCore) onChecked(p2pType uint32, privateIP string, publicIP string) {
	nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{IP: net.ParseIP(publicIP), Port: 0}, 0)
	nc.natType = p2pType
	nc.peerManager.onChecked(p2pType, privateIP, publicIP)
	Logger.Debugf("OnChecked, nat type :%v public ip: %v private ip :%v", p2pType, publicIP, privateIP)

	if p2pType == 4 || p2pType == 5 {
		fmt.Printf("Your router does not support NAT traversal, please upgrade your router.\n")
	}
}

// OnRecved callback when data is received
func (nc *NetCore) onRecved(netID uint64, session uint32, data []byte) {
	nc.recvData(netID, session, data)
}

func (nc *NetCore) recvData(netID uint64, session uint32, data []byte) {

	p := nc.peerManager.peerByNetID(netID)
	if p == nil {
		Logger.Errorf("recv data, but peer not in peer manager ! net id:%v session:%v", netID, session)
		return
	}

	p.addRecvData(data)
	nc.unhandled <- p
}

func (nc *NetCore) expirationTime() uint64 {
	return uint64(zvTime.TSInstance.Now().Unix()) + uint64(expiration.Seconds())

}
func (nc *NetCore) genDataMessage(data []byte,
	dataType DataType,
	code uint32,
	groupID string,
	msgDigest MsgDigest,
	relayCount int32) *MsgData {

	bizMessageIDBytes := make([]byte, 0)
	if msgDigest != nil {
		bizMessageIDBytes = msgDigest[:]
	}
	msgData := &MsgData{
		Data:         data,
		DataType:     dataType,
		GroupID:      groupID,
		MessageID:    nc.messageManager.genMessageID(),
		MessageCode:  uint32(code),
		SrcNodeID:    nc.ID.Bytes(),
		BizMessageID: bizMessageIDBytes,
		RelayCount:   relayCount,
		MessageInfo:  encodeMessageInfo(nc.chainID, nc.protocolVersion),
		Expiration:   nc.expirationTime()}

	return msgData
}

func (nc *NetCore) encodeDataPacket(data []byte,
	dataType DataType,
	code uint32,
	groupID string,
	msgDigest MsgDigest,
	relayCount int32) (msg *bytes.Buffer, hash []byte, err error) {

	dataMsg := nc.genDataMessage(data, dataType, code, groupID, msgDigest, relayCount)
	return nc.encodePacket(MessageType_MessageData, dataMsg)
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
	defer func() {
		if r := recover(); r != nil {
			Logger.Errorf("handleMessage error：%v\n", r)
		}
	}()
	msgType, packetSize, msg, buf, err := nc.decodeMessage(p)

	if err != nil {
		return err
	}

	switch msgType {
	case MessageType_MessagePing:
		err = nc.handlePing(msg.(*MsgPing), p)
	case MessageType_MessagePong:

		err = nc.handlePong(msg.(*MsgPong), p)
	case MessageType_MessageFindnode:
		err = nc.handleFindNode(msg.(*MsgFindNode), p)
	case MessageType_MessageNeighbors:
		err = nc.handleNeighbors(msg.(*MsgNeighbors), p)
	case MessageType_MessageData:
		err = nc.handleData(msg.(*MsgData), buf.Bytes()[0:packetSize], p)
	default:
		Logger.Errorf("unknown type: %d \n", msgType)
		return errUnknownMsg
	}
	if buf != nil {
		nc.bufferPool.freeBuffer(buf)
	}
	return err
}

func (nc *NetCore) decodeMessage(p *Peer) (MessageType, int, proto.Message, *bytes.Buffer, error) {

	msgType, packetSize, packetBuffer, data, err := p.decodePacket()

	if err != nil {
		return msgType, packetSize, nil, packetBuffer, err
	}

	var req proto.Message
	switch msgType {
	case MessageType_MessagePing:
		req = new(MsgPing)
	case MessageType_MessagePong:
		req = new(MsgPong)
	case MessageType_MessageFindnode:
		req = new(MsgFindNode)
	case MessageType_MessageNeighbors:
		req = new(MsgNeighbors)
	case MessageType_MessageData:
		req = new(MsgData)
	default:
		return msgType, packetSize, nil, packetBuffer, fmt.Errorf("unknown type: %d", msgType)
	}

	err = proto.Unmarshal(data, req)

	if msgType != MessageType_MessageData {
		nc.flowMeter.recv(P2PMessageCodeBase+int64(msgType), int64(packetSize))
	}

	return msgType, packetSize, req, packetBuffer, err
}

func (nc *NetCore) handlePing(req *MsgPing, p *Peer) error {

	if expired(req.Expiration) {
		return errExpired
	}

	ip := net.ParseIP(req.From.IP)
	port := int(req.From.Port)

	p.chainID = uint16(req.ChainID)

	from := net.UDPAddr{IP: net.ParseIP(req.From.IP), Port: int(req.From.Port)}

	if !nc.handleReply(p.ID, MessageType_MessagePing, req) {
		_, err := nc.kad.onPingNode(p.ID, &from)
		if err != nil {
			return err
		}
	}

	if len(req.PK) > 0 && len(req.Sign) > 0 && req.CurTime > 0 {
		pac := &PeerAuthContext{PK: req.PK, Sign: req.Sign, CurTime: req.CurTime}
		p.verify(pac)
	}

	pongMsg := MsgPong{Version: 0, VerifyResult: p.verifyResult}

	nc.sendMessageToNode(p.ID, nil, MessageType_MessagePong, &pongMsg, P2PMessageCodeBase+uint32(MessageType_MessagePong))

	if !p.remoteVerifyResult && p.ID.IsValid() {
		go nc.ping(p.ID, nil)
	}
	Logger.Debugf("ping from:%v, ID:%v, port:%d, VerifyResult:%v, RemoteVerifyResult:%v,isAuthSucceed:%v ",
		p.ID.GetHexString(), ip, port, p.verifyResult, p.remoteVerifyResult, p.isAuthSucceed)

	return nil
}

func (nc *NetCore) handlePong(req *MsgPong, p *Peer) error {

	p.setRemoteVerifyResult(req.VerifyResult)
	Logger.Debugf("Pong from:%v, VerifyResult:%v, RemoteVerifyResult:%v,isAuthSucceed:%v",
		p.ID.GetHexString(), p.verifyResult, p.remoteVerifyResult, p.isAuthSucceed)
	if !req.VerifyResult {
		p.resetRemoteVerifyContext()
		go nc.ping(p.ID, nil)
	}
	return nil
}

func (nc *NetCore) handleFindNode(req *MsgFindNode, p *Peer) error {

	if expired(req.Expiration) {
		return errExpired
	}

	target := req.Target
	nc.kad.mutex.Lock()
	closest := nc.kad.closest(target, bucketSize).entries
	nc.kad.mutex.Unlock()

	msg := MsgNeighbors{Expiration: nc.expirationTime()}

	for _, n := range closest {
		node := nodeToRPC(n)
		if len(node.IP) > 0 && node.Port > 0 {
			msg.Nodes = append(msg.Nodes, &node)
		}
	}
	if len(msg.Nodes) > 0 {
		nc.sendMessageToNode(p.ID, nil, MessageType_MessageNeighbors, &msg, P2PMessageCodeBase+uint32(MessageType_MessageNeighbors))
	}

	return nil
}

func (nc *NetCore) handleNeighbors(req *MsgNeighbors, p *Peer) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.handleReply(p.ID, MessageType_MessageNeighbors, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (nc *NetCore) handleData(req *MsgData, packet []byte, p *Peer) error {
	if expired(req.Expiration) {
		Logger.Infof("message expired!")
		return errExpired
	}
	srcNodeID := NodeID{}
	srcNodeID.SetBytes(req.SrcNodeID)

	statistics.AddCount("net.handleData", uint32(req.DataType), uint64(len(req.Data)))
	if req.DataType == DataType_DataNormal {
		nc.onHandleDataMessage(req, srcNodeID)
		return nil
	}

	forwarded := false
	handled := false
	if req.BizMessageID != nil {
		bizID := nc.messageManager.byteToBizID(req.BizMessageID)
		forwarded = nc.messageManager.isForwardedBiz(bizID)
		handled = nc.messageManager.isHandledBiz(bizID)

	} else {
		forwarded = nc.messageManager.isForwarded(req.MessageID)
		handled = nc.messageManager.isHandled(req.MessageID)
	}
	if !handled {
		nc.messageManager.handle(req.MessageID)
		if req.BizMessageID != nil {
			bizID := nc.messageManager.byteToBizID(req.BizMessageID)
			nc.messageManager.handleBiz(bizID)
		}
		nc.onHandleDataMessage(req, srcNodeID)
	}

	//group row message just handle it,but don't forward
	if req.DataType == DataType_DataGroupRow {
		return nil
	}

	if forwarded {
		return nil
	}

	nc.messageManager.forward(req.MessageID)
	if req.BizMessageID != nil {
		bizID := nc.messageManager.byteToBizID(req.BizMessageID)
		nc.messageManager.forwardBiz(bizID)
	}
	// Need to deal with

	broadcast := true
	// Need to be broadcast

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

		if dataBuffer != nil {
			if req.DataType == DataType_DataGroup || req.DataType == DataType_DataGroupColumn {
				nc.groupManager.onBroadcast(req.GroupID, req)
			} else if req.DataType == DataType_DataGlobal {
				nc.peerManager.broadcast(dataBuffer, uint32(req.MessageCode))
			}
		}
	}
	return nil
}

func (nc *NetCore) onHandleDataMessage(data *MsgData, fromID NodeID) {
	if atomic.LoadInt32(&nc.unhandledDataMsg) > MaxUnhandledMessageCount {
		Logger.Info("unhandled message too much , drop this message !")
		return
	}

	chainID, protocolVersion := decodeMessageInfo(data.MessageInfo)
	p := nc.peerManager.peerByID(fromID)
	if p != nil {
		p.chainID = chainID
		if !p.IsCompatible() {
			Logger.Info("Node chain ID not compatible, drop this message !")
			return
		}

		if !p.isAuthSucceed {
			Logger.Info("Peer Authentication is not succeed , drop this message !")
			return
		}
	}

	if netServerInstance != nil {
		netServerInstance.handleMessage(data.Data, fromID.GetHexString(), chainID, protocolVersion)
	}

}

func (nc *NetCore) onHandleDataMessageStart() {
	atomic.AddInt32(&nc.unhandledDataMsg, 1)
}

func (nc *NetCore) onHandleDataMessageDone() {
	atomic.AddInt32(&nc.unhandledDataMsg, -1)
}

func expired(ts uint64) bool {
	return zvTime.TSInstance.Now().Unix() > int64(ts)
}

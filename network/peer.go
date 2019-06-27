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
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

type PeerSource int32

const (
	PeerSourceUnkown PeerSource = 0
	PeerSourceKad    PeerSource = 1
	PeerSourceGroup  PeerSource = 2
)

<<<<<<< HEAD
type SendPriorityType uint32

const (
	SendPriorityHigh   SendPriorityType = 0
	SendPriorityMedium SendPriorityType = 1
	SendPriorityLow    SendPriorityType = 2
)
const MaxSendPriority = 3
const MaxPendingSend = 10
const MaxSendListSize = 256
const WaitTimeout = 3 * time.Second
const RelayTestTimeOut = 30 * time.Minute

var priorityTable map[uint32]SendPriorityType

type SendListItem struct {
	priority int
	list     *list.List
	quota    int
	curQuota int
}

func newSendListItem(priority int, quota int) *SendListItem {

	item := &SendListItem{priority: priority, quota: quota, list: list.New()}

	return item
}

type SendList struct {
	list        [MaxSendPriority]*SendListItem
	pendingSend int
	totalQuota  int
	curQuota    int
	lastOnWait  time.Time
}

func newSendList() *SendList {
	PriorityQuota := [MaxSendPriority]int{5, 3, 2}

	sl := &SendList{lastOnWait: time.Now()}

	for i := 0; i < MaxSendPriority; i++ {
		sl.list[i] = newSendListItem(i, PriorityQuota[i])
		sl.totalQuota += PriorityQuota[i]
	}

	return sl
}

func (sendList *SendList) send(peer *Peer, packet *bytes.Buffer, code int) {

	if peer == nil || packet == nil {
		return
	}

	diff := time.Since(sendList.lastOnWait)

	if diff > WaitTimeout {
		sendList.pendingSend = 0
		Logger.Infof("send list  WaitTimeout ！ net id:%v session:%v ", peer.ID.GetHexString(), peer.sessionID)
	}

	priority, isExist := priorityTable[uint32(code)]
	if !isExist {
		priority = MaxSendPriority - 1
	}
	sendListItem := sendList.list[priority]
	if sendListItem.list.Len() > MaxSendListSize {
		Logger.Infof("send list send is full, drop this message!  net id:%v session:%v code:%v", peer.ID.GetHexString(), peer.sessionID, code)
		return
	}
	sendListItem.list.PushBack(packet)
	netCore.flowMeter.send(int64(code), int64(len(packet.Bytes())))
	sendList.autoSend(peer)
}

func (sendList *SendList) isSendAvailable() bool {
	return sendList.pendingSend < MaxPendingSend
}

func (sendList *SendList) onSendWaited(peer *Peer) {
	sendList.lastOnWait = time.Now()
	sendList.lastOnWait = time.Now()
	sendList.pendingSend = 0
	sendList.autoSend(peer)
}

func (sendList *SendList) autoSend(peer *Peer) {
	
	if peer.sessionID == 0 || !sendList.isSendAvailable() {
		return
	}

	remain := 0
	for i := 0; i < MaxSendPriority && sendList.isSendAvailable(); i++ {
		item := sendList.list[i]

		for item.list.Len() > 0 && sendList.isSendAvailable() {
			e := item.list.Front()
			if e == nil {
				break
			} else if e.Value == nil {
				item.list.Remove(e)
				break
			}

			buf := e.Value.(*bytes.Buffer)
			Logger.Debugf("P2PSend  net id:%v session:%v size:%v ", peer.ID.GetHexString(), peer.sessionID, buf.Len())
			P2PSend(peer.sessionID, buf.Bytes())

			netCore.bufferPool.freeBuffer(buf)

			item.list.Remove(e)
			sendList.pendingSend++

			item.curQuota++
			sendList.curQuota++

			if item.curQuota >= item.quota {
				break
			}
		}
		remain += item.list.Len()
		if sendList.curQuota >= sendList.totalQuota {
			sendList.resetQuota()
		}
	}
}

func (sendList *SendList) resetQuota() {

	sendList.curQuota = 0

	for i := 0; i < MaxSendPriority; i++ {
		item := sendList.list[i]
		item.curQuota = 0
	}

}

func (sendList *SendList) getDataSize() int {
	size := 0
	for i := 0; i < MaxSendPriority; i++ {
		item := sendList.list[i]

		for e := item.list.Front(); e != nil; e = e.Next() {
			buf := e.Value.(*bytes.Buffer)
			size += buf.Len()
		}
	}
	return size
}

// Peer is node connection object
type Peer struct {
	ID             NodeID
	relayID        NodeID
	relayTestTime  time.Time
	sessionID      uint32
	IP             net.IP
	Port           int
	sendList       *SendList
	recvList       *list.List
	connectTimeout uint64
	mutex          sync.RWMutex
	connecting     bool

	isPinged       bool

	source         PeerSource

	bytesReceived   int
	bytesSend       int
	sendWaitCount   int
	disconnectCount int
	chainID         uint16
}

func newPeer(ID NodeID, sessionID uint32) *Peer {

	p := &Peer{ID: ID, sessionID: sessionID, sendList: newSendList(), recvList: list.New(), source: PeerSourceUnkown}

	return p
}

func (p *Peer) addRecvData(data []byte) {

	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := netCore.bufferPool.getBuffer(len(data))
	b.Write(data)
	p.recvList.PushBack(b)
	p.bytesReceived += len(data)
}

func (p *Peer) addRecvDataToHead(data *bytes.Buffer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.recvList.PushFront(data)
}

func (p *Peer) popData() *bytes.Buffer {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.recvList.Len() == 0 {
		return nil
	}
	buf := p.recvList.Front().Value.(*bytes.Buffer)
	p.recvList.Remove(p.recvList.Front())

	return buf
}

func (p *Peer) onSendWaited() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.sendList.onSendWaited(p)
	p.sendWaitCount++
}

func (p *Peer) resetData() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.recvList = list.New()
}

func (p *Peer) isEmpty() bool {

	empty := true
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.recvList.Len() > 0 {
		empty = false
	}

	return empty
}

func (p *Peer) write(packet *bytes.Buffer, code uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := netCore.bufferPool.getBuffer(packet.Len())
	b.Write(packet.Bytes())

	p.sendList.send(p, b, int(code))
}

func (p *Peer) getDataSize() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	size := 0
	for e := p.recvList.Front(); e != nil; e = e.Next() {
		buf := e.Value.(*bytes.Buffer)
		size += buf.Len()
	}

	return size
}

func (p *Peer) IsCompatible() bool {
	return netCore.chainID == p.chainID
}

// PeerManager is node connection management
type PeerManager struct {
	peers              map[uint64]*Peer // Key is the network ID
	mutex              sync.RWMutex
	natTraversalEnable bool
	natPort            uint16
	natIP              string
}

func newPeerManager() *PeerManager {

<<<<<<< HEAD
	pm := &PeerManager{
		peers: make(map[uint64]*Peer),
=======
	if !p.isAuthSucceed && p.verifyResult && p.remoteVerifyResult {
		p.isAuthSucceed = true
>>>>>>> 5247e09... Merge pull request #57 from zvchain/dev_crypto
	}
	priorityTable = map[uint32]SendPriorityType{
		BlockInfoNotifyMsg: SendPriorityHigh,
		NewBlockMsg:        SendPriorityHigh,
		ReqBlock:           SendPriorityHigh,
		BlockResponseMsg:   SendPriorityHigh,
		GroupChainCountMsg: SendPriorityHigh,
		ReqGroupMsg:        SendPriorityHigh,
		GroupMsg:           SendPriorityHigh,
		ReqChainPieceBlock: SendPriorityHigh,
		ChainPieceBlock:    SendPriorityHigh,
		CastVerifyMsg:      SendPriorityHigh,
		VerifiedCastMsg:    SendPriorityHigh,
		CastRewardSignReq:  SendPriorityMedium,
		CastRewardSignGot:  SendPriorityMedium,
	}
	return pm
}

func (pm *PeerManager) write(toid NodeID, toaddr *net.UDPAddr, packet *bytes.Buffer, code uint32, relay bool) {

	netID := genNetID(toid)
	p := pm.peerByNetID(netID)
	if p == nil {
		p = newPeer(toid, 0)
		p.connectTimeout = 0
		p.connecting = false
		pm.addPeer(netID, p)
	}
	if p.relayID.IsValid() && relay {
		relayPeer := pm.peerByID(p.relayID)

		if relayPeer != nil && relayPeer.sessionID > 0 {
			Logger.Infof("[Relay] send with relay , relay node ID: %v ,to id :%v", p.relayID.GetHexString(), toid.GetHexString())
			go pm.write(p.relayID, nil, packet, code, false)
			return
		}
	} else {
		p.write(packet, code)
		p.bytesSend += packet.Len()
	}

	if p.sessionID != 0 {
		return
	}
	if ((toaddr != nil && toaddr.IP != nil && toaddr.Port > 0) || pm.natTraversalEnable) && !p.connecting {
		p.connectTimeout = uint64(time.Now().Add(connectTimeout).Unix())
		p.connecting = true

		if toaddr != nil {
			p.IP = toaddr.IP
			p.Port = toaddr.Port
		}

		if pm.natTraversalEnable {
			P2PConnect(netID, pm.natIP, pm.natPort)
			Logger.Infof("connect node ,[nat]: %v ", toid.GetHexString())
		} else {
			P2PConnect(netID, toaddr.IP.String(), uint16(toaddr.Port))
			Logger.Infof("connect node ,[direct]: id: %v ip: %v port:%v ", toid.GetHexString(), toaddr.IP.String(), uint16(toaddr.Port))
		}
	}

	if !p.relayID.IsValid() && p.disconnectCount > 1 && p.bytesReceived == 0 && time.Since(p.relayTestTime) > RelayTestTimeOut {
		p.relayTestTime = time.Now()
		netCore.RelayTest(toid)
	}
}

<<<<<<< HEAD
// newConnection handling callbacks for successful connections
func (pm *PeerManager) newConnection(id uint64, session uint32, p2pType uint32, isAccepted bool) {

	p := pm.peerByNetID(id)
	if p == nil {
		p = newPeer(NodeID{}, session)
		p.connectTimeout = uint64(time.Now().Add(connectTimeout).Unix())
		pm.addPeer(id, p)
	} else if session > 0 {
		p.recvList = list.New()
		p.sessionID = session
	}
	p.connecting = false
=======
func (p *Peer) onConnect(id uint64, session uint32, p2pType uint32, isAccepted bool) {
	p.resetData()
	p.connecting = false
	if session > p.sessionID {
		p.sessionID = session
	}
	p.connectTime = time.Now()
>>>>>>> 5247e09... Merge pull request #57 from zvchain/dev_crypto

	if len(p.ID.GetHexString()) > 0 && !p.isPinged {
		netCore.ping(p.ID, nil)
		p.isPinged = true
	}

	p.sendList.pendingSend = 0
	p.sendList.autoSend(p)
	Logger.Infof("new connection, node id:%v  netid :%v session:%v isAccepted:%v ", p.ID.GetHexString(), id, session, isAccepted)
}

// onSendWaited  when the send queue is idle
func (pm *PeerManager) onSendWaited(id uint64, session uint32) {
	p := pm.peerByNetID(id)
	if p != nil {
		p.onSendWaited()
	}
}

<<<<<<< HEAD
// onDisconnected handles callbacks for disconnected connections
func (pm *PeerManager) onDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	if p != nil {

		Logger.Infof("OnDisconnected id：%v  session:%v ip:%v port:%v ", p.ID.GetHexString(), session, p.IP, p.Port)
		p.disconnectCount++
		p.connecting = false
		if p.sessionID == session {
			p.sessionID = 0
		}
	} else {
		Logger.Infof("OnDisconnected net id：%v session:%v port:%v code:%v", id, session, p2pCode)
	}
}

func (pm *PeerManager) disconnect(id NodeID) {
	netID := genNetID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	if p != nil {

		Logger.Infof("disconnect ip:%v port:%v ", p.IP, p.Port)

		p.connecting = false
		delete(pm.peers, netID)
	}
}

func (pm *PeerManager) onChecked(p2pType uint32, privateIP string, publicIP string) {

}
=======
func (p *Peer) verify(pac *PeerAuthContext) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.isAuthSucceed {
		return true
	}
	p.remoteAuthContext = pac
	verifyResult, verifyID := p.remoteAuthContext.Verify()

	p.verifyResult = verifyResult
	p.ID = NewNodeID(verifyID)
	p.verifyUpdate()
	return p.verifyResult
}

func (p *Peer) write(packet *bytes.Buffer, code uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := netCore.bufferPool.getBuffer(packet.Len())
	b.Write(packet.Bytes())
>>>>>>> 5247e09... Merge pull request #57 from zvchain/dev_crypto

func (pm *PeerManager) checkPeers() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	for _, p := range pm.peers {
		if p.bytesReceived == 0 {
			Logger.Infof("[PeerManager] [checkPeers] peer ip:%v port:%v bytes recv:%v ,bytes send:%v disconnect count:%v send wait count:%v ",
				p.IP, p.Port, p.bytesReceived, p.bytesSend, p.disconnectCount, p.sendWaitCount)

		}
	}
}

func (pm *PeerManager) broadcast(packet *bytes.Buffer, code uint32) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	Logger.Infof("broadcast total peer size:%v code:%v", len(pm.peers), code)

	for _, p := range pm.peers {
		if p.sessionID > 0 && p.IsCompatible() {
			p.write(packet, code)
		}
	}

	return
}

func (pm *PeerManager) checkPeerSource() {
	for _, p := range pm.peers {
		if p.sessionID > 0 && p.source == PeerSourceUnkown {
			node := netCore.kad.find(p.ID)
			if node != nil {
				p.source = PeerSourceKad
			} else {
				p.source = PeerSourceGroup
			}
		}
	}
}

<<<<<<< HEAD
func (pm *PeerManager) broadcastRandom(packet *bytes.Buffer, code uint32) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	Logger.Infof("broadcast random total peer size:%v code:%v", len(pm.peers), code)

	pm.checkPeerSource()
	availablePeers := make([]*Peer, 0, 0)
=======
func (p *Peer) disconnect() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
>>>>>>> 5247e09... Merge pull request #57 from zvchain/dev_crypto

	for _, p := range pm.peers {
		if p.sessionID > 0 && p.IsCompatible() {
			availablePeers = append(availablePeers, p)
		}
	}
	peerSize := len(availablePeers)
	maxCount := int(math.Sqrt(float64(peerSize)))
	if maxCount < 2 {
		maxCount = 2
	}

	if len(availablePeers) < maxCount {
		for _, p := range availablePeers {
			p.write(packet, code)
		}
	} else {
		nodesHasSend := make(map[int]bool)
		r := rand.New(rand.NewSource(time.Now().Unix()))

		for i := 0; i < peerSize && len(nodesHasSend) < maxCount; i++ {
			peerIndex := r.Intn(peerSize)
			if nodesHasSend[peerIndex] == true {
				continue
			}
			nodesHasSend[peerIndex] = true
			if peerIndex < len(availablePeers) {
				p := availablePeers[peerIndex]
				if p != nil {
					p.write(packet, code)
				}
			}
		}
	}

	return
}
<<<<<<< HEAD

func (pm *PeerManager) peerByID(id NodeID) *Peer {
	netID := genNetID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	return p
}

func (pm *PeerManager) peerByNetID(netID uint64) *Peer {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	p, _ := pm.peers[netID]
	return p
}

func (pm *PeerManager) addPeer(netID uint64, peer *Peer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.peers[netID] = peer

}
=======
>>>>>>> 5247e09... Merge pull request #57 from zvchain/dev_crypto

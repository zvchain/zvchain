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
	"math"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

const maxPeersPerIP = 16

// PeerManager is node connection management
type PeerManager struct {
	peers              map[uint64]*Peer // Key is the network ID
	mutex              sync.RWMutex
	natTraversalEnable bool
	natPort            uint16
	natIP              string
	peerIPSet          PeerIPSet
}

func newPeerManager() *PeerManager {
	pm := &PeerManager{
		peers:     make(map[uint64]*Peer),
		peerIPSet: PeerIPSet{Limit: maxPeersPerIP, members: make(map[string]uint)},
	}
	priorityTable = map[uint32]SendPriorityType{
		BlockInfoNotifyMsg:       SendPriorityHigh,
		NewBlockMsg:              SendPriorityHigh,
		ReqBlock:                 SendPriorityHigh,
		BlockResponseMsg:         SendPriorityHigh,
		ForkFindAncestorResponse: SendPriorityHigh,
		ForkFindAncestorReq:      SendPriorityHigh,
		CastVerifyMsg:            SendPriorityHigh,
		VerifiedCastMsg:          SendPriorityHigh,
		CastRewardSignReq:        SendPriorityMedium,
		CastRewardSignGot:        SendPriorityMedium,
	}
	return pm
}

func (pm *PeerManager) write(toid NodeID, toaddr *net.UDPAddr, packet *bytes.Buffer, code uint32) {

	if packet == nil {
		return
	}
	if !toid.IsValid() {
		return
	}
	netID := genNetID(toid)
	p := pm.peerByNetID(netID)

	if p == nil {
		p = newPeer(toid, 0)
		p.connectTimeout = 0
		p.connecting = false
		if toaddr != nil {
			p.IP = toaddr.IP
			p.Port = toaddr.Port
		}
		if !pm.addPeer(netID, p) {
			return
		}
	}
	p.write(packet, code)
	p.bytesSend += packet.Len()

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
}

// newConnection handling callbacks for successful connections
func (pm *PeerManager) newConnection(id uint64, session uint32, p2pType uint32, isAccepted bool, ip string, port uint16) {
	ip = net.ParseIP(ip).String()
	Logger.Infof("new connection, net id:%v session:%v isAccepted:%v ip:%v port:%v, peer count:%v ", id, session, isAccepted, ip, port, pm.peerIPSet.Count(ip))
	if len(ip) > 0 && !pm.peerIPSet.Add(ip) {
		P2PShutdown(session)
		Logger.Infof("new connection , peer in same IP exceed limit size !Max size:%v, ip:%v, peer count:%v", pm.peerIPSet.Limit, ip, pm.peerIPSet.Count(ip))
		return
	}
	p := pm.peerByNetID(id)
	if p == nil {
		p = newPeer(NodeID{}, session)
		pm.addPeer(id, p)
	} else {
		Logger.Infof("new connection ,peer != nil, net id:%v session:%v ip:%v port:%v, peer count:%v ", id, p.sessionID, p.IP.String(), p.Port, pm.peerIPSet.Count(p.IP.String()))
		if session < p.sessionID && time.Since(p.connectTime) < 3*time.Second {
			pm.peerIPSet.Remove(ip)
			Logger.Infof("new connection ,session less than p.sessionID, ip:%v, peer count:%v", ip, pm.peerIPSet.Count(ip))
			P2PShutdown(session)
			return
		} else if p.sessionID > 0 {
			pm.peerIPSet.Remove(p.IP.String())
			Logger.Infof("new connection ,p.sessionID greater than  0, ip:%v, peer count:%v", p.IP.String(), pm.peerIPSet.Count(p.IP.String()))
			p.disconnect()
		}
	}

	p.onConnect(id, session, p2pType, isAccepted, ip, port)
}

// onSendWaited  when the send queue is idle
func (pm *PeerManager) onSendWaited(id uint64, session uint32) {
	p := pm.peerByNetID(id)
	if p != nil {
		p.onSendWaited()
	}
}

// onDisconnected handles callbacks for disconnected connections
func (pm *PeerManager) onDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if p != nil {
		ip := p.IP.String()
		Logger.Infof("disconnected,  node id：%v, netid：%v, session:%v ip:%v port:%v,peers:%v, peer count:%v", p.ID.GetHexString(), id, session, ip, p.Port, pm.peerIPSet.members, pm.peerIPSet.Count(ip))
		if p.sessionID == session {
			pm.peerIPSet.Remove(ip)

			delete(pm.peers, id)
		}

		p.onDisonnect(id, session, p2pCode)

	} else {
		Logger.Infof("disconnected, but session id is unused, net id：%v session:%v", id, session)
	}
}

func (pm *PeerManager) disconnect(id NodeID) {
	netID := genNetID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p := pm.peers[netID]
	if p != nil {
		Logger.Infof("disconnect ip:%v port:%v ", p.IP, p.Port)

		p.disconnect()
		delete(pm.peers, netID)
		pm.peerIPSet.Remove(p.IP.String())
	}
}

func (pm *PeerManager) onChecked(p2pType uint32, privateIP string, publicIP string) {

}

func (pm *PeerManager) checkPeers() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	for _, p := range pm.peers {
		if !p.isAuthSucceed {

			if !p.remoteVerifyResult && p.sessionID > 0 && p.ID.IsValid() {
				go netServerInstance.netCore.ping(p.ID, nil)
			}
			if !p.verifyResult && p.sessionID > 0 {
				pongMsg := MsgPong{Version: 0, VerifyResult: p.verifyResult}

				packet, _, err := netServerInstance.netCore.encodePacket(MessageType_MessagePong, &pongMsg)
				if err != nil {
					return
				}
				p.write(packet, P2PMessageCodeBase+uint32(MessageType_MessagePong))
			}
		}
	}
}

func (pm *PeerManager) broadcast(packet *bytes.Buffer, code uint32) {
	if packet == nil {
		return
	}
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	Logger.Infof("broadcast total peer size:%v code:%v", len(pm.peers), code)

	for _, p := range pm.peers {
		if p.sessionID > 0 && p.IsCompatible() {
			p.write(packet, code)
		}
	}
}

func (pm *PeerManager) broadcastRandom(packet *bytes.Buffer, code uint32, maxCount int) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	Logger.Infof("broadcast random total peer size:%v, code:%v, max count:%v", len(pm.peers), code, maxCount)

	availablePeers := make([]*Peer, 0)

	for _, p := range pm.peers {
		if p.sessionID > 0 && p.IsCompatible() {
			availablePeers = append(availablePeers, p)
		}
	}
	peerSize := len(availablePeers)
	if maxCount == 0 {
		maxCount = int(math.Sqrt(float64(peerSize)))
	}

	if len(availablePeers) <= maxCount {
		for _, p := range availablePeers {
			p.write(packet, code)
		}
	} else {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomNodes := r.Perm(peerSize)

		for i := 0; i < maxCount; i++ {
			p := availablePeers[randomNodes[i]]
			p.write(packet, code)
		}
	}
}

func (pm *PeerManager) peerByID(id NodeID) *Peer {
	netID := genNetID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p := pm.peers[netID]
	return p
}

func (pm *PeerManager) peerByNetID(netID uint64) *Peer {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	p := pm.peers[netID]
	return p
}

func (pm *PeerManager) addPeer(netID uint64, peer *Peer) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.peers[netID] = peer
	return true
}

func (pm *PeerManager) removePeer(netID uint64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	delete(pm.peers, netID)
}

func (pm *PeerManager) ConnInfo() []Conn {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	connInfos := make([]Conn, 0)

	for _, p := range pm.peers {
		if p.sessionID > 0 && p.IP != nil && p.Port > 0 && p.isAuthSucceed {
			c := Conn{ID: p.ID.GetHexString(), IP: p.IP.String(), Port: strconv.Itoa(p.Port)}
			connInfos = append(connInfos, c)
		}
	}
	return connInfos

}

//PeerIPSet tracks IP of peers
type PeerIPSet struct {
	Limit uint // maximum number of IPs in each subnet
	mutex sync.RWMutex

	members map[string]uint
}

// Add add an IP to the set.
func (s *PeerIPSet) Add(ip string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	n := s.members[ip]
	if n < s.Limit {
		s.members[ip] = n + 1
		return true
	}
	return false
}

// Remove removes an IP from the set.
func (s *PeerIPSet) Remove(ip string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if n, ok := s.members[ip]; ok {
		if n == 1 {
			delete(s.members, ip)
		} else {
			s.members[ip] = n - 1
		}
	}
}

// Count the count of the given IP.
func (s PeerIPSet) Count(ip string) uint {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	c := s.members[ip]
	return c
}

// Len returns the number of tracked IPs.
func (s PeerIPSet) Len() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	n := uint(0)
	for _, i := range s.members {
		n += i
	}
	return int(n)
}

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
	"net"
	"strconv"
	"sync"
	"time"
)

const MAX_PEER_SIZE = 512

// PeerManager is node connection management
type PeerManager struct {
	peers              map[uint64]*Peer // Key is the network ID
	mutex              sync.RWMutex
	natTraversalEnable bool
	natPort            uint16
	natIP              string
}

func newPeerManager() *PeerManager {

	pm := &PeerManager{
		peers: make(map[uint64]*Peer),
	}
	priorityTable = map[uint32]SendPriorityType{
		BlockInfoNotifyMsg: SendPriorityHigh,
		NewBlockMsg:        SendPriorityHigh,
		ReqBlock:           SendPriorityHigh,
		BlockResponseMsg:   SendPriorityHigh,
		ReqChainPieceBlock: SendPriorityHigh,
		ChainPieceBlock:    SendPriorityHigh,
		CastVerifyMsg:      SendPriorityHigh,
		VerifiedCastMsg:    SendPriorityHigh,
		CastRewardSignReq:  SendPriorityMedium,
		CastRewardSignGot:  SendPriorityMedium,
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
func (pm *PeerManager) newConnection(id uint64, session uint32, p2pType uint32, isAccepted bool) {

	p := pm.peerByNetID(id)
	if p == nil {
		p = newPeer(NodeID{}, session)
		p.connectTimeout = uint64(time.Now().Add(connectTimeout).Unix())
		if !pm.addPeer(id, p) {
			P2PShutdown(session)
			return
		}
	}
	p.onConnect(id, session, p2pType, isAccepted)
	Logger.Infof("new connection, node id:%v  netid :%v session:%v isAccepted:%v ", p.ID.GetHexString(), id, session, isAccepted)
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
	if p != nil {

		Logger.Infof("OnDisconnected id：%v  session:%v ip:%v port:%v ", p.ID.GetHexString(), session, p.IP, p.Port)
		p.onDisonnect(id, session, p2pCode)

	} else {
		Logger.Infof("OnDisconnected net id：%v session:%v code:%v", id, session, p2pCode)
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
	}
}

func (pm *PeerManager) onChecked(p2pType uint32, privateIP string, publicIP string) {

}

func (pm *PeerManager) checkPeers() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	for nid, p := range pm.peers {
		if !p.isAuthSucceed {
			Logger.Infof("[PeerManager] [checkPeers] peer id:%v netid:%v ip:%v port:%v session:%v bytes recv:%v ,bytes send:%v  data size%v disconnect count:%v send wait count:%v ping count:%d isAuthSucceed:%v",
				p.ID.GetHexString(), nid, p.IP, p.Port, p.sessionID, p.bytesReceived, p.bytesSend, p.getDataSize(), p.disconnectCount, p.sendWaitCount, p.pingCount, p.isAuthSucceed)

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
	if len(pm.peers) > MAX_PEER_SIZE {
		Logger.Infof("addPeer failed, peer size over %v ", MAX_PEER_SIZE)
		return false
	}
	pm.peers[netID] = peer
	return true
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

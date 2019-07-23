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
	"net"
	"sync"
	"time"

	"github.com/zvchain/zvchain/common"
)

type PeerSource int32

const (
	PeerSourceUnkown PeerSource = 0
	PeerSourceKad    PeerSource = 1
	PeerSourceGroup  PeerSource = 2
)

type PeerAuthContext struct {
	PK      []byte
	Sign    []byte
	CurTime uint64
}

func (pa *PeerAuthContext) Verify() (bool, string) {
	pubkey := common.BytesToPublicKey(pa.PK)
	if pubkey == nil {
		return false, ""
	}

	if math.Abs(float64(time.Since(time.Unix(int64(pa.CurTime), 0)))) > float64(time.Minute*5) {
		return false, ""
	}
	buffer := bytes.Buffer{}
	source := pubkey.GetAddress()
	data := common.Uint64ToByte(pa.CurTime)
	buffer.Write(data)
	if netServerInstance != nil {
		buffer.Write(netServerInstance.netCore.ID.Bytes())
	}

	hash := common.BytesToHash(common.Sha256(buffer.Bytes()))
	sign := common.BytesToSign(pa.Sign)
	if sign == nil{
		return false, ""
	}

	result := pubkey.Verify(hash.Bytes(), sign)

	return result, source.Hex()
}

func genPeerAuthContext(PK string, SK string, toID *NodeID) *PeerAuthContext {
	privateKey := common.HexToSecKey(SK)
	pubkey := common.HexToPubKey(PK)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		return nil
	}
	buffer := bytes.Buffer{}
	curTime := uint64(time.Now().UTC().Unix())
	data := common.Uint64ToByte(curTime)
	buffer.Write(data)
	if toID != nil {
		buffer.Write(toID.Bytes())
	}
	hash := common.BytesToHash(common.Sha256(buffer.Bytes()))

	sign, err := privateKey.Sign(hash.Bytes())
	if err != nil {
		return nil
	}

	return &PeerAuthContext{PK: pubkey.Bytes(), Sign: sign.Bytes(), CurTime: curTime}
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
	pingCount      int
	lastPingTime   time.Time
	isPinged       bool
	source         PeerSource

	bytesReceived   int
	bytesSend       int
	sendWaitCount   int
	disconnectCount int
	chainID         uint16

	connectTime        time.Time
	authContext        *PeerAuthContext
	remoteAuthContext  *PeerAuthContext
	verifyResult       bool
	remoteVerifyResult bool
	isAuthSucceed      bool
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

func (p *Peer) isAvailable() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.isAuthSucceed && p.sessionID > 0 && p.IsCompatible()
}

func (p *Peer) resetData() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.recvList = list.New()
}

func (p *Peer) setRemoteVerifyResult(result bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.remoteVerifyResult = result

	p.verifyUpdate()
}

func (p *Peer) verifyUpdate() {

	if !p.isAuthSucceed && p.verifyResult && p.remoteVerifyResult {
		p.isAuthSucceed = true
	}
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

func (p *Peer) onConnect(id uint64, session uint32, p2pType uint32, isAccepted bool) {
	p.resetData()
	p.resetAuthContext()
	p.connecting = false
	if session > p.sessionID {

		p.sessionID = session
	}
	p.connectTime = time.Now()

	netCore.ping(p.ID, nil)

	p.sendList.pendingSend = 0
	p.sendList.autoSend(p)
}

func (p *Peer) resetAuthContext() {
	p.isAuthSucceed = false
	p.authContext = nil
	p.remoteAuthContext = nil
	p.remoteVerifyResult = false
	p.verifyResult = false
}

func (p *Peer) resetRemoteVerifyContext() {
	p.remoteAuthContext = nil
	p.remoteVerifyResult = false
	p.isAuthSucceed = false
}

func (p *Peer) onDisonnect(id uint64, session uint32, p2pCode uint32) {
	p.connecting = false
	p.disconnectCount++
	if session == p.sessionID {
		p.resetData()
		p.sessionID = 0
		p.sendList.pendingSend = 0
	}

}

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

func (p *Peer) disconnect() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.sessionID > 0 {
		P2PShutdown(p.sessionID)
		p.sessionID = 0
	}
	p.resetData()
}

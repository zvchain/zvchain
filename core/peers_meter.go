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

package core

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
)

const (
	evilTimeoutMeterMax = 3
	maxReqBlockCount    = 16
)

var peerManagerImpl *peerManager

type peerMeter struct {
	id            string
	timeoutMeter  int
	lastHeard     time.Time
	mu			  sync.Mutex
	reqBlockCount int // Maximum number of blocks per request
}

func (m *peerMeter) isEvil() bool {
	return time.Since(m.lastHeard).Seconds() > 30 || m.timeoutMeter > evilTimeoutMeterMax
}

func (m *peerMeter) increaseTimeout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeoutMeter++
}
func (m *peerMeter) decreaseTimeout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeoutMeter--
	if m.timeoutMeter < 0 {
		m.timeoutMeter = 0
	}
}

func (m *peerMeter) updateReqCnt(increase bool) {
	if !increase {
		m.reqBlockCount -= 4
		if m.reqBlockCount <= 0 {
			m.reqBlockCount = 1
		}
	} else {
		m.reqBlockCount++
		if m.reqBlockCount > maxReqBlockCount {
			m.reqBlockCount = maxReqBlockCount
		}
	}
}

func (m *peerMeter) updateLastHeard() {
	m.lastHeard = time.Now()
}

type peerManager struct {
	peerMeters *lru.Cache //peerMeters map[string]*peerMeter
}

func initPeerManager() {
	badPeerMeter := peerManager{
		peerMeters: common.MustNewLRUCache(100),
	}
	peerManagerImpl = &badPeerMeter
}

func (bpm *peerManager) getOrAddPeer(id string) *peerMeter {
	v, exit := bpm.peerMeters.Get(id)
	if !exit {
		v = &peerMeter{
			id:            id,
			reqBlockCount: maxReqBlockCount,
		}
		if exit, _ = bpm.peerMeters.ContainsOrAdd(id, v); exit {
			v, _ = bpm.peerMeters.Get(id)
		}
	}
	return v.(*peerMeter)
}

func (bpm *peerManager) getPeerReqBlockCount(id string) int {
	pm := bpm.getOrAddPeer(id)
	if pm == nil {
		return maxReqBlockCount
	}
	return pm.reqBlockCount
}

func (bpm *peerManager) heardFromPeer(id string) {
	if id == "" {
		return
	}
	pm := bpm.getOrAddPeer(id)
	pm.updateLastHeard()
	pm.decreaseTimeout()
}

func (bpm *peerManager) timeoutPeer(id string) {
	if id == "" {
		return
	}
	pm := bpm.getOrAddPeer(id)
	pm.increaseTimeout()
}

func (bpm *peerManager) isEvil(id string) bool {
	if id == "" {
		return false
	}
	pm := bpm.getOrAddPeer(id)
	return pm.isEvil()
}
func (bpm *peerManager) updateReqBlockCnt(id string, increase bool) {
	pm := bpm.getOrAddPeer(id)
	if pm == nil {
		return
	}
	pm.updateReqCnt(increase)
}

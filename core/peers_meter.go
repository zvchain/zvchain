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
	evilTimeoutMeterMax = 5
	evilTimAdd          = 10
	maxReqBlockCount    = 16
	maxEvilCount        = 100
	eachEvilExpireTime  = "1m"
)

var peerManagerImpl *peerManager

type peerMeter struct {
	id             string
	timeoutMeter   int
	mu             sync.Mutex
	reqBlockCount  int // Maximum number of blocks per request
	evilCount      time.Duration
	evilExpireTime time.Time
}

func (m *peerMeter) isEvil() bool {
	return time.Since(m.evilExpireTime) < 0
}

func (m *peerMeter) addEvilCountWithLock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addEvilCount()
}

func (m *peerMeter) addEvilCount() {
	if m.evilCount > maxEvilCount {
		return
	}
	m.evilCount++
	addTime, _ := time.ParseDuration(eachEvilExpireTime)
	m.evilExpireTime = time.Now().Add(m.evilCount * evilTimAdd * addTime)
}

func (m *peerMeter) resetEvilCountWithLock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetEvilCount()
}

func (m *peerMeter) resetEvilCount() {
	m.evilCount = 0
}

func (m *peerMeter) increaseTimeout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeoutMeter++
	if m.timeoutMeter >= evilTimeoutMeterMax {
		m.addEvilCount()
		m.timeoutMeter = 0
	}
}
func (m *peerMeter) resetTimeoutMeter() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeoutMeter = 0
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

type peerManager struct {
	peerMeters *lru.Cache //peerMeters map[string]*peerMeter
}

func initPeerManager() {
	badPeerMeter := peerManager{
		peerMeters: common.MustNewLRUCache(100),
	}
	peerManagerImpl = &badPeerMeter
}

func (bpm *peerManager) isPeerExists(id string) bool {
	_, ok := bpm.peerMeters.Get(id)
	return ok
}

func (bpm *peerManager) getOrAddPeer(id string) *peerMeter {
	v, exit := bpm.peerMeters.Get(id)
	if !exit {
		v = &peerMeter{
			id:             id,
			reqBlockCount:  maxReqBlockCount,
			evilExpireTime: time.Now(),
		}
		bpm.peerMeters.Add(id, v)
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
	pm.resetTimeoutMeter()
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
	if !bpm.isPeerExists(id) {
		return false
	}
	pm := bpm.getOrAddPeer(id)
	return pm.isEvil()
}

func (bpm *peerManager) addEvilCount(id string) {
	if id == "" {
		return
	}
	pm := bpm.getOrAddPeer(id)
	pm.addEvilCountWithLock()
}

func (bpm *peerManager) resetEvilCount(id string) {
	if id == "" {
		return
	}
	pm := bpm.getOrAddPeer(id)
	pm.resetEvilCountWithLock()
}

func (bpm *peerManager) updateReqBlockCnt(id string, increase bool) {
	pm := bpm.getOrAddPeer(id)
	if pm == nil {
		return
	}
	pm.updateReqCnt(increase)
}

//   Copyright (C) 2019 ZVChain
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

package group

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"sync/atomic"
)

type createStatus int

const (
	createStatusIdle createStatus = iota
	createStatusSuccess
	createStatusFail
)

type createStat struct {
	eraCache *lru.Cache
	idle     int32
	success  int32
	fail     int32
	outCh    chan struct{}
}

func newCreateStat() *createStat {
	st := &createStat{
		outCh: make(chan struct{}, 5),
	}
	st.eraCache = common.MustNewLRUCacheWithEvictCB(5, st.onEvict)
	return st
}

func (st *createStat) onEvict(k, v interface{}) {
	logger.Debugf("evict stat era %v %v", k, v)
	switch v.(createStatus) {
	case createStatusIdle:
		atomic.AddInt32(&st.idle, 1)
	case createStatusSuccess:
		atomic.AddInt32(&st.success, 1)
	case createStatusFail:
		atomic.AddInt32(&st.fail, 1)
	}
}

func (st *createStat) loop() {
	for {
		select {
		case <-st.outCh:
			st.statLog()
		}
	}
}

func (st *createStat) markStatus(eraSeed uint64, status createStatus) {
	st.eraCache.Add(eraSeed, status)
}

func (st *createStat) statLog() {
	var idle, success, fail = atomic.LoadInt32(&st.idle), atomic.LoadInt32(&st.success), atomic.LoadInt32(&st.fail)
	for _, seed := range st.eraCache.Keys() {
		v, ok := st.eraCache.Peek(seed)
		if !ok {
			continue
		}
		switch v.(createStatus) {
		case createStatusIdle:
			idle++
		case createStatusSuccess:
			success++
		case createStatusFail:
			fail++
		}
	}
	total := idle + success + fail
	start := success + fail
	if start == 0 {
		start = 1
	}
	logger.Debugf("create group stat: eraCnt=%v, idle=%v, successCnt=%v, failCnt=%v, successRate=%v", total, idle, success, fail, float64(success)/float64(start))
}

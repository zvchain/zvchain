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

// Package ticker implements a cron task schedule tool wrappered from the go timer sdk
package ticker

import (
	"sync"
	"sync/atomic"
	"time"
)

// RoutineFunc is the routine function which will be called at specified moment
type RoutineFunc func() bool

// the ticker status
const (
	stopped = int32(0)
	running = int32(1)
)

// the ticker type
const (
	rTypePeriodic = 1 // scheduled task will be executed periodically
	rTypeOneTime  = 2 // scheduled task will be executed just once
)

// TickerRoutine define the infos of the scheduled task
type TickerRoutine struct {
	id              string
	handler         RoutineFunc // Executive function
	interval        uint32      // Triggered heartbeat interval
	lastTicker      uint64      // Last executed heartbeat
	status          int32       // The current state : stopped, running
	triggerNextTick int32       // Should be executed in next heartbeat
	rType           int8        // type of the task
}

// GlobalTicker is the schedule tool structure
type GlobalTicker struct {
	beginTime time.Time
	timer     *time.Ticker
	ticker    uint64
	id        string
	routines  sync.Map // key: string, value: *TickerRoutine
}

func NewGlobalTicker(id string) *GlobalTicker {
	ticker := &GlobalTicker{
		id:        id,
		beginTime: time.Now(),
	}

	go ticker.routine()

	return ticker
}

func (gt *GlobalTicker) addRoutine(name string, tr *TickerRoutine) {
	gt.routines.Store(name, tr)
}

func (gt *GlobalTicker) getRoutine(name string) *TickerRoutine {
	if v, ok := gt.routines.Load(name); ok {
		return v.(*TickerRoutine)
	}
	return nil
}

func (gt *GlobalTicker) routine() {
	gt.timer = time.NewTicker(1 * time.Second)
	for range gt.timer.C {
		gt.ticker++
		gt.routines.Range(func(key, value interface{}) bool {
			rt := value.(*TickerRoutine)
			if (atomic.LoadInt32(&rt.status) == running && gt.ticker-rt.lastTicker >= uint64(rt.interval)) || atomic.LoadInt32(&rt.triggerNextTick) == 1 {
				atomic.CompareAndSwapInt32(&rt.triggerNextTick, 1, 0)
				go gt.trigger(rt)
			}
			return true
		})
	}
}

// RegisterPeriodicRoutine registers a specified periodic task to the scheduler instance.
// The task denoted by the routine param will be executed every interval heartbeats (in seconds)
func (gt *GlobalTicker) RegisterPeriodicRoutine(name string, routine RoutineFunc, interval uint32) {
	if rt := gt.getRoutine(name); rt != nil {
		return
	}
	r := &TickerRoutine{
		rType:           rTypePeriodic,
		interval:        interval,
		handler:         routine,
		lastTicker:      gt.ticker,
		id:              name,
		status:          stopped,
		triggerNextTick: 0,
	}
	gt.addRoutine(name, r)
}

// RegisterOneTimeRoutine registers a specified once-task to the scheduler instance.
// It will be executed after the given heartbeats (in seconds) denoted by delay
func (gt *GlobalTicker) RegisterOneTimeRoutine(name string, routine RoutineFunc, delay uint32) {
	if rt := gt.getRoutine(name); rt != nil {
		rt.lastTicker = gt.ticker
		return
	}

	r := &TickerRoutine{
		rType:           rTypeOneTime,
		interval:        delay,
		handler:         routine,
		lastTicker:      gt.ticker,
		id:              name,
		status:          running,
		triggerNextTick: 0,
	}
	gt.addRoutine(name, r)
}

func (gt *GlobalTicker) RemoveRoutine(name string) {
	gt.routines.Delete(name)
}

// StartTickerRoutine starts the specified routine.
// Note that, the task won't work if this function wasn't called after registered
func (gt *GlobalTicker) StartTickerRoutine(name string, triggerNextTicker bool) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}
	atomic.CompareAndSwapInt32(&routine.triggerNextTick, 0, 1)
	atomic.CompareAndSwapInt32(&routine.status, stopped, running)
}

// StartAndTriggerRoutine starts the specified routine.
// Note that, the task won't work if this function wasn't called after registered
func (gt *GlobalTicker) StartAndTriggerRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	atomic.CompareAndSwapInt32(&routine.status, stopped, running)
}

// StopTickerRoutine stops the specified task
func (gt *GlobalTicker) StopTickerRoutine(name string) {
	routine := gt.getRoutine(name)
	if routine == nil {
		return
	}

	atomic.CompareAndSwapInt32(&routine.status, running, stopped)
}

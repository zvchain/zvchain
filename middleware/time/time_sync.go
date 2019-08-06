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

// Package time provides time-zone and local-machine independent time service
package time

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/zvchain/zvchain/common"

	"github.com/beevik/ntp"
	"github.com/zvchain/zvchain/middleware/ticker"
)

// TimeStamp in seconds
type TimeStamp int64

func Int64ToTimeStamp(sec int64) TimeStamp {
	return TimeStamp(sec)
}

func TimeToTimeStamp(t time.Time) TimeStamp {
	return TimeStamp(t.Unix())
}

func (ts TimeStamp) Bytes() []byte {
	return common.Int64ToByte(int64(ts))
}

func (ts TimeStamp) UTC() time.Time {
	return time.Unix(ts.Unix(), 0).UTC()
}

func (ts TimeStamp) Local() time.Time {
	return time.Unix(ts.Unix(), 0).Local()
}

func (ts TimeStamp) Unix() int64 {
	return int64(ts)
}

func (ts TimeStamp) After(t TimeStamp) bool {
	return ts > t
}

func (ts TimeStamp) Since(t TimeStamp) int64 {
	return int64(ts - t)
}

func (ts TimeStamp) Add(sec int64) TimeStamp {
	return ts + Int64ToTimeStamp(sec)
}

func (ts TimeStamp) String() string {
	return ts.Local().String()
}

// ntpServer defines the ntp servers used for time synchronization
var ntpServer = []string{"ntp.aliyun.com", "ntp1.aliyun.com", "ntp2.aliyun.com", "ntp3.aliyun.com", "ntp4.aliyun.com", "ntp4.aliyun.com", "ntp5.aliyun.com", "ntp6.aliyun.com", "ntp7.aliyun.com"}

// TimeSync implements time synchronization from ntp servers
type TimeSync struct {
	currentOffset time.Duration // The offset of the local time to the ntp server
	ticker        *ticker.GlobalTicker
}

// TimeService is a time service, it return a timestamp in seconds
type TimeService interface {
	// Now returns the current timestamp calibrated with ntp server
	Now() TimeStamp

	// Now returns the current timestamp calibrated with ntp server
	NowTime() time.Time

	// Since returns the time duration from the given timestamp to current moment
	Since(t TimeStamp) int64

	// NowAfter checks if current timestamp greater than the given one
	NowAfter(t TimeStamp) bool
}

var TSInstance TimeService

func InitTimeSync() {
	ts := &TimeSync{
		currentOffset: 0,
		ticker:        ticker.NewGlobalTicker("time_sync"),
	}

	ts.ticker.RegisterPeriodicRoutine("time_sync", ts.syncRoutine, 60)
	ts.ticker.StartTickerRoutine("time_sync", false)
	ts.syncRoutine()
	TSInstance = ts
}

func (ts *TimeSync) syncRoutine() bool {
	r := rand.Intn(len(ntpServer))
	rsp, err := ntp.QueryWithOptions(ntpServer[r], ntp.QueryOptions{Timeout: 100 * time.Millisecond})
	if err != nil {
		fmt.Printf("time sync from %v err: %v\n", ntpServer[r], err)
		ts.ticker.StartTickerRoutine("time_sync", true)
		return false
	}
	ts.currentOffset = rsp.ClockOffset
	fmt.Printf("time offset from %v is %v\n", ntpServer[r], ts.currentOffset.String())
	return true
}

// Now returns the current timestamp calibrated with ntp server
func (ts *TimeSync) Now() TimeStamp {
	return TimeToTimeStamp(time.Now().Add(ts.currentOffset).UTC())
}

// Now returns the current timestamp calibrated with ntp server( with nano)
func (ts *TimeSync) NowTime() time.Time {
	return time.Now().Add(ts.currentOffset).UTC()
}


// Since returns the time duration from the given timestamp to current moment
func (ts *TimeSync) Since(t TimeStamp) int64 {
	return ts.Now().Since(t)
}

// NowAfter checks if current timestamp greater than the given one
func (ts *TimeSync) NowAfter(t TimeStamp) bool {
	return ts.Now().After(t)
}

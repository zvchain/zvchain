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
	"sync"
)

type FlowMeterItem struct {
	code  int64
	count int64
	size  int64
}

func newFlowMeterItem(code int64) *FlowMeterItem {

	item := &FlowMeterItem{code: code}

	return item
}

// FlowMeter network dataflow statistics by protocal code
type FlowMeter struct {
	name      string
	sendItems map[int64]*FlowMeterItem
	sendSize  int64
	recvItems map[int64]*FlowMeterItem
	recvSize  int64
	mutex     sync.RWMutex
}

func newFlowMeter(name string) *FlowMeter {

	return &FlowMeter{name: name,
		sendItems: make(map[int64]*FlowMeterItem),
		recvItems: make(map[int64]*FlowMeterItem)}

}

func (fm *FlowMeter) send(code int64, size int64) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	item := fm.sendItems[code]
	if item == nil {
		item = newFlowMeterItem(code)
		fm.sendItems[code] = item
	}
	item.count++
	item.size += size
	fm.sendSize += size
}

func (fm *FlowMeter) recv(code int64, size int64) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	item := fm.recvItems[code]
	if item == nil {
		item = newFlowMeterItem(code)
		fm.recvItems[code] = item
	}
	item.count++
	item.size += size
	fm.recvSize += size
}

func (fm *FlowMeter) reset() {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	fm.sendItems = make(map[int64]*FlowMeterItem)
	fm.recvItems = make(map[int64]*FlowMeterItem)

	fm.sendSize = 0
	fm.recvSize = 0

}

func (fm *FlowMeter) print() {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	if fm.sendSize > 0 {
		Logger.Infof("[FlowMeter][%v_send]  total send size:%v", fm.name, fm.sendSize)
		for _, item := range fm.sendItems {
			Logger.Infof("[FlowMeter][%v_send] code:%v  count:%v  size:%v percentage：%v%%", fm.name, item.code, item.count, item.size, float64(item.size)/float64(fm.sendSize)*100.0)
		}
	}

	if fm.recvSize > 0 {
		Logger.Infof("[FlowMeter][%v_recv]  total recv size:%v", fm.name, fm.recvSize)
		for _, item := range fm.recvItems {
			Logger.Infof("[FlowMeter][%v_recv] code:%v  count:%v  size:%v percentage：%v%%", fm.name, item.code, item.count, item.size, float64(item.size)/float64(fm.recvSize)*100.0)
		}
	}
	return
}

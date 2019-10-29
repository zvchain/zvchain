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
	"time"
)

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
		sendList.lastOnWait = time.Now()
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
	if peer == nil {
		return
	}
	sendList.lastOnWait = time.Now()
	sendList.pendingSend = 0
	sendList.autoSend(peer)
}

func (sendList *SendList) autoSend(peer *Peer) {

	if peer == nil {
		return
	}
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

func (sendList *SendList) reset() int {
	size := 0
	for i := 0; i < MaxSendPriority; i++ {
		item := sendList.list[i]

		item.list = list.New()
	}
	return size
}

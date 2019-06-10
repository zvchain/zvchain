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
	"hash/fnv"
	"sync"
	"time"
)

const BizMessageIDLength = 32

type BizMessageID = [BizMessageIDLength]byte

//MessageManager is a message management
type MessageManager struct {
	messages      map[uint64]time.Time
	bizMessages   map[BizMessageID]time.Time
	index         uint32
	id            NodeID
	forwardNodeID uint32
	mutex         sync.Mutex
}

func decodeMessageInfo(info uint32) (chainID uint16, protocolVersion uint16) {

	chainID = uint16(info >> 16)
	protocolVersion = uint16(info)

	return chainID, protocolVersion
}

func encodeMessageInfo(chainID uint16, protocolVersion uint16) uint32 {

	return uint32(chainID)<<16 | uint32(protocolVersion)
}

func newMessageManager(id NodeID) *MessageManager {

	mm := &MessageManager{
		messages:    make(map[uint64]time.Time),
		bizMessages: make(map[BizMessageID]time.Time),
	}
	mm.id = id
	mm.index = 0
	h := fnv.New32a()
	h.Write(id[:])
	mm.forwardNodeID = uint32(h.Sum32())
	return mm
}

// genMessageID generate a new message id
func (mm *MessageManager) genMessageID() uint64 {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.index++
	messageID := uint64(mm.forwardNodeID)
	messageID = messageID << 32
	messageID = messageID | uint64(mm.index)
	mm.messages[messageID] = time.Now()
	return messageID
}

func (mm *MessageManager) forward(messageID uint64) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.messages[messageID] = time.Now()
}

func (mm *MessageManager) isForwarded(messageID uint64) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	_, ok := mm.messages[messageID]
	return ok
}

func (mm *MessageManager) forwardBiz(messageID BizMessageID) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.bizMessages[messageID] = time.Now()
}

func (mm *MessageManager) isForwardedBiz(messageID BizMessageID) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	_, ok := mm.bizMessages[messageID]
	return ok
}

func (mm *MessageManager) byteToBizID(bid []byte) BizMessageID {
	var id [BizMessageIDLength]byte
	for i := 0; i < len(bid) && i < BizMessageIDLength; i++ {
		id[i] = bid[i]
	}
	return id
}

func (mm *MessageManager) clear() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	now := time.Now()
	MessageCacheTime := 5 * time.Minute

	for mid, t := range mm.messages {
		if now.Sub(t) > MessageCacheTime {
			delete(mm.messages, mid)
		}
	}

	for mid, t := range mm.bizMessages {
		if now.Sub(t) > MessageCacheTime {
			delete(mm.bizMessages, mid)
		}
	}

	return
}

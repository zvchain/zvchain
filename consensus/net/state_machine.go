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

package net

import (
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
)

type stateHandleFunc func(msg interface{})

type stateNode struct {
	//state code, unique in a machine
	code uint32

	//The minimum number of repetitions that need to occur
	//in order to transit to the next state
	leastRepeat int32

	//The maximum number of repetitions that would occur
	//at the state
	mostRepeat int32

	//the state transit handler func
	handler stateHandleFunc
	next    *stateNode

	currentIdx int32
	execNum    int32

	//future state msgs cached in the queue
	queue []*stateMsg
}

type stateMsg struct {
	Code uint32
	Data interface{}
	ID   string
}

type stateMachine struct {
	ID      string
	Current *stateNode
	Head    *stateNode
	Time    time.Time
	lock    sync.Mutex
}

type stateMachines struct {
	name      string
	machines  *lru.Cache
	generator StateMachineGenerator
	ticker    *ticker.GlobalTicker
}

var GroupInsideMachines stateMachines

var logger taslog.Logger

func initStateMachines() {
	logger = taslog.GetLoggerByIndex(taslog.StateMachineLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	GroupInsideMachines = stateMachines{
		name:      "GroupInsideMachines",
		generator: &groupInsideMachineGenerator{},
		machines:  common.MustNewLRUCache(50),
		ticker:    ticker.NewGlobalTicker("state_machine"),
	}

	GroupInsideMachines.startCleanRoutine()

}

func newStateMsg(code uint32, data interface{}, id string) *stateMsg {
	return &stateMsg{
		Code: code,
		Data: data,
		ID:   id,
	}
}

func newStateNode(st uint32, lr, mr int, h stateHandleFunc) *stateNode {
	return &stateNode{
		code:        st,
		leastRepeat: int32(lr),
		mostRepeat:  int32(mr),
		queue:       make([]*stateMsg, 0),
		handler:     h,
	}
}

func newStateMachine(id string) *stateMachine {
	return &stateMachine{
		ID:   id,
		Time: time.Now(),
	}
}

func (n *stateNode) queueSize() int32 {
	return int32(len(n.queue))
}

func (n *stateNode) state() string {
	return fmt.Sprintf("%v[%v/%v]", n.code, n.currentIdx, n.leastRepeat)
}

func (n *stateNode) dataIndex(id string) int32 {
	for idx, d := range n.queue {
		if d.ID == id {
			return int32(idx)
		}
	}
	return -1
}

func (n *stateNode) addData(stateMsg *stateMsg) (int32, bool) {
	idx := n.dataIndex(stateMsg.ID)
	if idx >= 0 {
		return idx, false
	}
	n.queue = append(n.queue, stateMsg)
	return int32(len(n.queue)) - 1, true
}

func (n *stateNode) leastFinished() bool {
	return n.currentIdx >= n.leastRepeat
}

func (n *stateNode) mostFinished() bool {
	return n.execNum >= n.mostRepeat
}

func (m *stateMachine) findTail() *stateNode {
	p := m.Head
	for p != nil && p.next != nil {
		p = p.next
	}
	return p
}

func (m *stateMachine) currentNode() *stateNode {
	return m.Current
}

func (m *stateMachine) setCurrent(node *stateNode) {
	m.Current = node
}

func (m *stateMachine) appendNode(node *stateNode) {
	if node == nil {
		// this must not happen
		panic("cannot add nil node to the state machine!")
	}

	tail := m.findTail()
	if tail == nil {
		m.setCurrent(node)
		m.Head = node
	} else {
		tail.next = node
	}
}

func (m *stateMachine) findNode(code uint32) *stateNode {
	p := m.Head
	for p != nil && p.code != code {
		p = p.next
	}
	return p
}

func (m *stateMachine) finish() bool {
	current := m.currentNode()
	return current.next == nil && current.leastFinished()
}

func (m *stateMachine) allFinished() bool {
	for n := m.Head; n != nil; n = n.next {
		if !n.mostFinished() {
			return false
		}
	}
	return true
}

func (m *stateMachine) expire() bool {
	return int(time.Since(m.Time).Seconds()) >= model.Param.GroupInitMaxSeconds
}

func (m *stateMachine) doTransform() {
	node := m.currentNode()
	qs := node.queueSize()

	d := qs - node.currentIdx
	switch d {
	case 0:
		return
	case 1:
		msg := node.queue[node.currentIdx]
		node.handler(msg.Data)
		// Free memory
		node.queue[node.currentIdx].Data = true
		node.currentIdx++
		node.execNum++
		logger.Debugf("machine %v handling exec state %v, from %v", m.ID, node.state(), msg.ID)
	default:
		wg := sync.WaitGroup{}
		for node.currentIdx < qs {
			msg := node.queue[node.currentIdx]
			wg.Add(1)
			go func() {
				defer wg.Done()
				node.handler(msg.Data)
				// Free memory
				msg.Data = true
			}()
			node.currentIdx++
			node.execNum++
			logger.Debugf("machine %v handling exec state %v in parallel, from %v", m.ID, node.state(), msg.ID)
		}
		wg.Wait()
	}

	if node.leastFinished() && node.next != nil {
		m.setCurrent(node.next)
		m.doTransform()
	}

}

func (m *stateMachine) transform(msg *stateMsg) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	defer func() {
		if !m.finish() {
			curr := m.currentNode()
			logger.Debugf("machine %v waiting state %v[%v/%v]", m.ID, curr.code, curr.currentIdx, curr.leastRepeat)
		} else {
			logger.Debugf("machine %v finished", m.ID)
		}
	}()
	node := m.findNode(msg.Code)
	if node == nil {
		return false
	}
	if node.code < m.currentNode().code {
		logger.Debugf("machine %v handle pre state %v, exec state %v", m.ID, node.code, m.currentNode().state())
		node.handler(msg.Data)
		node.execNum++
	} else if node.code > m.currentNode().code {
		logger.Debugf("machine %v cache future state %v from %v, current state %v", m.ID, node.code, msg.ID, m.currentNode().state())
		node.addData(msg)
	} else {
		_, add := node.addData(msg)
		if !add {
			logger.Debugf("machine %v ignore redundant state %v, current state %v", m.ID, node.code, m.currentNode().state())
			return false
		}
		m.doTransform()
	}
	return true
}

type StateMachineGenerator interface {
	Generate(id string, cnt int) *stateMachine
}

type groupInsideMachineGenerator struct{}

func (m *groupInsideMachineGenerator) Generate(id string, cnt int) *stateMachine {
	machine := newStateMachine(id)
	memNum := cnt
	machine.appendNode(newStateNode(network.GroupInitMsg, 1, 1, func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInit(msg.(*model.ConsensusGroupRawMessage))
	}))
	machine.appendNode(newStateNode(network.KeyPieceMsg, memNum, memNum, func(msg interface{}) {
		MessageHandler.processor.OnMessageSharePiece(msg.(*model.ConsensusSharePieceMessage))
	}))
	machine.appendNode(newStateNode(network.SignPubkeyMsg, 1, memNum, func(msg interface{}) {
		MessageHandler.processor.OnMessageSignPK(msg.(*model.ConsensusSignPubKeyMessage))
	}))
	machine.appendNode(newStateNode(network.GroupInitDoneMsg, model.Param.GetGroupK(memNum), model.Param.GetGroupK(memNum), func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInited(msg.(*model.ConsensusGroupInitedMessage))
	}))
	return machine
}

func (stm *stateMachines) startCleanRoutine() {
	stm.ticker.RegisterPeriodicRoutine(stm.name, stm.cleanRoutine, 2)
	stm.ticker.StartTickerRoutine(stm.name, false)
}

func (stm *stateMachines) cleanRoutine() bool {
	for _, k := range stm.machines.Keys() {
		id := k.(string)
		value, ok := stm.machines.Get(id)
		if !ok {
			continue
		}
		m := value.(*stateMachine)
		if m.allFinished() {
			logger.Infof("%v state machine allFinished, id=%v", stm.name, m.ID)
			stm.machines.Remove(m.ID)
		}
		if m.expire() {
			logger.Infof("%v state machine expire, id=%v", stm.name, m.ID)
			stm.machines.Remove(m.ID)
		}
	}
	return true
}

func (stm *stateMachines) GetMachine(id string, cnt int) *stateMachine {
	if v, ok := stm.machines.Get(id); ok {
		return v.(*stateMachine)
	}
	m := stm.generator.Generate(id, cnt)
	contains, _ := stm.machines.ContainsOrAdd(id, m)
	if !contains {
		return m
	}
	if v, ok := stm.machines.Get(id); ok {
		return v.(*stateMachine)
	}
	// this case must not happen
	panic("get machine fail, id " + id)
}

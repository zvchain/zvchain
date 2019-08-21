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
	"math"
	"math/rand"
	nnet "net"
	"sort"
	"sync"
	"time"
)

const GroupMinRowSize = 4

func groupRowSize(groupSize int) int {
	rowSize := int(math.Ceil(math.Sqrt(float64(groupSize))))
	if rowSize < GroupMinRowSize {
		rowSize = GroupMinRowSize
	}
	return rowSize
}

func groupColumnSendCount(groupSize int) int {
	sendSize := int(math.Ceil(float64(groupRowSize(groupSize)) / 2))

	return sendSize
}

func genGroupRandomEntranceNodes(members []string) []NodeID {

	totalSize := len(members)

	nodesIndex := make([]int, 0)
	nodes := make([]NodeID, 0)

	connectNodes := make([]NodeID, 0)

	if totalSize == 0 {
		return nodes
	}
	maxSize := groupColumnSendCount(totalSize)

	for i := 0; i < totalSize; i++ {
		ID := NewNodeID(members[i])
		if ID == nil || *ID == netCore.ID {
			continue
		}

		p := netCore.peerManager.peerByID(*ID)
		if p != nil && p.sessionID > 0 {
			continue
		}
		connectNodes = append(connectNodes, *ID)
	}

	if len(connectNodes) >= maxSize {
		return connectNodes[0:maxSize]
	}

	rowSize := groupRowSize(totalSize)

	rowCount := int(math.Ceil(float64(totalSize) / float64(rowSize)))

	columnIndex := rand.Intn(rowCount)
	nIndex := columnIndex * rowSize
	nID := NewNodeID(members[nIndex])
	if nID != nil {
		nodesIndex = append(nodesIndex, nIndex)
		nodes = append(nodes, *nID)
	}

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < totalSize; i++ {
		peerIndex := rand.Intn(totalSize)
		columnIndex := peerIndex % rowSize
		rowIndex := int(math.Floor(float64(peerIndex) / float64(rowSize)))

		selected := true
		for n := 0; n < len(nodesIndex); n++ {
			indexSelected := nodesIndex[n]
			columnIndexSelected := indexSelected % rowSize
			rowIndexSelected := int(math.Floor(float64(indexSelected) / float64(rowSize)))
			if rowIndex == rowIndexSelected || columnIndex == columnIndexSelected {
				selected = false
				break
			}
		}
		if selected {
			nID := NewNodeID(members[peerIndex])
			if nID != nil {
				nodesIndex = append(nodesIndex, peerIndex)
				nodes = append(nodes, *nID)
			}
		}
		if len(nodesIndex) >= maxSize-len(connectNodes) {
			break
		}
	}
	nodes = append(nodes, connectNodes...)
	return nodes
}

// Group network is Ring topology network with several accelerate links,to implement group broadcast
type Group struct {
	ID               string
	members          []NodeID
	needConnectNodes []NodeID // the nodes group network need connect
	mutex            sync.Mutex
	resolvingNodes   map[NodeID]time.Time //nodes is finding in kad

	curIndex int //current node index of this group

	rowSize  int
	rowCount int

	rowIndex    int
	columnIndex int
	rowNodes    []NodeID
	columnNodes []NodeID
}

func (g *Group) Len() int {
	return len(g.members)
}

func (g *Group) Less(i, j int) bool {
	return g.members[i].GetHexString() < g.members[j].GetHexString()
}

func (g *Group) Swap(i, j int) {
	g.members[i], g.members[j] = g.members[j], g.members[i]
}

func newGroup(ID string, members []NodeID) *Group {

	g := &Group{ID: ID, members: members, needConnectNodes: make([]NodeID, 0), resolvingNodes: make(map[NodeID]time.Time)}

	Logger.Infof("[group]new group ID：%v", ID)
	g.genConnectNodes()
	return g
}

func (g *Group) rebuildGroup(members []NodeID) {

	Logger.Infof("[group]rebuild group ID：%v", g.ID)
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.members = members
	g.genConnectNodes()

	go g.doRefresh()
}

func (g *Group) onRemove() {

	Logger.Infof("[group]group on remove  group ID：%v", g.ID)
	g.mutex.Lock()
	defer g.mutex.Unlock()
	memberSize := len(g.needConnectNodes)

	for i := 0; i < memberSize; i++ {
		ID := g.needConnectNodes[i]
		if ID == netCore.ID {
			continue
		}
		p := netCore.peerManager.peerByID(ID)
		if p == nil {
			continue
		}
		p.removeGroup(g.ID)
		if p.isGroupEmpty() {
			node := netCore.kad.find(ID)
			if node == nil {
				Logger.Infof("[group]group on remove, member ID: %v", ID)
				netCore.peerManager.disconnect(ID)
			}
		}
	}

}

// genConnectNodes Generate the nodes group work need to connect
func (g *Group) genConnectNodes() {

	groupSize := len(g.members)
	if groupSize == 0 {
		return
	}
	g.needConnectNodes = make([]NodeID, 0)
	g.rowNodes = make([]NodeID, 0)
	g.columnNodes = make([]NodeID, 0)
	sort.Sort(g)
	g.curIndex = 0
	for i := 0; i < len(g.members); i++ {
		if g.members[i] == netCore.ID {
			g.curIndex = i
			break
		}
	}

	Logger.Infof("[group][genConnectNodes] curIndex: %v", g.curIndex)
	for i := 0; i < len(g.members); i++ {
		Logger.Infof("[group][genConnectNodes] members ID: %v", g.members[i].GetHexString())
	}

	g.rowSize = groupRowSize(groupSize)

	g.rowCount = int(math.Ceil(float64(groupSize) / float64(g.rowSize)))
	g.rowIndex = int(math.Floor(float64(g.curIndex) / float64(g.rowSize)))
	g.columnIndex = g.curIndex % g.rowSize

	g.rowNodes = make([]NodeID, 0)

	for i := 0; i < g.rowSize; i++ {
		index := g.rowIndex*g.rowSize + i
		Logger.Infof("[group][genConnectNodes] row, i : %v ,index:%v", i, index)
		if index >= groupSize {
			break
		}
		if index != g.curIndex {
			g.rowNodes = append(g.rowNodes, g.members[index])
			g.needConnectNodes = append(g.needConnectNodes, g.members[index])
			Logger.Infof("[group][genConnectNodes] row member ID: %v", g.members[index].GetHexString())
		}
	}

	for i := 0; i < g.rowCount; i++ {
		index := i*g.rowSize + g.columnIndex
		Logger.Infof("[group][genConnectNodes] column, i : %v ,index:%v", i, index)
		if index >= groupSize {
			break
		}
		if index != g.curIndex {
			g.columnNodes = append(g.columnNodes, g.members[index])
			g.needConnectNodes = append(g.needConnectNodes, g.members[index])
			Logger.Infof("[group][genConnectNodes] column member ID: %v", g.members[index].GetHexString())
		}
	}
	Logger.Infof("[group][genConnectNodes] row size: %v, row count:%v,"+
		" row Index:%v column index:%v rowNodesCount:%v, columnNodesCount:%v",
		g.rowSize, g.rowCount, g.rowIndex, g.columnIndex, len(g.rowNodes), len(g.columnNodes))

}

// doRefresh Check all nodes need to connect is connecting，if not then connect that node
func (g *Group) doRefresh() {

	g.mutex.Lock()
	defer g.mutex.Unlock()

	memberSize := len(g.needConnectNodes)

	for i := 0; i < memberSize; i++ {
		ID := g.needConnectNodes[i]
		if ID == netCore.ID {
			continue
		}

		p := netCore.peerManager.peerByID(ID)
		if p != nil && p.sessionID > 0 && p.isAuthSucceed {
			p.addGroup(g.ID)
			continue
		}

		go netCore.ping(ID, nil)

		Logger.Debugf("[group] group doRefresh  ping node ID：%v ", ID.GetHexString())
	}
}

func (g *Group) resolve(ID NodeID) {
	resolveTimeout := 3 * time.Minute
	t, ok := g.resolvingNodes[ID]
	if ok && time.Since(t) < resolveTimeout {
		return
	}
	g.resolvingNodes[ID] = time.Now()
	go netCore.kad.resolve(ID)
}

func sendNodes(nodes []NodeID, packet *bytes.Buffer, code uint32) {
	if packet == nil {
		return
	}

	for i := 0; i < len(nodes); i++ {
		ID := nodes[i]
		if ID == netCore.ID {
			continue
		}
		p := netCore.peerManager.peerByID(ID)
		if p != nil {
			netCore.peerManager.write(ID, &nnet.UDPAddr{IP: p.IP, Port: int(p.Port)}, packet, code)
		} else {
			Logger.Debugf("[group] SendGroup node not connected : ID：%v", ID.GetHexString())
			netCore.peerManager.write(ID, nil, packet, code)
		}
	}
	netCore.bufferPool.freeBuffer(packet)
}

func (g *Group) sendGroupMessage(msgType DataType, nodes []NodeID, msg *MsgData) {
	Logger.Infof("[group] sendGroupMessage type:%v,nodes size:%v", msgType, len(nodes))

	msg.DataType = msgType
	buffer, _, err := netCore.encodePacket(MessageType_MessageData, msg)
	if err != nil {
		Logger.Errorf("[group] on group broadcast encode packet error：%v", err)
		return
	}
	if buffer != nil {
		sendNodes(nodes, buffer, msg.MessageCode)
	}
}

func (g *Group) Broadcast(msg *MsgData) {

	Logger.Infof("[group] Broadcast ID:%v", g.ID)
	if msg == nil {
		Logger.Infof("[group] Broadcast ID:%v ,msg is nil", g.ID)
		return
	}
	g.sendGroupMessage(DataType_DataGroupColumn, g.columnNodes, msg)

	groupSendCount := int(math.Ceil(float64(g.rowSize)/2)) - 1

	groupMsgMap := make(map[int]bool)

	if g.columnIndex != 0 { //if 0 position is not sent, keep it sent.
		groupMsgMap[0] = true
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for len(groupMsgMap) < groupSendCount {
		column := rand.Intn(g.rowSize)
		if !groupMsgMap[column] && column != g.columnIndex {
			groupMsgMap[column] = true
		}
	}

	groupMsgNodes := make([]NodeID, 0)
	rowMsgNodes := make([]NodeID, 0)
	for i := 0; i < len(g.rowNodes); i++ {
		if groupMsgMap[i] {
			groupMsgNodes = append(groupMsgNodes, g.rowNodes[i])
		} else {
			rowMsgNodes = append(rowMsgNodes, g.rowNodes[i])
		}
	}
	Logger.Infof("[group] Broadcast ID:%v, groupSendCount:%v, group msg count:%v, row msg count:%v ", g.ID, groupSendCount, len(groupMsgNodes), len(rowMsgNodes))

	if len(groupMsgNodes) > 0 {
		g.sendGroupMessage(DataType_DataGroup, groupMsgNodes, msg)
	}
	if len(rowMsgNodes) > 0 {
		g.sendGroupMessage(DataType_DataGroupRow, rowMsgNodes, msg)
	}

}

func (g *Group) onBroadcast(msg *MsgData) {
	Logger.Infof("[group] onBroadcast ID:type:%v type:%v", g.ID, msg.DataType)
	if msg == nil {
		Logger.Infof("[group] onBroadcast ID:%v ,msg is nil", g.ID)
		return
	}
	sendColumn := false
	sendRow := false
	if msg.DataType == DataType_DataGroup {
		sendColumn = true
		sendRow = true
	} else if msg.DataType == DataType_DataGroupColumn {
		sendRow = true
	}

	if sendColumn {
		g.sendGroupMessage(DataType_DataGroupColumn, g.columnNodes, msg)
	}

	if sendRow {
		g.sendGroupMessage(DataType_DataGroupRow, g.rowNodes, msg)
	}
}

// GroupManager represents group management
type GroupManager struct {
	groups map[string]*Group
	mutex  sync.RWMutex
}

func newGroupManager() *GroupManager {

	gm := &GroupManager{
		groups: make(map[string]*Group),
	}
	return gm
}

//buildGroup create a group, or rebuild the group network if the group already exists
func (gm *GroupManager) buildGroup(ID string, members []NodeID) *Group {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	Logger.Infof("[group] build group, ID:%v, count:%v", ID, len(members))

	g, isExist := gm.groups[ID]
	if !isExist {
		g = newGroup(ID, members)
		gm.groups[ID] = g
	} else {
		g.rebuildGroup(members)
	}
	go g.doRefresh()
	return g
}

//RemoveGroup remove the group
func (gm *GroupManager) removeGroup(ID string) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	Logger.Infof("[group] remove group, ID:%v.", ID)
	g := gm.groups[ID]
	if g == nil {
		Logger.Infof("[group] group not found.")
		return
	}
	g.onRemove()
	delete(gm.groups, ID)
}

func (gm *GroupManager) doRefresh() {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	for _, group := range gm.groups {
		go group.doRefresh()
	}
}

func (gm *GroupManager) onBroadcast(ID string, msg *MsgData) {

	Logger.Infof("[group] on group broadcast, ID:%v ,type:%v", ID, msg.DataType)
	if msg == nil {
		Logger.Errorf("[group] on group broadcast, msg is nil, ID:%v ", ID)
		return
	}
	gm.mutex.RLock()
	g := gm.groups[ID]
	if g == nil {
		Logger.Infof("[group] on group broadcast, group not found.")
		gm.mutex.RUnlock()
		return
	}

	gm.mutex.RUnlock()

	g.onBroadcast(msg)
}

func (gm *GroupManager) Broadcast(ID string, msg *MsgData, members []string, code uint32) {
	if msg == nil {
		Logger.Errorf("[group] group broadcast,msg is nil, ID:%v code:%v", ID, code)
		return
	}
	Logger.Infof("[group] group broadcast, ID:%v code:%v", ID, code)
	gm.mutex.RLock()
	g := gm.groups[ID]
	if g != nil {
		gm.mutex.RUnlock()
		g.Broadcast(msg)
		return
	}
	gm.mutex.RUnlock()

	gm.BroadcastExternal(ID, msg, members, code)
}

func (gm *GroupManager) BroadcastExternal(ID string, msg *MsgData, members []string, code uint32) {

	Logger.Infof("[group] group external broadcast, ID:%v code:%v", ID, code)
	if msg == nil {
		Logger.Errorf("[group] group external broadcast,msg is nil, ID:%v code:%v", ID, code)
		return
	}
	gm.mutex.RLock()
	g := gm.groups[ID]
	if g != nil {
		gm.mutex.RUnlock()
		g.Broadcast(msg)
		return
	}
	gm.mutex.RUnlock()

	msg.DataType = DataType_DataGroup
	groupBuffer, _, err := netCore.encodePacket(MessageType_MessageData, msg)
	if err != nil {
		Logger.Errorf("[group] on group external broadcast encode column packet error：%v", err)
		return
	}
	if groupBuffer == nil {
		Logger.Errorf("[group] on group external broadcast encode column packet is nil")
		return
	}

	nodes := genGroupRandomEntranceNodes(members)
	sendNodes(nodes, groupBuffer, msg.MessageCode)
}

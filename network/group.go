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
	nnet "net"
	"sort"
	"sync"
	"time"
)

const GroupBaseConnectNodeCount = 2

// Group network is Ring topology network with several accelerate links,to implement group broadcast
type Group struct {
	ID               string
	members          []NodeID
	needConnectNodes []NodeID // the nodes group network need connect
	mutex            sync.Mutex
	resolvingNodes   map[NodeID]time.Time //nodes is finding in kad
	curIndex         int                  //current node index of this group
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

	Logger.Infof("new group ID：%v", ID)
	g.genConnectNodes()
	return g
}

func (g *Group) rebuildGroup(members []NodeID) {

	Logger.Infof("rebuild group ID：%v", g.ID)
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.members = members
	g.genConnectNodes()

	go g.doRefresh()
}
func (g *Group) onRemove() {

	Logger.Infof("group on remove  group ID：%v", g.ID)
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
				Logger.Infof("group on remove, member ID: %v", ID)
				//netCore.peerManager.disconnect(ID)
			}
		}
	}

}

// genConnectNodes Generate the nodes group work need to connect
// at first sort group members,get current node index in this group,then add next two nodes to connect list
// then calculate accelerate link nodes,add to connect list
func (g *Group) genConnectNodes() {

	peerSize := len(g.members)
	if peerSize == 0 {
		return
	}
	g.needConnectNodes = make([]NodeID, 0, 0)
	sort.Sort(g)
	g.curIndex = 0
	for i := 0; i < len(g.members); i++ {
		if g.members[i] == netCore.ID {
			g.curIndex = i
			break
		}
	}

	Logger.Debugf("[genConnectNodes] curIndex: %v", g.curIndex)
	for i := 0; i < len(g.members); i++ {
		Logger.Debugf("[genConnectNodes] members ID: %v", g.members[i].GetHexString())
	}

	connectCount := GroupBaseConnectNodeCount
	if connectCount > len(g.members)-1 {
		connectCount = len(g.members) - 1
	}

	nextIndex := g.getNextIndex(g.curIndex)
	g.needConnectNodes = append(g.needConnectNodes, g.members[nextIndex])
	if peerSize >= 5 {

		nextIndex = g.getNextIndex(nextIndex)
		g.needConnectNodes = append(g.needConnectNodes, g.members[nextIndex])

		maxCount := int(math.Sqrt(float64(peerSize)) * 0.8)
		maxCount -= len(g.needConnectNodes)
		step := 1

		if maxCount > 0 {
			step = len(g.members) / maxCount
		}

		for i := 0; i < maxCount; i++ {
			nextIndex += step
			if nextIndex >= len(g.members) {
				nextIndex %= len(g.members)
			}
			g.needConnectNodes = append(g.needConnectNodes, g.members[nextIndex])
		}
	}

	for i := 0; i < len(g.needConnectNodes); i++ {
		Logger.Debugf("[genConnectNodes] needConnectNodes ID: %v", g.needConnectNodes[i].GetHexString())
	}

}

func (g *Group) getNextIndex(index int) int {
	index = index + 1
	if index >= len(g.members) {
		index = 0
	}
	return index
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
		if p != nil && p.sessionID > 0 {
			p.addGroup(g.ID)
			continue
		}
		node := netCore.kad.find(ID)
		if node != nil && node.IP != nil && node.Port > 0 {
			Logger.Debugf("Group doRefresh node found in KAD ID：%v ip: %v  port:%v", ID.GetHexString(), node.IP, node.Port)
			go netCore.ping(node.ID, &nnet.UDPAddr{IP: node.IP, Port: int(node.Port)})
		} else {
			go netCore.ping(ID, nil)

			Logger.Debugf("Group doRefresh node can not find in KAD ,resolve ....  ID：%v ", ID.GetHexString())
			g.resolve(ID)
		}
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

func (g *Group) send(packet *bytes.Buffer, code uint32) {
	if packet == nil {
		return
	}
	Logger.Debugf("Group Send ID：%v ", g.ID)

	for i := 0; i < len(g.needConnectNodes); i++ {
		ID := g.needConnectNodes[i]
		if ID == netCore.ID {
			continue
		}
		p := netCore.peerManager.peerByID(ID)
		if p != nil {
			netCore.peerManager.write(ID, &nnet.UDPAddr{IP: p.IP, Port: int(p.Port)}, packet, code, false)
		} else {
			node := netCore.kad.find(ID)
			if node != nil && node.IP != nil && node.Port > 0 {
				Logger.Debugf("SendGroup node not connected ,but in KAD : ID：%v ip: %v  port:%v", ID.GetHexString(), node.IP, node.Port)
				netCore.peerManager.write(node.ID, &nnet.UDPAddr{IP: node.IP, Port: int(node.Port)}, packet, code, false)
			} else {
				Logger.Debugf("SendGroup node not connected and not in KAD : ID：%v", ID.GetHexString())
				netCore.peerManager.write(ID, nil, packet, code, false)
			}
		}
	}
	netCore.bufferPool.freeBuffer(packet)
	return
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

	Logger.Infof("build group, ID:%v, count:%v", ID, len(members))

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

	Logger.Debugf("remove group, ID:%v.", ID)
	g := gm.groups[ID]
	if g == nil {
		Logger.Infof("group not found.")
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

func (gm *GroupManager) groupBroadcast(ID string, packet *bytes.Buffer, code uint32) {
	if packet == nil {
		return
	}
	Logger.Infof("group broadcast, ID:%v code:%v", ID, code)
	gm.mutex.RLock()
	g := gm.groups[ID]
	if g == nil {
		Logger.Infof("group not found.")
		gm.mutex.RUnlock()
		return
	}
	buf := netCore.bufferPool.getBuffer(packet.Len())
	buf.Write(packet.Bytes())
	gm.mutex.RUnlock()

	g.send(buf, code)
}

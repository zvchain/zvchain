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
	"errors"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/zvchain/zvchain/log"

	"github.com/zvchain/zvchain/common"
)

const (
	BasePort = 22000

	SuperBasePort = 1122

	NodeIDLength = 32
)

type NodeID [NodeIDLength]byte

func (netID NodeID) IsValid() bool {
	for i := 0; i < NodeIDLength; i++ {
		if netID[i] > 0 {
			return true
		}
	}
	return false
}

func (nid NodeID) GetHexString() string {
	return common.ToHex(nid.Bytes())
}

func NewNodeID(hex string) *NodeID {
	var nid NodeID

	if len(hex) == 0 {
		return nil
	}
	if !strings.HasPrefix(hex, "0x") {
		hex = "0x" + hex
	}
	nid.SetBytes(common.FromHex(hex))
	return &nid
}

func (nid *NodeID) SetBytes(b []byte) {
	if len(nid) < len(b) {
		b = b[:len(nid)]
	}
	copy(nid[:], b)
}

func (nid NodeID) Bytes() []byte {
	return nid[:]
}

// Node Kad node struct
type Node struct {
	ID      NodeID
	IP      net.IP
	Port    int
	NatType int

	sha     []byte
	addedAt time.Time
	fails   int
	pingAt  time.Time
	pinged  bool
}

// NewNode create a new node
func NewNode(ID NodeID, IP net.IP, port int) *Node {
	if ipv4 := IP.To4(); ipv4 != nil {
		IP = ipv4
	}
	return &Node{
		IP:   IP,
		Port: port,
		ID:   ID,
		sha:  makeSha256Hash(ID[:]),
	}
}

func (n *Node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP, Port: int(n.Port)}
}

// Incomplete is address is Incomplete
func (n *Node) Incomplete() bool {
	return n.IP == nil
}

func (n *Node) validateComplete() error {
	if n.Incomplete() {
		return errors.New("incomplete node")
	}
	if n.Port == 0 {
		return errors.New("missing port")
	}

	if n.IP.IsMulticast() || n.IP.IsUnspecified() {
		return errors.New("invalid IP (multicast/unspecified)")
	}
	return nil
}

func distanceCompare(target, a, b []byte) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

var leadingZeroCount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

func logDistance(a, b []byte) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += leadingZeroCount[x]
			break
		}
	}
	return len(a)*8 - lz
}

// InitSelfNode initialize local user's node
func InitSelfNode(isSuper bool, ID NodeID) (*Node, error) {
	ip := getLocalIP()
	port := SuperBasePort
	if !isSuper {
		port = getAvailablePort(ip, BasePort)
	}

	n := Node{ID: ID, IP: net.ParseIP(ip), Port: port}
	log.DefaultLogger.Info(n.String())

	return &n, nil
}

// getLocalIP is get intranet IP
func getLocalIP() string {
	addrs, _ := net.InterfaceAddrs()

	for _, address := range addrs {
		// Check the IP address to determine whether to loop the address
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getAvailablePort(ip string, port int) int {
	if port < 1024 {
		port = BasePort
	}

	if port > 65535 {
		Logger.Debugf("[Network]No available port!")
		return -1
	}

	rand.Seed(time.Now().UnixNano())
	port += rand.Intn(1000)

	return port
}

//String return  node detail description
func (n *Node) String() string {
	str := "Self node net info:\n" + "ID is:" + n.ID.GetHexString() + "\nIP is:" + n.IP.String() + "\nTcp port is:" + strconv.Itoa(n.Port) + "\n"
	return str
}

func getIPByAddress(address string) (net.IP, error) {

	IP := net.ParseIP(address)
	if IP != nil {
		return IP, nil
	}
	IPs, err := net.LookupIP(address)
	if err != nil || len(IPs) == 0 {
		Logger.Errorf("network  address :%v, LookupIP error:%v", address, err.Error())
		return nil, err
	}

	return IPs[0], nil
}

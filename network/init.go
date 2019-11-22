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

// Package network module implements p2p network, It uses a Kademlia-like protocol to maintain and discover Nodes.
// network transfer protocol use  KCP, a open source RUDP implementation,it provide NAT Traversal ability,let nodes
// under NAT can be connecting with other.
package network

import (
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/statistics"
)

// NetworkConfig is the network configuration
type NetworkConfig struct {
	NodeIDHex       string
	NatAddr         string
	NatPort         uint16
	SeedAddr        string
	ChainID         uint16 // Chain id
	ProtocolVersion uint16 // Protocol version
	TestMode        bool
	IsSuper         bool
	SeedIDs         []string
	PK              string
	SK              string
}

const (
	configMaxBroadcastCount = "max_broadcast_count"
	maxBroadcastCount       = 256
	configSection           = "p2p"
)

var netServerInstance *Server

var Logger *logrus.Logger

// Init initialize network instance,register message handler,join p2p network
func Init(config *common.ConfManager, consensusHandler MsgHandler, networkConfig NetworkConfig) (err error) {
	Logger = log.P2PLogger
	if config != nil {
		statistics.InitStatistics(*config)
	}

	nodeID := NewNodeID(networkConfig.NodeIDHex)
	if nodeID == nil {
		Logger.Error("Node ID is nil ")
		return errBadPeer
	}
	self, err := InitSelfNode(networkConfig.IsSuper, *nodeID)
	if err != nil {
		Logger.Error("InitSelfNode error:", err.Error())
		return err
	}

	if networkConfig.SeedAddr == "" {
		networkConfig.SeedAddr = self.IP.String()
	}

	seedPort := SuperBasePort

	seeds := make([]*Node, 0, 1)

	listenAddr := net.UDPAddr{IP: self.IP, Port: self.Port}

	var natEnable bool
	if networkConfig.TestMode {
		natEnable = false
		listenIP, err := getIPByAddress(networkConfig.SeedAddr)
		if err != nil || listenIP == nil {
			Logger.Errorf("Network SeedAddr:%v is wrong:%v", networkConfig.SeedAddr, err.Error())
			return err
		}
		listenAddr = net.UDPAddr{IP: listenIP, Port: self.Port}
		seedID := ""
		if len(networkConfig.SeedIDs) > 0 {
			seedID = networkConfig.SeedIDs[0]
		}
		Logger.Errorf("Seed ID:%v ", seedID)

		if !networkConfig.IsSuper {
			nID := NewNodeID(seedID)
			if nID != nil {
				bnNode := NewNode(*nID, net.ParseIP(networkConfig.SeedAddr), seedPort)
				if bnNode.ID != self.ID {
					seeds = append(seeds, bnNode)
				}
			}
		}
	} else {
		natEnable = true
		randomSeeds := genRandomSeeds(networkConfig.SeedIDs)
		for _, sid := range randomSeeds {
			nID := NewNodeID(sid)
			if nID != nil {
				bnNode := NewNode(*nID, net.ParseIP(networkConfig.SeedAddr), seedPort)
				Logger.Errorf("Seed ID:%v ", sid)

				if bnNode.ID != self.ID {
					seeds = append(seeds, bnNode)
				}
			}
		}
	}
	natIP := ""
	if len(networkConfig.NatAddr) > 0 {
		IP, err := getIPByAddress(networkConfig.NatAddr)
		if err != nil || IP == nil {
			Logger.Errorf("Network Lookup NatAddr:%v is wrong:%v", networkConfig.SeedAddr, err.Error())
			return err
		}
		natIP = IP.String()
	}

	netConfig := NetCoreConfig{ID: self.ID,
		ListenAddr:         &listenAddr,
		Seeds:              seeds,
		NatTraversalEnable: natEnable,
		NatIP:              natIP,
		NatPort:            networkConfig.NatPort,
		ChainID:            networkConfig.ChainID,
		ProtocolVersion:    networkConfig.ProtocolVersion}

	var netCore NetCore
	n, _ := netCore.InitNetCore(netConfig)

	maxCount := int(common.GlobalConf.GetInt(configSection, configMaxBroadcastCount, maxBroadcastCount))

	netServerInstance = &Server{Self: self,
		netCore:           n,
		consensusHandler:  consensusHandler,
		config:            &networkConfig,
		maxBroadcastCount: maxCount}

	return nil
}

func genRandomSeeds(seeds []string) []string {
	nodesSelect := make(map[int]bool)

	totalSize := len(seeds)

	//always select first
	nodesSelect[0] = true

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	maxSize := int(math.Ceil(float64(totalSize) / 3))
	for i := 0; i < totalSize; i++ {
		peerIndex := r.Intn(totalSize)
		if nodesSelect[peerIndex] {
			continue
		}
		nodesSelect[peerIndex] = true
		if len(nodesSelect) >= maxSize {
			break
		}
	}
	seedsRandom := make([]string, 0)

	for key := range nodesSelect {
		seedsRandom = append(seedsRandom, seeds[key])
	}
	return seedsRandom
}

func GetNetInstance() Network {
	if netServerInstance == nil {
		return nil
	}
	return netServerInstance
}

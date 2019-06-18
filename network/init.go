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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/taslog"

	"net"

	"github.com/zvchain/zvchain/middleware/statistics"
)

const (
	seedDefaultID = "0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4"

	seedDefaultIP = "47.105.51.161"

	seedDefaultPort = 1122
)

// NetworkConfig is the network configuration
type NetworkConfig struct {
	NodeIDHex       string
	NatAddr         string
	NatPort         uint16
	SeedAddr        string
	SeedID          string
	ChainID         uint16 // Chain id
	ProtocolVersion uint16 // Protocol version
	TestMode        bool
	IsSuper         bool
}

var netServerInstance *Server

var Logger taslog.Logger

// Init initialize network instance,register message handler,join p2p network
func Init(config common.ConfManager, consensusHandler MsgHandler, networkConfig NetworkConfig) (err error) {
	index := common.GlobalConf.GetString("instance", "index", "")
	Logger = taslog.GetLoggerByIndex(taslog.P2PLogConfig, index)
	statistics.InitStatistics(config)

	self, err := InitSelfNode(config, networkConfig.IsSuper, NewNodeID(networkConfig.NodeIDHex))
	if err != nil {
		Logger.Errorf("InitSelfNode error:", err.Error())
		return err
	}

	if networkConfig.SeedAddr == "" {
		networkConfig.SeedAddr = seedDefaultIP
	}
	if networkConfig.SeedID == "" {
		networkConfig.SeedID = seedDefaultID
	}

	seedPort := seedDefaultPort

	seeds := make([]*Node, 0, 1)

	bnNode := NewNode(NewNodeID(networkConfig.SeedID), net.ParseIP(networkConfig.SeedAddr), seedPort)

	if bnNode.ID != self.ID && !networkConfig.IsSuper {
		seeds = append(seeds, bnNode)
	}
	listenAddr := net.UDPAddr{IP: self.IP, Port: self.Port}

	var natEnable bool
	if networkConfig.TestMode {
		natEnable = false
		listenIP, err := getIPByAddress(networkConfig.SeedAddr)
		if err != nil || listenIP == nil {
			Logger.Errorf("network SeedAddr:%v is wrong:%v", networkConfig.SeedAddr, err.Error())
			return err
		}
		listenAddr = net.UDPAddr{IP: listenIP, Port: self.Port}
	} else {
		natEnable = true
	}
	natIP :=""
	if len(networkConfig.NatAddr) > 0 {
		IP, err := getIPByAddress(networkConfig.NatAddr)
		if err != nil || IP == nil {
			Logger.Errorf("network Lookup NatAddr:%v is wrong:%v", networkConfig.SeedAddr, err.Error())
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

	netServerInstance = &Server{Self: self, netCore: n, consensusHandler: consensusHandler}
	return nil
}

func GetNetInstance() Network {
	return netServerInstance
}

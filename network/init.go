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

	nnet "net"

	"github.com/zvchain/zvchain/middleware/statistics"
)

const (
	seedDefaultID = "0x10b94f335f1842befc329f996b9bee0d3f4fe034306842bb301023ca38711779"

	seedDefaultIP = "47.105.51.161"

	seedDefaultPort = 1122
)

// NetworkConfig is the network configuration
type NetworkConfig struct {
	NodeIDHex       string
	NatIP           string
	NatPort         uint16
	SeedIP          string
	SeedID          string
	ChainID         uint16 // Chain id
	ProtocolVersion uint16 // Protocol version
	TestMode        bool
	IsSuper         bool
}

var net *Server

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

	if networkConfig.SeedIP == "" {
		networkConfig.SeedIP = seedDefaultIP
	}
	if networkConfig.SeedID == "" {
		networkConfig.SeedID = seedDefaultID
	}

	seedPort := seedDefaultPort

	seeds := make([]*Node, 0, 16)

	bnNode := NewNode(NewNodeID(networkConfig.SeedID), nnet.ParseIP(networkConfig.SeedIP), seedPort)

	if bnNode.ID != self.ID && !networkConfig.IsSuper {
		seeds = append(seeds, bnNode)
	}
	listenAddr := nnet.UDPAddr{IP: self.IP, Port: self.Port}

	var natEnable bool
	if networkConfig.TestMode {
		natEnable = false
		listenAddr = nnet.UDPAddr{IP: nnet.ParseIP(networkConfig.SeedIP), Port: self.Port}
	} else {
		natEnable = true
	}
	netConfig := NetCoreConfig{ID: self.ID,
		ListenAddr: &listenAddr, Seeds: seeds,
		NatTraversalEnable: natEnable,
		NatIP:              networkConfig.NatIP,
		NatPort:            networkConfig.NatPort,
		ChainID:            networkConfig.ChainID,
		ProtocolVersion:    networkConfig.ProtocolVersion}

	var netcore NetCore
	n, _ := netcore.InitNetCore(netConfig)

	net = &Server{Self: self, netCore: n, consensusHandler: consensusHandler}
	return nil
}

func GetNetInstance() Network {
	return net
}

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

package cli

import (
	"github.com/zvchain/zvchain/consensus/group"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/log"
	"net"

	"github.com/zvchain/zvchain/cmd/gzv/rpc"

	"fmt"
	"strings"
)

// rpcLevel indicate the rpc service function
type rpcLevel int

const (
	rpcLevelNone     rpcLevel = iota // Won't start rpc service which is the default value if not set
	rpcLevelGtas                     // Only enable the core rpc service functions used by miners or dapp developers
	rpcLevelExplorer                 // Enable both above and explorer related functions
	rpcLevelDev                      // Enable all functions including functions for debug or developer use
)

// rpcApi defines rpc service instance interface
type rpcApi interface {
	Namespace() string
	Version() string
}

func (gtas *Gtas) addInstance(inst rpcApi) {
	gtas.rpcInstances = append(gtas.rpcInstances, inst)
}

func (gtas *Gtas) initRpcInstances() error {
	level := gtas.config.rpcLevel
	if level < rpcLevelNone || level > rpcLevelDev {
		return fmt.Errorf("rpc level error:%v", level)
	}
	base := &rpcBaseImpl{gr: getGroupReader(), br: core.BlockChainImpl}
	gtas.rpcInstances = make([]rpcApi, 0)
	if level >= rpcLevelGtas {
		gtas.addInstance(&RpcGtasImpl{rpcBaseImpl: base, routineChecker: group.GroupRoutine})
	}
	if level >= rpcLevelExplorer {
		gtas.addInstance(&RpcExplorerImpl{rpcBaseImpl: base})
	}
	if level >= rpcLevelDev {
		gtas.addInstance(&RpcDevImpl{rpcBaseImpl: base})
	}
	return nil
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewHTTPServer(cors, vhosts, handler).Serve(listener)
	return nil
}

// StartRPC RPC function
func (gtas *Gtas) startRPC() error {
	var err error

	// init api instance
	if err = gtas.initRpcInstances(); err != nil {
		return err
	}

	apis := make([]rpc.API, 0)
	for _, inst := range gtas.rpcInstances {
		apis = append(apis, rpc.API{Namespace: inst.Namespace(), Version: inst.Version(), Service: inst, Public: true})
	}
	host, port := gtas.config.rpcAddr, gtas.config.rpcPort

	var cors []string
	switch gtas.config.cors {
	case "all":
		cors = []string{"*"}
	case "":
		cors = []string{}
	default:
		cors = strings.Split(gtas.config.cors, ",")
	}

	for plus := 0; plus < 40; plus++ {
		endpoint := fmt.Sprintf("%s:%d", host, port+uint16(plus))
		err = startHTTP(endpoint, apis, []string{}, cors, []string{})
		if err == nil {
			log.DefaultLogger.Errorf("RPC serving on %v\n", endpoint)
			return nil
		}
		if strings.Contains(err.Error(), "address already in use") {
			log.DefaultLogger.Errorf("port:%d already in use\n", port+uint16(plus))
			continue
		}
		return err
	}
	return err
}

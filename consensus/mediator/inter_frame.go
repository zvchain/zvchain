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

package mediator

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/logical"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
)

// Proc is the global unique instance of the consensus engine
var Proc logical.Processor

// ConsensusInit means consensus initialization
//
// Returns: true - the initialization is successful
// The internal will interact with the chain for initial data loading and pre-processing.
// False - failed.
func ConsensusInit(mi model.SelfMinerDO, conf common.ConfManager) bool {
	logical.InitConsensus()
	ret := Proc.Init(mi, conf)
	net.MessageHandler.Init(&Proc)
	return ret
}

// Start the miner process and participate in the consensus
// Returns true if successful, false returns false
func StartMiner() bool {
	return Proc.Start()
}

// StopMiner ends the miner process and no longer participate in the consensus
func StopMiner() {
	Proc.Stop()
	Proc.Finalize()
	return
}

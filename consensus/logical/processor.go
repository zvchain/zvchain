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

// Package logical implements the whole logic of the consensus engine.
// Including the verifyGroup manager process
package logical

import (
	group2 "github.com/zvchain/zvchain/consensus/group"
	"github.com/zvchain/zvchain/consensus/groupsig"

	"fmt"
	"strings"
	"sync/atomic"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
)

var ProcTestMode bool

// Processor is the consensus engine implementation struct that implements all the consensus logic
// and contextual information needed in the consensus process
type Processor struct {

	// miner releted
	mi            *model.SelfMinerDO // Current miner information
	genesisMember bool               // Whether current node is one of the genesis verifyGroup members
	minerReader   *MinerPoolReader   // Miner info reader

	// block generate related
	blockContexts    *castBlockContexts   // Stores the proposal messages for proposal role and the verification context for verify roles
	futureVerifyMsgs *FutureMessageHolder // Store the verification messages non-processable because of absence of the proposal message
	futureRewardReqs *FutureMessageHolder // Store the reward sign request messages non-processable because of absence of the corresponding block
	proveChecker     *proveChecker        // Check the vrf prove and the full-book

	Ticker *ticker.GlobalTicker // Global timer responsible for some cron tasks

	ready bool // Whether it has been initialized

	MainChain  core.BlockChain  // Blockchain access interface
	GroupChain *core.GroupChain // Groupchain access interface

	vrf atomic.Value // VrfWorker

	NetServer net.NetworkServer  // Responsible for network messaging
	conf      common.ConfManager // The config

	isCasting int32 // Proposal check status: 0 idle, 1 casting

	castVerifyCh   chan *types.BlockHeader
	futureVerifyCh chan *types.BlockHeader
	futureRewardCh chan *types.BlockHeader

	groupReader *groupReader

	ts time.TimeService // Network-wide time service, regardless of local time

}

func (p Processor) getPrefix() string {
	return p.GetMinerID().GetHexString()
}

// getMinerInfo is a private function for testing, official version not available
func (p Processor) getMinerInfo() *model.SelfMinerDO {
	return p.mi
}

func (p Processor) GetPubkeyInfo() model.PubKeyInfo {
	return model.NewPubKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultPubKey())
}

// Init initialize the process engine
func (p *Processor) Init(mi model.SelfMinerDO, conf common.ConfManager) bool {
	p.ready = false
	p.conf = conf
	p.futureVerifyMsgs = NewFutureMessageHolder()
	p.futureRewardReqs = NewFutureMessageHolder()
	p.MainChain = core.BlockChainImpl
	p.GroupChain = core.GroupChainImpl
	p.mi = &mi

	p.castVerifyCh = make(chan *types.BlockHeader, 5)
	p.futureVerifyCh = make(chan *types.BlockHeader, 5)
	p.futureRewardCh = make(chan *types.BlockHeader, 5)

	p.blockContexts = newCastBlockContexts(p.MainChain)
	p.NetServer = net.NewNetworkServer()
	p.proveChecker = newProveChecker(p.MainChain)
	p.ts = time.TSInstance
	p.isCasting = 0

	p.minerReader = newMinerPoolReader(p, core.MinerManagerImpl)
	pkPoolInit(p.minerReader)

	p.Ticker = ticker.NewGlobalTicker("consensus")

	provider := &core.GroupManagerImpl
	sr := group2.InitRoutine(p.minerReader, p.MainChain, provider)
	p.groupReader = newGroupReader(provider.GetGroupStoreReader(), sr)

	if stdLogger != nil {
		stdLogger.Debugf("proc(%v) inited 2.\n", p.getPrefix())
		consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	}

	notify.BUS.Subscribe(notify.BlockAddSucc, p.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.GroupAddSucc, p.onGroupAddSuccess)

	jgFile := conf.GetString(ConsensusConfSection, "joined_group_store", "")
	if strings.TrimSpace(jgFile) == "" {
		jgFile = "joined_group.config." + common.GlobalConf.GetString("instance", "index", "")
	}

	return true
}

// GetMinerID get current miner ID
func (p Processor) GetMinerID() groupsig.ID {
	return p.mi.GetMinerID()
}

func (p Processor) GetMinerInfo() *model.MinerDO {
	return &p.mi.MinerDO
}

// isCastLegal check if the block header is legal
func (p *Processor) isCastLegal(bh *types.BlockHeader, preHeader *types.BlockHeader) (err error) {
	castor := groupsig.DeserializeID(bh.Castor)
	minerDO := p.minerReader.getProposeMinerByHeight(castor, preHeader.Height)
	if minerDO == nil {
		err = fmt.Errorf("minerDO is nil, id=%v", castor)
		return
	}
	if !minerDO.CanPropose() {
		err = fmt.Errorf("miner can't cast at height, id=%v, height=%v, status=%v", castor, bh.Height, minerDO.Status)
		return
	}
	totalStake := p.minerReader.getTotalStake(preHeader.Height)
	// Check if the vrf threshold is satisfied
	if ok2, err2 := vrfVerifyBlock(bh, preHeader, minerDO, totalStake); !ok2 {
		err = fmt.Errorf("vrf verify block fail, err=%v", err2)
		return
	}

	gSeed := bh.Group
	vGroupSeed := p.calcVerifyGroup(preHeader, bh.Height)
	// Check if the gSeed of the block equal to the calculated one
	if gSeed != vGroupSeed {
		err = fmt.Errorf("calc verify group not equal, expect %v infact %v", vGroupSeed, gSeed)
		return
	}
	return nil
}

// getProposerPubKey get the public key of proposer miner in the specified block
func (p Processor) getProposerPubKeyInBlock(bh *types.BlockHeader) *groupsig.Pubkey {
	castor := groupsig.DeserializeID(bh.Castor)
	castorMO := p.minerReader.getLatestProposeMiner(castor)
	if castorMO != nil {
		return &castorMO.PK
	}
	return nil
}

func (p *Processor) getEncryptPrivateKey() (common.PrivateKey, error) {
	seed := p.mi.SK.GetHexString() + p.mi.ID.GetHexString()
	return common.GenerateKey(seed)
}

func (p *Processor) getDefaultSeckeyInfo() model.SecKeyInfo {
	return model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
}

// Start starts miner process
func (p *Processor) Start() bool {
	p.Ticker.RegisterPeriodicRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 1)
	p.Ticker.RegisterPeriodicRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.RegisterPeriodicRoutine(p.getBroadcastRoutineName(), p.broadcastRoutine, 1)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.Ticker.StartTickerRoutine(p.getBroadcastRoutineName(), false)

	p.Ticker.RegisterPeriodicRoutine(p.getUpdateMonitorNodeInfoRoutine(), p.updateMonitorInfo, 3)
	p.Ticker.StartTickerRoutine(p.getUpdateMonitorNodeInfoRoutine(), false)

	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

// Stop is reserved interface
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {

	topHeight := p.MainChain.QueryTopBlock().Height

	groups := p.groupReader.getAvailableGroupsByHeight(topHeight)
	stdLogger.Infof("current group size:%v", len(groups))
	for _, g := range groups {
		// Genesis group
		if g.header.WorkHeight() == 0 && g.hasMember(p.GetMinerID()) {
			p.genesisMember = true
			break
		}
	}
}

// Ready check if the processor engine is initialized and ready for message processing
func (p *Processor) Ready() bool {
	return p.ready
}

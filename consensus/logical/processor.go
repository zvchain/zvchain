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
	"bytes"
	"io/ioutil"
	"strings"
	"sync"

	group2 "github.com/zvchain/zvchain/consensus/group"
	"github.com/zvchain/zvchain/consensus/groupsig"

	"fmt"
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
	minerReader   *MinerPoolReader   // Miner info storeReader

	// block generate related
	blockContexts    *castBlockContexts   // Stores the proposal messages for proposal role and the verification context for verify roles
	futureVerifyMsgs *FutureMessageHolder // Store the verification messages non-processable because of absence of the proposal message
	proveChecker     *proveChecker        // Check the vrf prove and the full-book

	Ticker *ticker.GlobalTicker // Global timer responsible for some cron tasks

	ready bool // Whether it has been initialized

	MainChain types.BlockChain // Blockchain access interface

	vrf atomic.Value // VrfWorker

	NetServer net.NetworkServer  // Responsible for network messaging
	conf      common.ConfManager // The config

	isCasting int32 // Proposal check status: 0 idle, 1 casting

	castVerifyCh chan *types.BlockHeader
	blockAddCh   chan *types.BlockHeader

	groupReader *groupReader

	ts time.TimeService // Network-wide time service, regardless of local time

	groupNetBuilt sync.Map // Store groups that have built group-network

	rewardHandler *RewardHandler
}

func (p *Processor) GetRewardManager() types.RewardManager {
	return p.MainChain.GetRewardManager()
}

func (p *Processor) GetVctxByHeight(height uint64) *VerifyContext {
	return p.blockContexts.getVctxByHeight(height)
}

func (p *Processor) GetGroupBySeed(seed common.Hash) *verifyGroup {
	return p.groupReader.getGroupBySeed(seed)
}

func (p *Processor) GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey {
	return p.groupReader.getGroupSignatureSeckey(seed)
}

func (p *Processor) AddTransaction(tx *types.Transaction) (bool, error) {
	return p.MainChain.GetTransactionPool().AddTransaction(tx)
}

func (p *Processor) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	p.NetServer.SendCastRewardSign(msg)
}

func (p *Processor) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	p.NetServer.SendCastRewardSignReq(msg)
}

func (p Processor) getPrefix() string {
	return p.GetMinerID().GetAddrString()
}

// getMinerInfo is a private function for testing, official version not available
func (p Processor) getMinerInfo() *model.SelfMinerDO {
	return p.mi
}

// Init initialize the process engine
func (p *Processor) Init(mi model.SelfMinerDO, conf common.ConfManager) bool {
	p.ready = false
	p.conf = conf
	p.futureVerifyMsgs = NewFutureMessageHolder()
	p.rewardHandler = NewRewardHandler(p)

	p.MainChain = core.BlockChainImpl
	p.mi = &mi

	p.castVerifyCh = make(chan *types.BlockHeader, 5)
	p.blockAddCh = make(chan *types.BlockHeader, 5)

	p.blockContexts = newCastBlockContexts(p.MainChain)
	p.NetServer = net.NewNetworkServer()
	p.proveChecker = newProveChecker()
	p.ts = time.TSInstance
	p.isCasting = 0

	p.minerReader = newMinerPoolReader(p, core.MinerManagerImpl)

	p.Ticker = ticker.NewGlobalTicker("consensus")

	provider := core.GroupManagerImpl
	sr := group2.InitRoutine(p.minerReader, p.MainChain, provider, provider, &mi)
	p.groupReader = newGroupReader(provider, sr)

	if stdLogger != nil {
		stdLogger.Debugf("proc(%v) inited 2.\n", p.getPrefix())
		consensusLogger.Infof("ProcessorId:%v", p.getPrefix())
	}

	notify.BUS.Subscribe(notify.BlockAddSucc, p.onBlockAddSuccess)

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
	seed := p.mi.SK.GetHexString() + p.mi.ID.GetAddrString()
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
	go p.chLoop()
	p.initLivedGroup()

	p.ready = true
	return true
}

// Stop is reserved interface
func (p *Processor) Stop() {
	p.groupReader.skStore.Close()
	return
}

func (p *Processor) initLivedGroup() {
	genesisGroup := group2.GenerateGenesis()

	for i, mem := range genesisGroup.Group.Members() {
		if bytes.Equal(mem.ID(), p.GetMinerID().Serialize()) {
			p.genesisMember = true
			msks, err := ioutil.ReadFile("genesis_msk.info")
			if err != nil {
				panic(fmt.Errorf("genesis miner storeReader genesis_msk.info fail:%v", err))
			}
			stdLogger.Debugf("store genesis member msk")
			arr := strings.Split(string(msks), ",")
			var sk groupsig.Seckey
			sk.SetHexString(arr[i])
			p.groupReader.skStore.StoreGroupSignatureSeckey(genesisGroup.Group.Header().Seed(), sk, common.MaxUint64)

			break
		}
	}

	currentHeight := p.MainChain.Height()
	currEpoch := types.EpochAt(currentHeight)

	// Build group-net of groups activated at current epoch
	p.buildGroupNetOfActivateEpochAt(currEpoch)
	// Try to build group-net of groups will be activated at next epoch
	p.buildGroupNetOfNextEpoch(currentHeight)
}

// Ready check if the processor engine is initialized and ready for message processing
func (p *Processor) Ready() bool {
	return p.ready
}

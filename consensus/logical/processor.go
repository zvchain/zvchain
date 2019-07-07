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
// Including the group manager process
package logical

import (
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

	// group related info
	joiningGroups *JoiningGroups // Group information in the process of being established joined by the current node
	belongGroups  *BelongGroups  // Join (successful) group information (miner node data)
	globalGroups  *GlobalGroups  // All available group infos, including groups can work currently or in the future
	groupManager  *GroupManager  // Responsible for group creating process

	// miner releted
	mi            *model.SelfMinerDO // Current miner information
	genesisMember bool               // Whether current node is one of the genesis group members
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

	castVerifyCh chan common.Hash

	ts time.TimeService // Network-wide time service, regardless of local time

}

func (p Processor) getPrefix() string {
	return p.GetMinerID().ShortS()
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
	p.globalGroups = newGlobalGroups(p.GroupChain)
	p.joiningGroups = NewJoiningGroups()
	encryptPrivateKey, err := p.getEncryptPrivateKey()
	if err != nil {
		return false
	}
	p.castVerifyCh = make(chan common.Hash, 5)
	p.belongGroups = NewBelongGroups(p.genBelongGroupStoreFile(), encryptPrivateKey)
	p.blockContexts = newCastBlockContexts(p.MainChain)
	p.NetServer = net.NewNetworkServer()
	p.proveChecker = newProveChecker(p.MainChain)
	p.ts = time.TSInstance
	p.isCasting = 0

	p.minerReader = newMinerPoolReader(p, core.MinerManagerImpl)
	pkPoolInit(p.minerReader)

	p.groupManager = newGroupManager(p)
	p.Ticker = ticker.NewGlobalTicker("consensus")

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
	p.belongGroups.joinedGroup2DBIfConfigExists(jgFile)

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
func (p *Processor) isCastLegal(bh *types.BlockHeader, preHeader *types.BlockHeader) (ok bool, group *StaticGroupInfo, err error) {
	castor := groupsig.DeserializeID(bh.Castor)
	minerDO := p.minerReader.getProposeMinerByHeight(castor, preHeader.Height)
	if minerDO == nil {
		err = fmt.Errorf("minerDO is nil, id=%v", castor.ShortS())
		return
	}
	if !minerDO.CanPropose() {
		err = fmt.Errorf("miner can't cast at height, id=%v, height=%v, status=%v", castor.ShortS(), bh.Height, minerDO.Status)
		return
	}
	totalStake := p.minerReader.getTotalStake(preHeader.Height)
	if ok2, err2 := vrfVerifyBlock(bh, preHeader, minerDO, totalStake); !ok2 {
		err = fmt.Errorf("vrf verify block fail, err=%v", err2)
		return
	}

	var gid = groupsig.DeserializeID(bh.Group)

	selectGroupIDFromCache := p.calcVerifyGroupFromCache(preHeader, bh.Height)

	if selectGroupIDFromCache == nil {
		err = common.ErrSelectGroupNil
		stdLogger.Errorf("selectGroupId is nil")
		return
	}
	var verifyGid = *selectGroupIDFromCache

	// It is possible that the group has been disbanded and needs to be taken from the chain.
	if !selectGroupIDFromCache.IsEqual(gid) {
		selectGroupIDFromChain := p.calcVerifyGroupFromChain(preHeader, bh.Height)
		if selectGroupIDFromChain == nil {
			err = common.ErrSelectGroupNil
			return
		}
		// Start the update if the memory does not match the chain
		if !selectGroupIDFromChain.IsEqual(*selectGroupIDFromCache) {
			go p.updateGlobalGroups()
		}
		if !selectGroupIDFromChain.IsEqual(gid) {
			err = common.ErrSelectGroupInequal
			stdLogger.Errorf("selectGroupId from both cache and chain not equal, expect %v, receive %v", selectGroupIDFromChain.ShortS(), gid.ShortS())
			return
		}
		verifyGid = *selectGroupIDFromChain
	}

	// Obtain legal ingot group
	group = p.GetGroup(verifyGid)
	if group == nil {
		err = fmt.Errorf("group is nil:groupID=%v", verifyGid)
		return
	}
	if !group.GroupID.IsValid() {
		err = fmt.Errorf("selectedGroup is not valid, expect gid=%v, real gid=%v", verifyGid.ShortS(), group.GroupID.ShortS())
		return
	}

	ok = true
	return
}

func (p *Processor) getMinerPos(gid groupsig.ID, uid groupsig.ID) int32 {
	sgi := p.GetGroup(gid)
	return int32(sgi.GetMinerPos(uid))
}

// GetGroup get a specific group
func (p Processor) GetGroup(gid groupsig.ID) *StaticGroupInfo {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		// this must not happen
		panic("GetSelfGroup failed.")
	} else {
		return g
	}
}

// getGroupPubKey get the public key of an ingot group (loaded from
// the chain when the processer is initialized)
func (p Processor) getGroupPubKey(gid groupsig.ID) groupsig.Pubkey {
	if g, err := p.globalGroups.GetGroupByID(gid); err != nil {
		// hold it for now
		panic("GetSelfGroup failed.")
	} else {
		return g.GetPubKey()
	}

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

func (p *Processor) getInGroupSeckeyInfo(gid groupsig.ID) model.SecKeyInfo {
	return model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid))
}

func (p *Processor) chLoop() {
	for {
		select {
		case hash := <-p.castVerifyCh:
			p.verifyCachedMsg(hash)
		}
	}
}

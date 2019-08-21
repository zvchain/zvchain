package crontab

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/transfer"
	"github.com/zvchain/zvchain/cmd/gzv/cli"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"strings"
	"sync"
	"time"
)

var lock sync.Mutex

const checkInterval = time.Second * 10

type Crontab struct {
	blockTransfer    *transfer.Transfer
	storage          *mysql.Storage
	blockHeight      uint64
	accountPrimaryId uint64
	isFetchingReward bool
	isFetchingStake  bool
	rpcExplore       *cli.RpcExplorerImpl
}

func NewServer(dbAddr string, dbPort int, dbUser string, dbPassword string, rpcAddr string, rpcPort int, reset bool) *Crontab {

	server := &Crontab{}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, rpcAddr, rpcPort, reset)
	server.blockHeight = server.storage.TopBlockRewardHeight()
	if server.blockHeight > 0 {
		server.blockHeight += 1
	}
	go server.loop()
	return server
}

func (crontab *Crontab) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
}

func (crontab *Crontab) fetchBlockRewards() {
	if crontab.isFetchingReward {
		return
	}
	crontab.isFetchingReward = true
	fmt.Println("[crontab]  fetchBlockRewards height:", crontab.blockHeight)
	blockRewards, _ := crontab.rpcExplore.ExplorerBlockReward(crontab.blockHeight)
	if blockRewards.Data != nil {
		rewards := blockRewards.Data.(cli.ExploreBlockReward)
		sys := &models.Sys{
			Variable: mysql.Blockrewardtophight,
			SetBy:    "carrie.cxl",
		}
		lock.Lock()
		crontab.storage.AddBlockRewardSystemconfig(sys)
		crontab.blockHeight += 1
		lock.Unlock()
		accounts := crontab.blockTransfer.BlockRewardTOAccount(rewards)
		for _, account := range accounts {

			crontab.storage.UpdateAccountByColumn(account, map[string]interface{}{"rewards": gorm.Expr("rewards + ?", account.Rewards)})

		}
		go crontab.fetchBlockRewards()

	}
	crontab.isFetchingReward = false

}

func (crontab *Crontab) fetchBlockStake() {
	if crontab.isFetchingStake {
		return
	}
	//按页数和标记信息更新数据，标记信息更新sys数据
	accounts := crontab.storage.GetAccountByMaxPrimaryId(10)
	for _, account := range accounts {
		minerinfo, stakefrom := crontab.GetMinerInfo(account.Address)
		crontab.storage.UpdateAccountByColumn(account, map[string]interface{}{
			"proposal_stake": minerinfo[0].Stake,
			"other_stake":    minerinfo[1].Stake,
			"verify_stake":   minerinfo[2].Stake,
			"stake_from":     stakefrom})
	}
}

func (crontab *Crontab) GetMinerInfo(addr string) ([]*cli.MortGage, string) {
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return nil, ""
	}

	morts := make([]*cli.MortGage, 0)
	address := common.StringToAddress(addr)
	proposalInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeProposal)
	var stakefrom = ""
	if proposalInfo != nil {
		mort := cli.NewMortGageFromMiner(proposalInfo)
		morts = append(morts, mort)
		//get stakeinfo by miners themselves
		details := core.MinerManagerImpl.GetStakeDetails(address, address)
		var selfStakecount uint64 = 0
		for _, detail := range details {
			if detail.MType == types.MinerTypeProposal {
				selfStakecount += detail.Value
			}
		}
		morts = append(morts, &cli.MortGage{
			Stake:       mort.Stake - selfStakecount,
			ApplyHeight: 0,
			Type:        "proposal node",
			Status:      "normal",
		})
		if selfStakecount > 0 {
			stakefrom = addr
		}
		// check if contain other stake ,
		//todo pool identify
		if selfStakecount < mort.Stake {
			stakefrom = stakefrom + "," + crontab.getStakeFrom(address)
		}
	}
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeVerify)
	if verifierInfo != nil {
		morts = append(morts, cli.NewMortGageFromMiner(verifierInfo))
	}

	return morts, stakefrom
}

func (crontab *Crontab) getStakeFrom(address common.Address) string {
	allStakeDetails := core.MinerManagerImpl.GetAllStakeDetails(address)
	var stakeFrom = ""
	index := 0
	for from, _ := range allStakeDetails {
		if from != address.String() {
			index += 1
			if index > 1 {
				break
			}
			stakeFrom = stakeFrom + from + ","
		}
	}
	return strings.Trim(stakeFrom, ",")
}

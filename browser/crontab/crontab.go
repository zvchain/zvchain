package crontab

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/transfer"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/cmd/gtas/cli"
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
	rewards := blockRewards.Data.(cli.ExploreBlockReward)
	if rewards != nil {
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
	//按页数和标记信息更新数据
	accounts := crontab.storage.GetAccountByMaxPrimaryId(10)
	for _, account := range accounts {
		minerinfo := crontab.GetMinerInfo(account.Address)
		crontab.storage.UpdateAccountByColumn(account, map[string]interface{}{"proposal_stake": minerinfo[0].Stake, "verify_stake": minerinfo[1].Stake})

	}

}

func (crontab *Crontab) GetMinerInfo(addr string) []*cli.MortGage {
	if !util.ValidateAddress(strings.TrimSpace(addr)) {
		return nil
	}

	morts := make([]*cli.MortGage, 0)
	address := common.HexToAddress(addr)
	proposalInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeProposal)
	if proposalInfo != nil {
		morts = append(morts, cli.NewMortGageFromMiner(proposalInfo))
	}
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeVerify)
	if verifierInfo != nil {
		morts = append(morts, cli.NewMortGageFromMiner(verifierInfo))
	}
	return morts
}

package crontab

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/transfer"
	"github.com/zvchain/zvchain/cmd/gtas/cli"
	"sync"
	"time"
)

var lock sync.Mutex

const checkInterval = time.Second * 10

type Crontab struct {
	blockTransfer    *transfer.Transfer
	storage          *mysql.Storage
	blockHeight      uint64
	isFetchingReward bool
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

func (server *Crontab) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
}

func (server *Crontab) fetchBlockRewards() {
	if server.isFetchingReward {
		return
	}
	server.isFetchingReward = true
	fmt.Println("[crontab]  fetchBlockRewards height:", server.blockHeight)
	blockRewards, _ := server.rpcExplore.ExplorerBlockReward(server.blockHeight)
	rewards := blockRewards.Data.(cli.ExploreBlockReward)
	if rewards != nil {
		sys := &models.Sys{}
		sys.Variable = mysql.Blockrewardtophight
		sys.SetBy = "carrie.cxl"
		lock.Lock()
		server.storage.AddBlockRewardSystemconfig(sys)
		server.blockHeight += 1
		lock.Unlock()
		accounts := server.blockTransfer.BlockRewardTOAccount(rewards)
		for _, account := range accounts {
			server.storage.UpdateAccountByColumn(account, "rewards", gorm.Expr("rewards + ?", account.Rewards))

		}
		go server.fetchBlockRewards()

	}
	server.isFetchingReward = false

}

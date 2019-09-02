package crontab

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"sync/atomic"
	"time"
)

const checkInterval = time.Second * 10

type Crontab struct {
	storage          *mysql.Storage
	blockHeight      uint64
	page             uint64
	maxid            uint
	accountPrimaryId uint64
	isFetchingReward int32

	isFetchingPoolvotes int32
	rpcExplore          *Explore
	transfer            *Transfer
}

func NewServer(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *Crontab {

	server := &Crontab{}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset)
	server.blockHeight = server.storage.TopBlockRewardHeight(mysql.Blockrewardtophight)
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
	go crontab.fetchPoolVotes()
	go crontab.fetchBlockRewards()
	//go crontab.fetchBlockStakeAll()

	for {
		select {
		case <-check.C:
			go crontab.fetchPoolVotes()
			go crontab.fetchBlockRewards()
			//go crontab.fetchBlockStakeAll()

		}
	}
}
func (crontab *Crontab) excutePoolVotes() {
	accountsPool := crontab.storage.GetAccountByRoletype(crontab.maxid, types.MinerPool)
	if accountsPool != nil && len(accountsPool) > 0 {
		blockheader := core.BlockChainImpl.CheckPointAt(mysql.CheckpointMaxHeight)
		var db types.AccountDB
		var err error
		if err != nil || db == nil {
			return
		}
		db, err = core.BlockChainImpl.AccountDBAt(blockheader.Height)
		total := len(accountsPool) - 1
		for num, pool := range accountsPool {
			if num == total {
				crontab.maxid = pool.ID
			}
			//pool to be normal miner
			proposalInfo := core.MinerManagerImpl.GetMiner(common.StringToAddress(pool.Address), types.MinerTypeProposal, blockheader.Height)
			attrs := make(map[string]interface{})
			if uint64(proposalInfo.Type) != pool.RoleType {
				attrs["role_type"] = types.InValidMinerPool
			}
			tickets := core.MinerManagerImpl.GetTickets(db, common.StringToAddress(pool.Address))
			var extra = &models.PoolExtraData{}
			if pool.ExtraData != "" {
				if err := json.Unmarshal([]byte(pool.ExtraData), extra); err != nil {
					fmt.Println("Unmarshal json", err.Error())
					if attrs != nil {
						crontab.storage.UpdateAccountByColumn(pool, attrs)
					}
					continue
				}
				//different vote need update
				if extra.Vote != tickets {
					extra.Vote = tickets
					result, _ := json.Marshal(extra)
					attrs["extra_data"] = string(result)
				}
			} else if tickets > 0 {
				extra.Vote = tickets
				result, _ := json.Marshal(extra)
				attrs["extra_data"] = string(result)
			}
			crontab.storage.UpdateAccountByColumn(pool, attrs)
		}
		crontab.excutePoolVotes()
	}
}

//uopdate invalid guard and pool
func (crontab *Crontab) fetchPoolVotes() {

	if !atomic.CompareAndSwapInt32(&crontab.isFetchingPoolvotes, 0, 1) {
		return
	}
	crontab.excutePoolVotes()
	atomic.CompareAndSwapInt32(&crontab.isFetchingPoolvotes, 1, 0)

}

func (crontab *Crontab) fetchBlockRewards() {
	if !atomic.CompareAndSwapInt32(&crontab.isFetchingReward, 0, 1) {
		return
	}
	crontab.excuteBlockRewards()
	atomic.CompareAndSwapInt32(&crontab.isFetchingReward, 1, 0)
}

func (crontab *Crontab) excuteBlockRewards() {
	height, _ := crontab.storage.TopBlockHeight()
	if crontab.blockHeight > height {
		return
	}
	//blockheader := core.BlockChainImpl.CheckPointAt(mysql.CheckpointMaxHeight)
	/*if crontab.blockHeight > blockheader.Height {
		return
	}*/
	rewards := crontab.rpcExplore.GetPreHightRewardByHeight(crontab.blockHeight)
	fmt.Println("[crontab]  fetchBlockRewards height:", crontab.blockHeight, 0)

	if rewards != nil {
		fmt.Println("[crontab]  fetchBlockRewards ObjectTojson:", util.ObjectTojson(rewards), crontab.blockHeight)

		accounts := crontab.transfer.RewardsToAccounts(rewards)

		if crontab.storage.AddBlockRewardMysqlTransaction(accounts) {
			crontab.blockHeight += 1
		}
		crontab.excuteBlockRewards()

	} else {
		crontab.blockHeight += 1

		fmt.Println("[crontab]  fetchBlockRewards rewards nil:", crontab.blockHeight, rewards)

	}
}

func (crontab *Crontab) Produce(header map[string]uint64, pipe chan map[string]uint64) {
	for {
		pipe <- header
		time.Sleep(time.Second)
	}

}
func (crontab *Crontab) Consume(pipe chan map[string]uint64) bool {
	for {
		accounts := <-pipe
		return crontab.storage.AddBlockRewardMysqlTransaction(accounts)
	}

}

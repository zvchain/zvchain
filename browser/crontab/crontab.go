package crontab

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"time"
)

const checkInterval = time.Second * 10

type Crontab struct {
	storage             *mysql.Storage
	blockHeight         uint64
	page                uint64
	maxid               uint
	accountPrimaryId    uint64
	isFetchingReward    bool
	isFetchingStake     bool
	isFetchingPoolvotes bool
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

//uopdate invalid guard and pool
func (crontab *Crontab) fetchPoolVotes() {
	if crontab.isFetchingPoolvotes {
		return
	}
	crontab.isFetchingPoolvotes = true
	//todo 守护节点失效
	accounts := crontab.storage.GetAccountByRoletype(0, types.MinerGuard)
	for _, account := range accounts {
		crontab.storage.UpdateAccountByColumn(account, map[string]interface{}{"role_type": types.MinerNormal})
	}
	accountsPool := crontab.storage.GetAccountByRoletype(crontab.maxid, types.MinerPool)
	if accountsPool != nil && len(accountsPool) > 0 {
		var db types.AccountDB
		var err error
		if err != nil || db == nil {
			return
		}
		db, err = core.BlockChainImpl.LatestAccountDB()
		total := len(accountsPool) - 1
		for num, pool := range accountsPool {
			if num == total {
				crontab.maxid = pool.ID
			}
			//pool to be normal miner
			proposalInfo := core.MinerManagerImpl.GetLatestMiner(common.StringToAddress(pool.Address), types.MinerTypeProposal)
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
		go crontab.fetchPoolVotes()
	}
	crontab.isFetchingPoolvotes = false
	//vote

}

/*func (crontab *Crontab) fetchBlockStakeAll() {
	if crontab.isFetchingStake {
		return
	}
	crontab.isFetchingStake = true

	//按页数和标记信息更新数据，标记信息更新sys数据
	accounts := crontab.storage.GetAccountByPage(crontab.page)
	if accounts != nil {
		for _, account := range accounts {
		}
		crontab.page += 1
		go crontab.fetchBlockStakeAll()
	}

	crontab.isFetchingStake = false

}*/

func (crontab *Crontab) fetchBlockRewards() {
	if crontab.isFetchingReward {
		return
	}
	crontab.isFetchingReward = true
	fmt.Println("[crontab]  fetchBlockRewards height:", crontab.blockHeight)
	rewards := crontab.rpcExplore.GetPreHightRewardByHeight(crontab.blockHeight)
	if rewards != nil {

		accounts := crontab.transfer.RewardsToAccounts(rewards)
		if crontab.storage.AddBlockRewardMysqlTransaction(accounts) {
			crontab.blockHeight += 1
		}
		go crontab.fetchBlockRewards()

	}
	crontab.isFetchingReward = false

}

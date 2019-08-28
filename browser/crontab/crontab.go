package crontab

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"strings"
	"time"
)

const checkInterval = time.Second * 10

type Crontab struct {
	storage             *mysql.Storage
	blockHeight         uint64
	page                uint64
	accountPrimaryId    uint64
	isFetchingReward    bool
	isFetchingStake     bool
	isFetchingPoolvotes bool
	rpcExplore          *Explore
	transfer            *Transfer
}

func NewServer(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *Crontab {

	server := &Crontab{}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, "", 0, reset)
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
	go crontab.fetchBlockRewards()
	go crontab.fetchBlockStakeAll()

	for {
		select {
		case <-check.C:
			go crontab.fetchBlockRewards()
			go crontab.fetchBlockStakeAll()

		}
	}
}

func (crontab *Crontab) fetchPoolVotes() {
	accounts := crontab.storage.GetAccountByRoletype(0, types.MinerGuard)
	//todo 守护节点失效
	for _, account := range accounts {
		crontab.storage.UpdateAccountByColumn(account, map[string]interface{}{"role_type": types.MinerNormal})
	}
	accountspool := crontab.storage.GetAccountByRoletype(0, types.MinerPool)
	if accountspool != nil && len(accountspool) > 0 {
		var db types.AccountDB
		var err error
		if err != nil || db == nil {
			return
		}
		db, err = core.BlockChainImpl.LatestAccountDB()

		for _, pool := range accountspool {
			//pool to be normal miner
			proposalInfo := core.MinerManagerImpl.GetLatestMiner(common.StringToAddress(pool.Address), types.MinerTypeProposal)
			if uint64(proposalInfo.Type) != pool.RoleType {
				crontab.storage.UpdateAccountByColumn(pool, map[string]interface{}{"role_type": types.InValidMinerPool})
			}
			tickets := core.MinerManagerImpl.GetTickets(db, common.StringToAddress(pool.Address))

			if pool.ExtraData != "" {
				var extra = &models.PoolExtraData{}
				if err := json.Unmarshal([]byte(pool.ExtraData), extra); err != nil {
					fmt.Println("Unmarshal json", err.Error())
					continue
				}
				if extra.Vote != tickets {
					extra.Vote = tickets
					crontab.storage.UpdateAccountByColumn(pool, map[string]interface{}{"extra_data": json.Marshal(extra)})
				}
			}
		}
	}
	//vote

}
func (crontab *Crontab) fetchBlockStakeAll() {
	if crontab.isFetchingStake {
		return
	}
	crontab.isFetchingStake = true

	//按页数和标记信息更新数据，标记信息更新sys数据
	accounts := crontab.storage.GetAccountByPage(crontab.page)
	if accounts != nil {
		for _, account := range accounts {
			crontab.updateAccountStake(account)
		}
		crontab.page += 1
		go crontab.fetchBlockStakeAll()
	}

	crontab.isFetchingStake = false

}

func (crontab *Crontab) updateAccountStake(account *models.Account) {
	if account == nil {
		return

	}
	minerinfo, stakefrom := crontab.GetMinerInfo(account.Address)
	if len(minerinfo) > 0 {
		crontab.storage.UpdateAccountByColumn(account, map[string]interface{}{
			"proposal_stake": minerinfo[0].Stake,
			"other_stake":    minerinfo[1].Stake,
			"verify_stake":   minerinfo[2].Stake,
			"total_stake":    minerinfo[0].Stake + minerinfo[2].Stake,
			"stake_from":     stakefrom,
			"status":         minerinfo[0].Status,
			"role_type":      minerinfo[0].Identity,
		})
	}
}

func (crontab *Crontab) fetchBlockStakeByAdress(address string) {
	//根据账户地址更新质押信息，前提是存在交易类型为质押时
	account := &models.Account{
		Address: address,
	}
	crontab.updateAccountStake(account)

}

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

func (crontab *Crontab) GetMinerInfo(addr string) ([]*MortGage, string) {
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return nil, ""
	}

	morts := make([]*MortGage, 0, 0)
	address := common.StringToAddress(addr)
	proposalInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeProposal)
	var stakefrom = ""
	if proposalInfo != nil {
		mort := NewMortGageFromMiner(proposalInfo)
		morts = append(morts, mort)
		//get stakeinfo by miners themselves
		details := core.MinerManagerImpl.GetStakeDetails(address, address)
		var selfStakecount uint64 = 0
		for _, detail := range details {
			if detail.MType == types.MinerTypeProposal {
				selfStakecount += detail.Value
			}
		}
		morts = append(morts, &MortGage{
			Stake:       mort.Stake - selfStakecount,
			ApplyHeight: 0,
			Type:        "proposal node",
			Status:      types.MinerStatusActive,
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
		morts = append(morts, NewMortGageFromMiner(verifierInfo))
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

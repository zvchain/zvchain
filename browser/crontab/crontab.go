package crontab

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/zvchain/zvchain/browser"
	common2 "github.com/zvchain/zvchain/browser/common"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/cmd/gzv/cli"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	check10SecInterval = time.Second * 10
	check30MinInterval = time.Minute * 30
	check1HourInterval = time.Minute * 60

	turnoverKey = "turnover"
	cpKey       = "checkpoint"

	GuardAndPoolContract = "zv6082d5468cd02e55f438401e204c0682ad8f77ee998db7c98ea476801f10225d"
	makerdaoOrder        = "makerdao.order_contract"
	makerdaoprice        = "makerdao.price_contract"
)

var (
	GlobalCP                         uint64
	FinishSyncSupplementBlockToMiner bool = false
	FinishSyncFetchOldAccounts       bool = false
	SupplementBlockToMinerHeight     uint64
)

const (
	ProposalStakeType = 0
	VerifyStakeType   = 1
)

var (
	GlobalCrontab *Crontab
	PoolNodeMap   = make(map[string]struct{})
)

type Crontab struct {
	storage      *mysql.Storage
	slaveStorage *mysql.SlaveStorage

	blockRewardHeight             uint64
	blockTopHeight                uint64
	rewardStorageDataHeight       uint64
	blockToMinerStorageDataHeight uint64
	curblockcount                 uint64
	curTrancount                  uint64
	ConfirmRewardHeight           uint64

	page                  uint64
	maxid                 uint
	accountPrimaryId      uint64
	isFetchingReward      int32
	isFetchingConsume     int32
	isFetchingGroups      bool
	groupHeight           uint64
	isInited              bool
	isInitedReward        bool
	makerdaoOrderContratc string
	makerdaoPriceContratc string

	isFetchingPoolvotes               int32
	isFetchingBlockToMiner            int32
	isFetchingOldTxCountToAccountList int32
	isConfirmBlockReward              int32
	rpcExplore                        *Explore
	transfer                          *Transfer
	fetcher                           *common2.Fetcher
	isFetchingBlocks                  bool
	initdata                          chan *models.ForkNotify
	initRewarddata                    chan *models.ForkNotify

	isFetchingVerfications bool
}

func NewServer(dbAddr string, dbPort int, dbUser string,
	dbPassword string, reset bool,
	browerslavedbaddr string, slavedbUser string, slavePassword string) *Crontab {
	server := &Crontab{
		initdata:       make(chan *models.ForkNotify, 10000),
		initRewarddata: make(chan *models.ForkNotify, 10000),
	}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, false)
	server.slaveStorage = mysql.NewSlaveStorage(browerslavedbaddr, dbPort, slavedbUser, slavePassword, reset, false)
	server.addGenisisblockAndReward()
	server.storage.InitCurConfig()
	_, server.rewardStorageDataHeight = server.storage.RewardTopBlockHeight()
	addr, _ := server.storage.GetMakerdaoOrderaddress(makerdaoOrder)
	if addr != "" {
		server.makerdaoOrderContratc = addr
	}
	addrprice, _ := server.storage.GetMakerdaoOrderaddress(makerdaoprice)
	if addrprice != "" {
		server.makerdaoPriceContratc = addrprice
	}
	go server.ConsumeContractTransfer()
	//notify.BUS.Subscribe(notify.BlockAddSucc, server.OnBlockAddSuccess)

	server.blockRewardHeight = server.storage.TopBlockRewardHeight(mysql.Blockrewardtopheight)
	confirmRewardHeight := server.storage.MinConfirmBlockRewardHeight()
	if confirmRewardHeight > 0 {
		server.ConfirmRewardHeight = confirmRewardHeight
	}
	server.blockTopHeight = server.storage.GetTopblock()
	if server.blockRewardHeight > 0 {
		server.blockRewardHeight += 1
	}
	go server.HandleOnBlockSuccess()
	go server.loop()
	GlobalCrontab = server
	return server
}

func (crontab *Crontab) HandleOnBlockSuccess() {
	for {
		simpleBlockHeader := <-core.OnBlockSuccessChan
		crontab.OnBlockAddSuccess(simpleBlockHeader)
	}
}

func (crontab *Crontab) loop() {
	var (
		check10Sec = time.NewTicker(check10SecInterval)
		check30Min = time.NewTicker(check30MinInterval)
		check1Hour = time.NewTicker(check1HourInterval)
	)
	defer check10Sec.Stop()
	go crontab.fetchOldLogs()
	go crontab.fetchOldReceiptToTransaction()
	go crontab.fetchPoolVotes()
	go crontab.fetchGroups()
	go crontab.fetchOldConctactCreate()
	//go crontab.GetMinerToblocksByPage()

	go crontab.fetchBlockRewards()
	go crontab.Consume()
	go crontab.ConsumeReward()
	go crontab.UpdateTurnOver()
	go crontab.UpdateCheckPoint()
	go crontab.SearchTempDeployToken()
	//go crontab.supplementProposalReward()
	go crontab.fetchOldBlockToMiner()
	//go crontab.fetchConfirmRewardsToMinerBlock()
	go crontab.fetchOldTxCountToAccountList()
	//go crontab.GetMinerToblocksByPage()
	go ResetVoteTimer()

	for {
		select {
		case <-check10Sec.C:
			go crontab.fetchPoolVotes()
			go crontab.fetchBlockRewards()
			go crontab.fetchGroups()
			go crontab.UpdateCheckPoint()
			go crontab.fetchOldTxCountToAccountList()

			//go crontab.fetchOldBlockToMiner()

		case <-check30Min.C:
			go crontab.UpdateTurnOver()
			go crontab.SearchTempDeployToken()
		case <-check1Hour.C:
			go crontab.fetchConfirmRewardsToMinerBlock()

		}
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

//update ConfirmRewards
func (crontab *Crontab) fetchConfirmRewardsToMinerBlock() {

	if !atomic.CompareAndSwapInt32(&crontab.isConfirmBlockReward, 0, 1) {
		return
	}
	crontab.ConfirmRewardsToMinerBlock()
	atomic.CompareAndSwapInt32(&crontab.isConfirmBlockReward, 1, 0)

}

func (crontab *Crontab) Reward2MinerBlockByAddress() {

	topHeight := crontab.storage.MaxConfirmBlockRewardHeight()
	//checkpoint := core.BlockChainImpl.LatestCheckPoint()
	maxHeight := topHeight - 1000

	//addr := storage.MinConfirmBlockReward()
	for i := 0; i < 24; i++ {

		addrs := crontab.storage.MinToMaxAccountverReward(uint64(i*100), 100)
		for _, addr := range addrs {
			crontab.Reward2MinerBlockNew(addr.Address, 0, maxHeight)
			crontab.Reward2MinerBlockNew(addr.Address, 1, maxHeight)
		}
	}
	proaddr := crontab.storage.MinToMaxAccountproposalReward(0, 100)
	for _, paddr := range proaddr {
		crontab.Reward2MinerBlockNew(paddr.Address, 1, maxHeight)
	}

}
func (crontab *Crontab) Reward2MinerBlockNew(address string, typeId uint64, maxHeight uint64) bool {
	fmt.Println("Reward2MinerBlockNew,", address, time.Now())

	timestamp := time.Now()
	miners, count := crontab.storage.GetExistminerBlock(address, typeId)
	total, idPrimarys := crontab.slaveStorage.SlaveRewardDatas(address, typeId, maxHeight)
	if len(total) < 1 {
		return true
	}
	sort.Ints(total)
	hights := make([]int, 0)
	if len(total) <= count {
		hights = total[0:]
		crontab.storage.AddminerBlock(address, hights, typeId, miners)
	} else {
		hights = total[0:count]
		crontab.storage.AddminerBlock(address, hights, typeId, miners)
		backward := total[count:]
		size := len(backward) / mysql.MAXCONFIRMREWARDCOUNT
		start := 0
		for l := 0; l <= size; l++ {
			miners, count := crontab.storage.GetExistminerBlock(address, typeId)
			if len(backward[start:]) <= count {
				hights = backward[start:]
				start = start + len(hights)
			} else {
				hights = backward[start : start+count]
				start = start + count
			}
			crontab.storage.AddminerBlock(address, hights, typeId, miners)
		}
	}
	primsize := len(idPrimarys) / 100
	ids := make([]uint64, 0)
	ll := 0
	for z := 0; z <= primsize; z++ {

		if len(idPrimarys[z*100:]) <= 100 {
			ids = idPrimarys[z*100:]

		} else {
			ids = idPrimarys[z*100 : z*100+100]
		}
		ll += len(ids)
		if !util.Errors(crontab.storage.DeleteRewardByIds(ids)) {
			fmt.Println("Reward2MinerBlockNew,delete error", address, ids)
			return false
		}
	}
	fmt.Println("lenth,maxheight,total,address,", address, ",", ll, ",", maxHeight, ",", len(total))
	fmt.Println("cost time,addr:", address, ",", typeId, ",", time.Since(timestamp))
	return true
}

func (crontab *Crontab) fetchBlockRewards() {
	if !atomic.CompareAndSwapInt32(&crontab.isFetchingReward, 0, 1) {
		return
	}
	crontab.excuteBlockRewards()
	atomic.CompareAndSwapInt32(&crontab.isFetchingReward, 1, 0)
}

func (crontab *Crontab) fetchGroups() {

	if crontab.isFetchingGroups {
		return
	}
	crontab.isFetchingGroups = true
	fmt.Println("[server]  fetchGroup height:", crontab.groupHeight)

	groups := crontab.rpcExplore.ExplorerGroupsAfter(crontab.groupHeight)
	fmt.Println("[server]  groups :", groups)

	if groups != nil {
		for i := 0; i < len(groups); i++ {
			crontab.storage.AddGroup(groups[i])
			if groups[i].Height >= crontab.groupHeight {
				crontab.groupHeight = groups[i].Height + 1
			}
		}
	}

	crontab.isFetchingGroups = false

}

//uopdate invalid guard and pool
func (crontab *Crontab) fetchOldBlockToMiner() {

	if !atomic.CompareAndSwapInt32(&crontab.isFetchingBlockToMiner, 0, 1) {
		return
	}
	crontab.supplementBlockToMiner()
	atomic.CompareAndSwapInt32(&crontab.isFetchingBlockToMiner, 1, 0)

}

func (crontab *Crontab) fetchOldTxCountToAccountList() {

	if !atomic.CompareAndSwapInt32(&crontab.isFetchingOldTxCountToAccountList, 0, 1) {
		return
	}
	crontab.supplementOldTxCountToAccountList()
	atomic.CompareAndSwapInt32(&crontab.isFetchingOldTxCountToAccountList, 1, 0)

}

func (crontab *Crontab) supplementOldTxCountToAccountList() {
	if FinishSyncFetchOldAccounts {
		return
	}
	accountCurId, err := crontab.storage.SuppAccountCurID()
	if err != nil {
		return
	}
	// sys中无此变量
	if accountCurId == 0 {
		var success bool
		if accountCurId, success = crontab.storage.SetSuppAccountCurID(0); !success {
			return
		}
	}

	accountMaxId, err := crontab.storage.SuppAccountMaxID()
	if err != nil {
		return
	}
	// sys中无此变量
	if accountMaxId == 0 {
		var success bool
		crontab.storage.GetDB().Model(&models.AccountList{}).Limit(1).Select("id").Order("id desc").Row().Scan(&accountMaxId)
		if accountMaxId, success = crontab.storage.SetSuppAccountMaxID(accountMaxId); !success {
			return
		}
	}

	topHeight, err := crontab.storage.SuppAccountMaxBlockTop()
	if err != nil {
		return
	}
	// sys中无此变量
	if topHeight == 0 {
		topHeight, _ = crontab.storage.TopBlockHeight()
		var success bool
		if topHeight, success = crontab.storage.SetSuppAccountMaxBlockTop(topHeight); !success {
			return
		}
	}
	accounts := make([]*models.AccountList, 0)
	crontab.storage.GetDB().Model(&models.AccountList{}).Order("id asc").Where("id between ? and  ?", accountCurId+1, accountMaxId).Find(&accounts)
	fmt.Println(unsafe.Sizeof(accounts))

	for _, account := range accounts {
		var txCount uint64
		sql := fmt.Sprintf("select count(1) from transactions where source <> '%s' and target = '%s' and block_height <= %d", account.Address, account.Address, topHeight)
		crontab.storage.GetDB().Raw(sql).Row().Scan(&txCount)
		updateAccount := map[string]interface{}{
			"total_transaction": gorm.Expr("total_transaction + ?", txCount),
		}
		updateSys := map[string]interface{}{
			"value": account.ID,
		}
		tx := crontab.storage.GetDB().Begin()
		if tx.Model(&models.AccountList{}).Where("id = ?", account.ID).Updates(updateAccount).Error == nil &&
			tx.Model(&models.Sys{}).Where("variable = ?", mysql.FetchAccountCurID).Updates(updateSys).Error == nil {
			tx.Commit()
			browserlog.BrowserLog.WithFields(logrus.Fields{
				"update":     "success",
				"address":    account.Address,
				"countDelta": txCount,
			}).Info("[supplementOldTxCountToAccountList]", "success update account,commit")
		} else {
			tx.Rollback()
			browserlog.BrowserLog.WithFields(logrus.Fields{
				"update":     "fail",
				"address":    account.Address,
				"countDelta": txCount,
			}).Info("[supplementOldTxCountToAccountList]", "success update account,rollback")
			return
		}
	}
	FinishSyncFetchOldAccounts = true
}

func (crontab *Crontab) supplementBlockToMiner() {
	if FinishSyncSupplementBlockToMiner {
		return
	}
	var aimHeight uint64 = 0
	var height uint64 = 0

	if SupplementBlockToMinerHeight == 0 {
		crontab.storage.GetDB().Model(&models.Sys{}).Select("value").Where("variable = ?", mysql.BlockSupplementAimHeight).Row().Scan(&aimHeight)
		SupplementBlockToMinerHeight = aimHeight
		if SupplementBlockToMinerHeight == 0 {
			crontab.storage.GetDB().Model(&models.BlockToMiner{}).Select("min(reward_height)").Where("reward_height <> 0").Row().Scan(&height)
			if height != 0 {
				sys1 := &models.Sys{
					Variable: mysql.BlockSupplementAimHeight,
					Value:    height,
					SetBy:    "dan",
				}
				sys2 := &models.Sys{
					Variable: mysql.BlockSupplementCurHeight,
					SetBy:    "dan",
				}
				crontab.storage.GetDB().Create(sys1)
				crontab.storage.GetDB().Create(sys2)
				crontab.storage.GetDB().Model(&models.Sys{}).Select("value").Where("variable = ?", mysql.BlockSupplementAimHeight).Row().Scan(&aimHeight)
				SupplementBlockToMinerHeight = aimHeight
			}
		}
	}

	var curHeight uint64 = 0
	crontab.storage.GetDB().Model(&models.Sys{}).Select("value").Where("variable = ?", mysql.BlockSupplementCurHeight).Row().Scan(&curHeight)

	for i := curHeight; i <= SupplementBlockToMinerHeight; i++ {
		blocks, proposalReward := crontab.rpcExplore.GetPreHightRewardByHeight(uint64(i))
		if proposalReward == nil && len(blocks) == 0 {
			fmt.Println("[server]  fetchVerfications empty:", i)
			continue
		}
		fmt.Println("[server]  fetchVerfications:", len(blocks))
		blockToMinersVerfs := make([]*models.BlockToMiner, 0, 0)
		blockToMinersPrpses := make([]*models.BlockToMiner, 0, 0)
		blockToMinerVerf := &models.BlockToMiner{}
		blockToMinerPrps := &models.BlockToMiner{}

		if proposalReward != nil && i != SupplementBlockToMinerHeight {

			blockToMinerPrps = &models.BlockToMiner{
				BlockHeight: proposalReward.BlockHeight,
				BlockHash:   proposalReward.BlockHash,
				CurTime:     proposalReward.CurTime,
				PrpsNodeID:  proposalReward.ProposalID,
				PrpsReward:  proposalReward.ProposalReward,
				PrpsGasFee:  proposalReward.ProposalGasFeeReward,
			}
			blockToMinersPrpses = append(blockToMinersPrpses, blockToMinerPrps)
		}
		for j := 0; j < len(blocks); j++ {
			block := blocks[j]
			if block.VerifierReward != nil {
				verifierBonus := block.VerifierReward
				if verifierBonus.TargetIDs == nil {
					continue
				}
				ids := verifierBonus.TargetIDs
				idsString := make([]string, 0)
				for n := 0; n < len(ids); n++ {
					idsString = append(idsString, ids[n].GetAddrString())

				}

				blockToMinerVerf = &models.BlockToMiner{
					BlockHeight:     block.BlockHeight,
					RewardHeight:    uint64(i),
					VerfNodeCnts:    uint64(len(ids)),
					VerfReward:      verifierBonus.Value,
					VerfTotalGasFee: block.VerifierGasFeeReward,
				}

				idsJSONString, err := json.Marshal(idsString)
				if err == nil {
					blockToMinerVerf.VerfNodeIDs = string(idsJSONString)
				}
				if v, err := strconv.ParseFloat(fmt.Sprintf("%.9f", float64(block.VerifierGasFeeReward)/float64(len(ids))), 64); err == nil {
					blockToMinerVerf.VerfSingleGasFee = v
				}
			}
			blockToMinersVerfs = append(blockToMinersVerfs, blockToMinerVerf)
		}
		tx := crontab.storage.GetDB().Begin()
		if crontab.storage.AddBlockToMinerSupplement(blockToMinersPrpses, blockToMinersVerfs, tx) {
			if tx.Model(&models.Sys{}).Where("variable = ?", mysql.BlockSupplementCurHeight).Update("value", i).Error == nil {
				tx.Commit()
			} else {
				tx.Rollback()
			}
		} else {
			tx.Rollback()
		}
	}
	if aimHeight != 0 {
		FinishSyncSupplementBlockToMiner = true
	}
}

func (crontab *Crontab) proposalsupplyrewarddata(chain *core.FullBlockChain,
	minheight uint64,
	maxheight uint64, upsys bool) {
	fmt.Println("proposalsupplyrewarddata, height,", minheight, ",", maxheight)

	for height := int(minheight); height < int(maxheight); height++ {
		existProposalReward := crontab.storage.ExistRewardBlockHeight(height)
		if !existProposalReward {
			fmt.Println("proposalsupplyrewarddata, sys,", height)
			b := chain.QueryBlockByHeight(uint64(height))
			proposalRewardBlock := crontab.rpcExplore.GetProposalRewardByBlock(b)
			verifications := make([]*models.Reward, 0, 0)
			if proposalRewardBlock != nil {
				proposalReward := &models.Reward{
					Type:         uint64(types.MinerTypeProposal),
					BlockHash:    proposalRewardBlock.BlockHash,
					BlockHeight:  proposalRewardBlock.BlockHeight,
					NodeId:       proposalRewardBlock.ProposalID,
					Value:        proposalRewardBlock.ProposalReward,
					CurTime:      proposalRewardBlock.CurTime,
					RewardHeight: uint64(height),
					GasFee:       float64(proposalRewardBlock.ProposalGasFeeReward),
				}
				verifications = append(verifications, proposalReward)
			}
			crontab.storage.AddRewards(verifications)
		}
		if upsys {
			crontab.storage.GetDB().Model(&models.Sys{}).Where("variable = ?", mysql.BlockSupplementProposalrewardprocessHeight).Update("value", height)
		}
	}
}
func (crontab *Crontab) supplementProposalReward() {

	sysConfig := make([]models.Sys, 0, 0)
	crontab.storage.GetDB().Limit(1).Where("variable = ?", mysql.BlockSupplementProposalrewardprocessHeight).Find(&sysConfig)
	maxsysConfig := make([]models.Sys, 0, 0)
	crontab.storage.GetDB().Limit(1).Where("variable = ?", mysql.BlockSupplementProposalrewardEndHeight).Find(&maxsysConfig)
	var minheight, max uint64
	if len(sysConfig) > 0 {
		minheight = sysConfig[0].Value
		max = maxsysConfig[0].Value
		if max == 0 {
			max = crontab.storage.MaxConfirmBlockRewardHeight()
			crontab.storage.GetDB().Model(&models.Sys{}).Where("variable = ?", mysql.BlockSupplementProposalrewardEndHeight).Update("value", max)
		}
	} else {
		max = crontab.storage.MaxConfirmBlockRewardHeight()
		sys1 := &models.Sys{
			Variable: mysql.BlockSupplementProposalrewardEndHeight,
			Value:    max,
			SetBy:    "xiaoli",
		}
		sys2 := &models.Sys{
			Variable: mysql.BlockSupplementProposalrewardprocessHeight,
			SetBy:    "xiaoli",
			Value:    0,
		}
		crontab.storage.GetDB().Create(sys1)
		crontab.storage.GetDB().Create(sys2)
	}
	if minheight < max {
		chain := core.BlockChainImpl
		crontab.proposalsupplyrewarddata(chain, minheight, max, true)
	}
}

func (crontab *Crontab) fetchReward(localHeight uint64) {

	blocks, proposalReward := crontab.rpcExplore.GetPreHightRewardByHeight(localHeight)
	if proposalReward == nil && len(blocks) == 0 {
		fmt.Println("[server]  fetchVerfications empty:", localHeight)
		return
	}
	fmt.Println("[server]  fetchVerfications:", len(blocks))
	verifications := make([]*models.Reward, 0, 0)
	blockToMinersVerfs := make([]*models.BlockToMiner, 0, 0)
	blockToMinersPrpses := make([]*models.BlockToMiner, 0, 0)
	blockToMinerVerf := &models.BlockToMiner{}
	blockToMinerPrps := &models.BlockToMiner{}

	if proposalReward != nil {
		subProposalReward := &models.Reward{
			Type:         uint64(types.MinerTypeProposal),
			BlockHash:    proposalReward.BlockHash,
			BlockHeight:  proposalReward.BlockHeight,
			NodeId:       proposalReward.ProposalID,
			Value:        proposalReward.ProposalReward,
			CurTime:      proposalReward.CurTime,
			RewardHeight: localHeight,
			GasFee:       float64(proposalReward.ProposalGasFeeReward),
		}
		verifications = append(verifications, subProposalReward)

		blockToMinerPrps = &models.BlockToMiner{
			BlockHeight: proposalReward.BlockHeight,
			BlockHash:   proposalReward.BlockHash,
			CurTime:     proposalReward.CurTime,
			PrpsNodeID:  proposalReward.ProposalID,
			PrpsReward:  proposalReward.ProposalReward,
			PrpsGasFee:  proposalReward.ProposalGasFeeReward,
		}
		blockToMinersPrpses = append(blockToMinersPrpses, blockToMinerPrps)
	}
	for i := 0; i < len(blocks); i++ {
		block := blocks[i]

		if block.VerifierReward != nil {
			verifierBonus := block.VerifierReward
			if verifierBonus.TargetIDs == nil {
				continue
			}
			ids := verifierBonus.TargetIDs
			value := verifierBonus.Value
			gas := fmt.Sprintf("%.9f", float64(block.VerifierGasFeeReward)/float64(len(ids)))
			rewarMoney, _ := strconv.ParseFloat(gas, 64)

			idsString := make([]string, 0)
			for n := 0; n < len(ids); n++ {
				v := models.Reward{}
				v.BlockHash = block.BlockHash
				v.BlockHeight = block.BlockHeight
				v.NodeId = ids[n].GetAddrString()
				v.Value = value
				v.CurTime = block.CurTime
				v.Type = uint64(types.MinerTypeVerify)
				v.RewardHeight = localHeight
				v.GasFee = rewarMoney
				verifications = append(verifications, &v)
				idsString = append(idsString, ids[n].GetAddrString())
			}

			blockToMinerVerf = &models.BlockToMiner{
				BlockHeight:     block.BlockHeight,
				RewardHeight:    localHeight,
				VerfNodeCnts:    uint64(len(ids)),
				VerfReward:      verifierBonus.Value,
				VerfTotalGasFee: block.VerifierGasFeeReward,
			}

			idsJSONString, err := json.Marshal(idsString)
			if err == nil {
				blockToMinerVerf.VerfNodeIDs = string(idsJSONString)
			}

			if v, err := strconv.ParseFloat(fmt.Sprintf("%.9f", float64(block.VerifierGasFeeReward)/float64(len(ids))), 64); err == nil {
				blockToMinerVerf.VerfSingleGasFee = v
			}

			blo := &models.Block{}
			blo.Hash = block.BlockHash
			crontab.storage.SetLoadVerified(blo)

		}
		blockToMinersVerfs = append(blockToMinersVerfs, blockToMinerVerf)
	}
	if len(verifications) > 0 {
		crontab.storage.AddRewards(verifications)
	}
	crontab.storage.AddBlockToMiner(blockToMinersPrpses, blockToMinersVerfs)
}

func getMinerDetail(addr string, height uint64, bizType types.MinerType) *common2.MortGage {
	address := common.StringToAddress(addr)

	minerInfo := core.MinerManagerImpl.GetMiner(address, bizType, height)

	if minerInfo != nil {
		mort := common2.NewMortGageFromMiner(minerInfo)
		return mort
	}
	return nil
}

func (crontab *Crontab) GetMinerToblocksByPage() {
	for h := 0; h < 13000; h++ {
		minerBlock := crontab.storage.GetMinerToblocksByPage(h)
		if len(minerBlock) < 1 {
			break
		}
		for _, block := range minerBlock {
			crontab.storage.UpMinerBlockMaxAndMin(block)
		}
	}
}

func (crontab *Crontab) excutePoolVotes() {
	accountsPool := crontab.storage.GetAccountByRoletype(crontab.maxid, types.MinerPool)
	if accountsPool != nil && len(accountsPool) > 0 {
		//blockheader := core.BlockChainImpl.LatestCheckPoint()
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
			fmt.Println("pool tickets", tickets)
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

func (crontab *Crontab) excuteBlockRewards() {
	height, _ := crontab.storage.TopBlockHeight()
	//checkpoint := core.BlockChainImpl.LatestCheckPoint()
	if crontab.blockRewardHeight > height {
		return
	}
	//topblock := core.BlockChainImpl.QueryTopBlock()
	//topheight := topblock.Height
	rewards, proposalReward := crontab.rpcExplore.GetPreHightRewardByHeight(crontab.blockRewardHeight)
	beginTime := time.Now()
	fmt.Println("[crontab]  fetchBlockRewards height:", crontab.blockRewardHeight, "delay:", time.Since(beginTime))
	if (rewards != nil && len(rewards) > 0) || proposalReward != nil {
		blockrewarddata := crontab.transfer.RewardsToAccounts(rewards, proposalReward)
		accounts := blockrewarddata.MapReward
		mapcountplus := blockrewarddata.MapBlockCount
		//mapMineBlockCount := blockrewarddata.MapMineBlockCount

		mapbalance := make(map[string]float64)
		var balance float64

		for k := range accounts {
			balance = crontab.fetcher.Fetchbalance(k)
			mapbalance[k] = balance
		}
		if crontab.storage.AddBlockRewardMysqlTransaction(accounts,
			mapbalance,
			mapcountplus,
			crontab.blockRewardHeight) {
			crontab.blockRewardHeight += 1
		}
		fmt.Println("Size excuteBlockRewards:", unsafe.Sizeof(blockrewarddata))
		crontab.excuteBlockRewards()
	} else {
		crontab.blockRewardHeight += 1
		fmt.Println("[crontab]  fetchBlockRewards rewards nil:", crontab.blockRewardHeight)
		crontab.excuteBlockRewards()
	}
	fmt.Println("[out excuteBlockRewards] blockHeight:", crontab.blockRewardHeight)

}

func (server *Crontab) consumeReward(localHeight uint64, pre uint64) {
	fmt.Println("[server]  consumeReward height:", localHeight)
	var maxHeight uint64
	_, maxHeight = server.storage.RewardTopBlockHeight()
	chain := core.BlockChainImpl
	blockDetail := chain.QueryBlockCeil(localHeight)
	if blockDetail != nil {
		if maxHeight > pre {
			server.storage.DeleteForkReward(pre, localHeight)
		}
		server.fetchReward(blockDetail.Header.Height)

	}
	//server.isFetchingBlocks = false

}

func (server *Crontab) consumeBlock(localHeight uint64, pre uint64) {

	fmt.Println("[server]  consumeBlock process height:", localHeight)
	var maxHeight uint64
	maxHeight = server.storage.GetTopblock()
	blockDetail, _ := server.fetcher.ExplorerBlockDetail(localHeight)
	if blockDetail != nil {
		if maxHeight > pre {
			server.storage.DeleteForkblock(pre, localHeight, blockDetail.CurTime)
		}
		if server.storage.AddBlock(&blockDetail.Block) {
			trans := make([]*models.Transaction, 0, 0)
			transContract := make([]*models.Transaction, 0, 0)
			for i := 0; i < len(blockDetail.Trans); i++ {
				tran := server.fetcher.ConvertTempTransactionToTransaction(blockDetail.Trans[i])
				tran.BlockHash = blockDetail.Block.Hash
				tran.BlockHeight = blockDetail.Block.Height
				tran.CurTime = blockDetail.Block.CurTime
				tran.CumulativeGasUsed = blockDetail.Receipts[i].CumulativeGasUsed
				if tran.Type == types.TransactionTypeContractCreate {
					tran.ContractAddress = blockDetail.Receipts[i].ContractAddress
					go server.HandleTempTokenTable(tran.Hash, tran.ContractAddress, tran.Source, blockDetail.Receipts[i].Status)
					if blockDetail.Receipts[i] != nil && blockDetail.Receipts[i].Status == 0 {
						go server.HandleVoteContractDeploy(tran.ContractAddress, tran.Source, blockDetail.Block.CurTime)
					}
				}
				if tran.Type == types.TransactionTypeContractCall {
					transContract = append(transContract, tran)

					//是否有transfer log
					if blockDetail.Receipts[i] != nil {
						for _, log := range blockDetail.Receipts[i].Logs {
							if blockDetail.Receipts[i].Status == 0 && common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("transfer"))) {
								server.storage.AddTokenContract(tran, log)
							}
						}
					}
					addr := server.makerdaoOrderContratc
					if addr == "" {
						addr, _ = server.storage.GetMakerdaoOrderaddress(makerdaoOrder)
						if addr != "" {
							server.makerdaoOrderContratc = addr
						}
					}
					priceaddr := server.makerdaoPriceContratc
					if priceaddr == "" {
						priceaddr, _ = server.storage.GetMakerdaoOrderaddress(makerdaoprice)
						if priceaddr != "" {
							server.makerdaoPriceContratc = priceaddr
						}
					}
					if tran.Target != "" && tran.Target == addr && blockDetail.Receipts[i].Status == 0 {
						for _, log := range blockDetail.Receipts[i].Logs {
							server.Handlemakerdao(addr, log, blockDetail.CurTime, tran.Hash)
						}
					}
					if tran.Target != "" && tran.Target == priceaddr && addr != "" && blockDetail.Receipts[i].Status == 0 {
						for _, log := range blockDetail.Receipts[i].Logs {
							server.Handlemakerdaobite(addr, log, blockDetail.CurTime, tran.Hash)
						}

					}
				}
				trans = append(trans, tran)
			}
			server.storage.AddTransactions(trans)
			for i := 0; i < len(blockDetail.Receipts); i++ {
				blockDetail.Receipts[i].BlockHash = blockDetail.Block.Hash
				blockDetail.Receipts[i].BlockHeight = blockDetail.Block.Height
			}
			server.storage.AddReceipts(blockDetail.Receipts)
			server.storage.AddLogs(blockDetail.Receipts, trans, false)
			server.ProcessContract(transContract)
		}
		server.NewConsumeTokenContractTransfer(blockDetail.Block.Height, blockDetail.Block.Hash)
	}
	//server.isFetchingBlocks = false
}

func loadliqitationrate(item string, contract string) uint64 {
	if item == "" || contract == "" {
		return 0
	}
	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		fmt.Errorf("loadliqitationrate err:%v ", err)
		return 0
	}
	contractAddress := common.StringToAddress(contract)
	key := "liquidation@" + item
	iter := db.DataIterator(contractAddress, []byte(key))
	if iter == nil {
		fmt.Errorf("loadliqitationrate err,iter is nil ")
		return 0
	}
	for iter.Next() {
		k := string(iter.Key[:])
		v := tvm.VmDataConvert(iter.Value[:])
		if key == k {
			if num, ok := v.(int64); ok {
				return uint64(num)

			}
		}
	}
	return 0
}
func loadbiteorder(item string, set map[uint64]struct{}, contract string) map[uint64]struct{} {

	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		fmt.Errorf("loadbiteorder err:%v ", err)
		return nil
	}
	contractAddress := common.StringToAddress(contract)
	key := "orderstatus@"
	iter := db.DataIterator(contractAddress, []byte(key))
	if iter == nil {
		fmt.Errorf("loadbiteorder err,iter is nil ")
		return nil
	}
	bite := make(map[uint64]struct{})
	for iter.Next() {
		k := string(iter.Key[:])
		v := tvm.VmDataConvert(iter.Value[:])

		order := strings.Replace(k, "orderstatus@", "", -1)
		orderno, _ := strconv.Atoi(order)
		if _, ok := set[uint64(orderno)]; ok {
			if status, ok := v.(int64); ok && status == 4 {
				bite[uint64(orderno)] = struct{}{}

			}
		}

	}
	return bite
}
func (server *Crontab) Handlemakerdaobite(addr string, log *models.Log, time time.Time, hash string) {
	//bite
	if common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("bite"))) {
		itemname := gjson.Get(log.Data, "args.0").String()
		avergeprice := gjson.Get(log.Data, "args.1").Uint()
		rate := gjson.Get(log.Data, "args.2").Uint()
		status := gjson.Get(log.Data, "args.3").Uint()

		mapData := make(map[string]interface{})
		mapData["status"] = status
		mapData["liquidation_price"] = avergeprice
		mapData["real_liquidation"] = rate

		orders := server.storage.GetDaiPriceContract(itemname)
		// 初始化map
		set := make(map[uint64]struct{})
		// 上面2部可替换为set := make(map[string]struct{})
		for _, value := range orders {
			set[value.OrderId] = struct{}{}
		}
		if len(set) > 0 {
			biteorders := loadbiteorder(itemname, set, addr)
			for k, _ := range biteorders {
				server.storage.Upmakerdao(k, mapData)
			}
		}
	}
}

func (server *Crontab) Handlemakerdao(addr string, log *models.Log, time time.Time, hash string) {
	//applylock
	if common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("applylock"))) {
		addr := gjson.Get(log.Data, "args.0").String()
		price := gjson.Get(log.Data, "args.1").Uint()
		order := gjson.Get(log.Data, "args.2").Uint()
		itemname := gjson.Get(log.Data, "args.3").String()
		num := gjson.Get(log.Data, "args.4").Uint()
		coin := gjson.Get(log.Data, "args.5").Uint()
		rate := loadliqitationrate(itemname, server.makerdaoPriceContratc)
		dai := &models.DaiPriceContract{OrderId: order,
			Address:  addr,
			Price:    price,
			ItemName: itemname,
			Num:      num,
			Coin:     coin,
			Status:   0,
			CurTime:  time,
			TxHash:   hash,
		}
		if rate > 0 {
			dai.Liquidation = rate
		}
		j, _ := json.Marshal(dai)
		server.storage.AddMakerdao(dai)
	}
	//lock
	if common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("lock"))) {
		order := gjson.Get(log.Data, "args.0").Uint()
		status := gjson.Get(log.Data, "args.1").Uint()
		mapData := make(map[string]interface{})
		mapData["status"] = status
		server.storage.Upmakerdao(order, mapData)
	}
	//applywipe
	if common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("applywipe"))) {
		order := gjson.Get(log.Data, "args.0").Uint()
		status := gjson.Get(log.Data, "args.1").Uint()
		mapData := make(map[string]interface{})
		mapData["status"] = status
		server.storage.Upmakerdao(order, mapData)
	}
	//wipe
	if common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("wipe"))) {
		order := gjson.Get(log.Data, "args.0").Uint()
		status := gjson.Get(log.Data, "args.1").Uint()
		mapData := make(map[string]interface{})
		mapData["status"] = status
		server.storage.Upmakerdao(order, mapData)
	}
	//refuselock
	if common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("refuselock"))) {
		order := gjson.Get(log.Data, "args.0").Uint()
		status := gjson.Get(log.Data, "args.1").Uint()
		mapData := make(map[string]interface{})
		mapData["status"] = status
		server.storage.Upmakerdao(order, mapData)
	}

}
func (crontab *Crontab) HandleVoteContractDeploy(contractAddr, promoter string, blockTime time.Time) {
	if !isVoteContract(contractAddr) {
		return
	}

	if !validPoolFromContract(promoter) {
		return
	}

	vote := parseContractKey(contractAddr)
	if vote == nil {
		return
	}
	voteId := vote.VoteId
	voteItems := make([]models.Vote, 0)
	crontab.storage.GetDB().Where("vote_id = ?", voteId).Find(&voteItems)

	// make sure the contract has exist in the vote contract
	if len(voteItems) == 0 {
		return
	}

	// make sure never set the contract address ever
	if len(voteItems[0].ContractAddr) > 0 {
		return
	}

	// check vote status
	if blockTime.Before(voteItems[0].StartTime) {
		vote.Status = models.VoteStatusNotBegin
	} else if (blockTime.After(voteItems[0].StartTime) ||
		blockTime.Equal(voteItems[0].StartTime)) &&
		blockTime.Before(voteItems[0].EndTime) {
		vote.Status = models.VoteStatusInProcess
	} else if blockTime.Equal(voteItems[0].EndTime) ||
		blockTime.After(voteItems[0].EndTime) {
		vote.Status = models.VoteStatusEnded
	}

	vote.Promoter = promoter
	vote.ContractAddr = contractAddr
	vote.Valid = true
	crontab.storage.GetDB().Model(&models.Vote{}).Where("vote_id = ?", voteId).Updates(*vote)

	voteTimer := NewVoteTimer()
	voteTimer.SetVoteStage(vote.Status).
		SetStartTime(vote.StartTime).
		SetEndTime(vote.EndTime)

	HandleVoteTimer(vote.VoteId, voteTimer)

}

func updatePoolMap() {
	PoolNodeMap = make(map[string]struct{})
	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("validPoolFromContract err: ", err)
		return
	}

	iter := db.DataIterator(common.StringToAddress(GuardAndPoolContract), []byte{})
	if iter == nil {
		browserlog.BrowserLog.Error("validPoolFromContract err: ", "iter is nil")
		return
	}

	for iter.Next() {
		k := string(iter.Key[:])
		if strings.HasPrefix(k, "pool_lists@") {
			addr := strings.TrimLeft(k, "pool_lists@")
			PoolNodeMap[addr] = struct{}{}
		}
	}
}

func validPoolFromContract(promoter string) bool {

	updatePoolMap()
	if len(PoolNodeMap) > 0 {
		_, ok := PoolNodeMap[promoter]
		if ok {
			return true
		}
	}

	return false
}

func filterVotes(allVotes map[string]int64) (models.VoteDetails, int, int, bool) {

	validVotes := make(map[int64][]models.Voter)
	totalGuards := make(map[string]int64)

	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("filterVotes err: ", err)
		return nil, 0, 0, false
	}

	iter := db.DataIterator(common.StringToAddress(GuardAndPoolContract), []byte{})
	if iter == nil {
		browserlog.BrowserLog.Error("filterVotes err: ", "iter is nil")
		return nil, 0, 0, false
	}

	var totalGuardsWeight int64
	for iter.Next() {
		k := string(iter.Key[:])
		v := tvm.VmDataConvert(iter.Value[:])
		if strings.HasPrefix(k, "guard_lists@") {
			addr := strings.TrimLeft(k, "guard_lists@")
			weight, ok := v.(int64)
			if !ok {
				continue
			}
			totalGuards[addr] = weight
			totalGuardsWeight += weight
		}
	}

	for addr, option := range allVotes {
		if weight, ok := totalGuards[addr]; ok {
			if _, exists := validVotes[option]; !exists {
				validVotes[option] = []models.Voter{{
					Addr:   addr,
					Weight: weight,
				}}
			} else {
				validVotes[option] = append(validVotes[option], models.Voter{
					Addr:   addr,
					Weight: weight,
				})
			}
		}
	}

	voteDetails := make(models.VoteDetails)

	maxWeight := 0
	for k, voters := range validVotes {
		var totalWeight int64
		var voteStat = new(models.VoteStat)
		for _, voter := range voters {
			totalWeight += voter.Weight
		}
		voteStat.Count = len(voters)
		voteStat.TotalWeight = int(totalWeight)
		voteStat.Voter = voters

		if voteStat.TotalWeight > maxWeight {
			maxWeight = voteStat.TotalWeight
		}
		voteDetails[uint64(k)] = voteStat
	}

	if maxWeight > int(totalGuardsWeight)/2 {
		return voteDetails, len(totalGuards), int(totalGuardsWeight), true
	}

	return voteDetails, len(totalGuards), int(totalGuardsWeight), false

}

func isGuardNode(address string) bool {
	proposalInfo := core.MinerManagerImpl.GetLatestMiner(common.StringToAddress(address), types.MinerTypeProposal)
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(common.StringToAddress(address), types.MinerTypeVerify)

	identify1 := false
	identify2 := false

	if proposalInfo != nil {
		identify1 = proposalInfo.IsGuard()
	}

	if verifierInfo != nil {
		identify2 = verifierInfo.IsGuard()
	}

	return identify1 || identify2
}

func isMinerPool(address string) bool {
	proposalInfo := core.MinerManagerImpl.GetLatestMiner(common.StringToAddress(address), types.MinerTypeProposal)
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(common.StringToAddress(address), types.MinerTypeVerify)

	identify1 := false
	identify2 := false

	if proposalInfo != nil {
		identify1 = proposalInfo.IsMinerPool()
	}

	if verifierInfo != nil {
		identify2 = verifierInfo.IsMinerPool()
	}

	return identify1 || identify2

}

func parseContractKey(contractAddr string) *models.Vote {
	vote := new(models.Vote)
	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("parseContractKey err: ", err)
		return nil
	}
	iter := db.DataIterator(common.StringToAddress(contractAddr), []byte{})
	for iter.Next() {
		k := string(iter.Key[:])
		v := tvm.VmDataConvert(iter.Value[:])
		switch k {
		case "vote_id":
			if v, ok := v.(int64); ok {
				vote.VoteId = uint64(v)
			}
		case "start_time":
			if v, ok := v.(int64); ok {
				vote.StartTime = time.Unix(v, 0)
			}
		case "end_time":
			if v, ok := v.(int64); ok {
				vote.EndTime = time.Unix(v, 0)
			}
		}
	}
	return vote
}

// 统计票数
func CountVotes(voteId uint64) {

	votes := make([]models.Vote, 0)
	err := GlobalCrontab.storage.GetDB().Model(&models.Vote{}).Where("vote_id = ?", voteId).Find(&votes).Error
	if err != nil || len(votes) == 0 {
		browserlog.BrowserLog.Error("CountVotes err: ", err)
		return
	}

	contractAddr := votes[0].ContractAddr

	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("CountVotes err: ", err)
		return
	}

	allVotes := make(map[string]int64)

	iter := db.DataIterator(common.StringToAddress(contractAddr), []byte{})
	if iter == nil {
		browserlog.BrowserLog.Error("CountVotes err: ", "iter is nil")
		return
	}

	for iter.Next() {
		k := string(iter.Key[:])
		v := tvm.VmDataConvert(iter.Value[:])
		if strings.HasPrefix(k, "data@") {
			addr := strings.TrimLeft(k, "data@")
			// 用户所选的选项不能比数据库的多或者选项值不能小于0
			option, ok := v.(int64)
			if !ok {
				return
			}
			if int64(votes[0].OptionsCount) <= option || option < 0 {
				continue
			}
			allVotes[addr] = option
		}
	}

	validVotesDetails, guardCounts, totalGuardsWeight, passed := filterVotes(allVotes)
	if validVotesDetails != nil {
		v, err := json.Marshal(validVotesDetails)
		if err != nil {
			browserlog.BrowserLog.Error("hasEssentialKey err: ", err)
			return
		}
		GlobalCrontab.storage.GetDB().Model(&models.Vote{}).Where("vote_id = ?", voteId).Updates(map[string]interface{}{
			"options_details": string(v),
			"guard_count":     guardCounts,
			"passed":          passed,
			"status":          models.VoteStatusEnded,
			"total_weight":    totalGuardsWeight,
		})
	}
}

func isVoteContract(contractAddr string) bool {
	return hasEssentialKey(contractAddr) && hasChooseFunc(contractAddr) && namedVote(contractAddr)
}

func (crontab *Crontab) isFromMinerPool(promoter string) bool {

	minerPoolList := make([]models.AccountList, 0)
	crontab.storage.GetDB().Model(&models.AccountList{}).Where("role_type = ?", 2).Find(&minerPoolList)
	for _, v := range minerPoolList {
		if v.Address == promoter {
			return true
		}
	}
	return false
}

func hasEssentialKey(addr string) bool {

	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("hasEssentialKey err: ", err)
		return false
	}
	symbol1 := db.GetData(common.StringToAddress(addr), []byte("data"))
	symbol2 := db.GetData(common.StringToAddress(addr), []byte("start_time"))
	symbol3 := db.GetData(common.StringToAddress(addr), []byte("end_time"))
	symbol4 := db.GetData(common.StringToAddress(addr), []byte("vote_id"))

	if (len(symbol1) > 0) &&
		(len(symbol2) > 0) &&
		(len(symbol3) > 0) &&
		(len(symbol4) > 0) {
		return true
	}
	return false
}

func hasChooseFunc(addr string) bool {
	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("hasChooseFunc err: ", err)
		return false
	}
	code := db.GetCode(common.StringToAddress(addr))
	contract := tvm.Contract{}
	err = json.Unmarshal(code, &contract)
	if err != nil {
		browserlog.BrowserLog.Error("hasChooseFunc err: ", err)
		return false
	}
	stringSlice := strings.Split(contract.Code, "\n")
	for k, targetString := range stringSlice {
		targetString = strings.TrimSpace(targetString)
		if strings.HasPrefix(targetString, "@register.public") {
			if len(stringSlice) > k+1 {
				if strings.Index(stringSlice[k+1], " choose(") != -1 {
					return true
				}
			}
		}
	}
	return false
}

// 查看合约class命名是否规范
func namedVote(addr string) bool {
	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("namedVote err: ", err)
		return false
	}
	code := db.GetCode(common.StringToAddress(addr))
	contract := tvm.Contract{}
	err = json.Unmarshal(code, &contract)
	if err != nil {
		browserlog.BrowserLog.Error("namedVote err: ", err)
		return false
	}
	stringSlice := strings.Split(contract.Code, "\n")
	for _, targetString := range stringSlice {
		targetString = strings.TrimSpace(targetString)
		if strings.HasPrefix(targetString, "class") && strings.Index(targetString, " Vote(") != -1 {
			return true
		}
	}
	return false
}

func (crontab *Crontab) HandleTempTokenTable(txHash, tokenAddr, source string, status uint) {
	tempDeployHashes := make([]*models.TempDeployToken, 0)
	crontab.storage.GetDB().Model(&models.TempDeployToken{}).Where("tx_hash = ?", txHash).Find(&tempDeployHashes)
	if len(tempDeployHashes) > 0 {

		if status != 0 {
			if crontab.handleDelSql(txHash) {
				fmt.Printf("success DELETE  FROM temp_deploy_tokens WHERE tx_hash = %s when status != 0\n", txHash)
			} else {
				fmt.Printf("delete TempDeployToken fail when status != 0,tx_hash = %s\n", txHash)
			}
			return
		}

		api := cli.RpcExplorerImpl{}
		tokenContract, err := api.ExplorerTokenMsg(tokenAddr)
		if err != nil {
			if crontab.handleDelSql(txHash) {
				fmt.Printf("success DELETE FROM temp_deploy_tokens WHERE tx_hash = %s and ExplorerTokenMsg err = %s\n", txHash, err.Error())
			} else {
				fmt.Printf("delete TempDeployToken fail WHERE tx_hash = %s and ExplorerTokenMsg err = %s\n", txHash, err.Error())
			}
			fmt.Printf("[HandleTempTokenTable] err :%s", err.Error())
			return
		}

		isSuccess1, isSuccess2, isSuccess3 := true, true, true
		tx := crontab.storage.GetDB().Begin()

		defer func() {
			if isSuccess1 && isSuccess2 && isSuccess3 {
				tx.Commit()
				fmt.Printf("success DELETE  FROM temp_deploy_tokens WHERE tx_hash = %s when tx commit", txHash)
			} else {
				tx.Rollback()
				fmt.Printf("roll back  FROM temp_deploy_tokens WHERE tx_hash = %s when tx commit", txHash)
			}
		}()

		subTokenContract := models.TokenContract{
			ContractAddr: tokenAddr,
			Creator:      source,
			Name:         tokenContract.Name,
			Symbol:       tokenContract.Symbol,
			Decimal:      tokenContract.Decimal,
			HolderNum:    tokenContract.HolderNum,
		}
		if tx.Create(&subTokenContract).Error != nil {
			isSuccess1 = false
			fmt.Printf("create subTokenContract fail,subTokenContract = %+v\n", subTokenContract)
		}

		for key, value := range tokenContract.TokenHolders {
			user := &models.TokenContractUser{
				ContractAddr: tokenAddr,
				Address:      key,
				Value:        value,
			}
			if tx.Create(&user).Error != nil {
				isSuccess2 = false
				fmt.Printf("create TokenContractUser fail,key = %s,value=%s\n", key, value)
				break
			}
		}

		sql := fmt.Sprintf("DELETE  FROM temp_deploy_tokens WHERE tx_hash = '%s'", txHash)
		if tx.Exec(sql).Error != nil {
			isSuccess3 = false
			fmt.Printf("delete TempDeployToken fail,tx_hash = %s\n", txHash)
		}
	}
}

func (crontab *Crontab) handleDelSql(txHash string) bool {
	sql := fmt.Sprintf("DELETE  FROM temp_deploy_tokens WHERE tx_hash = '%s'", txHash)
	if crontab.storage.GetDB().Exec(sql).Error != nil {
		fmt.Printf(" handleDelSql err,tx_hash = %s\n", txHash)
		return false
	}
	return true
}

func (crontab *Crontab) ProcessContract(trans []*models.Transaction) {
	chain := core.BlockChainImpl
	for _, tx := range trans {
		contract := &common2.ContractCall{
			Hash: tx.Hash,
		}
		addressList := crontab.storage.GetContractByHash(tx.Hash)
		wrapper := chain.GetTransactionPool().GetReceipt(common.HexToHash(tx.Hash))
		//contract address
		if wrapper != nil && wrapper.Status == 0 && len(addressList) > 0 {
			go crontab.ConsumeContract(contract, tx.Hash, tx.CurTime)
		}
	}
}

func (tm *Crontab) ConsumeContract(data *common2.ContractCall, hash string, curtime time.Time) {
	tm.storage.UpdateContractTransaction(hash, curtime)
	fmt.Println("for UpdateContractTransaction", util.ObjectTojson(hash))
	browserlog.BrowserLog.Info("for ConsumeContract:", util.ObjectTojson(data))
}

func (crontab *Crontab) OnBlockAddSuccess(simpleBlockHeader *core.SimpleBlockHeader) error {
	preHash := simpleBlockHeader.PreHash
	//preHash := bh.PreHash
	preBlock := core.BlockChainImpl.QueryBlockByHash(preHash)
	preHight := preBlock.Header.Height
	browserlog.BrowserLog.Info("BrowserForkProcessor,pre:", preHight, simpleBlockHeader.Height)
	data := &models.ForkNotify{
		PreHeight:   preHight,
		LocalHeight: simpleBlockHeader.Height,
	}
	crontab.Produce(data)
	crontab.ProduceReward(data)
	//go crontab.UpdateProtectNodeStatus()
	crontab.GochanPunishment(simpleBlockHeader.Height)

	return nil
}
func (crontab *Crontab) GochanPunishment(height uint64) {
	if group.Punishment == nil {
		return
	}
	punish := group.Punishment.Punish
	groupPiece := group.Punishment.GroupPiece
	if punish != nil || groupPiece != nil {
		go crontab.ProcessPunishment(height)
	}
}

func (crontab *Crontab) ProcessPunishment(height uint64) {
	if group.Punishment == nil {
		return
	}
	punish := group.Punishment.Punish
	groupPiece := group.Punishment.GroupPiece
	fmt.Println("for ProcessPunishment,punish:", util.ObjectTojson(punish), ",piece:", util.ObjectTojson(groupPiece), ",height:", height)
	if punish != nil && punish.Height == height {
		for _, addr := range punish.AddressList {
			accounts := &models.AccountList{}
			accounts.Address = addr
			browser.UpdateAccountStake(accounts, 0, crontab.storage)
		}
	}
	if groupPiece != nil && groupPiece.Height == height {
		for _, addr := range groupPiece.AddressList {
			accounts := &models.AccountList{}
			accounts.Address = addr
			browser.UpdateAccountStake(accounts, 0, crontab.storage)
		}
	}
}

func (crontab *Crontab) addGenisisblockAndReward() {
	datablock := crontab.storage.GetBlockByHeight(0)
	if len(datablock) < 1 {
		data := &models.ForkNotify{
			PreHeight:   0,
			LocalHeight: 0,
		}
		crontab.Produce(data)
		crontab.ProduceReward(data)
	}
}

func (crontab *Crontab) Produce(data *models.ForkNotify) {
	crontab.initdata <- data
	fmt.Println("for Produce", util.ObjectTojson(data))
	browserlog.BrowserLog.Info("for Produce:", util.ObjectTojson(data))

}

func (crontab *Crontab) ProduceReward(data *models.ForkNotify) {
	crontab.initRewarddata <- data
	fmt.Println("for ProduceReward", util.ObjectTojson(data))
	browserlog.BrowserLog.Info("for ProduceReward", util.ObjectTojson(data))

}

func (crontab *Crontab) Consume() {

	var ok = true
	for ok {
		select {
		case data := <-crontab.initdata:
			crontab.dataCompensationProcess(data.LocalHeight, data.PreHeight)
			crontab.consumeBlock(data.LocalHeight, data.PreHeight)
			fmt.Println("for Consume", util.ObjectTojson(data))
			browserlog.BrowserLog.Info("for Consume", util.ObjectTojson(data))
		}
	}
}
func (crontab *Crontab) ConsumeContractTransfer() {

	var ok = true
	for ok {
		select {
		case data := <-tvm.ContractTransferData:
			contractTransaction := &models.ContractTransaction{
				ContractCode: data.ContractCode,
				Address:      data.Address,
				Value:        data.Value,
				TxHash:       data.TxHash,
				TxType:       0,
				Status:       0,
				BlockHeight:  data.BlockHeight,
			}
			fmt.Println("ConsumeContractTransfer:", data.Address, ",contractcode:", data.ContractCode, ",json:", util.ObjectTojson(contractTransaction))
			mysql.DBStorage.AddContractTransaction(contractTransaction)
			contractCall := &models.ContractCallTransaction{
				ContractCode: data.ContractCode,
				TxHash:       data.TxHash,
				TxType:       0,
				BlockHeight:  data.BlockHeight,
				Status:       0,
			}
			mysql.DBStorage.AddContractCallTransaction(contractCall)
		}
	}
}

func (crontab *Crontab) NewConsumeTokenContractTransfer(height uint64, hash string) {
	datas, err := tvm.GetTokenContractldbdata(hash)
	if datas == nil {
		return
	}
	if err != nil {
		browserlog.BrowserLog.Error("NewConsumeTokenContract,error")
		return
	}
	for _, data := range datas {
		if !crontab.storage.IsDbTokenContract(data.ContractAddr) {
			continue
		}
		browserlog.BrowserLog.Info("ConsumeTokenContract,json:", util.ObjectTojson(data))
		chain := core.BlockChainImpl
		wrapper := chain.GetTransactionPool().GetReceipt(common.HexToHash(data.TxHash))
		if wrapper != nil {
			if wrapper.Status == 0 && data.Value != "" {
				valuestring := data.Value
				addr := strings.TrimPrefix(data.Addr, "balanceOf@")
				crontab.storage.UpdateTokenUser(data.ContractAddr,
					addr,
					valuestring)
			}

		}

	}
	tvm.MapTokenContractData.Delete(hash)
	var length int
	tvm.MapTokenContractData.Range(func(k, v interface{}) bool {
		length++
		return true
	})
	fmt.Println("delete ConsumeTokenContract:", hash, ",maplen:", length)

}

func (crontab *Crontab) ConsumeTokenContractTransfer(height uint64, hash string) {
	var ok = true
	chanData := tvm.MapTokenChan[hash]
	if chanData == nil || len(chanData) < 1 {
		return
	}
	ticker := time.NewTicker(time.Second * 5)
	for ok {
		select {
		case data := <-chanData:
			if !crontab.storage.IsDbTokenContract(data.ContractAddr) {
				return
			}
			browserlog.BrowserLog.Info("ConsumeTokenContractTransfer,json:", data.TxHash, ",", height, ",", hash)
			chain := core.BlockChainImpl
			wrapper := chain.GetTransactionPool().GetReceipt(common.HexToHash(data.TxHash))
			if wrapper != nil {
				if wrapper.Status == 0 && data.Value != "" {
					valuestring := data.Value
					addr := strings.TrimPrefix(data.Addr, "balanceOf@")
					crontab.storage.UpdateTokenUser(data.ContractAddr,
						addr,
						valuestring)
				}

			}
		case <-ticker.C:
			topHeight := core.BlockChainImpl.Height()
			if height > topHeight || (len(chanData) < 1 && height < topHeight) {
				close(chanData)
				delete(tvm.MapTokenChan, hash)
				return
			}
		}
	}
	fmt.Println("deleete ConsumeTokenContractTransfer", hash)

}

func (crontab *Crontab) ConsumeReward() {

	var ok = true
	for ok {
		select {
		case data := <-crontab.initRewarddata:
			crontab.rewardDataCompensationProcess(data.LocalHeight, data.PreHeight)
			crontab.consumeReward(data.LocalHeight, data.PreHeight)
			fmt.Println("for ConsumeReward", util.ObjectTojson(data))
			browserlog.BrowserLog.Info("for ConsumeReward", util.ObjectTojson(data))

		}
	}
}

func (crontab *Crontab) ConfirmRewardsToMinerBlock() {
	//topHeight := core.BlockChainImpl.Height()
	topHeight := crontab.storage.MaxConfirmBlockRewardHeight()
	//checkpoint := core.BlockChainImpl.LatestCheckPoint()
	if crontab.ConfirmRewardHeight > topHeight-1000 {
		return
	}
	if GetHourMinut1e(time.Now()) == "00" {
		return
	}

	/*if checkpoint.Height > 0 && crontab.ConfirmRewardHeight > checkpoint.Height {
		return
	} else if checkpoint.Height == 0 && crontab.ConfirmRewardHeight > topHeight-100 {
		return
	}*/
	crontab.storage.Reward2MinerBlock(crontab.ConfirmRewardHeight)
	crontab.ConfirmRewardHeight = crontab.ConfirmRewardHeight + 1
	crontab.ConfirmRewardsToMinerBlock()
}

func GetHourMinut1e(tm time.Time) string {
	h := tm.Hour()
	m := tm.Minute()
	return strconv.Itoa(h) + strconv.Itoa(m)
}

func (crontab *Crontab) UpdateTurnOver() {
	filterAddr := []string{
		"zv88200d8e51a63301911c19f72439cac224afc7076ee705391c16f203109c0ccf",
		"zv1d676136438ef8badbc59c89bae08ea3cdfccbbe8f4b22ac8d47361d6a3d510d",
		"zvd675bea39b6f329919fc73c729292c7d8a3d305fb628e47f95986ec725c43824",
		"zvc2a3709ec6f183132faf8bacbc34cdb340eb0a45a69e9d4c26290e93829de64b",
	}

	var turnover float64
	crontab.storage.GetDB().Model(&models.AccountList{}).Select("(sum(total_stake)-sum(other_stake)+sum(balance)+sum(stake_to_other)) as turnover").Where("address not in (?)", filterAddr).Row().Scan(&turnover)
	turnoverString := strconv.FormatFloat(turnover, 'E', -1, 64)

	type TurnoverDB struct {
		turnover string
	}
	configs := make([]*models.Config, 0)
	crontab.storage.GetDB().Model(&models.Config{}).Where("variable = ?", turnoverKey).Find(&configs)
	if len(configs) > 0 {
		crontab.storage.GetDB().Model(&models.Config{}).Where("variable = ?", turnoverKey).Update("value", turnoverString)
	} else {
		config := models.Config{
			Variable: turnoverKey,
			Value:    turnoverString,
		}
		crontab.storage.GetDB().Model(&models.Config{}).Create(&config)
	}
}

func (crontab *Crontab) SearchTempDeployToken() {
	tempDeployHashes := make([]*models.TempDeployToken, 0)
	crontab.storage.GetDB().Model(&models.TempDeployToken{}).Find(&tempDeployHashes)
	if len(tempDeployHashes) > 0 {
		api := &cli.RpcGzvImpl{}
		for _, v := range tempDeployHashes {
			res, err := api.TxReceipt(v.TxHash)
			if err != nil {
				fmt.Println("[SearchTempDeployToken] err:", err)
				return
			}
			if res != nil && res.Transaction != nil && res.Receipt != nil {
				crontab.HandleTempTokenTable(res.Transaction.Hash.Hex(), res.Receipt.ContractAddress.AddrPrefixString(), res.Transaction.Source.AddrPrefixString(), uint(res.Receipt.Status))
			}
		}
	}
}

func (crontab *Crontab) UpdateCheckPoint() {
	checkpoint := core.BlockChainImpl.LatestCheckPoint()
	if checkpoint.Height != 0 {
		GlobalCP = checkpoint.Height
		configs := make([]*models.Config, 0)
		crontab.storage.GetDB().Model(&models.Config{}).Where("variable = ?", cpKey).Find(&configs)
		if len(configs) > 0 {
			crontab.storage.GetDB().Model(&models.Config{}).Where("variable = ?", cpKey).Update("value", strconv.FormatUint(GlobalCP, 10))
		} else {
			config := models.Config{
				Variable: cpKey,
				Value:    strconv.FormatUint(GlobalCP, 10),
			}
			crontab.storage.GetDB().Model(&models.Config{}).Create(&config)
		}
	}
}

func (tm *Crontab) fetchContractAccount() {
	AddressCacheList := tm.storage.GetContractAddressAll()
	for _, address := range AddressCacheList {
		accounts := &models.AccountList{}
		targetAddrInfo := tm.storage.GetAccountById(address)
		//不存在账号
		if targetAddrInfo == nil || len(targetAddrInfo) < 1 {
			accounts.Address = address
			//accounts.TotalTransaction = totalTx
			accounts.Balance = tm.fetcher.Fetchbalance(address)
			if !tm.storage.AddObjects(accounts) {
				return
			}
			//存在账号
		} else {
			accounts.Address = address
			if !tm.storage.UpdateAccountbyAddress(accounts, map[string]interface{}{"total_transaction": gorm.Expr("total_transaction + ?", 0), "balance": tm.fetcher.Fetchbalance(address)}) {
				return
			}

		}
		//update stake
	}

}
func (crontab *Crontab) fetchOldLogs() {

	logs := make([]*models.Log, 0)
	crontab.storage.GetDB().Model(&models.Log{}).Limit(1).Find(&logs)
	if len(logs) == 0 {
		txs := make([]*models.Transaction, 0)
		crontab.storage.GetDB().Model(&models.Transaction{}).Where("type = ?", types.TransactionTypeContractCall).Find(&txs)
		heights := make(map[uint64]bool)
		for _, tx := range txs {
			heights[tx.BlockHeight] = true
		}
		if len(txs) > 0 {
			for height, _ := range heights {
				blockDetail, _ := crontab.fetcher.ExplorerBlockDetail(height)
				if blockDetail != nil {
					for i := 0; i < len(blockDetail.Receipts); i++ {
						blockDetail.Receipts[i].BlockHash = blockDetail.Block.Hash
						blockDetail.Receipts[i].BlockHeight = blockDetail.Block.Height
					}
					trans := make([]*models.Transaction, 0, 0)
					for i := 0; i < len(blockDetail.Trans); i++ {
						tran := crontab.fetcher.ConvertTempTransactionToTransaction(blockDetail.Trans[i])
						tran.BlockHash = blockDetail.Block.Hash
						tran.BlockHeight = blockDetail.Block.Height
						tran.CurTime = blockDetail.Block.CurTime
						trans = append(trans, tran)
					}
					crontab.storage.AddLogs(blockDetail.Receipts, trans, true)
				}
			}
		}
	}
	crontab.fetchContractAccount()
}

func (crontab *Crontab) fetchOldConctactCreate() {

	txsCounts := 0
	crontab.storage.GetDB().Raw("select count(1) from transactions where contract_address <> ? or contract_address <> ?", "", nil).Row().Scan(&txsCounts)

	if txsCounts == 0 {
		txs := make([]models.Transaction, 0)
		crontab.storage.GetDB().Model(&models.Transaction{}).Where("type = ?", types.TransactionTypeContractCreate).Find(&txs)

		for _, tx := range txs {
			receipts := make([]*models.Receipt, 0)
			crontab.storage.GetDB().Limit(1).Model(&models.Receipt{}).Where("tx_hash = ?", tx.Hash).Find(&receipts)

			for _, receipt := range receipts {
				crontab.storage.GetDB().Model(&models.Transaction{}).Where("hash = ?", tx.Hash).Update("contract_address", receipt.ContractAddress)
			}
		}
	}
}

func (crontab *Crontab) fetchOldReceiptToTransaction() {
	trans := make([]*models.Transaction, 0)
	crontab.storage.GetDB().Model(&models.Transaction{}).Not("type = ?", types.TransactionTypeReward).Where("cumulative_gas_used = ?", 0).Last(&trans)
	if len(trans) > 0 {
		type Tx struct {
			Hash string
			Type uint64
		}
		var tx Tx

		receipts := make([]*models.Receipt, 0)

		for i := trans[0].CurIndex; i > 0; i-- {
			crontab.storage.GetDB().Model(&models.Transaction{}).Limit(1).Select("hash,type").Where("cur_index = ?", i).Scan(&tx)
			if tx != (Tx{}) {
				crontab.storage.GetDB().Model(&models.Receipt{}).Where("tx_hash = ?", tx.Hash).Limit(1).Find(&receipts)
				if len(receipts) > 0 {
					if tx.Type == types.TransactionTypeReward {
						continue
					}
					if tx.Type != types.TransactionTypeContractCreate {
						//只更新cumulative_gas_used 字段
						crontab.storage.GetDB().Model(&models.Transaction{}).Where("cur_index = ?", i).Update("cumulative_gas_used", receipts[0].CumulativeGasUsed)
					} else {
						//contract_address字段和cumulative_gas_used 都更新
						crontab.storage.GetDB().Model(&models.Transaction{}).Where("cur_index = ?", i).Updates(map[string]interface{}{
							"cumulative_gas_used": receipts[0].CumulativeGasUsed,
							"contract_address":    receipts[0].ContractAddress,
						})
					}
				}
			}
		}
	}
}

////todo
//func (crontab *Crontab) fetchOldStakeInfo() {
//
//	filterAddrs := []string{
//		"zv96ea1d09221beb7c97edcda812736fd58f44f2add466f36793fac481a94f5710",
//		"zv5de43446effa9bf38bff2cc8359b5ae05822e5a94bd953bfe07fbde6461d545a",
//		"zv426c91a09fef18d953687e89535a0161aa081c31a562b2a5902239c8030904a9",
//		"zvdfdac30bafeeba0825c77389b03089d5711bce48610b7c106658b0776abfd05b",
//	}
//
//	trans := make([]*models.Transaction, 0)
//	crontab.storage.GetDB().Model(&models.Transaction{}).Where("source in (?) and type in (?)", filterAddrs, []int{types.TransactionTypeStakeAdd, types.TransactionTypeStakeRefund}).Find(&trans)
//	if len(trans) > 0 {
//		wg := sync.WaitGroup{}
//		wg.Add(len(trans))
//		for _, tran := range trans {
//			go func(tran *models.Transaction) {
//				defer wg.Done()
//				if tran.Source != tran.Target{
//					if tran.Type == types.TransactionTypeStakeAdd{
//
//					}else if tran.Type ==types.TransactionTypeStakeRefund{
//
//					}
//				}
//			}(tran)
//		}
//		wg.Wait()
//	}
//}

// 更新守护节点和矿池状态
func (crontab *Crontab) UpdateProtectNodeStatus() {

	expiredNodes := core.ExpiredGuardNodes
	if len(expiredNodes) > 0 {
		// 更新守护节点状态
		protectNodes := make([]*models.AccountList, 0)
		for _, node := range expiredNodes {
			crontab.storage.GetDB().Model(&models.AccountList{}).Where("address = ?", node.AddrPrefixString()).Limit(1).Find(&protectNodes)
			if len(protectNodes) > 0 {
				browser.UpdateAccountStake(protectNodes[0], 0, crontab.storage)
			}
		}

		// 更新矿池状态
		browser.UpdatePoolStatus(crontab.storage)
	}
}

func (crontab *Crontab) dataCompensationProcess(notifyHeight uint64, notifyPreHeight uint64) {
	timenow := time.Now()
	if !crontab.isInited {
		//fmt.Println("[Storage]  dataCompensationProcess start: ", notifyHeight, notifyPreHeight)
		browserlog.BrowserLog.Info("[Storage]  dataCompensationProcess start: ", notifyHeight, notifyPreHeight)

		dbMaxHeight := crontab.blockTopHeight
		if dbMaxHeight > 0 && dbMaxHeight <= notifyPreHeight {
			blockceil := core.BlockChainImpl.QueryBlockCeil(dbMaxHeight)
			time := time.Now()
			if blockceil != nil {
				time = blockceil.Header.CurTime.Local()
			}
			crontab.storage.DeleteForkblock(dbMaxHeight-1, dbMaxHeight, time)
			crontab.dataCompensation(dbMaxHeight, notifyPreHeight)
		}
		crontab.isInited = true
		browserlog.BrowserLog.Info("[Storage]  dataCompensationProcess cost: ", time.Since(timenow))
	}
	//fmt.Println("[Storage]  dataCompensationProcess cost: ", time.Since(timenow))
}

func (crontab *Crontab) rewardDataCompensationProcess(notifyHeight uint64, notifyPreHeight uint64) {
	timenow := time.Now()
	if !crontab.isInitedReward {
		//fmt.Println("[Storage]  rewardDataCompensationProcess start: ", notifyHeight, notifyPreHeight)
		browserlog.BrowserLog.Info("[Storage]  rewardDataCompensationProcess start: ", notifyHeight, notifyPreHeight)

		dbMaxHeight := crontab.rewardStorageDataHeight
		if dbMaxHeight > 0 && dbMaxHeight <= notifyPreHeight {
			crontab.storage.DeleteForkReward(dbMaxHeight-1, dbMaxHeight)
			//crontab.proposalrewardSupplementarydata(dbMaxHeight)
			crontab.rewarddataCompensation(dbMaxHeight, notifyPreHeight)
		}
		crontab.isInitedReward = true
		browserlog.BrowserLog.Info("[Storage]  rewardDataCompensationProcess cost: ", time.Since(timenow))

	}
	//fmt.Println("[Storage]  rewardDataCompensationProcess cost: ", time.Since(timenow))

}

//data Compensation
func (crontab *Crontab) dataCompensation(dbMaxHeight uint64, notifyPreHeight uint64) {
	blockceil := core.BlockChainImpl.QueryBlockCeil(dbMaxHeight)
	if blockceil != nil {
		preBlockceil := core.BlockChainImpl.QueryBlockByHash(blockceil.Header.PreHash)
		crontab.consumeBlock(blockceil.Header.Height, preBlockceil.Header.Height)
		//fmt.Println("for dataCompensation,", blockceil.Header.Height, ",", preBlockceil.Header.Height)
		browserlog.BrowserLog.Info("for dataCompensation,", blockceil.Header.Height, ",", preBlockceil.Header.Height)
		crontab.blockTopHeight = blockceil.Header.Height + 1
	} else {
		crontab.blockTopHeight += 1
	}
	//fmt.Println("[Storage]  dataCompensationProcess procee: ", crontab.blockTopHeight)
	browserlog.BrowserLog.Info("[Storage]  dataCompensationProcess procee: ", crontab.blockTopHeight)
	if crontab.blockTopHeight <= notifyPreHeight {
		crontab.dataCompensation(crontab.blockTopHeight, notifyPreHeight)
	}

}

func (crontab *Crontab) proposalrewardSupplementarydata(height uint64) {
	chain := core.BlockChainImpl
	verrewardheight := crontab.storage.MinBlockHeightverReward()
	if verrewardheight > 0 {
		crontab.proposalsupplyrewarddata(chain, verrewardheight, height, false)
	}

}

//data Compensation
func (crontab *Crontab) rewarddataCompensation(dbMaxHeight uint64, notifyPreHeight uint64) {
	blockceil := core.BlockChainImpl.QueryBlockCeil(dbMaxHeight)
	if blockceil != nil {
		preBlockceil := core.BlockChainImpl.QueryBlockByHash(blockceil.Header.PreHash)
		crontab.consumeReward(blockceil.Header.Height, preBlockceil.Header.Height)
		//fmt.Println("for rewarddataCompensation,", blockceil.Header.Height, ",", preBlockceil.Header.Height)
		browserlog.BrowserLog.Info("for rewarddataCompensation,", blockceil.Header.Height, ",", preBlockceil.Header.Height)
		crontab.rewardStorageDataHeight = blockceil.Header.Height + 1
	} else {
		crontab.rewardStorageDataHeight += 1
	}
	//fmt.Println("[Storage]  rewarddataCompensation procee: ", crontab.rewardStorageDataHeight)
	browserlog.BrowserLog.Info("[Storage]  rewarddataCompensation procee: ", crontab.rewardStorageDataHeight)

	if crontab.rewardStorageDataHeight <= notifyPreHeight {
		crontab.rewarddataCompensation(crontab.rewardStorageDataHeight, notifyPreHeight)
	}

}

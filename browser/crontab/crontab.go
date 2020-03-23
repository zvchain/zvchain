package crontab

import (
	"encoding/json"
	"fmt"
	common2 "github.com/zvchain/zvchain/browser/common"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"strconv"
	"time"
)

var (
	GlobalCP                     uint64
	SupplementBlockToMinerHeight uint64
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

	page              uint64
	maxid             uint
	accountPrimaryId  uint64
	isFetchingReward  int32
	isFetchingConsume int32
	isFetchingGroups  bool
	groupHeight       uint64
	isInited          bool
	isInitedReward    bool

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
	dbPassword string, reset bool, dbName string) *Crontab {
	server := &Crontab{
		initdata:       make(chan *models.ForkNotify, 10000),
		initRewarddata: make(chan *models.ForkNotify, 10000),
	}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, false, dbName)
	server.storage.InitCurConfig()
	_, server.rewardStorageDataHeight = server.storage.RewardTopBlockHeight()
	//notify.BUS.Subscribe(notify.BlockAddSucc, server.OnBlockAddSuccess)
	server.blockRewardHeight = server.storage.TopBlockRewardHeight(mysql.Blockrewardtopheight)
	server.blockTopHeight = server.storage.GetTopblock()
	if server.blockRewardHeight > 0 {
		server.blockRewardHeight += 1
	}
	go server.HandleOnBlockSuccess()
	go server.loop()
	return server
}

func (crontab *Crontab) HandleOnBlockSuccess() {
	for {
		simpleBlockHeader := <-core.OnBlockSuccessChan
		crontab.OnBlockAddSuccess(simpleBlockHeader)
	}
}

func (crontab *Crontab) loop() {
	go crontab.Consume()
	go crontab.ConsumeReward()

}

func (crontab *Crontab) proposalsupplyrewarddata(chain *core.FullBlockChain,
	minheight uint64,
	maxheight uint64, upsys bool) {
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
	/*if len(verifications) > 0 {
		crontab.storage.AddRewards(verifications)
	}*/
	crontab.storage.AddBlockToMiner(blockToMinersPrpses, blockToMinersVerfs)
}

func (server *Crontab) consumeReward(localHeight uint64, pre uint64) {
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

	var maxHeight uint64
	maxHeight = server.storage.GetTopblock()
	blockDetail, _ := server.fetcher.ExplorerBlockDetail(localHeight)
	if blockDetail != nil {
		if maxHeight > pre {
			server.storage.DeleteForkblock(pre, localHeight, blockDetail.CurTime)
		}
		if server.storage.AddBlock(&blockDetail.Block) {
			trans := make([]*models.Transaction, 0, 0)
			for i := 0; i < len(blockDetail.Trans); i++ {
				tran := server.fetcher.ConvertTempTransactionToTransaction(blockDetail.Trans[i])
				tran.BlockHash = blockDetail.Block.Hash
				tran.BlockHeight = blockDetail.Block.Height
				tran.CurTime = blockDetail.Block.CurTime
				tran.CumulativeGasUsed = blockDetail.Receipts[i].CumulativeGasUsed
				if tran.Type == types.TransactionTypeContractCreate {
					tran.ContractAddress = blockDetail.Receipts[i].ContractAddress
				}
				trans = append(trans, tran)
			}
			server.storage.AddTransactions(trans)
			/*for i := 0; i < len(blockDetail.Receipts); i++ {
				blockDetail.Receipts[i].BlockHash = blockDetail.Block.Hash
				blockDetail.Receipts[i].BlockHeight = blockDetail.Block.Height
			}
			server.storage.AddReceipts(blockDetail.Receipts)*/
		}
	}
	//server.isFetchingBlocks = false
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

	return nil
}

func (crontab *Crontab) Produce(data *models.ForkNotify) {
	crontab.initdata <- data
	browserlog.BrowserLog.Info("for Produce:", util.ObjectTojson(data))

}

func (crontab *Crontab) ProduceReward(data *models.ForkNotify) {
	crontab.initRewarddata <- data
	browserlog.BrowserLog.Info("for ProduceReward", util.ObjectTojson(data))

}

func (crontab *Crontab) Consume() {

	var ok = true
	for ok {
		select {
		case data := <-crontab.initdata:
			crontab.dataCompensationProcess(data.LocalHeight, data.PreHeight)
			crontab.consumeBlock(data.LocalHeight, data.PreHeight)
			browserlog.BrowserLog.Info("for Consume", util.ObjectTojson(data))
		}
	}
}

func (crontab *Crontab) ConsumeReward() {

	var ok = true
	for ok {
		select {
		case data := <-crontab.initRewarddata:
			crontab.rewardDataCompensationProcess(data.LocalHeight, data.PreHeight)
			crontab.consumeReward(data.LocalHeight, data.PreHeight)
			browserlog.BrowserLog.Info("for ConsumeReward", util.ObjectTojson(data))

		}
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

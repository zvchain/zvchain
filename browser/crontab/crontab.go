package crontab

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
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
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"math/big"
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
)

var (
	GlobalCP                         uint64
	FinishSyncSupplementBlockToMiner bool = false
	SupplementBlockToMinerHeight     uint64
)

const (
	ProposalStakeType = 0
	VerifyStakeType   = 1
)

type Crontab struct {
	storage                       *mysql.Storage
	blockRewardHeight             uint64
	blockTopHeight                uint64
	rewardStorageDataHeight       uint64
	blockToMinerStorageDataHeight uint64
	curblockcount                 uint64
	curTrancount                  uint64
	ConfirmRewardHeight           uint64

	page              uint64
	maxid             uint
	accountPrimaryId  uint64
	isFetchingReward  int32
	isFetchingConsume int32
	isFetchingGroups  bool
	groupHeight       uint64
	isInited          bool
	isInitedReward    bool

	isFetchingPoolvotes  int32
	isConfirmBlockReward int32
	rpcExplore           *Explore
	transfer             *Transfer
	fetcher              *common2.Fetcher
	isFetchingBlocks     bool
	initdata             chan *models.ForkNotify
	initRewarddata       chan *models.ForkNotify

	isFetchingVerfications bool
}

func NewServer(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *Crontab {
	server := &Crontab{
		initdata:       make(chan *models.ForkNotify, 10000),
		initRewarddata: make(chan *models.ForkNotify, 10000),
	}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, false)
	server.addGenisisblockAndReward()
	server.storage.InitCurConfig()
	_, server.rewardStorageDataHeight = server.storage.RewardTopBlockHeight()
	go server.ConsumeContractTransfer()
	notify.BUS.Subscribe(notify.BlockAddSucc, server.OnBlockAddSuccess)

	server.blockRewardHeight = server.storage.TopBlockRewardHeight(mysql.Blockrewardtopheight)
	confirmRewardHeight := server.storage.MinConfirmBlockRewardHeight()
	if confirmRewardHeight > 0 {
		server.ConfirmRewardHeight = confirmRewardHeight
	}
	server.blockTopHeight = server.storage.GetTopblock()
	if server.blockRewardHeight > 0 {
		server.blockRewardHeight += 1
	}
	go server.loop()
	return server
}

func (crontab *Crontab) loop() {
	var (
		check10Sec = time.NewTicker(check10SecInterval)
		check30Min = time.NewTicker(check30MinInterval)
	)
	defer check10Sec.Stop()
	go crontab.fetchOldLogs()
	go crontab.fetchOldReceiptToTransaction()
	go crontab.fetchPoolVotes()
	go crontab.fetchGroups()
	go crontab.fetchOldConctactCreate()

	go crontab.fetchBlockRewards()
	go crontab.Consume()
	go crontab.ConsumeReward()
	go crontab.UpdateTurnOver()
	go crontab.UpdateCheckPoint()
	go crontab.supplementProposalReward()
	go crontab.fetchOldBlockToMiner()
	go crontab.fetchConfirmRewardsToMinerBlock()
	for {
		select {
		case <-check10Sec.C:
			go crontab.fetchPoolVotes()
			go crontab.fetchBlockRewards()
			go crontab.fetchGroups()
			go crontab.UpdateCheckPoint()
			go crontab.fetchOldBlockToMiner()

		case <-check30Min.C:
			go crontab.UpdateTurnOver()
			go crontab.SearchTempDeployToken()
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

	if !atomic.CompareAndSwapInt32(&crontab.isFetchingPoolvotes, 0, 1) {
		return
	}
	crontab.supplementBlockToMiner()
	atomic.CompareAndSwapInt32(&crontab.isFetchingPoolvotes, 1, 0)

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

	for i := curHeight; i < SupplementBlockToMinerHeight; i++ {
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

		if proposalReward != nil {

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
				idsString := ""
				for n := 0; n < len(ids); n++ {
					idsString = fmt.Sprintf(idsString+"%s\r\n", ids[n].GetAddrString())
				}

				blockToMinerVerf = &models.BlockToMiner{
					BlockHeight:     block.BlockHeight,
					RewardHeight:    uint64(i),
					VerfNodeIDs:     idsString,
					VerfNodeCnts:    uint64(len(ids)),
					VerfReward:      verifierBonus.Value,
					VerfTotalGasFee: block.VerifierGasFeeReward,
				}
				if v, err := strconv.ParseFloat(fmt.Sprintf("%.9f", float64(block.VerifierGasFeeReward)/float64(len(ids))), 64); err == nil {
					blockToMinerVerf.VerfSingleGasFee = v
				}
			}
			blockToMinersVerfs = append(blockToMinersVerfs, blockToMinerVerf)
		}
		crontab.storage.AddBlockToMinerSupplement(blockToMinersPrpses, blockToMinersVerfs)
		curHeight++
		crontab.storage.GetDB().Model(&models.Sys{}).Where("variable = ?", mysql.BlockSupplementCurHeight).Update("value", curHeight)
	}
	if aimHeight != 0 {
		FinishSyncSupplementBlockToMiner = true
	}
}

func (crontab *Crontab) supplementProposalReward() {
	chain := core.BlockChainImpl
	topheight := chain.Height()

	for height := 0; height < int(topheight); height++ {
		existProposalReward := crontab.storage.ExistRewardBlockHeight(height)
		if !existProposalReward {
			b := chain.QueryBlockByHeight(uint64(height))
			proposalReward := crontab.rpcExplore.GetProposalRewardByBlock(b)
			verifications := make([]*models.Reward, 0, 0)
			if proposalReward != nil {
				proposalReward := &models.Reward{
					Type:         uint64(types.MinerTypeProposal),
					BlockHash:    proposalReward.BlockHash,
					BlockHeight:  proposalReward.BlockHeight,
					NodeId:       proposalReward.ProposalID,
					Value:        proposalReward.ProposalReward,
					CurTime:      proposalReward.CurTime,
					RewardHeight: uint64(height),
					GasFee:       float64(proposalReward.ProposalGasFeeReward),
				}
				verifications = append(verifications, proposalReward)
			}
			crontab.storage.AddRewards(verifications)
		}
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

			idsString := ""
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
				idsString = fmt.Sprintf(idsString+"%s\r\n", ids[n].GetAddrString())
			}

			blockToMinerVerf = &models.BlockToMiner{
				BlockHeight:     block.BlockHeight,
				RewardHeight:    localHeight,
				VerfNodeIDs:     idsString,
				VerfNodeCnts:    uint64(len(ids)),
				VerfReward:      verifierBonus.Value,
				VerfTotalGasFee: block.VerifierGasFeeReward,
			}
			if v, err := strconv.ParseFloat(fmt.Sprintf("%.9f", float64(block.VerifierGasFeeReward)/float64(len(ids))), 64); err == nil {
				blockToMinerVerf.VerfSingleGasFee = v
			}

			blo := &models.Block{}
			blo.Hash = block.BlockHash
			crontab.storage.SetLoadVerified(blo)

		}
		blockToMinersVerfs = append(blockToMinersVerfs, blockToMinerVerf)
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
	if rewards != nil && len(rewards) > 0 {
		blockrewarddata := crontab.transfer.RewardsToAccounts(rewards, proposalReward)
		accounts := blockrewarddata.MapReward
		mapcountplus := blockrewarddata.MapBlockCount
		mapMineBlockCount := blockrewarddata.MapMineBlockCount

		mapbalance := make(map[string]float64)
		var balance float64

		for k := range accounts {
			balance = crontab.fetcher.Fetchbalance(k)
			mapbalance[k] = balance
		}
		if crontab.storage.AddBlockRewardMysqlTransaction(accounts,
			mapbalance,
			mapcountplus,
			crontab.blockRewardHeight) &&
			crontab.storage.UpdateMineBlocks(mapMineBlockCount) {
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
	fmt.Println("[server]  consumeBlock height:", localHeight)
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
				}
				if tran.Type == types.TransactionTypeContractCall {
					transContract = append(transContract, tran)

					//是否有transfer log
					for _, log := range blockDetail.Receipts[i].Logs {
						if blockDetail.Receipts[i].Status == 0 && common.HexToHash(log.Topic) == common.BytesToHash(common.Sha256([]byte("transfer"))) {
							server.storage.AddTokenContract(tran, log)
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

func (crontab *Crontab) HandleTempTokenTable(txHash, tokenAddr, source string, status uint) {
	fmt.Printf("[in HandleTempTokenTable] txHash:%v, tokenAddr:%v, source:%v, status:%v\n", txHash, tokenAddr, source, status)
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

func (crontab *Crontab) OnBlockAddSuccess(message notify.Message) error {
	block := message.GetData().(*types.Block)
	bh := block.Header
	preHash := bh.PreHash
	preBlock := core.BlockChainImpl.QueryBlockByHash(preHash)
	preHight := preBlock.Header.Height
	browserlog.BrowserLog.Info("BrowserForkProcessor,pre:", preHight, bh.Height)
	data := &models.ForkNotify{
		PreHeight:   preHight,
		LocalHeight: bh.Height,
	}
	go crontab.Produce(data)
	go crontab.ProduceReward(data)
	go crontab.UpdateProtectNodeStatus()
	crontab.GochanPunishment(bh.Height)

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
			if wrapper.Status == 0 && data.Value != nil {
				var valuestring string
				if value, ok := data.Value.(int64); ok {
					valuestring = big.NewInt(value).String()
				} else if value, ok := data.Value.(*big.Int); ok {
					valuestring = value.String()
				}
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
				if wrapper.Status == 0 && data.Value != nil {
					var valuestring string
					if value, ok := data.Value.(int64); ok {
						valuestring = big.NewInt(value).String()
					} else if value, ok := data.Value.(*big.Int); ok {
						valuestring = value.String()
					}
					addr := strings.TrimPrefix(string(data.Addr), "balanceOf@")
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
	/*if checkpoint.Height > 0 && crontab.ConfirmRewardHeight > checkpoint.Height {
		return
	} else if checkpoint.Height == 0 && crontab.ConfirmRewardHeight > topHeight-100 {
		return
	}*/
	crontab.storage.Reward2MinerBlock(crontab.ConfirmRewardHeight)
	crontab.ConfirmRewardHeight = crontab.ConfirmRewardHeight + 1
	crontab.ConfirmRewardsToMinerBlock()
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
		if len(txs) > 0 {
			for _, tx := range txs {
				blockDetail, _ := crontab.fetcher.ExplorerBlockDetail(tx.BlockHeight)
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

	/*expiredNodes := core.ExpiredGuardNodes
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
	}*/
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

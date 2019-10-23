package crontab

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/browser"
	common2 "github.com/zvchain/zvchain/browser/common"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	check10SecInterval = time.Second * 10
)

const (
	ProposalStakeType = 0
	VerifyStakeType   = 1
)

type Crontab struct {
	storage                 *mysql.Storage
	blockRewardHeight       uint64
	blockTopHeight          uint64
	rewardStorageDataHeight uint64
	curblockcount           uint64
	curTrancount            uint64

	page              uint64
	maxid             uint
	accountPrimaryId  uint64
	isFetchingReward  int32
	isFetchingConsume int32
	isFetchingGroups  bool
	groupHeight       uint64
	isInited          bool
	isInitedReward    bool

	isFetchingPoolvotes int32
	rpcExplore          *Explore
	transfer            *Transfer
	fetcher             *common2.Fetcher
	isFetchingBlocks    bool
	initdata            chan *models.ForkNotify
	initRewarddata      chan *models.ForkNotify

	isFetchingVerfications bool
}

func NewServer(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *Crontab {
	server := &Crontab{
		initdata:       make(chan *models.ForkNotify, 10000),
		initRewarddata: make(chan *models.ForkNotify, 10000),
	}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, false)
	server.addGenisisblock()
	server.storage.InitCurConfig()
	_, server.rewardStorageDataHeight = server.storage.RewardTopBlockHeight()
	go server.ConsumeContractTransfer()
	notify.BUS.Subscribe(notify.BlockAddSucc, server.OnBlockAddSuccess)

	server.blockRewardHeight = server.storage.TopBlockRewardHeight(mysql.Blockrewardtopheight)
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
	)
	defer check10Sec.Stop()
	go crontab.fetchOldLogs()
	go crontab.fetchOldReceiptToTransaction()
	go crontab.fetchPoolVotes()
	go crontab.fetchGroups()

	go crontab.fetchBlockRewards()
	go crontab.Consume()
	go crontab.ConsumeReward()

	for {
		select {
		case <-check10Sec.C:
			go crontab.fetchPoolVotes()
			go crontab.fetchBlockRewards()
			go crontab.fetchGroups()
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

func (crontab *Crontab) fetchReward(localHeight uint64) {

	blocks := crontab.rpcExplore.GetPreHightRewardByHeight(localHeight)

	if blocks == nil || len(blocks) == 0 {
		fmt.Println("[server]  fetchVerfications empty:", localHeight)
		return
	}
	fmt.Println("[server]  fetchVerfications:", len(blocks))
	for i := 0; i < len(blocks); i++ {
		block := blocks[i]
		verifications := make([]*models.Reward, 0, 0)
		if block.ProposalReward > 0 {
			//mort := getMinerDetail(block.ProposalID, block.BlockHeight, types.MinerTypeProposal)
			proposalReward := &models.Reward{
				Type:        uint64(types.MinerTypeProposal),
				BlockHash:   block.BlockHash,
				BlockHeight: block.BlockHeight,
				NodeId:      block.ProposalID,
				Value:       block.ProposalReward,
				//RoleType:     uint64(mort.Identity),
				CurTime:      block.CurTime,
				RewardHeight: localHeight,
				GasFee:       float64(block.ProposalGasFeeReward),
			}
			//if mort != nil {
			//	proposalReward.Stake = mort.Stake
			//	proposalReward.RoleType = uint64(mort.Identity)
			//}
			verifications = append(verifications, proposalReward)

		}
		if block.VerifierReward != nil {
			verifierBonus := block.VerifierReward
			if verifierBonus.TargetIDs == nil {
				continue
			}
			ids := verifierBonus.TargetIDs
			value := verifierBonus.Value
			gas := fmt.Sprintf("%.9f", float64(block.VerifierGasFeeReward)/float64(len(ids)))
			rewarMoney, _ := strconv.ParseFloat(gas, 64)

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
				//mort := getMinerDetail(v.NodeId, block.BlockHeight, types.MinerTypeVerify)
				//if mort != nil {
				//	v.Stake = mort.Stake
				//	v.RoleType = uint64(mort.Identity)
				//}
				verifications = append(verifications, &v)
			}
			blo := &models.Block{}
			blo.Hash = block.BlockHash
			crontab.storage.SetLoadVerified(blo)

		}
		crontab.storage.AddRewards(verifications)
	}

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
	rewards := crontab.rpcExplore.GetPreHightRewardByHeight(crontab.blockRewardHeight)
	beginTime := time.Now()
	fmt.Println("[crontab]  fetchBlockRewards height:", crontab.blockRewardHeight, "delay:", time.Since(beginTime))
	if rewards != nil && len(rewards) > 0 {
		blockrewarddata := crontab.transfer.RewardsToAccounts(rewards)
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
				}
				if tran.Type == types.TransactionTypeContractCall {
					transContract = append(transContract, tran)
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
	}
	//server.isFetchingBlocks = false

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
		if wrapper.Status == 0 && len(addressList) > 0 {
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

func (crontab *Crontab) addGenisisblock() {
	datablock := crontab.storage.GetBlockByHeight(0)
	if len(datablock) < 1 {
		data := &models.ForkNotify{
			PreHeight:   0,
			LocalHeight: 0,
		}
		crontab.Produce(data)
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
			fmt.Println("ConsumeContractTransfer:", data.Address, ",contractcode:", data.ContractCode)
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

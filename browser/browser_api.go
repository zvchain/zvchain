package browser

import (
	"fmt"
	"github.com/jinzhu/gorm"
	common2 "github.com/zvchain/zvchain/browser/common"
	"github.com/zvchain/zvchain/browser/crontab"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/cmd/gzv/rpc"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const checkInterval = time.Second * 5

const (
	dismissGroup = iota
	workGroup
	prepareGroup
)

//var AddressCacheList map[string]uint64

type DBMmanagement struct {
	sync.Mutex
	blockHeight        uint64
	prepareGroupHeight uint64
	groupHeight        uint64
	dismissGropHeight  uint64
	storage            *mysql.Storage //待迁移

	isFetchingBlocks        int32
	isFetchingWorkGroups    bool
	isFetchingPrepareGroups bool
	isFetchingDismissGroups bool
	fetcher                 *common2.Fetcher
}

func NewDBMmanagement(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool, resetcrontab bool) *DBMmanagement {
	tvm.ContractTransferData = make(chan *tvm.ContractTransfer, 500)
	tablMmanagement := &DBMmanagement{}
	tablMmanagement.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, resetcrontab)

	tablMmanagement.blockHeight, _ = tablMmanagement.storage.TopBlockHeight()
	if tablMmanagement.blockHeight > 0 {
		tablMmanagement.blockHeight += 1
	}
	tablMmanagement.groupHeight, _ = tablMmanagement.storage.TopGroupHeight()
	tablMmanagement.prepareGroupHeight, _ = tablMmanagement.storage.TopPrepareGroupHeight()
	tablMmanagement.dismissGropHeight, _ = tablMmanagement.storage.TopDismissGroupHeight()
	go tablMmanagement.loop()
	return tablMmanagement
}

func (tm *DBMmanagement) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
	tm.fetchGenesisAndGuardianAccounts()
	go tm.fetchAccounts()
	go tm.fetchGroup()
	for {
		select {
		case <-check.C:
			go tm.fetchAccounts()
			go tm.fetchGroup()
		}
	}
}

func (tm *DBMmanagement) fetchAccounts() {
	// atomic operate
	if !atomic.CompareAndSwapInt32(&tm.isFetchingBlocks, 0, 1) {
		return
	}
	tm.excuteAccounts()
	atomic.CompareAndSwapInt32(&tm.isFetchingBlocks, 1, 0)

}

func (tm *DBMmanagement) fetchGenesisAndGuardianAccounts() {
	consensusImpl := mediator.ConsensusHelperImpl{}
	genesisInfo := consensusImpl.GenerateGenesisInfo()

	accounts := make([]string, 0)
	// genesis group members
	for _, member := range genesisInfo.Group.Members() {
		miner := common.ToAddrHex(member.ID())
		accounts = append(accounts, miner)
	}

	// guardian accounts
	for _, guardNode := range common.ExtractGuardNodes {
		accounts = append(accounts, guardNode.AddrPrefixString())
	}

	for _, miner := range accounts {
		targetAddrInfo := tm.storage.GetAccountById(miner)
		accounts := &models.AccountList{}
		// if the account doesn't exist
		if targetAddrInfo == nil || len(targetAddrInfo) < 1 {
			accounts.Address = miner
			accounts.Balance = tm.fetcher.Fetchbalance(miner)
			if !tm.storage.AddObjects(accounts) {
				return
			}
			UpdateAccountStake(accounts, 0, tm.storage)
		}
	}
}

func (tm *DBMmanagement) excuteAccountProposalAndVerifyCount() {
	for page := 1; page < 100; page++ {
		accounts := tm.storage.GetAccountByPage(uint64(page))
		for _, acc := range accounts {
			countVerify := tm.storage.GetProposalVerifyCount(uint64(types.MinerTypeVerify), acc.Address)
			countProposal := tm.storage.GetProposalVerifyCount(uint64(types.MinerTypeProposal), acc.Address)
			attrs := make(map[string]interface{})
			attrs["proposal_count"] = countProposal
			attrs["verify_count"] = countVerify
			account := &models.AccountList{}
			account.Address = acc.Address
			tm.storage.UpdateAccountByColumn(account, attrs)
		}
	}
}

func (tm *DBMmanagement) excuteAccounts() {

	topHeight := core.BlockChainImpl.Height()
	checkpoint := core.BlockChainImpl.LatestCheckPoint()
	if checkpoint.Height > 0 && tm.blockHeight > checkpoint.Height {
		return
	} else if checkpoint.Height == 0 && tm.blockHeight > topHeight-50 {
		return
	}
	browserlog.BrowserLog.Info("[DBMmanagement] excuteAccounts height:", tm.blockHeight, ",CheckPointHeight", checkpoint.Height, ",TopHeight", topHeight)
	//fmt.Println("[DBMmanagement]  fetchBlock height:", tm.blockHeight, "CheckPointHeight", topHeight)
	chain := core.BlockChainImpl
	block := chain.QueryBlockCeil(tm.blockHeight)

	if block != nil {
		if len(block.Transactions) > 0 {
			AddressCacheList := make(map[string]uint64)
			PoolList := make(map[string]uint64)
			stakelist := make(map[string]map[string]int64)
			set := &util.Set{}
			for _, tx := range block.Transactions {
				if tx.Type == types.TransactionTypeVoteMinerPool {
					if tx.Target != nil {
						if _, exists := PoolList[tx.Target.AddrPrefixString()]; exists {
							PoolList[tx.Target.AddrPrefixString()] += 1
						} else {
							PoolList[tx.Target.AddrPrefixString()] = 1
						}
						crontab.UpdatePoolStatus(tm.storage)
					}
				}

				if tx.Source != nil {
					//account list
					if _, exists := AddressCacheList[tx.Source.AddrPrefixString()]; exists {
						AddressCacheList[tx.Source.AddrPrefixString()] += 1
					} else {
						AddressCacheList[tx.Source.AddrPrefixString()] = 1
					}
					//if tx.Type == types.TransactionTypeStakeAdd || tx.Type == types.TransactionTypeStakeReduce{
					var target string
					if tx.Target != nil {
						target = tx.Target.AddrPrefixString()
						if _, exists := AddressCacheList[target]; exists {
							AddressCacheList[target] += 0
						} else {
							AddressCacheList[target] = 0
						}
					}

					if tx.Type == types.TransactionTypeContractCall {
						addressList := tm.storage.GetContractByHash(tx.GenHash().Hex())
						//wrapper := chain.GetTransactionPool().GetReceipt(tx.GenHash())
						//contract address
						if len(addressList) > 0 {
							for _, addr := range addressList {
								if _, exists := AddressCacheList[addr]; exists {
									AddressCacheList[addr] += 0
								} else {
									AddressCacheList[addr] = 0
								}
							}
							//go tm.ConsumeContract(contract, chain, tx.GenHash())
						}
					}
					//check update stake
					if checkStakeTransaction(tx.Type) {
						if tx.Target != nil {
							set.Add(tx.Target.AddrPrefixString())
						}
					}
					//stake list
					if tx.Type == types.TransactionTypeStakeAdd || tx.Type == types.TransactionTypeStakeReduce {
						if _, exists := stakelist[tx.Target.AddrPrefixString()][tx.Source.AddrPrefixString()]; exists {
							if tx.Type == types.TransactionTypeStakeAdd {
								stakelist[tx.Target.AddrPrefixString()][tx.Source.AddrPrefixString()] += tx.Value.Int64()
							}
							if tx.Type == types.TransactionTypeStakeReduce {
								stakelist[tx.Target.AddrPrefixString()][tx.Source.AddrPrefixString()] -= tx.Value.Int64()
							}
						} else {
							stakelist[tx.Target.AddrPrefixString()] = map[string]int64{}
							if tx.Type == types.TransactionTypeStakeAdd {
								stakelist[tx.Target.AddrPrefixString()][tx.Source.AddrPrefixString()] = tx.Value.Int64()
							}
							if tx.Type == types.TransactionTypeStakeReduce {
								stakelist[tx.Target.AddrPrefixString()][tx.Source.AddrPrefixString()] = -tx.Value.Int64()
							}
						}
					}

				}
			}
			//生成质押来源信息
			generateStakefromByTransaction(tm, stakelist)
			//begain
			for address, totalTx := range AddressCacheList {
				accounts := &models.AccountList{}
				targetAddrInfo := tm.storage.GetAccountById(address)
				//不存在账号
				if targetAddrInfo == nil || len(targetAddrInfo) < 1 {
					accounts.Address = address
					accounts.TotalTransaction = totalTx
					accounts.Balance = tm.fetcher.Fetchbalance(address)
					if !tm.storage.AddObjects(accounts) {
						return
					}
					//存在账号
				} else {
					accounts.Address = address
					//accounts.TotalTransaction = totalTx
					//accounts.ID = targetAddrInfo[0].ID
					//accounts.Balance = tm.fetchbalance(address)
					if !tm.storage.UpdateAccountbyAddress(accounts, map[string]interface{}{"total_transaction": gorm.Expr("total_transaction + ?", totalTx), "balance": tm.fetcher.Fetchbalance(address)}) {
						return
					}

				}
				//update stake
			}
			if set.M != nil {
				account := &models.AccountList{}
				for aa, _ := range set.M {
					account.Address = aa.(string)

					UpdateAccountStake(account, 0, tm.storage)
				}
			}
			for address, _ := range PoolList {
				accounts := &models.AccountList{}
				targetAddrInfo := tm.storage.GetAccountById(address)
				if targetAddrInfo == nil || len(targetAddrInfo) < 1 {
					accounts.Address = address
					accounts.ExtraData = tm.fetchTickets(address)
					if !tm.storage.AddObjects(accounts) {
						return
					}
				} else {
					accounts.Address = address
					accounts.ExtraData = tm.fetchTickets(address)

					if !tm.storage.UpdateObject(*accounts, address) {
						return
					}
				}
			}
		}
		//块高存储持久化
		sys := &models.Sys{
			Variable: mysql.Blocktopheight,
			SetBy:    "wujia",
			Value:    block.Header.Height,
		}
		tm.storage.AddBlockHeightSystemconfig(sys)
		tm.blockHeight = block.Header.Height + 1
		tm.excuteAccounts()
	}
}

func checkStakeTransaction(trtype int8) bool {
	if trtype == types.TransactionTypeStakeReduce || trtype == types.TransactionTypeStakeAdd ||
		trtype == types.TransactionTypeApplyGuardMiner || trtype == types.TransactionTypeVoteMinerPool ||
		trtype == types.TransactionTypeChangeFundGuardMode || trtype == types.TransactionTypeMinerAbort ||
		trtype == types.TransactionTypeStakeRefund {
		return true
	}
	return false
}

func (tm *DBMmanagement) fetchTickets(address string) string {
	voteLIst := make(map[string]interface{})
	db, err := core.BlockChainImpl.LatestAccountDB()
	if err != nil {
		return ""
	}
	voteCount := core.MinerManagerImpl.GetTickets(db, common.StringToAddress(address))
	voteLIst["vote"] = voteCount
	data := tm.storage.MapToJson(voteLIst)

	return data
}

func (tm *DBMmanagement) fetchGroup() {
	//fmt.Println("[DBMmanagement]  fetchGroup height:", tm.groupHeight)
	browserlog.BrowserLog.Info("[DBMmanagement] fetchGroup height:", tm.groupHeight)

	//读本地数据库表
	db := tm.storage.GetDB()
	if db == nil {
		fmt.Println("[DBMmanagement] storage.db == nil")
		return
	}

	//Prepare
	handelPrepareGroup(tm, tm.storage.GetDB())
	//Work
	handelWorkGroup(tm, tm.storage.GetDB())
	//Dissmiss
	handelDismissGroup(tm, tm.storage.GetDB())

}

func handelDismissGroup(tm *DBMmanagement, db *gorm.DB) {
	if tm.isFetchingDismissGroups {
		return
	}
	tm.isFetchingDismissGroups = true
	groups := make([]models.Group, 1)

	err := db.Where("dismiss_height <= ? AND height > ?", tm.blockHeight, tm.dismissGropHeight).Order("height").Find(&groups).Error
	if err != nil {
		fmt.Println("db err:", err)
		return
	}

	if groups == nil || len(groups) < 1 {
		tm.isFetchingDismissGroups = false
		return
	}
	//fmt.Println("[DBMmanagement]  fetchDismissGroup height:", groups[len(groups)-1].Height)
	//func() {
	if handelInGroup(tm, groups, dismissGroup) {
		if len(groups) > 0 {
			tm.dismissGropHeight = groups[len(groups)-1].Height
		} else {
			tm.dismissGropHeight = 0
		}

		//高度存储持久化
		hight, ifexist := tm.storage.TopDismissGroupHeight()
		AddGroupHeightSystemconfig(mysql.DismissGropHeight, tm, db, groups, hight, ifexist)
	} else {
		tm.isFetchingDismissGroups = false
		return
	}
	tm.isFetchingDismissGroups = false
	fmt.Println("《D--SUCCEED--》")
	//}()

}

func handelWorkGroup(tm *DBMmanagement, db *gorm.DB) {
	if tm.isFetchingWorkGroups {
		return
	}
	tm.isFetchingWorkGroups = true
	groups := make([]models.Group, 1)

	err := db.Where("work_height <= ? AND dismiss_height > ? AND height > ?", tm.blockHeight, tm.blockHeight, tm.groupHeight).Order("height").Find(&groups).Error
	if err != nil {
		fmt.Println("db err:", err)
		return
	}
	if groups == nil || len(groups) < 1 {
		tm.isFetchingWorkGroups = false
		return
	}
	//fmt.Println("[DBMmanagement]  fetchGroup height:", groups[len(groups)-1].Height)
	//func() {
	if handelInGroup(tm, groups, workGroup) {
		if len(groups) > 0 {
			tm.groupHeight = groups[len(groups)-1].Height
		} else {
			tm.groupHeight = 0
		}

		//高度存储持久化
		hight, ifexist := tm.storage.TopGroupHeight()
		AddGroupHeightSystemconfig(mysql.GroupTopHeight, tm, db, groups, hight, ifexist)
	} else {
		tm.isFetchingWorkGroups = false
		return
	}
	tm.isFetchingWorkGroups = false
	fmt.Println("《W--SUCCEED--》")
	//}()
}

func handelPrepareGroup(tm *DBMmanagement, db *gorm.DB) {
	if tm.isFetchingPrepareGroups {
		return
	}
	tm.isFetchingPrepareGroups = true
	groups := make([]models.Group, 1)

	err := db.Where("work_height >= ? AND height > ?", tm.blockHeight, tm.prepareGroupHeight).Order("height").Find(&groups).Error
	if err != nil {
		fmt.Println("db err:", err)
		return
	}

	if groups == nil || len(groups) < 1 {
		tm.isFetchingPrepareGroups = false
		return
	}
	//fmt.Println("[DBMmanagement]  fetchPrepareGroup height:", groups[len(groups)-1].Height)
	//func() {
	if handelInGroup(tm, groups, prepareGroup) {
		if len(groups) > 0 {
			tm.prepareGroupHeight = groups[len(groups)-1].Height
		} else {
			tm.prepareGroupHeight = 0
		}
		//高度存储持久化
		hight, ifexist := tm.storage.TopPrepareGroupHeight()
		AddGroupHeightSystemconfig(mysql.PrepareGroupTopHeight, tm, db, groups, hight, ifexist)
	} else {
		tm.isFetchingPrepareGroups = false
		return
	}
	tm.isFetchingPrepareGroups = false
	fmt.Println("《P--SUCCEED--》")
	//}()
}

func AddGroupHeightSystemconfig(groupstate string, tm *DBMmanagement, db *gorm.DB, groups []models.Group, hight uint64, ifexist bool) {
	sys := &models.Sys{
		Variable: groupstate,
		SetBy:    "wujia",
	}
	sysConfig := make([]models.Sys, 0, 0)
	db.Limit(1).Where("variable = ?", groupstate).Find(&sysConfig)

	//高度存储持久化
	if ifexist == false && groups[len(groups)-1].Height == 0 {
		db.Create(&sys)
	} else if ifexist == false && groups[len(groups)-1].Height != 0 {
		db.Create(&sys)
		db.Model(&sysConfig).Where("variable = ?", groupstate).UpdateColumn("value", groups[len(groups)-1].Height)
	} else {
		db.Model(&sysConfig[0]).Where("variable = ?", groupstate).UpdateColumn("value", groups[len(groups)-1].Height)
	}
}

func handelInGroup(tm *DBMmanagement, groups []models.Group, groupState int) bool {
	if len(groups) <= 0 {
		return false
	}

	for _, group := range groups {
		addresInfos := strings.Split(group.MembersStr, "\r\n")
		for _, addr := range addresInfos {
			if addr == "" {
				continue
			}

			switch groupState {
			case prepareGroup:
				tm.storage.GetDB().Table(mysql.ACCOUNTDBNAME).Where("address = ?", addr).Updates(map[string]interface{}{
					"prepare_group": gorm.Expr("prepare_group + ?", 1),
				})
			case workGroup:
				tm.storage.GetDB().Table(mysql.ACCOUNTDBNAME).Where("address = ?", addr).Updates(map[string]interface{}{
					"work_group":    gorm.Expr("work_group + ?", 1),
					"prepare_group": gorm.Expr("prepare_group - ?", 1),
				})
			case dismissGroup:
				tm.storage.GetDB().Table(mysql.ACCOUNTDBNAME).Where("address = ?", addr).Updates(map[string]interface{}{
					"dismiss_group": gorm.Expr("dismiss_group + ?", 1),
					"work_group":    gorm.Expr("work_group - ?", 1),
				})
			}
		}
	}
	return true
}

//genrate stake from by transaction
func generateStakefromByTransaction(tm *DBMmanagement, stakelist map[string]map[string]int64) {
	if stakelist == nil {
		return
	}
	poolstakefrom := make([]*models.PoolStake, 0, 0)
	for address, fromList := range stakelist {
		/*detail := tm.storage.GetAccountById(address)
		if detail != nil && len(detail) >0{
		}*/
		for from, stake := range fromList {
			poolstake := &models.PoolStake{
				Address: address,
				Stake:   stake,
				From:    from,
			}
			if from != "" {
				poolstakefrom = append(poolstakefrom, poolstake)
			}

		}
	}
	tm.storage.AddOrUpPoolStakeFrom(poolstakefrom)

}

func GetMinerInfo(addr string, height uint64) (map[string]*common2.MortGage, *common2.FronzenAndStakeFrom) {
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return nil, nil
	}

	morts := make(map[string]*common2.MortGage)
	address := common.StringToAddress(addr)
	var proposalInfo *types.Miner
	//if height == 0 {
	proposalInfo = core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeProposal)
	//} else {
	//	proposalInfo = core.MinerManagerImpl.GetMiner(address, types.MinerTypeProposal, height)

	//}
	details := core.MinerManagerImpl.GetStakeDetails(address, address)
	var selfStakecount, proposalfrozenStake, verifyfrozenStake uint64

	for _, detail := range details {
		if detail.MType == types.MinerTypeProposal && detail.Status == types.Staked {
			selfStakecount += detail.Value
		}
		if detail.Status == types.StakeFrozen && detail.MType == types.MinerTypeProposal {
			proposalfrozenStake += detail.Value
		}
		if detail.Status == types.StakeFrozen && detail.MType == types.MinerTypeVerify {
			verifyfrozenStake += detail.Value
		}
	}

	data := &common2.FronzenAndStakeFrom{
		ProposalFrozen: uint64(common.RA2TAS(proposalfrozenStake)),
		VerifyFrozen:   uint64(common.RA2TAS(verifyfrozenStake)),
	}

	var stakefrom = ""
	if proposalInfo != nil {
		mort := common2.NewMortGageFromMiner(proposalInfo)
		morts["proposal"] = mort
		//morts = append(morts, mort)
		//get stakeinfo by miners themselves

		fmt.Println("GetMinerInfo", proposalInfo.Stake, ",", selfStakecount, ",", address)
		other := &common2.MortGage{
			Stake:       mort.Stake - uint64(common.RA2TAS(selfStakecount)),
			ApplyHeight: 0,
			Type:        "proposal node",
			Status:      types.MinerStatusActive,
		}
		morts["other"] = other
		if selfStakecount > 0 {
			stakefrom = addr
		}
		// check if contain other stake ,
		//todo pool identify
		if selfStakecount < mort.Stake {
			stakefrom = stakefrom + "," + GetStakeFrom(address)
		}
	}
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeVerify)
	if verifierInfo != nil {
		morts["verify"] = common2.NewMortGageFromMiner(verifierInfo)
		if stakefrom == "" {
			stakefrom = addr
		}
	}
	data.StakeFrom = stakefrom
	return morts, data
}
func GetStakeFrom(address common.Address) string {
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

func UpdateAccountStake(account *models.AccountList, height uint64, storage *mysql.Storage) {
	if account == nil {
		return
	}
	minerinfo, frozensAndtakefrom := GetMinerInfo(account.Address, height)
	if len(minerinfo) > 0 {
		var verifystake uint64
		mapcolumn := make(map[string]interface{})
		if minerinfo["verify"] != nil {
			verifystake = minerinfo["verify"].Stake
			mapcolumn["verify_stake"] = verifystake
			mapcolumn["verify_status"] = minerinfo["verify"].Status
			mapcolumn["role_type"] = minerinfo["verify"].Identity
		}
		var prostake uint64
		if minerinfo["proposal"] != nil {
			prostake = minerinfo["proposal"].Stake
			mapcolumn["proposal_stake"] = prostake
			mapcolumn["other_stake"] = minerinfo["other"].Stake
			mapcolumn["status"] = minerinfo["proposal"].Status
			mapcolumn["role_type"] = minerinfo["proposal"].Identity
		}
		mapcolumn["total_stake"] = verifystake + prostake + frozensAndtakefrom.ProposalFrozen + frozensAndtakefrom.VerifyFrozen
		mapcolumn["stake_from"] = frozensAndtakefrom.StakeFrom
		mapcolumn["proposal_frozen_stake"] = frozensAndtakefrom.ProposalFrozen
		mapcolumn["verify_frozen_stake"] = frozensAndtakefrom.VerifyFrozen

		storage.UpdateAccountByColumn(account, mapcolumn)
	}
}

func (tm *DBMmanagement) GetGroups() bool {
	rpcAddr := "0.0.0.0"
	rpcPort := 8101
	client, err := rpc.Dial(fmt.Sprintf("http://%s:%d", rpcAddr, rpcPort))
	if err != nil {
		fmt.Println("[fetcher] Error in dialing. err:", err)
		return false
	}
	defer client.Close()

	//call remote procedure with args

	groups := make([]*models.Group, 0)
	for i := uint64(0); i < 2; i++ {
		fmt.Println("=======================GROUP HIGH", i, tm.groupHeight)
		var result []map[string]interface{}
		//var result interface{}

		err = client.Call(&result, "Explorer_explorerGroupsAfter", i)
		if err != nil {
			fmt.Println("[fetcher] GetGroups  client.Call error :", err)
			return false
		}
		fmt.Println("[fetcher] GetGroups  result :", len(result), result)
		if result == nil {
			return false
		}
		//groupsData := result
		for _, g := range result {
			group := dataToGroup(g)
			if group != nil {
				groups = append(groups, group)
			}
		}
		if groups != nil {
			for i := 0; i < len(groups); i++ {
				groupsInfo := tm.storage.GetGroupByHigh(groups[i].Height)
				fmt.Println("++++++++>>>>>", groupsInfo, len(groupsInfo))
				if len(groupsInfo) != 0 {
					continue
				}
				fmt.Println("数据库写入GROUP", groups[i].Height, groups[i].Id)
				tm.storage.AddGroup(groups[i])
				if groups[i].Height >= tm.groupHeight {
					tm.groupHeight = groups[i].Height + 1
				}
			}
		} else {
			return false
		}
	}
	return true
}

func dataToGroup(data map[string]interface{}) *models.Group {

	group := &models.Group{}
	group.Id = util.DataToString(data["id"])
	group.WorkHeight = uint64(data["begin_height"].(float64))
	group.DismissHeight = uint64(data["dismiss_height"].(float64))
	group.Threshold = uint64(data["threshold"].(float64))
	group.Height = uint64(data["group_height"].(float64))

	members := data["members"].([]interface{})
	group.Members = make([]string, 0)
	group.MemberCount = uint64(len(members))
	for _, memberId := range members {
		midStr := memberId.(string)
		if len(midStr) > 0 {
			group.MembersStr = group.MembersStr + midStr + "\r\n"
		}
	}
	return group
}

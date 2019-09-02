package browser

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/crontab"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/cmd/gzv/rpc"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"strings"
	"sync"
	"time"
)

const checkInterval = time.Second * 5

const (
	dismissGroup = iota
	workGroup
	prepareGroup
)

//var AddressCacheList map[string]uint64

type MortGage struct {
	Stake                uint64             `json:"stake"`
	ApplyHeight          uint64             `json:"apply_height"`
	Type                 string             `json:"type"`
	Status               types.MinerStatus  `json:"miner_status"`
	StatusUpdateHeight   uint64             `json:"status_update_height"`
	Identity             types.NodeIdentity `json:"identity"`
	IdentityUpdateHeight uint64             `json:"identity_update_height"`
}
type DBMmanagement struct {
	sync.Mutex
	blockHeight        uint64
	prepareGroupHeight uint64
	groupHeight        uint64
	dismissGropHeight  uint64
	storage            *mysql.Storage //待迁移
	crontab            *crontab.Crontab

	isFetchingBlocks        bool
	isFetchingWorkGroups    bool
	isFetchingPrepareGroups bool
	isFetchingDismissGroups bool
}

func NewDBMmanagement(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *DBMmanagement {
	tablMmanagement := &DBMmanagement{}
	tablMmanagement.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset)

	tablMmanagement.blockHeight, _ = tablMmanagement.storage.TopBlockHeight()
	tablMmanagement.groupHeight, _ = tablMmanagement.storage.TopGroupHeight()
	tablMmanagement.prepareGroupHeight, _ = tablMmanagement.storage.TopPrepareGroupHeight()
	tablMmanagement.dismissGropHeight, _ = tablMmanagement.storage.TopDismissGroupHeight()
	tablMmanagement.blockHeight = 0
	go tablMmanagement.loop()
	return tablMmanagement
}

func (tm *DBMmanagement) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
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
	if tm.isFetchingBlocks {
		return
	}

	blockheader := core.BlockChainImpl.CheckPointAt(mysql.CheckpointMaxHeight)
	if tm.blockHeight > blockheader.Height {
		return
	}
	tm.isFetchingBlocks = true
	fmt.Println("[DBMmanagement]  fetchBlock height:", tm.blockHeight)

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
					}
				}
				if tx.Source != nil && tx.Target != nil {
					//account list
					if _, exists := AddressCacheList[tx.Source.AddrPrefixString()]; exists {
						AddressCacheList[tx.Source.AddrPrefixString()] += 1
					} else {
						AddressCacheList[tx.Source.AddrPrefixString()] = 1
					}
					//check update stake
					if checkStakeTransaction(tx.Type) {
						set.Add(tx.Source.AddrPrefixString())
					}
					//stake list
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
			//生成质押来源信息
			generateStakefromByTransaction(tm, stakelist)
			//begain
			accounts := &models.Account{}
			for address, totalTx := range AddressCacheList {
				targetAddrInfo := tm.storage.GetAccountById(address)
				//不存在账号
				if targetAddrInfo == nil || len(targetAddrInfo) < 1 {
					accounts.Address = address
					accounts.TotalTransaction = totalTx
					accounts.Balance = tm.fetchbalance(address)
					if !tm.storage.AddObjects(accounts) {
						return
					}
					//存在账号
				} else {
					accounts.Address = address
					//accounts.TotalTransaction = totalTx
					//accounts.ID = targetAddrInfo[0].ID
					//accounts.Balance = tm.fetchbalance(address)
					if !tm.storage.UpdateAccountbyAddress(accounts, map[string]interface{}{"total_transaction": gorm.Expr("total_transaction + ?", totalTx), "balance": tm.fetchbalance(address)}) {
						return
					}

				}
				//update stake

			}
			if set.M != nil {
				account := &models.Account{}
				for aa, _ := range set.M {
					account.Address = aa.(string)
					tm.UpdateAccountStake(account, blockheader.Height)
				}
			}
			for address, _ := range PoolList {
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
					if !tm.storage.UpdateObject(accounts) {
						return
					}
				}
			}
		}

		//块高存储持久化
		sys := &models.Sys{
			Variable: mysql.Blocktophight,
			SetBy:    "wujia",
		}
		tm.storage.AddBlockHeightSystemconfig(sys)
		tm.blockHeight = block.Header.Height + 1

		tm.isFetchingBlocks = false
		go tm.fetchAccounts()
	}
	tm.isFetchingBlocks = false
}

func checkStakeTransaction(trtype int8) bool {
	if trtype == types.TransactionTypeStakeReduce || trtype == types.TransactionTypeStakeAdd {
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

func (tm *DBMmanagement) fetchbalance(addr string) float64 {
	b := core.BlockChainImpl.GetBalance(common.StringToAddress(addr))
	balance := common.RA2TAS(b.Uint64())

	return balance
}

func (tm *DBMmanagement) fetchGroup() {
	fmt.Println("[DBMmanagement]  fetchGroup height:", tm.groupHeight)

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

	tm.blockHeight += 4000
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
				tm.storage.GetDB().Table("accounts").Where("address = ?", addr).Updates(map[string]interface{}{
					"prepare_group": gorm.Expr("prepare_group + ?", 1),
				})
			case workGroup:
				tm.storage.GetDB().Table("accounts").Where("address = ?", addr).Updates(map[string]interface{}{
					"work_group":    gorm.Expr("work_group + ?", 1),
					"prepare_group": gorm.Expr("prepare_group - ?", 1),
				})
			case dismissGroup:
				tm.storage.GetDB().Table("accounts").Where("address = ?", addr).Updates(map[string]interface{}{
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

func GetMinerInfo(addr string, height uint64) ([]*MortGage, string) {
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return nil, ""
	}

	morts := make([]*MortGage, 0, 0)
	address := common.StringToAddress(addr)
	var proposalInfo *types.Miner
	//if height == 0 {
	proposalInfo = core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeProposal)
	//} else {
	//	proposalInfo = core.MinerManagerImpl.GetMiner(address, types.MinerTypeProposal, height)

	//}
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
			Stake:       mort.Stake - uint64(common.RA2TAS(selfStakecount)),
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
			stakefrom = stakefrom + "," + GetStakeFrom(address)
		}
	}
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeVerify)
	if verifierInfo != nil {
		morts = append(morts, NewMortGageFromMiner(verifierInfo))
	}
	return morts, stakefrom
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

func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "proposal node"
	if miner.IsVerifyRole() {
		t = "verify node"
	}
	status := types.MinerStatusPrepare
	if miner.IsActive() {
		status = types.MinerStatusActive
	} else if miner.IsFrozen() {
		status = types.MinerStatusFrozen
	}

	i := types.MinerNormal
	if miner.IsMinerPool() {
		i = types.MinerPool
	} else if miner.IsInvalidMinerPool() {
		i = types.InValidMinerPool
	} else if miner.IsGuard() {
		i = types.MinerGuard
	}
	mg := &MortGage{
		Stake:                uint64(common.RA2TAS(miner.Stake)),
		ApplyHeight:          miner.ApplyHeight,
		Type:                 t,
		Status:               status,
		StatusUpdateHeight:   miner.StatusUpdateHeight,
		Identity:             i,
		IdentityUpdateHeight: miner.IdentityUpdateHeight,
	}
	return mg
}
func (tm *DBMmanagement) UpdateAccountStake(account *models.Account, height uint64) {
	if account == nil {
		return

	}
	minerinfo, stakefrom := GetMinerInfo(account.Address, height)
	if len(minerinfo) > 0 {
		tm.storage.UpdateAccountByColumn(account, map[string]interface{}{
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

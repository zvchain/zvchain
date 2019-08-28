package browser

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
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

type DBMmanagement struct {
	sync.Mutex
	blockHeight        uint64
	prepareGroupHeight uint64
	groupHeight        uint64
	dismissGropHeight  uint64
	storage            *mysql.Storage //待迁移

	isFetchingBlocks bool
}

func NewDBMmanagement(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *DBMmanagement {
	tablMmanagement := &DBMmanagement{}
	tablMmanagement.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset)

	tablMmanagement.blockHeight, _ = tablMmanagement.storage.TopBlockHeight()
	tablMmanagement.groupHeight = tablMmanagement.storage.TopGroupHeight()
	tablMmanagement.prepareGroupHeight = tablMmanagement.storage.TopPrepareGroupHeight()
	tablMmanagement.dismissGropHeight = tablMmanagement.storage.TopDismissGroupHeight()

	go tablMmanagement.loop()
	return tablMmanagement
}

func (tm *DBMmanagement) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
	go tm.fetchAccounts()
	for {
		select {
		case <-check.C:
			go tm.fetchAccounts()
			//go tm.fetchGroup()
		}
	}
}

func (tm *DBMmanagement) fetchAccounts() {
	if tm.isFetchingBlocks {
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
	groups := make([]models.Group, 1, 1)

	//Dissmiss
	handelDismissGroup(tm, db, groups)

	//Work
	handelWorkGroup(tm, db, groups)

	//Prepare
	handelPrepareGroup(tm, db, groups)
}

func handelDismissGroup(tm *DBMmanagement, db *gorm.DB, groups []models.Group) {
	fmt.Println("[DBMmanagement]  fetchDismissGroup height:", tm.dismissGropHeight)
	db.Where("dismiss_height <= ? AND height > ?", tm.blockHeight, tm.dismissGropHeight).Order("height").Find(&groups)
	if groups == nil || len(groups) < 1 {
		return
	}
	//fmt.Println("[DBMmanagement]  fetchDismissGroup height:", groups[len(groups)-1].Height)
	go func() {
		if handelInGroup(tm, groups, dismissGroup) {
			if len(groups) > 0 {
				tm.dismissGropHeight = groups[len(groups)-1].Height
			} else {
				tm.dismissGropHeight = 0
			}

			//高度存储持久化
			hight := tm.storage.TopDismissGroupHeight()
			AddGroupHeightSystemconfig(mysql.DismissGropHeight, tm, db, groups, hight)
		} else {
			return
		}
	}()

}

func handelWorkGroup(tm *DBMmanagement, db *gorm.DB, groups []models.Group) {
	fmt.Println("[DBMmanagement]  fetchGroup height:", tm.groupHeight)
	db.Where("work_height <= ? AND dismiss_height > ? AND height > ?", tm.blockHeight, tm.blockHeight, tm.groupHeight).Order("height").Find(&groups)
	if groups == nil || len(groups) < 1 {
		return
	}
	//fmt.Println("[DBMmanagement]  fetchGroup height:", groups[len(groups)-1].Height)
	if groups == nil {
		return
	}
	go func() {
		if handelInGroup(tm, groups, workGroup) {
			if len(groups) > 0 {
				tm.groupHeight = groups[len(groups)-1].Height
			} else {
				tm.groupHeight = 0
			}

			//高度存储持久化
			hight := tm.storage.TopGroupHeight()
			AddGroupHeightSystemconfig(mysql.GroupTopHeight, tm, db, groups, hight)
		} else {
			return
		}
	}()
}

func handelPrepareGroup(tm *DBMmanagement, db *gorm.DB, groups []models.Group) {
	fmt.Println("[DBMmanagement]  fetchPrepareGroup height:", tm.prepareGroupHeight)
	db.Where("work_height > ? AND height > ?", tm.blockHeight, tm.prepareGroupHeight).Order("height").Find(&groups)
	if groups == nil || len(groups) < 1 {
		return
	}
	//fmt.Println("[DBMmanagement]  fetchPrepareGroup height:", groups[len(groups)-1].Height)
	go func() {
		if handelInGroup(tm, groups, prepareGroup) {
			if len(groups) > 0 {
				tm.prepareGroupHeight = groups[len(groups)-1].Height
			} else {
				tm.prepareGroupHeight = 0
			}
			//高度存储持久化
			hight := tm.storage.TopPrepareGroupHeight()
			AddGroupHeightSystemconfig(mysql.PrepareGroupTopHeight, tm, db, groups, hight)
		} else {
			return
		}
	}()
}

func AddGroupHeightSystemconfig(groupstate string, tm *DBMmanagement, db *gorm.DB, groups []models.Group, hight uint64) {
	sys := &models.Sys{
		Variable: groupstate,
		SetBy:    "wujia",
	}
	//高度存储持久化
	if hight > 0 {
		db.Model(&sys).UpdateColumn("value", gorm.Expr("value + ?", groups[len(groups)-1].Height-hight))
	} else {
		tm.storage.AddObjects(&sys)
	}
}

func handelInGroup(tm *DBMmanagement, groups []models.Group, groupState int) bool {
	tm.Lock()
	defer tm.Unlock()
	var account models.Account
	for _, grop := range groups {
		addresInfos := strings.Split(grop.MembersStr, "\r\n")
		for _, addr := range addresInfos {
			account.Address = addr
			tm.storage.GetDB().Where("address = ? ", addr).Find(&account)

			switch groupState {
			case prepareGroup:
				account.PrepareGroup += 1
			case workGroup:
				account.WorkGroup += 1
				account.PrepareGroup -= 1
			case dismissGroup:
				account.WorkGroup -= 1
				account.DismissGroup += 1
			}
			if !tm.storage.UpdateObject(&account) {
				return false
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
			poolstakefrom = append(poolstakefrom, poolstake)
		}
	}
	tm.storage.AddOrUpPoolStakeFrom(poolstakefrom)

}

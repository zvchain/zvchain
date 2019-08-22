package browser

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/core"
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

var AddressCacheList map[string]uint64

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

	tablMmanagement.blockHeight = tablMmanagement.storage.TopBlockHeight()
	tablMmanagement.groupHeight = tablMmanagement.storage.TopGroupHeight()
	tablMmanagement.prepareGroupHeight = tablMmanagement.storage.TopPrepareGroupHeight()
	tablMmanagement.dismissGropHeight = tablMmanagement.storage.TopDismissGroupHeight()

	return tablMmanagement
}

func (tm *DBMmanagement) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
	//go tm.fetchAccounts()
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
	tm.isFetchingBlocks = true
	fmt.Println("[server]  fetchBlock height:", tm.blockHeight)

	chain := core.BlockChainImpl
	block := chain.QueryBlockCeil(tm.blockHeight)

	if block != nil {
		AddressCacheList = make(map[string]uint64)
		for _, tx := range block.Transactions {
			if _, exists := AddressCacheList[tx.Source.AddrPrefixString()]; exists {
				AddressCacheList[tx.Source.AddrPrefixString()] += 1
			} else {
				AddressCacheList[tx.Source.AddrPrefixString()] = 1
			}
		}
		//块高存储持久化
		sys := &models.Sys{
			Variable: mysql.Blockrewardtophight,
			SetBy:    "wujia",
		}
		tm.storage.AddBlockHeightSystemconfig(sys)
		tm.blockHeight = block.Header.Height + 1

		//begain
		accounts := &models.Account{}
		for address, totalTx := range AddressCacheList {

			targetAddrInfo := tm.storage.GetAccountById(address)
			//不存在账号
			if targetAddrInfo == nil {
				accounts.Address = address
				//accounts.TotalTransaction = totalTx
				//高度存储持久化
				tm.storage.UpdateAccountByColumn(accounts, map[string]interface{}{"total_transaction": gorm.Expr("total_transaction = ?", totalTx)})
				//tm.storage.UpdateObject(accounts)

				//存在账号
			} else {
				accounts.Address = address
				//accounts.TotalTransaction = totalTx + targetAddrInfo[0].TotalTransaction
				//高度存储持久化
				tm.storage.UpdateAccountByColumn(accounts, map[string]interface{}{"total_transaction": gorm.Expr("total_transaction + ?", totalTx)})
				//tm.storage.UpdateObject(accounts)
			}
		}
	}
	go tm.fetchAccounts()
	tm.isFetchingBlocks = false

}

func (tm *DBMmanagement) fetchGroup() {

	fmt.Println("[server]  fetchGroup height:", tm.groupHeight)

	//读本地数据库表
	db := tm.storage.GetDB()
	if db == nil {
		fmt.Println("[Storage] storage.db == nil")
	}
	sys := make([]models.Sys, 0, 1)
	groups := make([]models.Group, 1, 1)
	//marke:=tm.dismissGropHeight

	//解散组
	db.Where("dismiss_height <= ? AND id > ?", tm.blockHeight, tm.dismissGropHeight).Find(&groups)
	fmt.Println("[server]  fetchDismissGroup height:", tm.dismissGropHeight)
	tm.storage.UpdateObject(sys)
	go func() {
		if handelInGroup(tm, groups, dismissGroup) {
			tm.dismissGropHeight = groups[len(groups)-1].Height
			sys := &models.Sys{
				Variable: mysql.DismissGropHeight,
				SetBy:    "wujia",
			}
			//tm.storage.AddBlockHeightSystemconfig(sys)
			//高度存储持久化
			hight := tm.storage.TopDismissGroupHeight()
			if hight > 0 {
				db.Model(&sys).UpdateColumn("value", gorm.Expr("value = ?", groups[len(groups)-1].Height))
				//db.Model(&sys).Update(Addr{RoleType:1})
			} else {
				sys.Value = 1
				tm.storage.AddObjects(&sys)
			}
		}
	}()

	//查找工作组
	db.Where("work_height <= ? AND dismiss_height > ? ? AND id > ?", tm.blockHeight, tm.blockHeight, tm.groupHeight).Find(&groups)
	fmt.Println("[server]  fetchGroup height:", tm.groupHeight)
	go func() {
		if handelInGroup(tm, groups, workGroup) {
			tm.groupHeight = groups[len(groups)-1].Height
			sys := &models.Sys{
				Variable: mysql.GroupTopHeight,
				SetBy:    "wujia",
			}
			//tm.storage.AddGroupHeightSystemconfig(sys)
			//高度存储持久化
			hight := tm.storage.TopGroupHeight()
			if hight > 0 {
				db.Model(&sys).UpdateColumn("value", gorm.Expr("value = ?", groups[len(groups)-1].Height))
			} else {
				sys.Value = 1
				tm.storage.AddObjects(&sys)
			}
		}
	}()

	//准备组
	db.Where("work_height > ? AND id > ?", tm.blockHeight, tm.prepareGroupHeight).Find(&groups)
	fmt.Println("[server]  fetchPrepareGroup height:", tm.prepareGroupHeight)
	go func() {
		if handelInGroup(tm, groups, prepareGroup) {
			tm.prepareGroupHeight = groups[len(groups)-1].Height
			sys := &models.Sys{
				Variable: mysql.PrepareGroupTopHeight,
				SetBy:    "wujia",
			}
			//tm.storage.AddBlockHeightSystemconfig(sys)
			//高度存储持久化
			hight := tm.storage.TopPrepareGroupHeight()
			if hight > 0 {
				db.Model(&sys).UpdateColumn("value", gorm.Expr("value = ?", groups[len(groups)-1].Height))
			} else {
				sys.Value = 1
				tm.storage.AddObjects(&sys)
			}
		}
	}()

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
			if !tm.storage.UpdateObject(account) {
				return false
			}
		}
	}
	return true
}

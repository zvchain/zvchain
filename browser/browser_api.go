package browser

import (
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/core"
	"strings"
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
	blockHeight       uint64
	gropHeight        uint64
	workGropHeight    uint64
	dismissGropHeight uint64
	storage           *mysql.Storage //待迁移

	isFetchingBlocks bool
}

func NewDBMmanagement(dbAddr string, dbPort int, dbUser string, dbPassword string, rpcAddr string, rpcPort int, reset bool) *DBMmanagement {
	tablMmanagement := &DBMmanagement{}
	tablMmanagement.blockHeight = uint64(0)

	tablMmanagement.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, rpcAddr, rpcPort, reset)

	return tablMmanagement
}

func (tm *DBMmanagement) loop() {
	var (
		check = time.NewTicker(checkInterval)
	)
	defer check.Stop()
	go tm.fetchBlocks()
	for {
		select {
		case <-check.C:
			go tm.fetchBlocks()
		}
	}
}

func (tm *DBMmanagement) fetchBlocks() {
	if tm.isFetchingBlocks {
		return
	}
	tm.isFetchingBlocks = true

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
		tm.blockHeight = block.Header.Height + 1

		//begain
		accounts := &models.Account{}
		for address, totalTx := range AddressCacheList {

			targetAddrInfo := tm.storage.GetAccountById(address)
			//不存在账号
			if targetAddrInfo == nil {
				accounts.Address = address
				accounts.TotalTransaction = totalTx
				tm.storage.UpdateObject(accounts)
				//存在账号
			} else {
				accounts.Address = address
				accounts.TotalTransaction = totalTx + targetAddrInfo[0].TotalTransaction
				tm.storage.UpdateObject(accounts)
			}

		}
	}
	go tm.fetchBlocks()

	tm.isFetchingBlocks = false

}

//storage.db.Where("id > ? ", maxid).Limit(LIMIT).Find(&accounts)

//func (tm *DBMmanagement)fetchMinerInfo()  {
//	//MSQ去数据
//	db:=tm.storage.GetDB()
//	var addresses []string
//	//data:=db.Find(&models.Account{}).Pluck("id",&addresses)
//	data:=tm.storage.GetDataByColumn(models.Account{},"address",addresses).([]string)
//
//	convertDetails := func(dts []*types.StakeDetail) []*StakeDetail {
//		details := make([]*StakeDetail, 0)
//		for _, d := range dts {
//			dt := &StakeDetail{
//				Value:        uint64(common.RA2TAS(d.Value)),
//				UpdateHeight: d.UpdateHeight,
//				MType:        mTypeString(d.MType),
//				Status:       statusString(d.Status),
//			}
//			details = append(details, dt)
//		}
//		return details
//	}
//
//
//	//查stak
//	for i,addr:=range data{
//		//根据地址拿数据
//		stakeinfos:=core.MinerManagerImpl.GetAllStakeDetails(common.HexToAddress(addr))
//		stakeinfo:=stakeinfos[addr]
//
//		//拿proposalStake
//
//		//verifyStake
//
//		//otherstake

func (tm *DBMmanagement) fetchGroup() {
	//读本地数据库表
	db := tm.storage.GetDB()
	if db == nil {
		fmt.Println("[Storage] storage.db == nil")
	}
	//sys:= make([]models.Sys, 0, 1)
	groups := make([]models.Group, 1, 1)
	//marke:=tm.dismissGropHeight

	//解散组
	db.Where("dismiss_height <= ? AND id > ?", tm.blockHeight, tm.dismissGropHeight).Find(&groups)
	tm.dismissGropHeight = groups[len(groups)-1].Height
	//sys[0] todo
	//tm.storage.UpdateObject(sys)
	go handelInGroup(tm, groups, dismissGroup)

	//marke=tm.dismissGropHeight-marke
	//查找工作组
	db.Where("work_height <= ? AND dismiss_height > ? ? AND id > ?", tm.blockHeight, tm.blockHeight, tm.workGropHeight).Find(&groups)
	tm.workGropHeight = groups[len(groups)-1].Height
	go handelInGroup(tm, groups, workGroup)

	//准备组
	db.Where("work_height > ? AND id > ?", tm.blockHeight, tm.gropHeight).Find(&groups)
	tm.gropHeight = groups[len(groups)-1].Height
	go handelInGroup(tm, groups, prepareGroup)

}

func handelInGroup(tm *DBMmanagement, groups []models.Group, groupState int) bool {
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

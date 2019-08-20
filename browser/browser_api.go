package browser

import (
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/core"
	"time"
)

const checkInterval = time.Second * 5

var AddressCacheList map[string]uint64

type DBMmanagement struct {
	blockHeight uint64
	storage     *mysql.Storage //待迁移

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
//
//		//
//
//		//
//
//		//for   {
//		//
//		//}
//
//		//再次插入MSQ
//	}
//}

func (tm *DBMmanagement) fetchGroup() {

}

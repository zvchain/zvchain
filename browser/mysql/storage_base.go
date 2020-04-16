//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package mysql

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"time"
)

const (
	REWARDStable = "rewards"
)
const PageSize uint64 = 20

var DBStorage *Storage
var SLAVEDBStorage *SlaveStorage

type Storage struct {
	db                        *gorm.DB
	dbAddr                    string
	dbPort                    int
	dbUser                    string
	dbPassword                string
	rpcAddrStr                string
	topBlockHigh              uint64
	topGroupHigh              uint64
	accounts                  []*models.AccountList
	topbrowserBlockHeight     uint64
	statisticsblockLastUpdate string
	statisticstranLastUpdate  string
}

type SlaveStorage struct {
	db           *gorm.DB
	dbAddr       string
	dbPort       int
	dbUser       string
	dbPassword   string
	rpcAddrStr   string
	topBlockHigh uint64
	topGroupHigh uint64
}

func NewStorage(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool, resetcrontab bool) *Storage {
	if DBStorage != nil {
		return DBStorage
	}
	DBStorage = &Storage{
		dbAddr:     dbAddr,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPassword: dbPassword,
	}
	DBStorage.Init(reset, resetcrontab)
	return DBStorage
}
func NewSlaveStorage(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool, resetcrontab bool) *SlaveStorage {
	if SLAVEDBStorage != nil {
		return SLAVEDBStorage
	}
	SLAVEDBStorage = &SlaveStorage{
		dbAddr:     dbAddr,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPassword: dbPassword,
	}
	SLAVEDBStorage.SLAVEInit(reset, resetcrontab)
	return SLAVEDBStorage
}

func (storage *SlaveStorage) SLAVEInit(reset bool, resetcrontab bool) {
	if storage.db != nil {
		return
	}
	//args := fmt.Sprintf("root:Jobs1955!@tcp(119.23.205.254:3306)/tas?charset=utf8&parseTime=True&loc=Local")
	args := fmt.Sprintf("%s:%s@tcp(%s:%d)/test?charset=utf8&parseTime=True&loc=Local",
		storage.dbUser,
		storage.dbPassword,
		storage.dbAddr,
		storage.dbPort)
	fmt.Println("[SlaveStorage] db args:", args)
	db, err := gorm.Open("mysql", args)
	if err != nil {
		fmt.Println("[SlaveStorage] gorm.Open err:", err)
		return
	}
	storage.db = db
}
func (storage *Storage) Init(reset bool, resetcrontab bool) {
	if storage.db != nil {
		return
	}
	//args := fmt.Sprintf("root:Jobs1955!@tcp(119.23.205.254:3306)/tas?charset=utf8&parseTime=True&loc=Local")
	args := fmt.Sprintf("%s:%s@tcp(%s:%d)/test?charset=utf8&parseTime=True&loc=Local",
		storage.dbUser,
		storage.dbPassword,
		storage.dbAddr,
		storage.dbPort)
	fmt.Println("[Storage] db args:", args)
	db, err := gorm.Open("mysql", args)
	if err != nil {
		fmt.Println("[Storage] gorm.Open err:", err)
		return
	}
	// orm logger
	db.LogMode(true)
	db.SetLogger(browserlog.OrmLog)

	storage.db = db
	if reset {
		db.DropTable(&models.Block{})
		db.DropTable(&models.Transaction{})
		db.DropTable(&models.Receipt{})
		db.DropTable(&models.Reward{})
	}

	if resetcrontab {
		db.DropTable(&models.AccountList{})
		db.DropTable(&models.Sys{})
		db.DropTable(&models.PoolStake{})
		db.DropTable(&models.Group{})
		db.DropTable(&models.ContractTransaction{})

	}
	if !db.HasTable(&models.AccountList{}) {
		db.CreateTable(&models.AccountList{})
	}
	if !db.HasTable(&models.ContractTransaction{}) {
		db.CreateTable(&models.ContractTransaction{})
	}
	if !db.HasTable(&models.ContractCallTransaction{}) {
		db.CreateTable(&models.ContractCallTransaction{})
	}
	if !db.HasTable(&models.Sys{}) {
		db.CreateTable(&models.Sys{})
	}
	if !db.HasTable(&models.PoolStake{}) {
		db.CreateTable(&models.PoolStake{})
	}
	if !db.HasTable(&models.Group{}) {
		db.CreateTable(&models.Group{})
	}
	if !db.HasTable(&models.Block{}) {
		db.CreateTable(&models.Block{})
	}
	if !db.HasTable(&models.Transaction{}) {
		db.CreateTable(&models.Transaction{})
	}
	if !db.HasTable(&models.Receipt{}) {
		db.CreateTable(&models.Receipt{})
	}
	if !db.HasTable(&models.Reward{}) {
		db.CreateTable(&models.Reward{})
	}
	if !db.HasTable(&models.RecentMineBlock{}) {
		db.CreateTable(&models.RecentMineBlock{})
	}
	if !db.HasTable(&models.Log{}) {
		db.CreateTable(&models.Log{})
	}
	if !db.HasTable(&models.StakeMapping{}) {
		db.CreateTable(&models.StakeMapping{})
	}
	if !db.HasTable(&models.Config{}) {
		db.CreateTable(&models.Config{})
	}
	if !db.HasTable(&models.TokenContract{}) {
		db.CreateTable(&models.TokenContract{})
	}
	if !db.HasTable(&models.TokenContractUser{}) {
		db.CreateTable(&models.TokenContractUser{})
	}
	if !db.HasTable(&models.TokenContractTransaction{}) {
		db.CreateTable(&models.TokenContractTransaction{})
	}
	if !db.HasTable(&models.TempDeployToken{}) {
		db.CreateTable(&models.TempDeployToken{})
	}
	if !db.HasTable(&models.BlockToMiner{}) {
		db.CreateTable(&models.BlockToMiner{})
	}
	if !db.HasTable(&models.MinerToBlock{}) {
		db.CreateTable(&models.MinerToBlock{})
	}
	if !db.HasTable(&models.MinerList{}) {
		db.CreateTable(&models.MinerList{})
	}
	if !db.HasTable(&models.Vote{}) {
		db.CreateTable(&models.Vote{})
	}
}

func (storage *Storage) GetDB() *gorm.DB {
	return storage.db
}

/**
 * common method
 * updateobject into mysqldb
 */
func (storage *Storage) UpdateObject(object interface{}, addr string) bool {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}

	storage.db.Model(&models.AccountList{}).Where("address = ?", addr).Updates(object)

	return true
}

/**
 * common method
 * addobject into mysqldb
 */
func (storage *Storage) AddObjects(object interface{}) bool {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	timeBegin := time.Now()
	tx := storage.db.Begin()
	tx.Create(object)
	tx.Commit()
	fmt.Println("[Storage]  objects cost: ", time.Since(timeBegin), "，len :")
	return true
}

func (storage *Storage) AddLoadVerifiedCount(block *models.Block) bool {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	block.LoadVerifyCount += 1
	storage.db.Model(&block).Updates(block)
	return true
}
func (storage *Storage) AddVerifications(verifications []*models.Verification) bool {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	timeBegin := time.Now()
	tx := storage.db.Begin()
	for i := 0; i < len(verifications); i++ {
		//fmt.Println("[Storage] add verification:",verifications[i])
		if verifications[i] != nil {
			tx.Create(&verifications[i])
		}
	}
	tx.Commit()
	fmt.Println("[Storage]  AddRewards cost: ", time.Since(timeBegin), "，len :", len(verifications))
	return true
}

func (storage *Storage) AddRewards(rewards []*models.Reward) bool {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	if len(rewards) < 1 {
		return false
	}
	timeBegin := time.Now()
	for i := 0; i < len(rewards); i++ {
		//fmt.Println("[Storage] add verification:",verifications[i])
		if rewards[i] != nil {
			if !errors(storage.db.Create(&rewards[i]).Error) {
				rewardsql := fmt.Sprintf("DELETE  FROM rewards WHERE  type = %d and block_hash = '%s'  and node_id = '%s'",
					rewards[i].Type, rewards[i].BlockHash, rewards[i].NodeId)
				storage.db.Exec(rewardsql)
				storage.db.Create(&rewards[i])
			}
		}
	}
	fmt.Println("[Storage]  AddRewards cost: ", time.Since(timeBegin), "，len :", len(rewards))
	return true
}

func (storage *Storage) AddBlockToMiner(blockToMinersPrpses, blockToMinersVerfs []*models.BlockToMiner) bool {

	createSuccess, updateSucceess := true, true

	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	if len(blockToMinersPrpses) == 0 && len(blockToMinersVerfs) == 0 {
		return false
	}
	timeBegin := time.Now()
	tx := storage.db.Begin()
	for i := 0; i < len(blockToMinersPrpses); i++ {
		if blockToMinersPrpses[i] != nil {
			if !errors(tx.Create(&blockToMinersPrpses[i]).Error) {
				createSuccess = false
				rewardsql := fmt.Sprintf("DELETE  FROM block_to_miners WHERE  block_height = '%d' ",
					blockToMinersPrpses[i].BlockHeight)
				if tx.Exec(rewardsql).Error == nil && tx.Create(&blockToMinersPrpses[i]).Error == nil {
					createSuccess = true
				}
			}
		}
	}
	for _, v := range blockToMinersVerfs {
		if v != nil {
			if tx.Model(&models.BlockToMiner{}).Where("block_height = ?", v.BlockHeight).Updates(*v).Error != nil {
				updateSucceess = false
			}
		}
	}

	if createSuccess && updateSucceess {
		tx.Commit()
		fmt.Println("[Storage]  AddBlockToMiner Success. cost: ", time.Since(timeBegin), "，len :", len(blockToMinersPrpses)+len(blockToMinersVerfs))
	} else {
		tx.Rollback()
		fmt.Println("[Storage]  AddBlockToMiner Fail. cost: ", time.Since(timeBegin), "，len :", len(blockToMinersPrpses)+len(blockToMinersVerfs))
	}

	return true
}

func (storage *Storage) AddBlockToMinerSupplement(blockToMinersPrpses, blockToMinersVerfs []*models.BlockToMiner, tx *gorm.DB) bool {

	createSuccess, updateSucceess := true, true

	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	if len(blockToMinersPrpses) == 0 && len(blockToMinersVerfs) == 0 {
		return false
	}
	timeBegin := time.Now()
	//tx := storage.db.Begin()
	for _, v := range blockToMinersPrpses {
		if v != nil {
			blockToMiners := make([]*models.BlockToMiner, 0)
			tx.Model(&models.BlockToMiner{}).Where("block_height = ?", v.BlockHeight).Find(&blockToMiners)
			if len(blockToMiners) == 0 {
				if !errors(storage.db.Create(&v).Error) {
					createSuccess = false
				}
			}
		}
	}

	for _, v := range blockToMinersVerfs {
		if v != nil {
			if tx.Model(&models.BlockToMiner{}).Where("block_height = ?", v.BlockHeight).Updates(*v).Error != nil {
				updateSucceess = false
			}
		}
	}

	if createSuccess && updateSucceess {
		fmt.Println("[Storage]  AddBlockToMiner Success. cost: ", time.Since(timeBegin), "，len :", len(blockToMinersPrpses)+len(blockToMinersVerfs))
		return true
	} else {
		fmt.Println("[Storage]  AddBlockToMiner Fail. cost: ", time.Since(timeBegin), "，len :", len(blockToMinersPrpses)+len(blockToMinersVerfs))
		return false
	}

	return false
}

func (storage *Storage) SetLoadVerified(block *models.Block) bool {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	block.LoadVerify = true
	storage.db.Model(&block).Where("hash = ?", block.Hash).Updates(block)
	return true
}

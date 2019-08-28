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
	"github.com/zvchain/zvchain/browser/models"
	"time"
)

const PageSize uint64 = 20

type Storage struct {
	db           *gorm.DB
	dbAddr       string
	dbPort       int
	dbUser       string
	dbPassword   string
	rpcAddrStr   string
	topBlockHigh uint64
	accounts     []*models.Account
}

func NewStorage(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool) *Storage {

	storage := &Storage{
		dbAddr:     dbAddr,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPassword: dbPassword,
	}
	storage.Init(reset)
	return storage
}

func (storage *Storage) Init(reset bool) {
	if storage.db != nil {
		return
	}
	//args := fmt.Sprintf("root:Jobs1955!@tcp(119.23.205.254:3306)/tas?charset=utf8&parseTime=True&loc=Local")
	args := fmt.Sprintf("%s:%s@tcp(%s:%d)/tas?charset=utf8&parseTime=True&loc=Local",
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
	storage.db = db
	if reset {
		db.DropTable(&models.Account{})
	}
	if !db.HasTable(&models.Account{}) {
		db.CreateTable(&models.Account{})
	}
	if !db.HasTable(&models.Sys{}) {
		db.CreateTable(&models.Sys{})
	}
	if !db.HasTable(&models.PoolStake{}) {
		db.CreateTable(&models.PoolStake{})
	}

}

func (storage *Storage) GetDB() *gorm.DB {
	return storage.db
}

/**
 * common method
 * updateobject into mysqldb
 */
func (storage *Storage) UpdateObject(object interface{}) bool {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	storage.db.Model(object).Updates(object)
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

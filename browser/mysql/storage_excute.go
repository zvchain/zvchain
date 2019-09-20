package mysql

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"time"
)

const (
	Blockrewardtophight   = "block_reward.top_block_height"
	Blocktophight         = "block.top_block_height"
	GroupTopHeight        = "group.top_group_height"
	PrepareGroupTopHeight = "group.top_prepare_group_height"
	DismissGropHeight     = "group.top_dismiss_group_height"
	LIMIT                 = 20
	CheckpointMaxHeight   = 1000000000
	ACCOUNTDBNAME         = "account_lists"
)

func (storage *Storage) MapToJson(mapdata map[string]interface{}) string {
	var data string
	if mapdata != nil {
		result, _ := json.Marshal(mapdata)
		data = string(result)
	}
	return data
}

func (storage *Storage) AddBatchAccount(accounts []*models.AccountList) bool {
	//fmt.Println("[Storage] add log ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	for i := 0; i < len(accounts); i++ {
		if accounts[i] != nil {
			storage.AddObjects(accounts[i])
		}
	}
	return true
}

func (storage *Storage) GetAccountById(address string) []*models.AccountList {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.AccountList, 0, 0)
	storage.db.Where("address = ? ", address).Find(&accounts)
	return accounts
}

func (storage *Storage) GetAccountByMaxPrimaryId(maxid uint64) []*models.AccountList {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.AccountList, LIMIT, LIMIT)
	storage.db.Where("id > ? ", maxid).Limit(LIMIT).Find(&accounts)
	return accounts
}

func (storage *Storage) GetAccountByPage(page uint64) []*models.AccountList {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.AccountList, LIMIT, LIMIT)
	storage.db.Offset(page * LIMIT).Limit(LIMIT).Find(&accounts)
	return accounts
}

func (storage *Storage) GetAccountByRoletype(maxid uint, roleType uint64) []*models.AccountList {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.AccountList, LIMIT, LIMIT)
	if maxid > 0 {
		storage.db.Limit(LIMIT).Where("role_type = ? and id < ?", roleType, maxid).Order("id desc").Find(&accounts)
	} else {
		storage.db.Limit(LIMIT).Where("role_type = ? ", roleType).Order("id desc").Find(&accounts)

	}
	return accounts
}

func (storage *Storage) GetGroupByHigh(height uint64) []*models.Group {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	groups := make([]*models.Group, 0, 0)
	storage.db.Where("height = ? ", height).Find(&groups)
	return groups
}

func (storage *Storage) AddBlockRewardMysqlTransaction(accounts map[string]uint64) bool {
	if storage.db == nil {
		return false
	}
	tx := storage.db.Begin()

	updateReward := func(addr string, reward uint64) error {
		return tx.Table(ACCOUNTDBNAME).
			Where("address = ?", addr).
			Updates(map[string]interface{}{"rewards": gorm.Expr("rewards + ?", reward)}).Error
	}
	for address, reward := range accounts {
		if address == "" {
			continue
		}
		if !errors(updateReward(address, reward)) {
			tx.Rollback()
			fmt.Println("AddBlockRewardMysqlTransaction,", address, ",", reward)
			return false
		}
	}
	if !storage.IncrewardBlockheightTosys(tx) {
		return false
	}
	tx.Commit()
	return true
}

func (storage *Storage) AddOrUpPoolStakeFrom(stakefrom []*models.PoolStake) bool {
	if storage.db == nil {
		return false
	}
	tx := storage.db.Begin()

	updateStakefrom := func(stake *models.PoolStake) error {
		expression := map[string]interface{}{"from": stake.From,
			"stake": gorm.Expr("stake + ?", stake.Stake)}
		if stake.Stake < 0 {
			expression = map[string]interface{}{"from": stake.From,
				"stake": gorm.Expr("stake - ?", -stake.Stake)}
		}
		return tx.Model(&stake).
			Where("id = ?", stake.ID).
			Updates(expression).Error
	}

	addStakefrom := func(stake *models.PoolStake) error {
		return tx.Create(&stake).Error
	}

	for _, stake := range stakefrom {
		stakeInfo := getstakefrom(tx, stake.Address, stake.From)
		if stakeInfo != nil {
			stake.ID = stakeInfo.ID
			if !errors(updateStakefrom(stake)) {
				tx.Rollback()
				return false
			}
		} else {
			if !errors(addStakefrom(stake)) {
				tx.Rollback()
				return false
			}
		}
	}
	tx.Commit()
	return true

}

func getstakefrom(tx *gorm.DB, address string, from string) *models.PoolStake {
	stake := &models.PoolStake{}
	tx.Limit(1).Where(map[string]interface{}{"address": address, "from": from}).Find(stake)
	if stake.Address == "" {
		return nil
	}
	return stake

}
func (storage *Storage) IncrewardBlockheightTosys(tx *gorm.DB) bool {

	sys := &models.Sys{
		Variable: Blockrewardtophight,
		SetBy:    "carrie.cxl",
	}
	sysConfig := make([]models.Sys, 0, 0)
	tx.Limit(1).Where("variable = ?", Blockrewardtophight).Find(&sysConfig)
	if sysConfig != nil && len(sysConfig) > 0 {
		if !errors(tx.Model(&sysConfig[0]).UpdateColumn("value", gorm.Expr("value + ?", 1)).Error) {
			tx.Rollback()
			return false
		}
	} else {
		if !errors(tx.Create(&sys).Error) {
			tx.Rollback()
			return false
		}

	}
	return true
}

func errors(error error) bool {
	if error != nil {
		fmt.Println("update/add error", error)
		return false
	}
	return true

}

func (storage *Storage) AddBlockHeightSystemconfig(sys *models.Sys) bool {
	hight, ifexist := storage.TopBlockHeight()
	if hight == 0 && ifexist == false {
		storage.AddObjects(&sys)
	} else {
		storage.db.Model(&sys).Where("variable=?", sys.Variable).UpdateColumn("value", gorm.Expr("value + ?", 1))
	}
	return true
}

//func (storage *Storage) AddGroupHeightSystemconfig(sys *models.Sys) bool {
//	hight := storage.TopGroupHeight()
//	if hight > 0 {
//		storage.db.Model(&sys).UpdateColumn("value", gorm.Expr("value + ?", 1))
//	} else {
//		sys.Value = 1
//		storage.AddObjects(&sys)
//	}
//	return true
//}

func (storage *Storage) UpdateAccountByColumn(account *models.AccountList, attrs map[string]interface{}) bool {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	if account.Address != "" {
		storage.db.Model(&account).Where("address = ?", account.Address).Updates(attrs)
	} else {
		storage.db.Model(&account).Updates(attrs)
	}
	return true

}

func (storage *Storage) UpdateAccountbyAddress(account *models.AccountList, attrs map[string]interface{}) bool {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	if account.Address != "" {
		storage.db.Model(&account).Where("address = ?", account.Address).Updates(attrs)
	} else {
		return false
	}
	return true

}

// get topblockreward height
func (storage *Storage) TopBlockRewardHeight(variable string) uint64 {
	if storage.db == nil {
		return 0
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", variable).Find(&sys)
	if len(sys) > 0 {
		storage.topBlockHigh = sys[0].Value
		return sys[0].Value
	}
	return 0
}

func (storage *Storage) TopBlockHeight() (uint64, bool) {
	if storage.db == nil {
		return 0, false
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", Blocktophight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value, true
	}
	return 0, false
}

func (storage *Storage) TopGroupHeight() (uint64, bool) {
	if storage.db == nil {
		return 0, false
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", GroupTopHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value, true
	}
	return 0, false
}

func (storage *Storage) TopPrepareGroupHeight() (uint64, bool) {
	if storage.db == nil {
		return 0, false
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", PrepareGroupTopHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value, true
	}
	return 0, false
}

func (storage *Storage) TopDismissGroupHeight() (uint64, bool) {
	if storage.db == nil {
		return 0, false
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", DismissGropHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value, true
	}
	return 0, false
}

func (storage *Storage) GetDataByColumn(table interface{}, column string, value interface{}) interface{} {
	if storage.db == nil {
		return nil
	}
	storage.db.Find(&table).Pluck(column, &value)

	return value
}

func (storage *Storage) AddGroup(group *models.Group) bool {
	fmt.Println("[Storage] add group ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}

	if storage.topGroupHigh < group.Height {
		storage.topGroupHigh = group.Height
	}
	storage.db.Create(&group)
	return true
}

func (storage *Storage) AddBlock(block *models.Block) bool {
	//fmt.Println("[Storage] add block ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	timeBegin := time.Now()

	if storage.topbrowserBlockHeight < block.Height {
		storage.topbrowserBlockHeight = block.Height
	}
	//storage.statistics.BlocksCountToday += 1
	//storage.statistics.TopBlockHeight = storage.topbrowserBlockHeight

	if !errors(storage.db.Create(&block).Error) {
		blocksql := fmt.Sprintf("DELETE  FROM blocks WHERE  hash = '%s'",
			block.Hash)
		storage.db.Exec(blocksql)
		storage.db.Create(&block)
	}
	fmt.Println("[Storage]  AddBlock cost: ", time.Since(timeBegin))
	return true
}

func (storage *Storage) AddTransactions(trans []*models.Transaction) bool {
	//fmt.Println("[Storage] add transaction ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	timeBegin := time.Now()
	//tx := storage.db.Begin()
	for i := 0; i < len(trans); i++ {

		if trans[i] != nil {
			if !errors(storage.db.Create(&trans[i]).Error) {
				transql := fmt.Sprintf("DELETE  FROM transactions WHERE  hash = '%s'",
					trans[i].Hash)
				storage.db.Exec(transql)
				storage.db.Create(&trans[i])
			}
		}
	}
	//storage.statistics.TransCountToday += uint64(len(trans))
	//storage.statistics.TotalTransCount += uint64(len(trans))
	//tx.Commit()
	fmt.Println("[Storage]  AddTransactions cost: ", time.Since(timeBegin))

	return true
}

func (storage *Storage) AddReceipts(receipts []*models.Receipt) bool {
	//fmt.Println("[Storage] add receipt ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	timeBegin := time.Now()

	//tx := storage.db.Begin()
	for i := 0; i < len(receipts); i++ {
		if !errors(storage.db.Create(&receipts[i]).Error) {
			transql := fmt.Sprintf("DELETE  FROM receipts WHERE  tx_hash = '%s'",
				receipts[i].TxHash)
			storage.db.Exec(transql)
			storage.db.Create(&receipts[i])
		}

	}
	//tx.Commit()
	fmt.Println("[Storage]  AddReceipts cost: ", time.Since(timeBegin))

	return true
}

func (storage *Storage) browserTopBlockHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	blocks := make([]models.Block, 0, 1)
	storage.db.Limit(1).Order("height desc").Find(&blocks)
	if len(blocks) > 0 {
		//storage.topBlockHeight = blocks[0].Height

		return uint64(blocks[0].Height)
	}
	return 0
}

func (storage *Storage) RewardTopBlockHeight() (uint64, uint64) {
	if storage.db == nil {
		return 0, 0
	}
	rewards := make([]models.Reward, 0, 0)
	storage.db.Limit(1).Order("reward_height desc").Find(&rewards)
	if len(rewards) > 0 {

		return rewards[0].BlockHeight, rewards[0].RewardHeight
	}
	return 0, 0
}
func (storage *Storage) GetTopblock() uint64 {
	var maxHeight uint64
	storage.db.Table("blocks").Select("max(height) as height").Row().Scan(&maxHeight)
	return maxHeight
}

func (storage *Storage) DeleteForkblock(preHeight uint64, localHeight uint64) (err error) {

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容
		}
	}()
	blockSql := fmt.Sprintf("DELETE  FROM blocks WHERE height > %d and height < %d", preHeight, localHeight)
	transactionSql := fmt.Sprintf("DELETE  FROM transactions WHERE block_height > %d and block_height < %d", preHeight, localHeight)
	receiptSql := fmt.Sprintf("DELETE  FROM receipts WHERE block_height > %d and block_height < %d", preHeight, localHeight)
	storage.db.Debug().Exec(blockSql)
	storage.db.Exec(transactionSql)
	storage.db.Exec(receiptSql)
	return err
}

func (storage *Storage) DeleteForkReward(preHeight uint64, localHeight uint64) (err error) {

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容
		}
	}()

	verifySql := fmt.Sprintf("DELETE  FROM rewards WHERE reward_height > %d and reward_height < %d", preHeight, localHeight)
	storage.db.Exec(verifySql)
	return err
}

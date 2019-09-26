package mysql

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"time"
)

const (
	Blockrewardtophight = "block_reward.top_block_height"
	Blocktophight       = "block.top_block_height"
	Blockcurblockhight  = "block.cur_block_height"
	BlockDeleteCount    = "block.delete_count"
	Blockcurtranhight   = "block.cur_tran_height"

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

func (storage *Storage) AddBlockRewardMysqlTransaction(accounts map[string]float64, updates map[string]float64) bool {
	if storage.db == nil {
		return false
	}
	isSuccess := false
	tx := storage.db.Begin()

	defer func() {
		if isSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()
	updateReward := func(addr string, reward float64) error {
		mapData := make(map[string]interface{})
		mapData["rewards"] = gorm.Expr("rewards + ?", reward)
		balance, ok := updates[addr]
		if ok {
			mapData["balance"] = balance
		}
		return tx.Table(ACCOUNTDBNAME).
			Where("address = ?", addr).
			Updates(mapData).Error
	}
	for address, reward := range accounts {
		if address == "" {
			continue
		}
		if !errors(updateReward(address, reward)) {
			fmt.Println("AddBlockRewardMysqlTransaction,", address, ",", reward)
			return false
		}
	}
	if !storage.IncrewardBlockheightTosys(tx, Blockrewardtophight) {
		return false
	}
	isSuccess = true
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
func (storage *Storage) IncrewardBlockheightTosys(tx *gorm.DB, variable string) bool {
	if variable == "" {
		return false
	}

	sys := &models.Sys{
		Variable: variable,
		SetBy:    "carrie.cxl",
	}
	sysConfig := make([]models.Sys, 0, 0)
	tx.Limit(1).Where("variable = ?", variable).Find(&sysConfig)
	if sysConfig != nil && len(sysConfig) > 0 {
		if !errors(tx.Model(&sysConfig[0]).UpdateColumn("value", gorm.Expr("value + ?", 1)).Error) {
			return false
		}
	} else {
		if !errors(tx.Create(&sys).Error) {
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

func (storage *Storage) AddSysConfig(variable string) {
	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	sysdata := make([]models.Sys, 0, 0)
	storage.db.Limit(1).Where("variable = ?", variable).Find(&sysdata)
	if len(sysdata) < 1 {
		sys.Value = 0
		storage.AddObjects(sys)
	}
}
func (storage *Storage) UpdateSysConfigValue(variable string, value int64) {
	if value < 1 {
		return
	}
	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", gorm.Expr("value + ?", value))
}

func (storage *Storage) AddCurCountconfig(curtime time.Time, variable string) bool {

	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	storage.addcursys(curtime, variable)
	t := time.Now()
	date := fmt.Sprintf("%d-%d-%d", t.Year(), t.Month(), t.Day())
	if variable == Blockcurblockhight {
		if storage.statisticsblockLastUpdate == "" {
			storage.statisticsblockLastUpdate = date
		}
		if date != storage.statisticsblockLastUpdate {
			storage.statisticsblockLastUpdate = date
			storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", 0)
		}
	} else {
		if storage.statisticstranLastUpdate == "" {
			storage.statisticstranLastUpdate = date
		}
		if date != storage.statisticstranLastUpdate {
			storage.statisticstranLastUpdate = date
			storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", 0)
		}
	}

	return true
}

func (storage *Storage) addcursys(curtime time.Time, variable string) {
	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	//timeBegin := time.Now()
	if curtime.After(GetTodayStartTs()) {
		sysdata := make([]models.Sys, 0, 0)
		storage.db.Limit(1).Where("variable = ?", variable).Find(&sysdata)
		if len(sysdata) < 1 {
			hight := storage.GetCurBlockCount(variable)
			sys.Value = hight + 1
			storage.AddObjects(sys)
		} else {
			storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", gorm.Expr("value + ?", 1))
		}
	}
}

func GetTodayStartTs() time.Time {
	t := time.Now()
	tm1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return tm1
}

func (storage *Storage) Deletecurcount(variable string) {
	blocksql := fmt.Sprintf("DELETE  FROM sys WHERE  variable = '%s'",
		variable)
	storage.db.Exec(blocksql)
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

func (storage *Storage) GetCurBlockCount(variable string) uint64 {
	var countToday uint64
	if variable == Blockcurblockhight {
		storage.db.Model(&models.Block{}).Where(" cur_time > CURDATE()").Count(&countToday)
	} else if variable == Blockcurtranhight {
		storage.db.Model(&models.Transaction{}).Where(" cur_time > CURDATE()").Count(&countToday)

	}
	return countToday

}

func (storage *Storage) GetCurTranCount() uint64 {
	var transCountToday uint64
	storage.db.Model(&models.Transaction{}).Where(" cur_time > CURDATE()").Count(&transCountToday)
	return transCountToday

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
	var maxIndex uint64 = 0
	blocks := make([]*models.Block, 1)
	storage.db.Limit(1).Order("cur_index desc").Find(&blocks)
	if len(blocks) > 0 {
		maxIndex = blocks[0].CurIndex
	}
	block.CurIndex = maxIndex + 1
	if !errors(storage.db.Create(&block).Error) {
		blocksql := fmt.Sprintf("DELETE  FROM blocks WHERE  hash = '%s'",
			block.Hash)
		storage.db.Exec(blocksql)
		storage.db.Create(&block)
	}
	storage.AddCurCountconfig(block.CurTime, Blockcurblockhight)
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
	var maxIndex uint64 = 0
	txs := make([]*models.Transaction, 1)
	storage.db.Limit(1).Order("cur_index desc").Find(&txs)
	if len(txs) > 0 {
		maxIndex = txs[0].CurIndex
	}
	for i := 0; i < len(trans); i++ {

		if trans[i] != nil {
			maxIndex++
			trans[i].CurIndex = maxIndex
			if !errors(storage.db.Create(&trans[i]).Error) {
				transql := fmt.Sprintf("DELETE  FROM transactions WHERE  hash = '%s'",
					trans[i].Hash)
				storage.db.Exec(transql)
				storage.db.Create(&trans[i])
			}
		}
		storage.AddCurCountconfig(trans[i].CurTime, Blockcurtranhight)

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
	blockSql := fmt.Sprintf("DELETE  FROM blocks WHERE height > %d and height <= %d", preHeight, localHeight)
	transactionSql := fmt.Sprintf("DELETE  FROM transactions WHERE block_height > %d and block_height <= %d", preHeight, localHeight)
	receiptSql := fmt.Sprintf("DELETE  FROM receipts WHERE block_height > %d and block_height <= %d", preHeight, localHeight)
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

	verifySql := fmt.Sprintf("DELETE  FROM rewards WHERE reward_height > %d and reward_height <= %d", preHeight, localHeight)
	storage.db.Exec(verifySql)
	return err
}

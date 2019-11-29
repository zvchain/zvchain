package mysql

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/tidwall/gjson"
	"github.com/zvchain/zvchain/browser/common"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/util"
	common2 "github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/tvm"
	"math/big"
	"sort"
	"strings"
	"time"
)

const (
	Blockrewardtopheight    = "block_reward.top_block_height"
	Blocktopheight          = "block.top_block_height"
	BlockStakeMappingHeight = "block.stake_mapping_height"
	Blockcurblockheight     = "block.cur_block_height"
	BlockDeleteCount        = "block.delete_count"
	Blockcurtranheight      = "block.cur_tran_height"

	GroupTopHeight        = "group.top_group_height"
	PrepareGroupTopHeight = "group.top_prepare_group_height"
	DismissGropHeight     = "group.top_dismiss_group_height"
	LIMIT                 = 20
	ACCOUNTDBNAME         = "account_lists"
	RECENTMINEBLOCKS      = "recent_mine_blocks"
	MAXCONFIRMREWARDCOUNT = 1000
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

func (storage *Storage) GetBlockByHeight(height uint64) []*models.Block {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.Block, 0, 0)
	storage.db.Where("height = ? ", height).Find(&accounts)
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

func (storage *Storage) AddBlockRewardMysqlTransaction(accounts map[string]float64,
	updates map[string]float64,
	mapblockcount map[string]map[string]uint64,
	forcount uint64) bool {
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
		blockcount, ok := mapblockcount[addr]
		if ok {
			if blockcount["verify_count"] > 0 {
				mapData["verify_count"] = gorm.Expr("verify_count + ?", blockcount["verify_count"])
			}
			if blockcount["proposal_count"] > 0 {
				mapData["proposal_count"] = gorm.Expr("proposal_count + ?", blockcount["proposal_count"])
			}
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
	if !storage.IncrewardBlockheightTosys(tx, Blockrewardtopheight, forcount) {
		return false
	}
	isSuccess = true
	return true
}

func (storage *Storage) UpdateMineBlocks(mapMineBlockCount map[string][]uint64) bool {

	const MaxCounts = 1000
	if storage.db == nil {
		return false
	}

	updateReward := func(addr string, mineCount []uint64) error {

		//mineCount, ok := mapMineBlockCount[addr]
		pendingBlockHeights := models.BlockHeights(mineCount)
		sort.Sort(pendingBlockHeights)

		baseData := make([]models.RecentMineBlock, 0)
		storage.db.Model(&models.RecentMineBlock{}).Where("address = ?", addr).Find(&baseData)

		if len(baseData) == 0 {
			byteVerifyBlocks, err := json.Marshal(pendingBlockHeights)
			if err != nil {
				return err
			}
			verifyBlocks := string(byteVerifyBlocks)

			RecentMineBlock := models.RecentMineBlock{
				Address:            addr,
				RecentVerifyBlocks: verifyBlocks,
			}
			return storage.db.Table(RECENTMINEBLOCKS).Create(&RecentMineBlock).Error
			//return storage.db.Create(RecentMineBlock).Error

		} else {
			blockHeights := make([]uint64, 0)
			if err := json.Unmarshal([]byte(baseData[0].RecentVerifyBlocks), &blockHeights); err != nil {
				return err
			}

			totalBlocks := pendingBlockHeights
			totalBlocks = append(totalBlocks, blockHeights...)

			delta := MaxCounts - len(blockHeights)

			if delta < len(mineCount) {
				totalBlocks = totalBlocks[:MaxCounts]
			}

			updateString, err := json.Marshal(totalBlocks)
			if err != nil {
				return err
			}
			RecentMineBlock := models.RecentMineBlock{
				Address:            addr,
				RecentVerifyBlocks: string(updateString),
			}
			return storage.db.Table(RECENTMINEBLOCKS).Where("address = ?", addr).Updates(RecentMineBlock).Error

		}
	}

	for addr, counts := range mapMineBlockCount {
		if !errors(updateReward(addr, counts)) {
			fmt.Println("UpdateMineBlocks,", addr)
			return false
		}
	}

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
func (storage *Storage) IncrewardBlockheightTosys(tx *gorm.DB, variable string, value uint64) bool {
	if variable == "" {
		return false
	}

	sys := &models.Sys{
		Variable: variable,
		SetBy:    "carrie.cxl",
		Value:    value,
	}
	sysConfig := make([]models.Sys, 0, 0)
	tx.Limit(1).Where("variable = ?", variable).Find(&sysConfig)
	if sysConfig != nil && len(sysConfig) > 0 {
		if !errors(tx.Model(&sysConfig[0]).UpdateColumn("value", value).Error) {
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
		browserlog.BrowserLog.Info("update/add error", error)
		return false
	}
	return true

}

func (storage *Storage) AddBlockHeightSystemconfig(sys *models.Sys) bool {
	hight, ifexist := storage.TopBlockHeight()
	if hight == 0 && ifexist == false {
		storage.AddObjects(&sys)
	} else {
		storage.db.Model(&sys).Where("variable=?", sys.Variable).UpdateColumn("value", sys.Value)
	}
	return true
}

func (storage *Storage) BlockStakeMappingHeightCfg(sys *models.Sys) bool {
	hight, ifexist := storage.TopStakeMappingHeight()
	if hight == 0 && ifexist == false {
		storage.AddObjects(&sys)
	} else {
		storage.db.Model(&sys).Where("variable=?", sys.Variable).UpdateColumn("value", sys.Value)
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
func (storage *Storage) UpdateSysConfigValue(variable string, value int64, isadd bool) {
	if value < 1 {
		return
	}
	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	if isadd {
		storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", gorm.Expr("value + ?", value))
	} else {
		//err := storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", gorm.Expr("value - ?", value)).Error
		//when value < 0 ,out of range
		sql := fmt.Sprintf("UPDATE sys  SET value =(CASE WHEN value < %d  THEN 0 ELSE value-%d END)  WHERE  variable = '%s' LIMIT 1",
			value,
			value,
			sys.Variable)
		storage.db.Exec(sql)
	}
}

func (storage *Storage) InitCurConfig() {
	t := time.Now()
	date := fmt.Sprintf("%d-%d-%d", t.Year(), t.Month(), t.Day())
	storage.statisticsblockLastUpdate = date
	storage.statisticstranLastUpdate = date
	storage.initVariable(Blockcurblockheight, 1)
	storage.initVariable(Blockcurtranheight, 0)
}

func (storage *Storage) initVariable(variable string, count uint64) {
	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	sysdata := make([]models.Sys, 0, 0)
	storage.db.Limit(1).Where("variable = ?", variable).Find(&sysdata)
	if len(sysdata) < 1 {
		sys.Value = count
		success := storage.AddObjects(sys)
		if !success {
			panic("init variable failed!")
		}
	}
}

func (storage *Storage) AddCurCountconfig(curtime time.Time, variable string) bool {
	sys := &models.Sys{
		Variable: variable,
		SetBy:    "xiaoli",
	}
	t := time.Now()
	date := fmt.Sprintf("%d-%d-%d", t.Year(), t.Month(), t.Day())
	if variable == Blockcurblockheight {
		if date != storage.statisticsblockLastUpdate {
			storage.statisticsblockLastUpdate = date
			storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", 0)
		}
	} else {
		if date != storage.statisticstranLastUpdate {
			storage.statisticstranLastUpdate = date
			storage.db.Model(sys).Where("variable=?", sys.Variable).UpdateColumn("value", 0)
		}
	}
	if GetTodayStartTs(curtime).Equal(GetTodayStartTs(t)) {
		storage.UpdateSysConfigValue(variable, 1, true)
	}
	return true
}

func GetTodayStartTs(tm time.Time) time.Time {
	tm1 := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())
	return tm1
}

func (storage *Storage) Deletecurcount(variable string) {
	blocksql := fmt.Sprintf("DELETE  FROM sys WHERE  variable = '%s'",
		variable)
	storage.db.Exec(blocksql)
}

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

func (storage *Storage) MinConfirmBlockRewardHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	rewards := make([]models.Reward, 0, 0)
	storage.db.Limit(1).Order("id ASC").Find(&rewards)
	if len(rewards) > 0 {
		return rewards[0].BlockHeight
	}
	return 0
}
func (storage *Storage) MaxConfirmBlockRewardHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	rewards := make([]models.Reward, 0, 0)
	storage.db.Limit(1).Order("id DESC").Find(&rewards)
	if len(rewards) > 0 {
		return rewards[0].BlockHeight
	}
	return 0
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
	storage.db.Limit(1).Where("variable = ?", Blocktopheight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value, true
	}
	return 0, false
}

func (storage *Storage) TopStakeMappingHeight() (uint64, bool) {
	if storage.db == nil {
		return 0, false
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", BlockStakeMappingHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value, true
	}
	return 0, false
}

func (storage *Storage) GetCurCount(variable string) uint64 {
	var countToday uint64
	if variable == Blockcurblockheight {
		storage.db.Model(&models.Block{}).Where(" cur_time >= CURDATE()").Count(&countToday)
	} else if variable == Blockcurtranheight {
		storage.db.Model(&models.Transaction{}).Where(" cur_time >= CURDATE()").Count(&countToday)

	}
	return countToday

}

func (storage *Storage) GetCurTranCount() uint64 {
	var transCountToday uint64
	storage.db.Model(&models.Transaction{}).Where(" cur_time >= CURDATE()").Count(&transCountToday)
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
	data := make([]*models.Group, 0, 0)
	storage.db.Limit(1).Where("id = ?", group.Id).Find(&data)
	if len(data) > 0 {
		return false
	}

	if storage.topGroupHigh < group.Height {
		storage.topGroupHigh = group.Height
	}
	storage.db.Create(&group)
	return true
}
func (storage *Storage) AddContractTransaction(contract *models.ContractTransaction) bool {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	storage.db.Create(&contract)
	return true
}

func (storage *Storage) IsDbTokenContract(contract string) bool {
	token := make([]models.TokenContract, 0, 0)
	storage.db.Where("contract_addr = ?", contract).Find(&token)
	if len(token) < 1 {
		return false
	}
	return true
}

func (storage *Storage) UpdateTokenUser(contract string, addr string, value string) {
	if value == "" {
		return
	}
	//token := make([]models.TokenContract, 0, 0)
	//storage.db.Where("contract_addr = ?", contract).Find(&token)
	//if len(token) < 1 {
	//	return
	//}

	users := make([]models.TokenContractUser, 0, 0)
	storage.db.Where("address =? and contract_addr = ?", addr, contract).Find(&users)
	if len(users) > 0 {
		value := getUseValue(contract, addr)
		storage.db.Model(&models.TokenContractUser{}).
			Where("contract_addr = ? and address = ? ", contract, addr).
			Update("value", value)
	} else {
		user := &models.TokenContractUser{
			ContractAddr: contract,
			Address:      addr,
			Value:        value,
		}
		storage.db.Create(&user)
	}

}
func (storage *Storage) AddContractCallTransaction(contract *models.ContractCallTransaction) bool {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	data := make([]*models.ContractCallTransaction, 0, 0)
	storage.db.Limit(1).Where("tx_hash = ?", contract.TxHash).Find(&data)
	if len(data) > 0 {
		return false
	}
	storage.db.Create(&contract)
	return true
}

func (storage *Storage) Reward2MinerBlock(height uint64) bool {
	rewards := make([]*models.Reward, 0)
	storage.db.Where("block_height = ?", height).Find(&rewards)
	if len(rewards) < 1 {
		return true
	}
	isSuccess := false
	tx := storage.db.Begin()
	for _, reward := range rewards {
		if !errors(upMinerBlock(tx, reward.NodeId, reward.BlockHeight, reward.Type)) {
			return false
		}
	}
	if !errors(DeleteRewardByHeight(tx, height)) {
		return false
	}

	defer func() {
		if isSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()
	isSuccess = true
	return isSuccess

}

func upMinerBlock(tx *gorm.DB, addr string,
	height uint64,
	typeId uint64) error {
	if addr == "" || height < 0 || typeId < 0 {
		return nil
	}
	rewards := make([]*models.MinerToBlock, 0)
	tx.Limit(1).Where("address = ? and type = ?", addr, typeId).
		Order("sequence desc").Find(&rewards)

	if len(rewards) > 0 && rewards[0].BlockCnts < MAXCONFIRMREWARDCOUNT {
		mapData := make(map[string]interface{})
		blockVerHeights := make([]uint64, 0)
		if err := json.Unmarshal([]byte(rewards[0].BlockIDs), &blockVerHeights); err != nil {
			return err
		}
		blockVerHeights = append(blockVerHeights, height)
		//blockVerHeights = append(blockVerHeights, height)
		//blockVerHeights = util.InsertUint64SliceCopy(blockVerHeights, []uint64{height}, 0)
		updateVerString, err := json.Marshal(blockVerHeights)
		if err != nil {
			return err
		}
		mapData["block_ids"] = updateVerString
		mapData["block_cnts"] = len(blockVerHeights)

		erraccount := upAccountConfirmCount(tx, typeId, rewards[0].Sequence, uint64(len(blockVerHeights)), addr)
		if erraccount != nil {
			return erraccount
		}

		return tx.Model(&models.MinerToBlock{}).
			Where("id = ?", rewards[0].ID).
			Updates(mapData).Error

	} else {
		sequence := uint64(0)
		if len(rewards) > 0 && rewards[0].BlockCnts >= MAXCONFIRMREWARDCOUNT {
			sequence = rewards[0].Sequence + 1
		}
		MineBlock := models.MinerToBlock{
			Address: addr,
		}
		problockList := make([]uint64, 0)
		if height > 0 {
			problockList = append(problockList, height)
			byteProBlocks, err := json.Marshal(problockList)
			if err != nil {
				return err
			}
			proBlockString := string(byteProBlocks)
			MineBlock.BlockIDs = proBlockString
			MineBlock.Sequence = sequence
			MineBlock.BlockCnts = len(problockList)
			MineBlock.Type = typeId
			erraccount := upAccountConfirmCount(tx, typeId, sequence, uint64(len(problockList)), addr)
			if erraccount != nil {
				return erraccount
			}

		}
		return tx.Model(models.MinerToBlock{}).Create(&MineBlock).Error
	}
}

func upAccountConfirmCount(tx *gorm.DB,
	typeId uint64,
	sequence uint64,
	size uint64,
	addr string) error {
	mapAccountData := make(map[string]interface{})
	if typeId == 0 {
		mapAccountData["verify_confirm_count"] = sequence*MAXCONFIRMREWARDCOUNT + size
	} else {
		mapAccountData["proposal_confirm_count"] = sequence*MAXCONFIRMREWARDCOUNT + size

	}
	err := tx.Table(ACCOUNTDBNAME).
		Where("address = ?", addr).
		Updates(mapAccountData).Error
	if err != nil {
		return err
	}
	return nil
}
func (storage *Storage) AddBlock(block *models.Block) bool {
	//fmt.Println("[Storage] add block ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	timeBegin := time.Now()
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
		browserlog.BrowserLog.Info("AddBlockDELETE", blocksql)
		storage.db.Exec(blocksql)
		storage.db.Create(&block)
	}
	storage.AddCurCountconfig(block.CurTime, Blockcurblockheight)
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
				browserlog.BrowserLog.Info("AddTransactionsDELETE", transql)
				storage.db.Exec(transql)
				storage.db.Create(&trans[i])
			}
		}
		storage.AddCurCountconfig(trans[i].CurTime, Blockcurtranheight)

	}
	//storage.statistics.TransCountToday += uint64(len(trans))
	//storage.statistics.TotalTransCount += uint64(len(trans))
	//tx.Commit()
	fmt.Println("[Storage]  AddTransactions cost: ", time.Since(timeBegin))

	return true
}

func (storage *Storage) AddTokenContract(tran *models.Transaction, log *models.Log) {
	fmt.Println("AddTokenContract", log)

	tokenContracts := make([]*models.TokenContract, 0)
	storage.db.Model(models.TokenContract{}).Where("contract_addr = ?", log.Address).Find(&tokenContracts)
	if log != nil {
		source := gjson.Get(log.Data, "args.0").String()
		target := gjson.Get(log.Data, "args.1").String()
		value := gjson.Get(log.Data, "args.2").Raw
		fmt.Println("AddTokenContract", source, target)
		if source == "" || target == "" {
			return
		}
		realValue := &big.Int{}
		realValue.SetString(value, 10)
		if len(tokenContracts) == 0 {
			fmt.Println("IsTokenContract", source, target, log.Address)

			if !common.IsTokenContract(common2.StringToAddress(log.Address)) {
				return
			}
			fmt.Println("IsTokenContract,", tran)

			//create
			chain := core.BlockChainImpl
			db, err := chain.LatestAccountDB()
			if err != nil {
				browserlog.BrowserLog.Error("AddTokenContract: ", err)
				return
			}

			// 查看balanceOf
			iter := db.DataIterator(common2.StringToAddress(log.Address), []byte{})
			if iter == nil {
				return
			}
			//balanceOf := make(map[string]interface{})
			for iter.Next() {
				if strings.HasPrefix(string(iter.Key[:]), "balanceOf@") {
					realAddr := strings.TrimPrefix(string(iter.Key[:]), "balanceOf@")
					if util.ValidateAddress(realAddr) {
						value := tvm.VmDataConvert(iter.Value[:])
						if value != nil {
							var valuestring string
							if value1, ok := value.(int64); ok {
								valuestring = big.NewInt(value1).String()
							} else if value2, ok := value.(*big.Int); ok {
								valuestring = value2.String()
							}
							storage.UpdateTokenUser(log.Address, realAddr, valuestring)
						}
					}
				}
			}

			tokenContract := models.TokenContract{}
			//mapInterface := make(map[string]interface{})
			keyMap := []string{"name", "symbol", "decimal"}
			for times, key := range keyMap {
				data := db.GetData(common2.StringToAddress(log.Address), []byte(key))
				if v, ok := tvm.VmDataConvert(data).(string); ok {
					switch times {
					case 0:
						tokenContract.Name = v
					case 1:
						tokenContract.Symbol = v
					}
				}
				if v, ok := tvm.VmDataConvert(data).(int64); ok {
					switch times {
					case 2:
						tokenContract.Decimal = v
					}
				}
			}
			tokenContract.ContractAddr = log.Address
			if util.ValidateAddress(source) && util.ValidateAddress(target) {
				if source == target {
					tokenContract.HolderNum = 1
				} else {
					tokenContract.HolderNum = 2
				}
			}

			src := ""
			storage.db.Model(models.Transaction{}).Select("source").Where("contract_address = ? ", tran.Target).Row().Scan(&src)
			tokenContract.Creator = src
			tokenContract.TransferTimes = 1
			storage.db.Create(&tokenContract)

		} else { //update
			storage.db.Model(models.TokenContract{}).Where("contract_addr = ?", log.Address).UpdateColumn("transfer_times", gorm.Expr("transfer_times + ?", 1))
			users := make([]*models.TokenContractUser, 0)
			storage.db.Model(models.TokenContractUser{}).Where("address = ?", target).Find(&users)
			if len(users) == 0 {
				storage.db.Model(models.TokenContract{}).Where("contract_addr = ?", log.Address).UpdateColumn("holder_num", gorm.Expr("holder_num + ?", 1))
			}
		}
		contract := &models.TokenContractTransaction{
			ContractAddr: log.Address,
			Source:       source,
			Target:       target,
			Value:        realValue.String(),
			TxHash:       tran.Hash,
			TxType:       0,
			Status:       0,
			BlockHeight:  tran.BlockHeight,
			CurTime:      tran.CurTime,
		}
		fmt.Println("AddTokenTran", contract)
		//update tokenContractTx and tokenContractUser
		storage.AddTokenTran(contract)
	}

}

func (storage *Storage) AddTokenTran(tokenContract *models.TokenContractTransaction) bool {
	fmt.Println("AddTokenTran,", tokenContract)

	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	tx := storage.db.Begin()
	isSuccess := true
	defer func() {
		if isSuccess {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	if !errors(tx.Create(&tokenContract).Error) {
		isSuccess = false
		return isSuccess
	}
	isSuccess = storage.AddTokenUser(tx, tokenContract)
	return isSuccess
}

func getUseValue(tokenaddr string, useraddr string) string {
	if tokenaddr == "" && useraddr == "" {
		return big.NewInt(0).String()
	}
	key := fmt.Sprintf("balanceOf@%s", useraddr)
	resultData, _ := common.QueryAccountData(tokenaddr, key, 0)
	result := resultData.(map[string]interface{})
	if result["value"] != nil {
		if value, ok := result["value"].(int64); ok {
			return big.NewInt(value).String()
		} else if value, ok := result["value"].(*big.Int); ok {
			return value.String()
		}
	}
	return big.NewInt(0).String()
}

/*
 add tokencontract user info
*/
func (storage *Storage) AddTokenUser(tx *gorm.DB, tokenContract *models.TokenContractTransaction) bool {
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	isSuccess := true

	addressList := []string{tokenContract.Source, tokenContract.Target}
	users := make([]models.TokenContractUser, 0, 0)
	tx.Where("address in (?) and contract_addr = ?", addressList, tokenContract.ContractAddr).Find(&users)
	createUser := make([]string, 0)
	set := &util.Set{}
	if len(users) > 0 {
		for _, user := range users {
			set.Add(user.Address)
			value := getUseValue(tokenContract.ContractAddr, user.Address)
			if !errors(tx.Model(&models.TokenContractUser{}).
				Where("contract_addr = ? and address = ? ", tokenContract.ContractAddr, user.Address).Update("value", value).Error) {
				isSuccess = false
				return isSuccess
			}
		}
		for _, user := range addressList {
			if _, ok := set.M[user]; !ok {
				createUser = append(createUser, user)
			}
		}
	} else {
		createUser = addressList
	}
	if len(createUser) > 0 {
		for _, user := range createUser {
			user := &models.TokenContractUser{
				ContractAddr: tokenContract.ContractAddr,
				Address:      user,
				Value:        getUseValue(tokenContract.ContractAddr, user),
			}
			if !errors(tx.Create(&user).Error) {
				isSuccess = false
				return isSuccess
			}
		}
	}
	return isSuccess

}

func (storage *Storage) AddLogs(receipts []*models.Receipt, trans []*models.Transaction, old bool) bool {
	//fmt.Println("[Storage] add receipt ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}
	maptran := make(map[string]*models.Transaction)
	for _, tr := range trans {
		maptran[tr.Hash] = tr
	}
	timeBegin := time.Now()

	//tx := storage.db.Begin()
	for i := 0; i < len(receipts); i++ {
		if receipts[i].Logs != nil {
			for j := 0; j < len(receipts[i].Logs); j++ {
				if !errors(storage.db.Create(&receipts[i].Logs[j]).Error) {
					transql := fmt.Sprintf("DELETE  FROM logs WHERE  block_number = '%d' and tx_index = '%d' and index='%d'",
						receipts[i].Logs[j].BlockNumber, receipts[i].Logs[j].TxIndex, receipts[i].Logs[j].LogIndex)
					browserlog.BrowserLog.Info("AddLogsDELETE", transql)
					storage.db.Exec(transql)
					storage.db.Create(&receipts[i].Logs[j])
				}
				if old && receipts[i].Logs[j] != nil && receipts[i].Logs[j].Data != "" {
					decodeBytes := receipts[i].Logs[j].Data
					fmt.Println("[Storage]  AddContractTransaction ", receipts[i].Logs[j].Data)
					if decodeBytes != "" {
						//log := string (decodeBytes)
						logData := &common.LogData{}
						if json.Unmarshal([]byte(decodeBytes), logData) == nil {
							value := logData.Value
							contract := &models.ContractTransaction{
								ContractCode: receipts[i].Logs[j].Address,
								Address:      logData.User,
								Value:        value,
								TxHash:       receipts[i].Logs[j].TxHash,
								TxType:       0,
								Status:       1,
								BlockHeight:  receipts[i].Logs[j].BlockNumber,
							}
							storage.AddContractTransaction(contract)
							contractCall := &models.ContractCallTransaction{
								ContractCode: receipts[i].Logs[j].Address,
								TxHash:       receipts[i].Logs[j].TxHash,
								TxType:       0,
								BlockHeight:  receipts[i].Logs[j].BlockNumber,
								Status:       1,
							}
							if maptran[receipts[i].Logs[j].TxHash] != nil {
								contractCall.CurTime = maptran[receipts[i].Logs[j].TxHash].CurTime
							}
							storage.AddContractCallTransaction(contractCall)

						}
					}

				}
			}
		}

	}
	//tx.Commit()
	fmt.Println("[Storage]  AddLogs cost: ", time.Since(timeBegin))

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
			browserlog.BrowserLog.Info("AddReceiptsDELETE", transql)
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

func (storage *Storage) GetContractByHash(hash string) []string {
	if storage.db == nil {
		return nil
	}
	rows, _ := storage.db.Model(models.ContractTransaction{}).Where("tx_hash = ?", hash).Select("address").Rows() // (*sql.Rows, error)
	defer rows.Close()
	s := make([]string, 0, 0)
	var addr string
	for rows.Next() {
		rows.Scan(&addr)
		s = append(s, addr)
	}
	return s
}
func (storage *Storage) GetContractAddressAll() []string {
	if storage.db == nil {
		return nil
	}
	rows, _ := storage.db.Model(models.ContractTransaction{}).Select("distinct(address) as addr").Rows() // (*sql.Rows, error)
	defer rows.Close()
	s := make([]string, 0, 0)
	var addr string
	for rows.Next() {
		rows.Scan(&addr)
		s = append(s, addr)
	}
	return s
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

func (storage *Storage) GetProposalVerifyCount(minerType uint64, hash string) uint64 {
	var count uint64
	storage.db.Model(&models.Reward{}).Where("type=? and node_id = ?", minerType, hash).Count(&count)
	return count
}

func (storage *Storage) GetTopblock() uint64 {
	var maxHeight uint64
	storage.db.Table("blocks").Select("max(height) as height").Row().Scan(&maxHeight)
	return maxHeight
}

func (storage *Storage) UpdateContractTransaction(txHash string, curTime time.Time) {
	contractSql := fmt.Sprintf("UPDATE contract_transactions SET status = 1 WHERE tx_hash = '%s'", txHash)
	//contractcallSql := fmt.Sprintf("UPDATE contract_call_transactions SET status = 1 WHERE tx_hash = '%s'", txHash)
	attrs := make(map[string]interface{})
	attrs["status"] = 1
	attrs["cur_time"] = curTime
	storage.db.Model(&models.ContractCallTransaction{}).Where("tx_hash = ?", txHash).Updates(attrs)
	storage.db.Exec(contractSql)
	//storage.db.Exec(contractcallSql)

}
func (storage *Storage) DeleteForkblock(preHeight uint64, localHeight uint64, curTime time.Time) (err error) {

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容
		}
	}()
	blockSql := fmt.Sprintf("DELETE  FROM blocks WHERE height > %d", preHeight)
	transactionSql := fmt.Sprintf("DELETE  FROM transactions WHERE block_height > %d", preHeight)
	receiptSql := fmt.Sprintf("DELETE  FROM receipts WHERE block_height > %d", preHeight)
	logSql := fmt.Sprintf("DELETE  FROM logs WHERE block_number > %d", preHeight)
	contractTransSql := fmt.Sprintf("DELETE  FROM contract_transactions WHERE block_height > %d", preHeight)
	blockCount := storage.db.Exec(blockSql)
	transactionCount := storage.db.Exec(transactionSql)
	storage.db.Exec(receiptSql)
	storage.db.Exec(logSql)
	go storage.db.Exec(contractTransSql)

	if GetTodayStartTs(curTime).Equal(GetTodayStartTs(time.Now())) {
		storage.UpdateSysConfigValue(Blockcurblockheight, blockCount.RowsAffected, false)
		storage.UpdateSysConfigValue(Blockcurtranheight, transactionCount.RowsAffected, false)
	}
	browserlog.BrowserLog.Info("[DeleteForkblock] DeleteForkblock preHeight:", preHeight, "localHeight", localHeight)

	return err
}

func (storage *Storage) DeleteForkReward(preHeight uint64, localHeight uint64) (err error) {

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容
		}
	}()

	verifySql := fmt.Sprintf("DELETE FROM rewards WHERE reward_height > %d ", preHeight)
	storage.db.Exec(verifySql)
	browserlog.BrowserLog.Info("[DeleteForkReward] DeleteForkReward preHeight:", preHeight, "localHeight", localHeight)
	return err
}

func DeleteRewardByHeight(tx *gorm.DB, height uint64) error {

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容
		}
	}()

	verifySql := fmt.Sprintf("DELETE FROM rewards WHERE block_height = %d ", height)

	browserlog.BrowserLog.Info("[DeleteRewardByHeight] DeleteRewardByHeight Height:", height)
	fmt.Println("DeleteRewardByHeight,", height)
	return tx.Exec(verifySql).Error

}

package mysql

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
)

const (
	Blockrewardtophight   = "block_reward.top_block_height"
	Blocktophight         = "block.top_block_height"
	GroupTopHeight        = "group.top_group_height"
	PrepareGroupTopHeight = "group.top_prepare_group_height"
	DismissGropHeight     = "group.top_dismiss_group_height"
	LIMIT                 = 20
)

func (storage *Storage) UpdateBatchAccount(accounts []*models.Account) bool {
	//fmt.Println("[Storage] add log ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return false
	}

	for i := 0; i < len(accounts); i++ {
		if accounts[i] != nil {
			storage.UpdateObject(&accounts[i])
		}
	}
	return true
}

func (storage *Storage) AddBatchAccount(accounts []*models.Account) bool {
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

func (storage *Storage) GetAccountById(address string) []*models.Account {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.Account, 0, 0)
	storage.db.Where("address = ? ", address).Find(&accounts)
	return accounts
}

func (storage *Storage) GetAccountByMaxPrimaryId(maxid uint64) []*models.Account {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.Account, LIMIT, LIMIT)
	storage.db.Where("id > ? ", maxid).Limit(LIMIT).Find(&accounts)
	return accounts
}

func (storage *Storage) GetAccountByPage(page uint64) []*models.Account {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.Account, LIMIT, LIMIT)
	storage.db.Offset((page) * LIMIT).Limit(LIMIT).Find(&accounts)
	return accounts
}

func (storage *Storage) AddBlockRewardMysqlTransaction(accounts []*models.Account) bool {
	if storage.db == nil {
		return false
	}
	tx := storage.db.Begin()
	for _, account := range accounts {
		if account.Address == "" {
			continue
		}
		if !errors(tx.Model(&account).
			Where("address = ?", account.Address).
			Updates(map[string]interface{}{"rewards": gorm.Expr("rewards + ?", account.Rewards)}).Error) {
			tx.Rollback()
			return false
		}
	}
	if !increwardBlockheightTosys(tx) {
		return false

	}
	tx.Commit()
	return true
}

func increwardBlockheightTosys(tx *gorm.DB) bool {

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
		fmt.Println("update addblockreward error", error)
		return false
	}
	return true

}

func (storage *Storage) AddBlockHeightSystemconfig(sys *models.Sys) bool {
	hight := storage.TopGroupHeight()
	if hight > 0 {
		storage.db.Model(&sys).UpdateColumn("value", gorm.Expr("value + ?", 1))
	} else {
		sys.Value = 1
		storage.AddObjects(&sys)
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

func (storage *Storage) UpdateAccountByColumn(account *models.Account, attrs map[string]interface{}) bool {
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

func (storage *Storage) TopBlockHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", Blocktophight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value
	}
	return 0
}

func (storage *Storage) TopGroupHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", GroupTopHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value
	}
	return 0
}

func (storage *Storage) TopPrepareGroupHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", PrepareGroupTopHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value
	}
	return 0
}

func (storage *Storage) TopDismissGroupHeight() uint64 {
	if storage.db == nil {
		return 0
	}
	sys := make([]models.Sys, 0, 1)
	storage.db.Limit(1).Where("variable = ?", DismissGropHeight).Find(&sys)
	if len(sys) > 0 {
		//storage.topBlockHigh = sys[0].Value
		return sys[0].Value
	}
	return 0
}

func (storage *Storage) GetDataByColumn(table interface{}, column string, value interface{}) interface{} {
	if storage.db == nil {
		return nil
	}
	storage.db.Find(&table).Pluck(column, &value)

	return value
}

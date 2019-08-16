package mysql

import (
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
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
			storage.AddObjects(&accounts[i])
		}
	}
	return true
}

func (storage *Storage) GetAccountById(id string) []*models.Account {
	//fmt.Println("[Storage] add Verification ")
	if storage.db == nil {
		fmt.Println("[Storage] storage.db == nil")
		return nil
	}
	accounts := make([]*models.Account, 1, 1)
	storage.db.Where("id = ? ", id).Find(&accounts)
	return accounts
}

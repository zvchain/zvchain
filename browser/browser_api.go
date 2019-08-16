package browser

import (
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/core"
	"time"
)

const checkInterval = time.Second * 5

var AddressCacheList map[string]uint64

type TablMmanagement struct {
	blockHeight uint64
	storage     *mysql.Storage //待迁移

	isFetchingBlocks bool
}

func NewTablMmanagement(dbAddr string, dbPort int, dbUser string, dbPassword string, rpcAddr string, rpcPort int, reset bool) *TablMmanagement {
	tablMmanagement := &TablMmanagement{}
	tablMmanagement.blockHeight = uint64(0)

	tablMmanagement.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, rpcAddr, rpcPort, reset)

	return tablMmanagement
}

func (tm *TablMmanagement) loop() {
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

func (tm *TablMmanagement) fetchBlocks() {
	if tm.isFetchingBlocks {
		return
	}
	tm.isFetchingBlocks = true

	chain := core.BlockChainImpl
	block := chain.QueryBlockCeil(tm.blockHeight)

	AddressCacheList = make(map[string]uint64)
	for _, tx := range block.Transactions {
		if _, exists := AddressCacheList[tx.Source.Hex()]; exists {
			AddressCacheList[tx.Source.Hex()] += 1
		} else {
			AddressCacheList[tx.Source.Hex()] = 1
		}
	}
	tm.blockHeight = block.Header.Height + 1

	accounts := &models.Account{}
	for address, totalTx := range AddressCacheList {
		accounts.Address = address
		accounts.TotalTransaction = totalTx
		if tm.storage.AddObjects(accounts) {
			go tm.fetchBlocks()
		}
	}

	tm.isFetchingBlocks = false

}

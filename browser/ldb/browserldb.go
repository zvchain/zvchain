package ldb

import (
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
)

type BrowserConfig struct {
	dbFile           string
	contractTransfer string
	tokenSetdata     string
}

var ContractTransfer *tasdb.PrefixedDatabase
var TokenSetdata *tasdb.PrefixedDatabase

type BrowserLdb struct {
	config *BrowserConfig
}

func getBrowserConfig() *BrowserConfig {
	return &BrowserConfig{
		dbFile:           common.GlobalConf.GetString("chain", "db_blocks", "browserd_b"),
		contractTransfer: "tr",
		tokenSetdata:     "set",
	}
}

func InitBrowserdb() {
	browser := &BrowserLdb{
		config: getBrowserConfig(),
	}
	ds, err := tasdb.NewDataSource(browser.config.dbFile, nil)
	if err != nil {
		browserlog.BrowserLog.Errorf("new datasource error:%v", err)
		return
	}
	ContractTransfer, err = ds.NewPrefixDatabase(browser.config.contractTransfer)
	TokenSetdata, err = ds.NewPrefixDatabase(browser.config.tokenSetdata)
}

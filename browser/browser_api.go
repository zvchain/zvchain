package browser

import (
	common2 "github.com/zvchain/zvchain/browser/common"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/core"
	"sync"
	"time"
)

const checkInterval = time.Second * 5

const (
	dismissGroup = iota
	workGroup
	prepareGroup
)

//var AddressCacheList map[string]uint64

type DBMmanagement struct {
	sync.Mutex
	blockHeight        uint64
	stakeMappingHeight uint64
	prepareGroupHeight uint64
	groupHeight        uint64
	dismissGropHeight  uint64
	storage            *mysql.Storage //待迁移

	isFetchingBlocks        int32
	isFetchingStakeMapping  int32
	isFetchingWorkGroups    bool
	isFetchingPrepareGroups bool
	isFetchingDismissGroups bool
	fetcher                 *common2.Fetcher
	mm                      *core.MinerManager
}

func NewDBMmanagement(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool, resetcrontab bool) *DBMmanagement {
	tablMmanagement := &DBMmanagement{}
	tablMmanagement.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, resetcrontab, "gzv")

	tablMmanagement.blockHeight, _ = tablMmanagement.storage.TopBlockHeight()
	if tablMmanagement.blockHeight > 0 {
		tablMmanagement.blockHeight += 1
	}
	tablMmanagement.stakeMappingHeight, _ = tablMmanagement.storage.TopStakeMappingHeight()
	tablMmanagement.groupHeight, _ = tablMmanagement.storage.TopGroupHeight()
	return tablMmanagement
}

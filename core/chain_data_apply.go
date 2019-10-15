//   Copyright (C) 2019 ZVChain
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

package core

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
)

func NewBlockChainByDB(db string) (*FullBlockChain, error) {
	notify.BUS.Publish(notify.MessageToConsole, &consoleMsg{msg: fmt.Sprintf("read chain data from %v", db)})
	chain := &FullBlockChain{
		config: &BlockChainConfig{
			dbfile:      db,
			block:       "bh",
			blockHeight: "hi",
			state:       "st",
			reward:      "nu",
			tx:          "tx",
			receipt:     "rc",
		},
		latestBlock:      nil,
		init:             true,
		isAdjusting:      false,
		ticker:           ticker.NewGlobalTicker("chain"),
		ts:               time.TSInstance,
		futureRawBlocks:  common.MustNewLRUCache(100),
		verifiedBlocks:   common.MustNewLRUCache(10),
		topRawBlocks:     common.MustNewLRUCache(20),
		newBlockMessages: common.MustNewLRUCache(100),
	}

	options := &opt.Options{
		OpenFilesCacheCapacity:        100,
		BlockCacheCapacity:            16 * opt.MiB,
		WriteBuffer:                   512 * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           4 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
		CompactionTotalSize:           16 * opt.MiB,
		BlockSize:                     64 * opt.KiB,
	}

	ds, err := tasdb.NewDataSource(chain.config.dbfile, options)
	if err != nil {
		Logger.Errorf("new datasource error:%v", err)
		return nil, err
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.txDb, err = ds.NewPrefixDatabase(chain.config.tx)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.stateDb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}

	chain.stateCache = account.NewDatabase(chain.stateDb)

	latestBH := chain.loadCurrentBlock()
	chain.latestBlock = latestBH
	return chain, nil
}

type consoleMsg struct {
	msg string
}

func (m *consoleMsg) GetRaw() []byte {
	return []byte{}
}
func (m *consoleMsg) GetData() interface{} {
	return m.msg
}

func (chain *FullBlockChain) ApplyChain(chain2 *FullBlockChain) error {
	top := chain2.Height()
	Logger.Debugf("db for apply height %v", top)
	notify.BUS.Publish(notify.MessageToConsole, &consoleMsg{msg: fmt.Sprintf("chain height at db %v", top)})
	for h := chain.Height() + 1; h <= top; h++ {
		b := chain2.QueryBlockByHeight(h)
		if b == nil {
			continue
		}
		add := chain.AddBlockOnChain("", b)
		//notify.BUS.Publish(notify.MessageToConsole, &consoleMsg{msg:fmt.Sprintf("add block %v sucess", h)})
		if add != types.AddBlockSucc && add != types.AddBlockExisted {
			Logger.Panicf("add block fail, height=%v, ret=%v", h, add)
		}
	}
	return nil
}

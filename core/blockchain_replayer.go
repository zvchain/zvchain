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
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
	"io"
	"time"
)

type replayer interface {
	Replay(provider BlockProvider, out io.Writer) error
}

// BlockProvider provides blocks of the given height range [start, end)
type BlockProvider interface {
	Provide(begin, end uint64) []*types.Block
	Height() uint64
}

func (chain *FullBlockChain) Replay(provider BlockProvider, out io.Writer) error {
	begin := chain.Height() + 1
	for {
		t := time.Now()
		blocks := provider.Provide(begin, begin+20)

		if len(blocks) == 0 {
			begin += 20
			continue
		}
		for _, b := range blocks {
			ret, err := chain.addBlockOnChain("", b)
			if ret != types.AddBlockSucc {
				return fmt.Errorf("replay block %v %v error:%v", b.Header.Hash, b.Header.Height, err)
			}
		}
		begin = blocks[len(blocks)-1].Header.Height + 1
		top := provider.Height()
		if top <= begin {
			break
		}

		cost := time.Since(t)
		bps := 20 / cost.Seconds()
		remainT := time.Duration(float64(top-begin)/bps) * time.Second

		out.Write([]byte(fmt.Sprintf("replay block %v finished, bps %v, remain %v\n", begin-1, bps, remainT.String())))
	}
	return nil
}

type localBlockProvider struct {
	dir   string
	chain *FullBlockChain
	top   uint64
}

func NewLocalBlockProvider(dir string) (BlockProvider, error) {
	config := getBlockChainConfig()
	config.dbfile = dir
	chain := &FullBlockChain{
		config:          config,
		latestBlock:     nil,
		init:            true,
		isAdjusting:     false,
		futureRawBlocks: common.MustNewLRUCache(10),
		verifiedBlocks:  common.MustNewLRUCache(10),
		topRawBlocks:    common.MustNewLRUCache(20),
	}

	options := &opt.Options{
		BlockCacheCapacity: 64 * opt.MiB,
		Filter:             filter.NewBloomFilter(10),
		ReadOnly:           true,
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

	bh := chain.queryBlockHeaderByHeightFloor(common.MaxUint64)
	if bh == nil {
		return nil, fmt.Errorf("query top block nil")
	}

	return &localBlockProvider{
		dir:   dir,
		chain: chain,
		top:   bh.Height,
	}, nil
}

func (p *localBlockProvider) Provide(begin, end uint64) []*types.Block {
	return p.chain.BatchGetBlocksBetween(begin, end)
}

func (p *localBlockProvider) Height() uint64 {
	return p.top
}

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
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
	"io"
	"time"
)

type traverser interface {
	Traverse(config *account.TraverseConfig) (bool, error)
}

func (chain *FullBlockChain) traverser(h uint64) traverser {
	db, err := chain.AccountDBAt(h)
	if err != nil {
		Logger.Errorf("get account db error at %v, err %v", h, err)
		return nil
	}
	return db.(*account.AccountDB)
}

func (chain *FullBlockChain) Traverse(height uint64, config *account.TraverseConfig) (bool, error) {
	v := chain.traverser(height)
	if v != nil {
		return v.Traverse(config)
	}
	return true, nil
}

type Replayer interface {
	Replay(provider BlockProvider, out io.Writer) error
}

// BlockProvider provides blocks of the given height range [start, end)
type BlockProvider interface {
	Provide(begin, end uint64) []*types.Block
	Height() uint64
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

type fastReplayer struct {
	blocksCh chan []*types.Block
	finishCh chan struct{}
	provider BlockProvider
	chain    *FullBlockChain
	out      io.Writer
}

func NewFastReplayer(bp BlockProvider, chain *FullBlockChain, out io.Writer) Replayer {
	return &fastReplayer{
		blocksCh: make(chan []*types.Block, 1),
		finishCh: make(chan struct{}),
		provider: bp,
		chain:    chain,
		out:      out,
	}
}

func (fr *fastReplayer) produce(begin uint64) {
	const step = 200
	for {
		blocks := fr.provider.Provide(begin, begin+step)

		if len(blocks) == 0 {
			begin += step
			continue
		}
		begin = blocks[len(blocks)-1].Header.Height + 1
		top := fr.provider.Height()

		fr.blocksCh <- blocks

		if top <= begin {
			fr.finishCh <- struct{}{}
			break
		}
	}
}

func (fr *fastReplayer) consume() error {
	begin := time.Now()
	cnt := 0
	for {
		select {
		case blocks := <-fr.blocksCh:
			t := time.Now()
			if len(blocks) == 0 {
				continue
			}
			for _, b := range blocks {
				ret, err := fr.chain.addBlockOnChain("", b)
				if ret != types.AddBlockSucc {
					return fmt.Errorf("consume block %v %v error:%v", b.Header.Hash, b.Header.Height, err)
				}
			}
			cnt += len(blocks)
			cost := time.Since(t)
			bps := float64(len(blocks)) / cost.Seconds()
			top := fr.provider.Height()
			last := blocks[len(blocks)-1].Header.Height + 1
			remainT := time.Duration(float64(top-last)/bps) * time.Second

			fr.out.Write([]byte(fmt.Sprintf("replay block %v finished, bps %v, remain %v\n", last-1, bps, remainT.String())))
		case <-fr.finishCh:
			fr.out.Write([]byte(fmt.Sprintf("replay total %v blocks finished, cost %v\n", cnt, time.Since(begin).String())))
			break
		}
	}
	return nil
}

func (fr *fastReplayer) Replay(provider BlockProvider, out io.Writer) error {
	fr.out.Write([]byte(fmt.Sprintf("source chain height %v\n", fr.provider.Height())))
	go fr.produce(fr.chain.Height() + 1)
	return fr.consume()
}

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
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
)

type mOperation interface {
	ParseTransaction() error
	Operation() error
}

type operContext struct {
	accountdb vm.AccountDB
	tx        *types.Transaction
}

func newOperContext(db vm.AccountDB, tx *types.Transaction) *operContext {
	return &operContext{
		accountdb: db,
		tx:        tx,
	}
}

func tx2Miner(tx *types.Transaction) (*types.Miner, error) {
	if tx.Data == nil {
		return nil, fmt.Errorf("tx data is nil:%v", tx.Hash.Hex())
	}
	data := common.FromHex(string(tx.Data))
	var miner = new(types.Miner)
	err := msgpack.Unmarshal(data, miner)
	if err != nil {
		return nil, err
	}
	miner.ID = tx.Source.Bytes()
	return miner, nil
}

type applyOper struct {
	ctx   *operContext
	miner *types.Miner
}

func (op *applyOper) ParseTransaction() error {
	m, err := tx2Miner(op.ctx.tx)
	if err != nil {
		return err
	}
	op.miner = m
	return nil
}

func (op *applyOper) Operation() error {

}

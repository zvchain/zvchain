//   Copyright (C) 2018 ZVChain
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
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
)

func (chain *FullBlockChain) initMessageHandler() {
	notify.BUS.Subscribe(notify.BlockAddSucc, chain.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.NewBlock, chain.newBlockHandler)
}

func (chain *FullBlockChain) newBlockHandler(msg notify.Message) error{
	m := notify.AsDefault(msg)

	source := m.Source()
	block, e := types.UnMarshalBlock(m.Body())
	if e != nil {
		err := fmt.Errorf("UnMarshal block error:%s", e.Error())
		Logger.Error(err)
		return err
	}

	Logger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions))
	chain.AddBlockOnChain(source, block)
	log.ELKLogger.WithFields(logrus.Fields{
		"height": block.Header.Height,
		"blockHash": block.Header.Hash.Hex(),
		"blockTime": block.Header.CurTime.String(),
	}).Debug("AddBlockOnChain")
	return nil
}

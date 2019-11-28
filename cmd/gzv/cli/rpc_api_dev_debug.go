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

package cli

import (
	"fmt"
	"strings"

	"github.com/zvchain/zvchain/log"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
)

type rewardTxHash struct {
	TxHash, BlockHash common.Hash
}

func (api *RpcDevImpl) DebugGetTxs(limit int) ([]string, error) {
	txs := core.BlockChainImpl.GetTransactionPool().GetReceived()

	hashs := make([]string, 0)
	for _, tx := range txs {
		hashs = append(hashs, tx.Hash.Hex())
		if len(hashs) >= limit {
			break
		}
	}
	return hashs, nil
}

func (api *RpcDevImpl) DebugGetRewardTxs(limit int) ([]*rewardTxHash, error) {
	txs := core.BlockChainImpl.GetTransactionPool().GetRewardTxs()

	hashs := make([]*rewardTxHash, 0)
	for _, tx := range txs {
		btx := &rewardTxHash{
			TxHash:    tx.Hash,
			BlockHash: common.BytesToHash(tx.Data),
		}
		hashs = append(hashs, btx)
		if len(hashs) >= limit {
			break
		}
	}
	return hashs, nil
}

func (api *RpcDevImpl) DebugGetRawTx(hash string) (*Transaction, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return nil, fmt.Errorf("wrong param format")
	}
	tx := core.BlockChainImpl.GetTransactionByHash(false, common.HexToHash(hash))

	if tx != nil {
		trans := convertTransaction(tx)
		return trans, nil
	}
	return nil, nil
}

func (api *RpcDevImpl) DebugGetDbProp(propName string) (string, error) {
	return core.BlockChainImpl.GetProperty(propName)
}

// DebugChaindbCompact starts a compaction with given range. if both start and limit are empty, it will start a full compaction.
func (api *RpcDevImpl) DebugChaindbCompact(start string, limit string) error {
	var (
		startByte []byte
		limitByte []byte
	)
	if len(start) > 0 {
		startByte = []byte(start)
	}
	if len(limit) > 0 {
		limitByte = []byte(limit)
	}

	log.DefaultLogger.Info("Compacting chain database:", "range", fmt.Sprintf("0x%0.2X-0x%0.2X", start, limit))
	if err := core.BlockChainImpl.Compact(startByte, limitByte); err != nil {
		log.DefaultLogger.Error("Database compaction failed", "err", err)
		return err
	}
	return nil
}

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
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
)

type applyFunc func()

type msgShower struct {
	ticker  *ticker.GlobalTicker
	out     io.Writer
	bchain  core.BlockChain
	gchain  *core.GroupChain
	id      []byte
	applied bool
	apply   applyFunc
}

var shower *msgShower

func initMsgShower(id []byte, apply applyFunc) {
	ii := &msgShower{
		ticker:  ticker.NewGlobalTicker("cli_ticker"),
		out:     os.Stdout,
		bchain:  core.BlockChainImpl,
		gchain:  core.GroupChainImpl,
		id:      id,
		apply:   apply,
		applied: false,
	}
	ii.ticker.RegisterPeriodicRoutine("cli_print_height", ii.showHeightRoutine, 10)
	ii.ticker.StartTickerRoutine("cli_print_height", true)

	notify.BUS.Subscribe(notify.BlockAddSucc, ii.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.BlockSync, ii.blockSync)
	notify.BUS.Subscribe(notify.GroupSync, ii.groupSync)

	shower = ii
}

func (ms *msgShower) showMsg(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Fprintf(ms.out, fmt.Sprintf("%v\n", s))
}

func (ms *msgShower) showHeightRoutine() bool {
	height := ms.bchain.Height()
	ms.showMsg("local height is %v %v", height, ms.gchain.Height())

	if ms.apply != nil && !ms.applied {
		balance := core.BlockChainImpl.GetBalance(common.BytesToAddress(ms.id))
		if balance.Int64() > common.VerifyStake {
			ms.showMsg("Balance enough! auto apply miner")
			ms.apply()
			ms.applied = true
		}
	}
	return true
}

func (ms *msgShower) txSuccess(tx common.Hash) bool {
	receipt := ms.bchain.GetTransactionPool().GetReceipt(tx)
	return receipt != nil && receipt.Status == types.ReceiptStatusSuccessful
}

func (ms *msgShower) onBlockAddSuccess(message notify.Message) {
	b := message.GetData().(*types.Block)
	if bytes.Equal(b.Header.Castor, ms.id) {
		ms.showMsg("congratulations, you mined block height %v success!", b.Header.Height)
	}
	if b.Transactions != nil && len(b.Transactions) > 0 {
		for _, tx := range b.Transactions {
			switch tx.Type {
			case types.TransactionTypeBonus:
				_, ids, blockHash, value, err := ms.bchain.GetBonusManager().ParseBonusTransaction(tx)
				if err != nil {
					ms.showMsg("failed to parse bonus transaction %s", err)
					continue
				}
				for _, id := range ids {
					if bytes.Equal(id, ms.id) {
						ms.showMsg("congratulations, you verified block hash %v success, bonus %v TAS", blockHash.Hex(), common.RA2TAS(value.Uint64()))
						break
					}
				}
			case types.TransactionTypeMinerApply:
				if bytes.Equal(tx.Source.Bytes(), ms.id) && ms.txSuccess(tx.Hash) {
					miner := core.MinerManagerImpl.Transaction2Miner(tx)
					role := "proposer"
					if miner.Type == types.MinerTypeVerify {
						role = "verifier"
					}
					ms.showMsg("congratulations to you on becoming a %v at height %v, start mining", role, b.Header.Height)
				}
			case types.TransactionTypeMinerAbort:
				if bytes.Equal(tx.Source.Bytes(), ms.id) && ms.txSuccess(tx.Hash) {
					role := "proposer"
					if tx.Data[0] == types.MinerTypeVerify {
						role = "verifier"
					}
					ms.showMsg("abort miner role %v success at height %v, stoping mining", role, b.Header.Height)
				}
			case types.TransactionTypeMinerRefund:
				if bytes.Equal(tx.Source.Bytes(), ms.id) && ms.txSuccess(tx.Hash) {
					role := "proposer"
					if tx.Data[0] == types.MinerTypeVerify {
						role = "verifier"
					}
					ms.showMsg("refund miner role %v success at %v", role, b.Header.Height)
				}
			}
		}
	}
}

func (ms *msgShower) blockSync(message notify.Message) {
	cand := message.GetData().(*core.SyncCandidateInfo)
	ms.showMsg("sync block from %v[height=%v], localHeight=%v, reqHeight %v", cand.Candidate, cand.CandidateHeight, core.BlockChainImpl.Height(), cand.ReqHeight)
}

func (ms *msgShower) groupSync(message notify.Message) {
	cand := message.GetData().(*core.SyncCandidateInfo)
	ms.showMsg("sync group from %v[height=%v], localHeight=%v, reqHeight %v", cand.Candidate, cand.CandidateHeight, core.GroupChainImpl.Height(), cand.ReqHeight)
}

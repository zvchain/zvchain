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
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/time"
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
	ticker      *ticker.GlobalTicker
	out         io.Writer
	bchain      types.BlockChain
	groupReader groupInfoReader
	id          []byte
	applied     bool
	apply       applyFunc
}

var shower *msgShower

func initMsgShower(id []byte, apply applyFunc) {
	ii := &msgShower{
		ticker:      ticker.NewGlobalTicker("cli_ticker"),
		out:         os.Stdout,
		bchain:      core.BlockChainImpl,
		id:          id,
		apply:       apply,
		applied:     false,
		groupReader: getGroupReader(),
	}
	ii.ticker.RegisterPeriodicRoutine("cli_print_height", ii.showHeightRoutine, 10)
	ii.ticker.StartTickerRoutine("cli_print_height", true)

	notify.BUS.Subscribe(notify.BlockAddSucc, ii.onBlockAddSuccess)
	notify.BUS.Subscribe(notify.BlockSync, ii.blockSync)
	notify.BUS.Subscribe(notify.MessageToConsole, ii.messageToConsole)

	shower = ii
}

func (ms *msgShower) showMsg(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Fprintf(ms.out, fmt.Sprintf("%v\n", s))
}

func (ms *msgShower) showHeightRoutine() bool {
	height := ms.bchain.Height()
	ms.showMsg("local height is %v %v", height, ms.groupReader.Height())

	if ms.apply != nil && !ms.applied {
		balance := core.BlockChainImpl.GetBalance(common.BytesToAddress(ms.id))
		if balance.Uint64() >= core.MinMinerStake {
			ms.showMsg("Balance enough! auto apply miner")
			ms.apply()
			ms.applied = true
		}
	}
	return true
}

func (ms *msgShower) txSuccess(tx common.Hash) bool {
	receipt := ms.bchain.GetTransactionPool().GetReceipt(tx)
	return receipt != nil && receipt.Success()
}

func (ms *msgShower) onBlockAddSuccess(message notify.Message) error {
	b := message.GetData().(*types.Block)
	castor := common.BytesToAddress(b.Header.Castor).AddrPrefixString()
	if bytes.Equal(b.Header.Castor, ms.id) {
		log.ELKLogger.WithFields(logrus.Fields{
			"minedHeight": b.Header.Height,
			"now":         time.TSInstance.Now().UTC(),
			"logType":     "proposalLog",
			"version":     common.GtasVersion,
			"castor":      castor,
		}).Info("mined block height")
		ms.showMsg("congratulations, you mined block height %v success!", b.Header.Height)
	}
	if b.Transactions != nil && len(b.Transactions) > 0 {
		for _, tx := range b.Transactions {
			switch tx.Type {
			case types.TransactionTypeReward:
				_, ids, blockHash, _, err := ms.bchain.GetRewardManager().ParseRewardTransaction(types.NewTransaction(tx, tx.GenHash()))
				if err != nil {
					ms.showMsg("failed to parse reward transaction %s", err)
					continue
				}
				for _, id := range ids {
					if bytes.Equal(id, ms.id) {
						log.ELKLogger.WithFields(logrus.Fields{
							"verifiedHeight": b.Header.Height,
							"now":            time.TSInstance.Now().UTC(),
							"logType":        "verifyLog",
							"version":        common.GtasVersion,
						}).Info("verifyLog")
						ms.showMsg("congratulations, you verified block hash %v success, reward %v ZVC", blockHash.Hex(), common.RA2TAS(tx.Value.Uint64()))
						break
					}
				}
			case types.TransactionTypeStakeAdd:
				if bytes.Equal(tx.Source.Bytes(), ms.id) && ms.txSuccess(tx.GenHash()) {
					miner, _ := types.DecodePayload(tx.Data)
					role := "proposer"
					if types.IsVerifyRole(miner.MType) {
						role = "verifier"
					}
					ms.showMsg("congratulations to you on becoming a %v at height %v, start mining", role, b.Header.Height)
				}
			case types.TransactionTypeMinerAbort:
				if bytes.Equal(tx.Source.Bytes(), ms.id) && ms.txSuccess(tx.GenHash()) {
					role := "proposer"
					if types.IsVerifyRole(types.MinerType(tx.Data[0])) {
						role = "verifier"
					}
					ms.showMsg("abort miner role %v success at height %v, stoping mining", role, b.Header.Height)
				}
			case types.TransactionTypeStakeReduce:
				if bytes.Equal(tx.Source.Bytes(), ms.id) && ms.txSuccess(tx.GenHash()) {
					role := "proposer"
					if types.IsVerifyRole(types.MinerType(tx.Data[0])) {
						role = "verifier"
					}
					ms.showMsg("refund miner role %v success at %v", role, b.Header.Height)
				}
			}
		}
	}
	return nil
}

func (ms *msgShower) blockSync(message notify.Message) error {
	cand := message.GetData().(*core.SyncCandidateInfo)
	ms.showMsg("sync block from %v[height=%v], localHeight=%v, reqHeight %v", cand.Candidate, cand.CandidateHeight, core.BlockChainImpl.Height(), cand.ReqHeight)
	return nil
}

func (ms *msgShower) messageToConsole(message notify.Message) error {
	ms.showMsg(message.GetData().(string))
	return nil
}

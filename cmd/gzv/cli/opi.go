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

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

var (
	ErrPassword    = fmt.Errorf("password error")
	ErrUnlocked    = fmt.Errorf("please unlock the account first")
	ErrUnConnected = fmt.Errorf("please connect to one node first")
	ErrInternal    = fmt.Errorf("internal error")
)

type txRawData struct {
	Target    string `json:"target"`
	Value     uint64 `json:"value"`
	Gas       uint64 `json:"gas"`
	Gasprice  uint64 `json:"gasprice"`
	TxType    int    `json:"tx_type"`
	Nonce     uint64 `json:"nonce"`
	Data      []byte `json:"data"`
	Sign      string `json:"sign"`
	ExtraData []byte `json:"extra_data"`
}

func opError(err error) *Result {
	ret, _ := failResult(err.Error())
	return ret
}

func opSuccess(data interface{}) *Result {
	ret, _ := successResult(data)
	return ret
}

type MinerInfo struct {
	PK          string
	VrfPK       string
	ID          string
	Stake       uint64
	NType       byte
	ApplyHeight uint64
	AbortHeight uint64
}

func txRawToTransaction(tx *txRawData) *types.Transaction {
	var target *common.Address
	if tx.Target != "" {
		t := common.HexToAddress(tx.Target)
		target = &t
	}
	var sign []byte
	if tx.Sign != "" {
		sign = common.HexToSign(tx.Sign).Bytes()
	} else {

	}

	return &types.Transaction{
		Data:      tx.Data,
		Value:     types.NewBigInt(tx.Value),
		Nonce:     tx.Nonce,
		Target:    target,
		Type:      int8(tx.TxType),
		GasLimit:  types.NewBigInt(tx.Gas),
		GasPrice:  types.NewBigInt(tx.Gasprice),
		Sign:      sign,
		ExtraData: tx.ExtraData,
	}
}

type accountOp interface {
	NewAccount(password string, miner bool) *Result

	AccountList() *Result

	Lock(addr string) *Result

	UnLock(addr string, password string, duration uint) *Result

	AccountInfo() *Result

	DeleteAccount() *Result

	NewAccountByImportKey(key string, password string, miner bool) *Result

	ExportKey(addr string) *Result

	Close()
}

type chainOp interface {
	// Connect connect node by ip and port
	Connect(ip string, port int) error
	// Endpoint returns current connected ip and port
	Endpoint() string
	// SendRaw send transaction to connected node
	SendRaw(tx *txRawData) *Result
	// Balance query Balance by address
	Balance(addr string) *Result
	// Nonce query Balance by address
	Nonce(addr string) *Result
	// MinerInfo query miner info by address
	MinerInfo(addr string, detail string) *Result

	ApplyGuardMiner(gas, gasprice uint64) *Result

	CancelGuard(addr string,gas, gasprice uint64) *Result

	VoteMinerPool(address string,gas, gasprice uint64)*Result

	BlockHeight() *Result

	GroupHeight() *Result

	StakeAdd(target string, mtype int, value uint64, gas, gasprice uint64) *Result

	MinerAbort(mtype int, gas, gasprice uint64, force bool) *Result

	StakeRefund(target string, mtype int, gas, gasprice uint64) *Result

	StakeReduce(target string, mtype int, value, gas, gasprice uint64) *Result

	TxInfo(hash string) *Result

	BlockByHash(hash string) *Result

	BlockByHeight(h uint64) *Result

	ViewContract(addr string) *Result

	TxReceipt(hash string) *Result

	GroupCheck(addr string) *Result
}

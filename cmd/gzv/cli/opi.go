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

type TxRawData struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Value     uint64 `json:"value"`
	GasLimit  uint64 `json:"gas_limit"`
	GasPrice  uint64 `json:"gas_price"`
	TxType    int    `json:"type"`
	Nonce     uint64 `json:"nonce"`
	Data      []byte `json:"data"`
	Sign      string `json:"sign"`
	ExtraData []byte `json:"extra_data"`
}

func opErrorRes(err error) *ErrorResult {
	return failErrResult(err.Error())
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

func txRawToTransaction(tx *TxRawData) *types.Transaction {
	var target *common.Address
	if tx.Target != "" {
		t := common.StringToAddress(tx.Target)
		target = &t
	}
	src := common.StringToAddress(tx.Source)
	var sign []byte
	if tx.Sign != "" {
		sign = common.HexToSign(tx.Sign).Bytes()
	}

	raw := &types.RawTransaction{
		Data:      tx.Data,
		Value:     types.NewBigInt(tx.Value),
		Nonce:     tx.Nonce,
		Target:    target,
		Type:      int8(tx.TxType),
		GasLimit:  types.NewBigInt(tx.GasLimit),
		GasPrice:  types.NewBigInt(tx.GasPrice),
		Sign:      sign,
		ExtraData: tx.ExtraData,
		Source:    &src,
	}
	return &types.Transaction{RawTransaction: raw, Hash: raw.GenHash()}
}

type accountOp interface {
	NewAccount(password string, miner bool) (string, error)

	AccountList() ([]string, error)

	Lock(addr string) error

	UnLock(addr string, password string, duration uint) error

	AccountInfo() (*Account, error)

	DeleteAccount() (string, error)

	NewAccountByImportKey(key string, password string, miner bool) (string, error)

	ExportKey(addr string) (string, error)

	Close()
}

type chainOp interface {
	// Connect connect node by ip and port
	Connect(ip string, port int) error
	// Endpoint returns current connected ip and port
	Endpoint() string
	// SendRaw send transaction to connected node
	SendRaw(tx *TxRawData) *RPCResObjCmd
	// Balance query Balance by address
	Balance(addr string) *RPCResObjCmd
	// Nonce query Balance by address
	Nonce(addr string) *RPCResObjCmd
	// MinerInfo query miner info by address
	MinerInfo(addr string, detail string) *RPCResObjCmd

	BlockHeight() *RPCResObjCmd

	MinerPoolInfo(addr string) *RPCResObjCmd

	TicketsInfo(addr string) *RPCResObjCmd

	ApplyGuardMiner(gas, gasprice uint64) *RPCResObjCmd

	VoteMinerPool(address string, gas, gasprice uint64) *RPCResObjCmd

	ChangeFundGuardMode(mode int, gas, gasprice uint64) *RPCResObjCmd

	GroupHeight() *RPCResObjCmd

	StakeAdd(target string, mtype int, value uint64, gas, gasprice uint64) *RPCResObjCmd

	MinerAbort(mtype int, gas, gasprice uint64, force bool) *RPCResObjCmd

	StakeRefund(target string, mtype int, gas, gasprice uint64) *RPCResObjCmd

	StakeReduce(target string, mtype int, value, gas, gasprice uint64) *RPCResObjCmd

	TxInfo(hash string) *RPCResObjCmd

	BlockByHash(hash string) *RPCResObjCmd

	BlockByHeight(h uint64) *RPCResObjCmd

	ViewContract(addr string) *RPCResObjCmd

	TxReceipt(hash string) *RPCResObjCmd

	GroupCheck(addr string) *RPCResObjCmd
}

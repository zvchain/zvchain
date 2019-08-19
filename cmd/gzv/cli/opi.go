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

type MinerInfo struct {
	PK          string
	VrfPK       string
	ID          string
	Stake       uint64
	NType       byte
	ApplyHeight uint64
	AbortHeight uint64
}

func TxRawToTransaction(tx *TxRawData) *types.Transaction {
	var target *common.Address
	if tx.Target != "" {
		t := common.StringToAddress(tx.Target)
		target = &t
	}
	var sign []byte
	if tx.Sign != "" {
		sign = common.HexToSign(tx.Sign).Bytes()
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

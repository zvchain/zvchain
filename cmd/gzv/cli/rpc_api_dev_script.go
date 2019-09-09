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
)

func (api *RpcDevImpl) ScriptTransferTx(privateKey string, from string, to string, amount uint64, nonce uint64, txType int, gasPrice uint64) (string, error) {
	return api.TxUnSafe(privateKey, to, amount, gasPrice, gasPrice, nonce, txType, "")
}

// TxUnSafe sends a transaction by submitting the privateKey.
// It is not safe for users, used for testing purpose
func (api *RpcDevImpl) TxUnSafe(privateKey, target string, value, gas, gasprice, nonce uint64, txType int, data string) (string, error) {
	sk := common.HexToSecKey(privateKey)
	if sk == nil {
		return "", fmt.Errorf("parse private key fail:%v", privateKey)
	}
	src := sk.GetPubKey().GetAddress()
	txRaw := &TxRawData{
		Source:   src.AddrPrefixString(),
		Target:   target,
		Value:    common.TAS2RA(value),
		GasLimit: gas,
		GasPrice: gasprice,
		Nonce:    nonce,
		TxType:   txType,
		Data:     []byte(data),
	}

	trans := txRawToTransaction(txRaw)
	sign, err := sk.Sign(trans.Hash.Bytes())
	if err != nil {
		return "", err
	}
	trans.Sign = sign.Bytes()

	if err := sendTransaction(trans); err != nil {
		return "", nil
	}
	return trans.Hash.Hex(), nil
}

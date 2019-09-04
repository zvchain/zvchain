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
	"encoding/json"
	"fmt"

	"github.com/zvchain/zvchain/cmd/gzv/rpc"
	"github.com/zvchain/zvchain/common"
)

type WalletServer struct {
	Port int
	aop  accountOp
}

func NewWalletServer(port int, aop accountOp) *WalletServer {
	ws := &WalletServer{
		Port: port,
		aop:  aop,
	}
	return ws
}

func (ws *WalletServer) Start() error {
	if ws.Port <= 0 {
		return fmt.Errorf("please input the rpcport")
	}
	apis := []rpc.API{
		{Namespace: "GZVWallet", Version: "1", Service: ws, Public: true},
	}
	host := fmt.Sprintf("127.0.0.1:%d", ws.Port)
	err := startHTTP(host, apis, []string{}, []string{}, []string{})
	if err == nil {
		fmt.Printf("Wallet RPC serving on http://%s\n", host)
		return nil
	}
	return err
}

func (ws *WalletServer) SignData(source, target, unlockPassword string, value uint64, gas uint64, gaspriceStr string, txType int, nonce uint64, data string) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	gp, err := common.ParseCoin(gaspriceStr)
	if err != nil {
		res.Error = opErrorRes(fmt.Errorf("%v:%v, correct example: 100RA,100kRA,1mRA,1ZVC", err, gaspriceStr))
		return res
	}
	txRaw := &TxRawData{
		Target:   target,
		Value:    value,
		GasLimit: gas,
		GasPrice: gp,
		TxType:   txType,
		Nonce:    nonce,
		Data:     []byte(data),
	}

	err = ws.aop.UnLock(source, unlockPassword, 10)
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	aci, err := ws.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}

	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		res.Error = opErrorRes(fmt.Errorf("privatekey or pubkey error"))
		return res
	}
	sourceAddr := pubkey.GetAddress()
	if sourceAddr.AddrPrefixString() != aci.Address {
		res.Error = opErrorRes(fmt.Errorf("address error"))
		return res
	}

	tranx := txRawToTransaction(txRaw)
	sign, err := privateKey.Sign(tranx.Hash.Bytes())
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	tranx.Sign = sign.Bytes()
	txRaw.Sign = sign.Hex()

	txMsg := RawMessage{}
	txMsg, err = json.Marshal(txRaw)
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	res.Result = txMsg
	return res
}

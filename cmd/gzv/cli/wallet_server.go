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

	"github.com/zvchain/zvchain/cmd/gzv/rpc"
	"github.com/zvchain/zvchain/common"
)

type WalletServer struct {
	Host string
	Port int
	aop  accountOp
}

func NewWalletServer(host string, port int, aop accountOp) *WalletServer {
	ws := &WalletServer{
		Host: host,
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
		{Namespace: "GzvWallet", Version: "1", Service: ws, Public: true},
	}
	host := fmt.Sprintf("%s:%d", ws.Host, ws.Port)
	err := startHTTP(host, apis, []string{}, []string{}, []string{})
	if err == nil {
		fmt.Printf("Wallet RPC serving on http://%s\n", host)
		return nil
	}
	return err
}

func (ws *WalletServer) SignTransaction(txRaw *TxRawData, unlockPassword string) (string, error) {
	err := ws.aop.UnLock(txRaw.Source, unlockPassword, 10)
	if err != nil {
		return "", err
	}
	aci, err := ws.aop.AccountInfo()
	if err != nil {
		return "", err
	}

	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		return "", fmt.Errorf("privatekey or pubkey error")
	}
	sourceAddr := pubkey.GetAddress()
	if sourceAddr.AddrPrefixString() != aci.Address {
		return "", fmt.Errorf("address error")
	}

	tranx := txRawToTransaction(txRaw)
	sign, err := privateKey.Sign(tranx.Hash.Bytes())
	if err != nil {
		return "", err
	}
	return sign.Hex(), nil
}

func (ws *WalletServer) SignData(signer string, data []byte, unlockPassword string) (string, error) {
	err := ws.aop.UnLock(signer, unlockPassword, 10)
	if err != nil {
		return "", err
	}
	aci, err := ws.aop.AccountInfo()
	if err != nil {
		return "", err
	}

	privateKey := common.HexToSecKey(aci.Sk)

	sign, err := privateKey.Sign(data)
	if err != nil {
		return "", err
	}
	return sign.Hex(), nil
}

func (ws *WalletServer) GenHash(txRaw *TxRawData) (string, error) {
	tranx := txRawToTransaction(txRaw)
	return tranx.Hash.Hex(), nil

}

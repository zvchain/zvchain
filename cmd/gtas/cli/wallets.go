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
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
)

// Wallets contains wallets
type wallets []wallet

var mutex sync.Mutex

// Store storage wallet account
func (ws *wallets) store() {
	js, err := json.Marshal(ws)
	if err != nil {
		common.DefaultLogger.Errorf("store wallets error")
	}
	common.GlobalConf.SetString(Section, "wallets", string(js))
}

func (ws *wallets) deleteWallet(key string) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, v := range *ws {
		if v.Address == key || v.PrivateKey == key {
			*ws = append((*ws)[:i], (*ws)[i+1:]...)
			break
		}
	}
	ws.store()
}

// newWallet create a new wallet and store it in the config file
func (ws *wallets) newWallet() (privKeyStr, walletAddress string, err error) {
	mutex.Lock()
	defer mutex.Unlock()
	priv, err := common.GenerateKey("")
	if err != nil {
		return "", "", err
	}
	pub := priv.GetPubKey()
	address := pub.GetAddress()
	privKeyStr, walletAddress = pub.Hex(), address.Hex()
	return
}

func (ws *wallets) getBalance(account string) (float64, error) {
	if account == "" && len(walletManager) > 0 {
		account = walletManager[0].Address
	}
	balance := core.BlockChainImpl.GetBalance(common.HexToAddress(account))

	return common.RA2TAS(balance.Uint64()), nil
}

func newWallets() wallets {
	var ws wallets
	s := common.GlobalConf.GetString(Section, "wallets", "")
	if s == "" {
		return ws
	}
	err := json.Unmarshal([]byte(s), &ws)
	if err != nil {
		common.DefaultLogger.Errorf(err.Error())
	}
	return ws
}

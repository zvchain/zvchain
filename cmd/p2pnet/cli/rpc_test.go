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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/zvchain/zvchain/core"
)

var cfg = &minerConfig{
	rpcLevel:      rpcLevelDev,
	rpcAddr:       "127.0.0.1",
	rpcPort:       8101,
	super:         false,
	testMode:      true,
	natIP:         "",
	natPort:       0,
	applyRole:     "",
	keystore:      "keystore",
	enableMonitor: false,
	chainID:       1,
	password:      "123",
}

func resetDb(dbPath string) error {
	core.BlockChainImpl.(*core.FullBlockChain).Close()
	//taslog.Close()
	fmt.Println("---reset db---")
	dir, err := ioutil.ReadDir(".")
	if err != nil {
		return err
	}
	for _, d := range dir {
		if d.IsDir() && strings.HasPrefix(d.Name(), "d_") {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				return err
			}
		}
		if d.IsDir() && strings.Compare(dbPath, d.Name()) == 0 {
			os.RemoveAll(d.Name())
		}

		if d.IsDir() && strings.Compare("logs", d.Name()) == 0 {
			os.RemoveAll(d.Name())
		}
	}
	os.RemoveAll(cfg.keystore)
	return nil
}

func TestMarshalTxRawData(t *testing.T) {
	tx := &txRawData{
		Target:   "0x123",
		Value:    100000000,
		Gas:      1304,
		Gasprice: 2324,
	}
	json, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(json))

}

func TestUnmarhsalTxRawData(t *testing.T) {
	s := `{"target":"0x123","value":23,"gas":99,"gasprice":2324,"tx_type":0,"nonce":0,"data":"","sign":"","extra_data":""}`
	tx := &txRawData{}

	err := json.Unmarshal([]byte(s), tx)
	if err != nil {
		t.Fatal(err)
	}
}

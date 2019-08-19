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

const code = `

# import account

class Token(object):
    def __init__(self):
        self.name = 'Tas Token'
        self.symbol = "TAS"
        self.decimal = 3

        self.totalSupply = 100000

        self.balanceOf = TasCollectionStorage()
        self.allowance = TasCollectionStorage()

        self.balanceOf['0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019'] = self.totalSupply

        # self.owner = msg.sender

    # @register.view()
    # def symbol(self):
    #     return self.symbol

    # @regsiter.view()
    # def blanceOf(self, key):
    #     return self.blanceOf[key] + 1000W

    def _transfer(self, _from, _to, _value):
        if self.balanceOf[_to] is None:
            self.balanceOf[_to] = 0
        if self.balanceOf[_from] is None:
            self.balanceOf[_from] = 0
        # 接收账户地址是否合法
        # require(Address(_to).invalid())
        # 账户余额是否满足转账金额
        if self.balanceOf[_from] < _value:
            raise Exception('账户余额小于转账金额')
        # 检查转账金额是否合法
        if _value <= 0:
            raise Exception('转账金额必须大于等于0')
        # 转账
        self.balanceOf[_from] -= _value
        self.balanceOf[_to] += _value
        # Event.emit("Transfer", _from, _to, _value)

    @register.public(str, int)
    def transfer(self, _to, _value):
        self._transfer(msg.sender, _to, _value)

    @register.public(str, int)
    def approve(self, _spender, _valuexj):
        if _value <= 0:
            raise Exception('授权金额必须大于等于0')
        if self.allowance[msg.sender] is None:
            self.allowance[msg.sender] = TasCollectionStorage()
        self.allowance[msg.sender][_spender] = _value
        # account.eventCall('Approval', 'index', 'data')
        # Event.emit("Approval", msg.sender, _spender, _value)

    @register.public(str, str, int)
    def transfer_from(self, _from, _to, _value):
        if _value > self.allowance[_from][msg.sender]:
            raise Exception('超过授权转账额度')
        self.allowance[_from][msg.sender] -= _value
        self._transfer(_from, _to, _value)

    # def approveAndCall(self, _spender, _value, _extraData):
    #         spender = Address(_spender)
    #     if self.approve(spender, _value):
    #         spender.call("receive_approval", msg.sender, _value, this, _extraData)
    #         return True
    #     else:
    #         return False

    @register.public()
    def test(self, _value):

    @register.public(int)
    def burn(self, _value):
        if _value <= 0:
            raise Exception('燃烧金额必须大于等于0')
        if self.balanceOf[msg.sender] < _value:
            raise Exception('账户余额不足')
        self.balanceOf[msg.sender] -= _value
        self.totalSupply -= _value
        # Event.emit("Burn", msg.sender, _value)

    # def burn_from(self, _from, _value):
    #     # if _from not in self.balanceOf:
    #     #     self.balanceOf[_from] = 0
    #     #检查账户余额
    #     require(self.balanceOf[_from] >= _value)
    #     require(_value <= self.allowance[_from][msg.sender])
    #     self.balanceOf[_from] -= _value
    #     self.allowance[_from][msg.sender] -= _value
    #     self.totalSupply -= _value
    #     Event.emit("Burn", _from, _value)
    #     return True
`

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

func TestParseABI(t *testing.T) {
	abi := parseABI(code)
	fmt.Println(abi)
}

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

package tvm

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"testing"
)

const contractExample1 = `

import account

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

    @register.public(list, dict, bool, str, int)
    def big(self, a, b, c, d, e):
        print(a)

`

const contractExample2 = `

import account

class A(object):
    def __init__(self):
        pass

    @register.public()
    def t(self):
        pass
`
const abiJSON1 = `
{
    "FuncName": "big",
    "Args": [[786, 2.23, 70.2], {"12": 123}, true,"goodday", 500]
}
`

const abiJSON2 = `
{
    "FuncName": "t",
    "Args": []
}
`

const evilJSON  = `
{
    "FuncName": "balance_of",
    "Args": ["0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd\")\ntas_Token.transfer('0x123',50)\n(\""]
}
`

func TestVmTest(t *testing.T) {
	//db, _ := tasdb.NewMemDatabase()
	//statedb, _ := core.NewAccountDB(common.Hash{}, core.NewDatabase(db))

	contract := &Contract{ContractName: "test"}
	vm := NewTVM(nil, contract, "")
	vm.SetGas(9999999999999999)
	vm.ContractName = "test"
	script := `
a = 1.2
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
	script = `
eval("a = 10")
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
	script = `
exec("a = 10")
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
	script = `
with open("a.txt", "w") as f:
	f.write("a")
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
}

func TestVm(t *testing.T) {
	vm := TVM{}
	vm.Contract = &Contract{}
	vm.ContractName = "Token"
	abi := ABI{
		FuncName: "balance_of",
		Args: []interface{}{"0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd\")\ntas_Token.transfer('0x123',50)=\n(\""},

	}
	fmt.Println(vm.generateScript(abi))
}

func TestController_ExecuteAbiEval(t *testing.T) {

}

func TestTVM_VerifyABI1(t *testing.T) {
	contractAddr := common.HexToAddress("0x123")
	senderAddr := common.HexToAddress("0x456")
	contract := &Contract{
		Code:         contractExample1,
		ContractName: "Token",
		ContractAddress:&contractAddr,
	}
	vm := NewTVM(&senderAddr, contract, "")
	vm.SetGas(9999999999999999)
	var addr common.Address
	addr = common.BytesToAddress([]byte("0x123"))
	vm.ContractAddress = &addr

	abi := ABI{}
	abiJSONError := json.Unmarshal([]byte(abiJSON1), &abi)
	if abiJSONError != nil {
		t.Error("abiJSONError Unmarshall err:",abiJSONError)
	}

	msg := Msg{
		Data:[]byte{},
		Value:0,
	}
	_, result,err := vm.CreateContractInstance(msg)
	if err != nil{
		t.Error("CreateContractInstance err:",err)
	}

	//result := vm.ExecuteScriptKindFile(contract.Code)
	fmt.Println("result:",result)

	if !vm.VerifyABI(result.Abi,abi){
		t.Error("VerifyABI err")
	}
}

func TestTVM_VerifyABI2(t *testing.T) {
	contractAddr := common.HexToAddress("0x123")
	senderAddr := common.HexToAddress("0x456")
	contract := &Contract{
		Code:         contractExample2,
		ContractName: "A",
		ContractAddress:&contractAddr,
	}
	vm := NewTVM(&senderAddr, contract, "")
	vm.SetGas(9999999999999999)
	var addr common.Address
	addr = common.BytesToAddress([]byte("0x123"))
	vm.ContractAddress = &addr

	abi := ABI{}
	abiJSONError := json.Unmarshal([]byte(abiJSON2), &abi)
	if abiJSONError != nil {
		t.Error("abiJSONError Unmarshall err:",abiJSONError)
	}

	msg := Msg{
		Data:[]byte{},
		Value:0,
	}
	_, result,err := vm.CreateContractInstance(msg)
	if err != nil{
		t.Error("CreateContractInstance err:",err)
	}

	//result := vm.ExecuteScriptKindFile(contract.Code)
	fmt.Println("result:",result)

	if !vm.VerifyABI(result.Abi,abi){
		t.Error("VerifyABI err")
	}
}

func BenchmarkAdd(b *testing.B) {
	vm := NewTVM(nil, nil, "")
	vm.SetGas(9999999999999999)
	script := `
a = 1
`
	vm.ExecuteScriptVMSucceed(script)
	script = `
a += 1
`
	for i := 0; i < b.N; i++ { //use b.N for looping
		vm.ExecuteScriptVMSucceed(script)
	}
}

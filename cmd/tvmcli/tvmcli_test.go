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

package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/account"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func _deployContract(contractName string, filePath string) string {
	tvmCli := NewTvmCli()
	f, err := os.Open(filePath) //读取文件
	if err != nil {
		panic("")
	}
	defer f.Close()
	codeStr, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	contractAddress, err := tvmCli.Deploy(contractName, string(codeStr))
	if err != nil {
		fmt.Println(err)
	}
	tvmCli.DeleteTvmCli()
	return contractAddress
}

func _callContract(contractAddress string, abiJSON string) {
	tvmCli := NewTvmCli()
	tvmCli.Call(contractAddress, abiJSON)
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_Call(t *testing.T) {
	contractAddress := _deployContract("Token", "erc20.py")
	abiJson := `{
	"func_name": "balance_of",
		"args": ["zv6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`
	_callContract(contractAddress, abiJson)
}

func TestTvmCli_QueryData(t *testing.T) {
	erc20Contract := _deployContract("Token", "erc20.py")

	tvmCli := NewTvmCli()
	result := tvmCli.QueryData(erc20Contract, "name", 0)
	if result["name"] != "sZVC Token" {
		t.FailNow()
	}
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_Call_ContractCallContract(t *testing.T) {
	erc20Contract := _deployContract("Token", "erc20.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
 "func_name": "call_contract",
 "Args": ["%s","balance_of","zv6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`, erc20Contract)
	_callContract(routerContract, abiJSON)
}

func TestTvmCli_Call_ContractCallContract_2(t *testing.T) {
	receiverContract := _deployContract("Receiver", "receiver.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "func_name": "call_contract",
  "Args": ["%s","private_set_name","test"]
}`, receiverContract)
	_callContract(routerContract, abiJSON)
}

func TestTvmCli_Call_ContractCallContract_3(t *testing.T) {
	receiverContract := _deployContract("Receiver", "receiver.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "func_name": "call_contract",
  "Args": ["%s","set_name","receiver1"]
}`, receiverContract)
	_callContract(routerContract, abiJSON)

	tvmCli := NewTvmCli()
	result := tvmCli.QueryData(receiverContract, "name", 0)
	if result["name"] != "sreceiver1" {
		t.FailNow()
	}
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_Call_ContractCallContract_Error_4(t *testing.T) {
	receiverContract := _deployContract("Receiver", "receiver.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "func_name": "call_contract2",
  "Args": ["%s",3]
}`, receiverContract)
	_callContract(routerContract, abiJSON)
}

func TestTvmCli_Call_ContractCallContract_5(t *testing.T) {
	receiverContract := _deployContract("Receiver", "receiver.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "func_name": "call_contract3",
  "Args": ["%s",3]
}`, receiverContract)
	_callContract(routerContract, abiJSON)
}

func getState(cli *TvmCli) *account.AccountDB {
	stateHash := cli.settings.GetString("root", "StateHash", "")
	state, _ := account.NewAccountDB(common.HexToHash(stateHash), cli.database)
	return state
}

func TestTvmCli_Call_Transfer(t *testing.T) {
	contract := _deployContract("Transfer", "transfer.py")

	tvmCli := NewTvmCli()
	state := getState(tvmCli)
	//addr := "123"
	state.SetBalance(common.StringToAddress(contract), big.NewInt(100))
	hash, _ := state.Commit(false)
	tvmCli.database.TrieDB().Commit(hash, false)
	tvmCli.settings.SetString("root", "StateHash", hash.Hex())

	abiJson := fmt.Sprintf(`{
 "func_name": "ckeckbalance",
 "Args": ["%s"]
}`, contract)
	fmt.Println("checkbalance\t" + contract + "________")
	tvmCli.Call(contract, abiJson)

	randHash := sha256.Sum256([]byte(strconv.Itoa(int(time.Now().UnixNano()))))
	randAddr := fmt.Sprintf("zv"+"%x", string(randHash[:]))

	abiJson2 := fmt.Sprintf(`{
"func_name": "transfer",
"Args": ["%s",10]
}`, randAddr)
	fmt.Printf("%s___transfer___to___%s\n", contract, randAddr)
	tvmCli.Call(contract, abiJson2)

	abiJson3 := fmt.Sprintf(`{
"func_name": "ckeckbalance",
"Args": ["%s"]
}`, contract)
	fmt.Println("checkbalance\t" + contract + "________")
	tvmCli.Call(contract, abiJson3)

	abiJson4 := fmt.Sprintf(`{
"func_name": "ckeckbalance",
"Args": ["%s"]
}`, randAddr)
	fmt.Println("checkbalance\t" + randAddr + "________")
	tvmCli.Call(contract, abiJson4)

	abiJson5 := fmt.Sprintf(`{
"func_name": "transfer",
"Args": ["%s",10]
}`, randAddr)
	fmt.Printf("%s___transfer___to___%s\n", contract, randAddr)
	tvmCli.Call(contract, abiJson5)

	abiJson6 := fmt.Sprintf(`{
"func_name": "ckeckbalance",
"Args": ["%s"]
}`, contract)
	fmt.Println("checkbalance\t" + contract + "________")
	tvmCli.Call(contract, abiJson6)

	abiJson7 := fmt.Sprintf(`{
"func_name": "ckeckbalance",
"Args": ["%s"]
}`, randAddr)
	fmt.Println("checkbalance\t" + randAddr + "________")
	tvmCli.Call(contract, abiJson7)

	tvmCli.DeleteTvmCli()
}

func TestTvmCli_Set_Data_Error(t *testing.T) {
	contract := _deployContract("Setandget", "setdata.py")

	tvmCli := NewTvmCli()
	state := getState(tvmCli)
	key := "123"
	hash, _ := state.Commit(false)
	tvmCli.database.TrieDB().Commit(hash, false)
	tvmCli.settings.SetString("root", "StateHash", hash.Hex())

	abiJson := fmt.Sprintf(`{
 "func_name": "setdata",
 "Args": ["%s","abcde"]
}`, key)
	tvmCli.Call(contract, abiJson)

	abiJson2 := fmt.Sprintf(`{
 "func_name": "getdata",
 "Args": ["%s"]
}`, key)
	tvmCli.Call(contract, abiJson2)

	abiJson3 := fmt.Sprintf(`{
"func_name": "removedata",
"Args": ["%s"]
}`, key)
	tvmCli.Call(contract, abiJson3)

	abiJson4 := fmt.Sprintf(`{
"func_name": "getdata",
"Args": ["%s"]
}`, key)
	tvmCli.Call(contract, abiJson4)

	tvmCli.DeleteTvmCli()

}

func TestTvmCli_ExecTime(t *testing.T) {
	contractAddress := _deployContract("Max", "exectime.py")

	abiJSON := `{
	"func_name": "exec1",
		"Args": [100000]
}`
	start := time.Now()
	_callContract(contractAddress, abiJSON)
	t.Log(time.Since(start).Seconds())

	abiJSON = `{
	"func_name": "exec2",
		"Args": [100000]
}`
	start = time.Now()
	_callContract(contractAddress, abiJSON)
	t.Log(time.Since(start).Seconds())

	abiJSON = `{
	"func_name": "exec3",
		"Args": [100000]
}`
	start = time.Now()
	_callContract(contractAddress, abiJSON)
	t.Log(time.Since(start).Seconds())
}

func TestTvmCli_TestABI_Error(t *testing.T) {
	contractAddress := _deployContract("TestABI", "testabi.py")
	abiJSON := `{
	"func_name": "exec1",
		"Args": [100000]
}`
	_callContract(contractAddress, abiJSON)
}

func TestTvmCli_TestABI2(t *testing.T) {
	contractAddress := _deployContract("TestABI", "testabi2.py")
	abiJSON := `{
	"func_name": "testint",
		"Args": [1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000]
}`
	_callContract(contractAddress, abiJSON)

	abiJSON = `{
	"func_name": "teststr",
		"Args": ["1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"]
}`
	_callContract(contractAddress, abiJSON)

	tvmCli := NewTvmCli()
	result := tvmCli.QueryData(contractAddress, "count", 0)
	if len(result) > 0 {
		t.FailNow()
	}
	result = tvmCli.QueryData(contractAddress, "string", 0)
	if result["string"] != "s1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" {
		t.FailNow()
	}
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_TestBigInt(t *testing.T) {
	contractAddress := _deployContract("TestBigInt", "testbigint.py")
	abiJSON := `{
	"func_name": "save",
		"Args": []
}`
	_callContract(contractAddress, abiJSON)
	tvmCli := NewTvmCli()
	result := tvmCli.QueryData(contractAddress, "bigint", 0)
	fmt.Println(result)
	//TODO assert
	tvmCli.DeleteTvmCli()

	abiJSON = `{
	"func_name": "add",
		"Args": []
}`
	_callContract(contractAddress, abiJSON)
	tvmCli = NewTvmCli()
	result = tvmCli.QueryData(contractAddress, "bigint", 0)
	fmt.Println(result)
	//TODO assert
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_TestStorage(t *testing.T) {
	_ = _deployContract("Token", "test_storage.py")
}

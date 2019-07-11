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
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/account"
	"io"
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
	codeStr := ""
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		codeStr = fmt.Sprintf("%s%s \n", codeStr, line)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic("")
			}
		}
	}
	contractAddress := tvmCli.Deploy(contractName, codeStr)
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
	"FuncName": "balance_of",
		"Args": ["0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`
	_callContract(contractAddress, abiJson)
}

func TestTvmCli_QueryData(t *testing.T) {
	erc20Contract := _deployContract("Token", "erc20.py")

	tvmCli := NewTvmCli()
	tvmCli.QueryData(erc20Contract, "name", 0)
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_Call_ContractCallContract(t *testing.T) {
	erc20Contract := _deployContract("Token", "erc20.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "FuncName": "call_contract",
  "Args": ["%s","balance_of","0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`, erc20Contract)
	_callContract(routerContract, abiJSON)
}

func TestTvmCli_Call_ContractCallContract_2(t *testing.T) {
	receiverContract := _deployContract("Receiver", "receiver.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "FuncName": "call_contract",
  "Args": ["%s","private_set_name","test"]
}`, receiverContract)
	_callContract(routerContract, abiJSON)
}

func TestTvmCli_Call_ContractCallContract_3(t *testing.T) {
	receiverContract := _deployContract("Receiver", "receiver.py")
	routerContract := _deployContract("Router", "router.py")

	abiJSON := fmt.Sprintf(`{
  "FuncName": "call_contract",
  "Args": ["%s","set_name","test"]
}`, receiverContract)
	_callContract(routerContract, abiJSON)

	tvmCli := NewTvmCli()
	tvmCli.QueryData(routerContract, "name", 0)
	tvmCli.QueryData(receiverContract, "name", 0)
	tvmCli.DeleteTvmCli()
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
	state.SetBalance(common.HexToAddress(contract), big.NewInt(100))
	hash, _ := state.Commit(false)
	tvmCli.database.TrieDB().Commit(hash, false)
	tvmCli.settings.SetString("root", "StateHash", hash.Hex())

	abiJson := fmt.Sprintf(`{
 "FuncName": "ckeckbalance",
 "Args": ["%s"]
}`, contract)
	fmt.Println("checkbalance\t" + contract + "________")
	tvmCli.Call(contract, abiJson)

	randHash := sha256.Sum256([]byte(strconv.Itoa(int(time.Now().UnixNano()))))
	randAddr := fmt.Sprintf("0x"+"%x", string(randHash[:]))

	abiJson2 := fmt.Sprintf(`{
"FuncName": "transfer",
"Args": ["%s",10]
}`, randAddr)
	fmt.Printf("%s___transfer___to___%s\n", contract, randAddr)
	tvmCli.Call(contract, abiJson2)

	abiJson3 := fmt.Sprintf(`{
"FuncName": "ckeckbalance",
"Args": ["%s"]
}`, contract)
	fmt.Println("checkbalance\t" + contract + "________")
	tvmCli.Call(contract, abiJson3)

	abiJson4 := fmt.Sprintf(`{
"FuncName": "ckeckbalance",
"Args": ["%s"]
}`, randAddr)
	fmt.Println("checkbalance\t" + randAddr + "________")
	tvmCli.Call(contract, abiJson4)

	abiJson5 := fmt.Sprintf(`{
"FuncName": "transfer",
"Args": ["%s",10]
}`, randAddr)
	fmt.Printf("%s___transfer___to___%s\n", contract, randAddr)
	tvmCli.Call(contract, abiJson5)

	abiJson6 := fmt.Sprintf(`{
"FuncName": "ckeckbalance",
"Args": ["%s"]
}`, contract)
	fmt.Println("checkbalance\t" + contract + "________")
	tvmCli.Call(contract, abiJson6)

	abiJson7 := fmt.Sprintf(`{
"FuncName": "ckeckbalance",
"Args": ["%s"]
}`, randAddr)
	fmt.Println("checkbalance\t" + randAddr + "________")
	tvmCli.Call(contract, abiJson7)

	tvmCli.DeleteTvmCli()
}

func TestTvmCli_Set_Data(t *testing.T) {
	contract := _deployContract("Setandget", "setdata.py")

	tvmCli := NewTvmCli()
	state := getState(tvmCli)
	key := "123"
	hash, _ := state.Commit(false)
	tvmCli.database.TrieDB().Commit(hash, false)
	tvmCli.settings.SetString("root", "StateHash", hash.Hex())

	abiJson := fmt.Sprintf(`{
 "FuncName": "setdata",
 "Args": ["%s","abcde"]
}`, key)
	tvmCli.Call(contract, abiJson)

	abiJson2 := fmt.Sprintf(`{
 "FuncName": "getdata",
 "Args": ["%s"]
}`, key)
	tvmCli.Call(contract, abiJson2)

	abiJson3 := fmt.Sprintf(`{
"FuncName": "removedata",
"Args": ["%s"]
}`, key)
	tvmCli.Call(contract, abiJson3)

	abiJson4 := fmt.Sprintf(`{
"FuncName": "getdata",
"Args": ["%s"]
}`, key)
	tvmCli.Call(contract, abiJson4)

	tvmCli.DeleteTvmCli()

}

func TestTvmCli_ExecTime(t *testing.T) {
	contractAddress := _deployContract("Max", "exectime.py")

	abiJSON := `{
	"FuncName": "exec1",
		"Args": [100000]
}`
	start := time.Now()
	_callContract(contractAddress, abiJSON)
	t.Log(time.Since(start).Seconds())

	abiJSON = `{
	"FuncName": "exec2",
		"Args": [100000]
}`
	start = time.Now()
	_callContract(contractAddress, abiJSON)
	t.Log(time.Since(start).Seconds())

	abiJSON = `{
	"FuncName": "exec3",
		"Args": [100000]
}`
	start = time.Now()
	_callContract(contractAddress, abiJSON)
	t.Log(time.Since(start).Seconds())
}

func TestTvmCli_TestABI(t *testing.T)  {
	contractAddress := _deployContract("TestABI", "testabi.py")
	abiJSON := `{
	"FuncName": "exec1",
		"Args": [100000]
}`
	_callContract(contractAddress, abiJSON)
}

func TestTvmCli_TestABI2(t *testing.T)  {
	contractAddress := _deployContract("TestABI", "testabi2.py")
	abiJSON := `{
	"FuncName": "testint",
		"Args": [1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000]
}`
	_callContract(contractAddress, abiJSON)

	abiJSON = `{
	"FuncName": "teststr",
		"Args": ["1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"]
}`
	_callContract(contractAddress, abiJSON)

	abiJSON = `{
	"FuncName": "testutf8",
		"Args": []
}`
	_callContract(contractAddress, abiJSON)

	abiJSON = `{
	"FuncName": "testutf82",
		"Args": ["你好，世界"]
}`
	_callContract(contractAddress, abiJSON)


	tvmCli := NewTvmCli()
	tvmCli.QueryData(contractAddress, "count", 0)
	tvmCli.QueryData(contractAddress, "string", 0)
	tvmCli.QueryData(contractAddress, "utf8", 0)
	tvmCli.QueryData(contractAddress, "utf82", 0)
	tvmCli.DeleteTvmCli()
}

func TestTvmCli_TestBigInt(t *testing.T)  {
	contractAddress := _deployContract("TestBigInt", "testbigint.py")
	abiJSON := `{
	"FuncName": "save",
		"Args": []
}`
	_callContract(contractAddress, abiJSON)
	tvmCli := NewTvmCli()
	tvmCli.QueryData(contractAddress, "bigint", 0)
	tvmCli.DeleteTvmCli()

	abiJSON = `{
	"FuncName": "add",
		"Args": []
}`
	_callContract(contractAddress, abiJSON)
	tvmCli = NewTvmCli()
	tvmCli.QueryData(contractAddress, "bigint", 0)
	tvmCli.DeleteTvmCli()
}


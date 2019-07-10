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
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
	"github.com/zvchain/zvchain/tvm"
)

const (
	TransactionGasLimitMax = 500000
)

type Transaction struct {
	tvm.ControllerTransactionInterface
}

func (Transaction) GetGasLimit() uint64 { return TransactionGasLimitMax }
func (Transaction) GetValue() uint64    { return 0 }
func (Transaction) GetSource() *common.Address {
	address := common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	return &address
}
func (Transaction) GetTarget() *common.Address {
	address := common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	return &address
}
func (Transaction) GetData() []byte      { return nil }
func (Transaction) GetHash() common.Hash { return common.Hash{} }

type FakeChainReader struct {
}

func (FakeChainReader) Height() uint64 {
	return 0
}

func (FakeChainReader) QueryTopBlock() *types.BlockHeader {
	return &types.BlockHeader{}
}
func (FakeChainReader) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	return &types.BlockHeader{}
}
func (FakeChainReader) QueryBlockHeaderByHeight(height uint64) *types.BlockHeader {
	return &types.BlockHeader{}
}
func (FakeChainReader) HasBlock(hash common.Hash) bool {
	return true
}
func (FakeChainReader) HasHeight(height uint64) bool {
	return true
}

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

var (
	DefaultAccounts = [...]string{"0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd",
		"0x3eed3f4a15d238dc2ab658dcaa069a7d072437c9c86e1605ce74cd9f4730bbf2",
		"0x36ae29871aed1bc21e708c4e2f5ff7c03218f5ffcd3eeae31d94a2985143abd7",
		"0xf798010011a0f17510ce4fdea9b3e7b458392b4bb8205ead3eb818609e93746c",
		"0xcd54640ff11b6ffe601566008872c87a4f3ec01a2890404b6ce30905ee3b2137"}
)

type TvmCli struct {
	settings common.ConfManager
	db       *tasdb.LDBDatabase
	database account.AccountDatabase
}

func NewTvmCli() *TvmCli {
	tvmCli := new(TvmCli)
	tvmCli.init()
	return tvmCli
}

func (t *TvmCli) DeleteTvmCli() {
	defer t.db.Close()
}

func (t *TvmCli) init() {

	currentPath, error := filepath.Abs(filepath.Dir(os.Args[0]))
	if error != nil {
		fmt.Println(error)
		return
	}
	fmt.Println(currentPath)
	t.db, _ = tasdb.NewLDBDatabase(currentPath+"/db", nil)
	t.database = account.NewDatabase(t.db)

	if Exists(currentPath + "/settings.ini") {
		t.settings = common.NewConfINIManager(currentPath + "/settings.ini")
		//stateHash := settings.GetString("root", "StateHash", "")
		//state, error := account.NewAccountDB(common.HexToHash(stateHash), database)
		//if error != nil {
		//	fmt.Println(error)
		//	return
		//}
		//fmt.Println(stateHash)
		//fmt.Println(state.GetBalance(common.StringToAddress(defaultAccounts[0])))
	} else {
		t.settings = common.NewConfINIManager(currentPath + "/settings.ini")
		state, _ := account.NewAccountDB(common.Hash{}, t.database)
		for i := 0; i < len(DefaultAccounts); i++ {
			accountAddress := common.HexToAddress(DefaultAccounts[i])
			state.SetBalance(accountAddress, big.NewInt(200))
		}
		hash, error := state.Commit(false)
		t.database.TrieDB().Commit(hash, false)
		if error != nil {
			fmt.Println(error)
			return
		} else {
			t.settings.SetString("root", "StateHash", hash.Hex())
			fmt.Println(hash.Hex())
		}
	}
}

func (t *TvmCli) Deploy(contractName string, contractCode string) string {
	stateHash := t.settings.GetString("root", "StateHash", "")
	state, _ := account.NewAccountDB(common.HexToHash(stateHash), t.database)
	transaction := Transaction{}
	controller := tvm.NewController(state, FakeChainReader{}, &types.BlockHeader{}, transaction, 0, nil)

	nonce := state.GetNonce(*transaction.GetSource())
	contractAddress := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.GetSource()[:], common.Uint64ToByte(nonce))))
	fmt.Println("contractAddress: ", contractAddress.Hex())
	state.SetNonce(*transaction.GetSource(), nonce+1)

	contract := tvm.Contract{
		ContractName: contractName,
		Code:         contractCode,
		//ContractAddress: &contractAddress,
	}

	jsonBytes, errMarsh := json.Marshal(contract)
	if errMarsh != nil {
		fmt.Println(errMarsh)
		return ""
	}
	state.CreateAccount(contractAddress)
	state.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.Deploy(&contract)
	fmt.Println("gas: ", TransactionGasLimitMax-controller.VM.Gas())

	hash, error := state.Commit(false)
	t.database.TrieDB().Commit(hash, false)
	if error != nil {
		fmt.Println(error)
	}
	t.settings.SetString("root", "StateHash", hash.Hex())
	fmt.Println(hash.Hex())
	return contractAddress.Hex()
}

func (t *TvmCli) Call(contractAddress string, abiJSON string) {
	stateHash := t.settings.GetString("root", "StateHash", "")
	state, _ := account.NewAccountDB(common.HexToHash(stateHash), t.database)

	controller := tvm.NewController(state, FakeChainReader{}, &types.BlockHeader{}, Transaction{}, 0, nil)

	//abi := tvm.ABI{}
	//abiJsonError := json.Unmarshal([]byte(abiJSON), &abi)
	//if abiJsonError != nil{
	//	fmt.Println(abiJSON, " json.Unmarshal failed ", abiJsonError)
	//	return
	//}
	_contractAddress := common.HexToAddress(contractAddress)
	contract := tvm.LoadContract(_contractAddress)
	//fmt.Println(contract.Code)
	sender := common.HexToAddress(DefaultAccounts[0])
	executeResult, _, logs, _ := controller.ExecuteAbiEval(&sender, contract, abiJSON)
	fmt.Println("gas: ", TransactionGasLimitMax-controller.VM.Gas())
	fmt.Printf("%d logs: \n", len(logs))
	for _, log := range logs {
		fmt.Printf("		string: %s, data: %s\n", log.String(), string(log.Data))
	}
	if executeResult == nil {
		fmt.Println("ExecuteAbiEval error")
		return
	} else if executeResult.ResultType == 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		fmt.Println("error code: ", executeResult.ErrorCode, " error info: ", executeResult.Content)
	} else {
		fmt.Println("executeResult: ", executeResult.Content)
	}

	hash, error := state.Commit(false)
	t.database.TrieDB().Commit(hash, false)
	if error != nil {
		fmt.Println(error)
	}
	t.settings.SetString("root", "StateHash", hash.Hex())
	fmt.Println(hash.Hex())
}

func (t *TvmCli) ExportAbi(contractName string, contractCode string) {
	contract := tvm.Contract{
		ContractName: contractName,
		//Code: contractCode,
		//ContractAddress: &contractAddress,
	}
	vm := tvm.NewTVM(nil, &contract)
	defer func() {
		vm.DelTVM()
	}()
	str := `
class Register(object):
    def __init__(self):
        self.funcinfo = {}
        self.abiinfo = []

    def public(self , *dargs):
        def wrapper(func):
            paranametuple = func.__para__
            paraname = list(paranametuple)
            paraname.remove("self")
            paratype = []
            for i in range(len(paraname)):
                paratype.append(dargs[i])
            self.funcinfo[func.__name__] = [paraname,paratype]
            tmp = {}
            tmp["FuncName"] = func.__name__
            tmp["Args"] = paratype
            self.abiinfo.append(tmp)
            abiexport(str(self.abiinfo))

            def _wrapper(*args , **kargs):
                return func(*args, **kargs)
            return _wrapper
        return wrapper

import builtins
builtins.register = Register()
`

	err := vm.ExecuteScriptVMSucceed(str)
	if err == nil {
		result := vm.ExecuteScriptKindFile(contractCode)
		fmt.Println(result.Abi)
	} else {
		fmt.Println(err)
	}

}

func (t *TvmCli) QueryData(address string, key string, count int) {
	stateHash := t.settings.GetString("root", "StateHash", "")
	state, _ := account.NewAccountDB(common.HexToHash(stateHash), t.database)

	hexAddr := common.HexToAddress(address)
	if count == 0 {
		value := state.GetData(hexAddr, []byte(key))
		if value != nil {
			fmt.Println("key:", key, "value:", string(value))
		}
	} else {
		iter := state.DataIterator(hexAddr, []byte(key))
		if iter != nil {
			for iter.Next() {
				k := string(iter.Key[:])
				if !strings.HasPrefix(k, key) {
					continue
				}
				v := string(iter.Value[:])
				fmt.Println("key:", k, "value:", v)
				count--
				if count <= 0 {
					break
				}
			}
		}
	}
}

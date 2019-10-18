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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

var bridgeInited = false

// Controller VM Controller
type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction types.TxMessage
	AccountDB   types.AccountDB
	Reader      types.ChainReader
	VM          *TVM
	VMStack     []*TVM
	GasLeft     uint64
	mm          MinerManager
}

// MinerManager MinerManager is the interface of the miner manager
type MinerManager interface {
	ExecuteOperation(accountdb types.AccountDB, msg types.TxMessage, height uint64) (success bool, err error)
}

// NewController New a TVM controller
func NewController(accountDB types.AccountDB,
	chainReader types.ChainReader,
	header *types.BlockHeader,
	transaction types.TxMessage,
	gasUsed uint64,
	manager MinerManager) *Controller {
	if controller == nil {
		controller = &Controller{}
	}
	if transaction.GetGasLimit() < gasUsed {
		panic(fmt.Sprintf("gasLimit less than gasUsed:%v %v", transaction.GetGasLimit(), gasUsed))
	}
	controller.BlockHeader = header
	controller.Transaction = transaction
	controller.AccountDB = accountDB
	controller.Reader = chainReader
	controller.VM = nil
	controller.VMStack = make([]*TVM, 0)
	controller.GasLeft = transaction.GetGasLimit() - gasUsed
	controller.mm = manager
	return controller
}

func transactionErrorWith(result *ExecuteResult) *types.TransactionError {
	if result.ResultType == 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		if result.ErrorCode == types.TVMGasNotEnoughError {
			return types.NewTransactionError(types.TVMGasNotEnoughError, "does not have enough gas to run!")
		} else if result.ErrorCode == types.TVMCheckABIError {
			return types.NewTransactionError(types.TVMCheckABIError, result.Content)
		} else {
			return types.NewTransactionError(types.TVMExecutedError, result.Content)
		}
	}

	return nil
}

// Deploy Deploy a contract instance
func (con *Controller) Deploy(contract *Contract) (*ExecuteResult, []*types.Log, *types.TransactionError) {
	var blockHeight uint64 = 0
	if con.BlockHeader != nil {
		blockHeight = con.BlockHeader.Height
	}
	con.VM = NewTVM(con.Transaction.Operator(), contract, blockHeight)
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	con.VM.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.GetValue()}

	result := con.VM.Deploy(msg)
	transactionError := transactionErrorWith(result)
	if transactionError != nil {
		return result, nil, transactionError
	}

	return result, con.VM.Logs, nil
}

// ExecuteAbiEval Execute the contract with abi and returns result
func (con *Controller) ExecuteAbiEval(sender *common.Address, contract *Contract, abiJSON string) (*ExecuteResult, []*types.Log, *types.TransactionError) {
	var blockHeight uint64 = 0
	if con.BlockHeader != nil {
		blockHeight = con.BlockHeader.Height
	}
	con.VM = NewTVM(sender, contract, blockHeight)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	msg := Msg{Data: con.Transaction.Payload(), Value: con.Transaction.GetValue()}
	result, err := con.VM.CreateContractInstance(msg)
	if err != nil {
		return result, nil, transactionErrorWith(result)
	}
	abi := ABI{}

	decoder := json.NewDecoder(bytes.NewReader([]byte(abiJSON)))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	abiJSONError := decoder.Decode(&abi)
	if abiJSONError != nil {
		return nil, nil, types.NewTransactionError(types.TVMCheckABIError, abiJSONError.Error())
	}

	//con.VM.SetLibLine(libLen)

	result = con.VM.executeABIKindEval(abi) //execute
	transactionError := transactionErrorWith(result)
	if transactionError != nil {
		return result, nil, transactionError
	}

	return result, con.VM.Logs, nil
}

// GetGasLeft get gas left
func (con *Controller) GetGasLeft() uint64 {
	return con.GasLeft
}

func BytesToBigInt(bs []byte) *big.Int {
	res := &big.Int{}
	isNeg := false
	var data []byte
	if len(bs) < 1 {
		return nil
	}
	if bs[0] & 0x80 != 0 {
		isNeg = true
		tmp := big.NewInt(0)
		tmp.SetBytes(bs)
		tmp.Sub(tmp, big.NewInt(1))
		data = tmp.Bytes()
		for i:=0; i<len(data);i++ {
			data[i] = ^data[i]
		}
	} else {
		data = bs
	}
	res.SetBytes(data)
	if isNeg {
		res = res.Neg(res)
	}
	return res
}

func VmDataConvert(value []byte) interface{} {
	//const char DICT_FORMAT_C = 'd';
	//const char STR_FORMAT_C = 's';
	//const char INT_FORMAT_C = 'i';
	//const char SMALLINT_FORMAT_C = 'm';
	//const char LIST_FORMAT_C = 'l';
	//const char BOOL_FORMAT_C = 'b';
	//const char NONE_FORMAT_C = 'n';
	//const char BYTE_FORMAT_C = 'y';

	if value == nil || len(value) == 0 {
		return nil
	}

	if value[0] == 'm' {
		bytesBuffer := bytes.NewBuffer(value[1:])
		var x int64
		err := binary.Read(bytesBuffer, binary.LittleEndian, &x)
		if err != nil {
			return nil
		}
		return x
	} else if value[0] == 'i' {
		r := BytesToBigInt(value[1:])
		if r == nil {
			return nil
		}
		return r
	} else if value[0] == 'd' {
		return map[string]interface{}{}
	} else if value[0] == 's' {
		return string(value[1:])
	} else if value[0] == 'l' {
		return []interface{}{}
	} else if value[0] == 'b' {
		if len(value) == 2 {
			if value[1] != '0'{
				return true
			} else {
				return false
			}
		} else {
			return nil
		}
	} else if value[0] == 'n'{
		return nil
	} else if value[0] == 'y' {
		return value[1:]
	} else {
		return nil
	}
}
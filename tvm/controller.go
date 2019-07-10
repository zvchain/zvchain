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
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
)

var bridgeInited = false

// ControllerTransactionInterface ControllerTransactionInterface is the interface that match Controller
type ControllerTransactionInterface interface {
	GetGasLimit() uint64
	GetValue() uint64
	GetSource() *common.Address
	GetTarget() *common.Address
	GetData() []byte
	GetHash() common.Hash
}

// Controller VM Controller
type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction ControllerTransactionInterface
	AccountDB   vm.AccountDB
	Reader      vm.ChainReader
	VM          *TVM
	VMStack     []*TVM
	GasLeft     uint64
	mm          MinerManager
}

// MinerManager MinerManager is the interface of the miner manager
type MinerManager interface {
	ExecuteOperation(accountdb vm.AccountDB, msg vm.MinerOperationMessage, height uint64) (success bool, err error)
}

// NewController New a TVM controller
func NewController(accountDB vm.AccountDB,
	chainReader vm.ChainReader,
	header *types.BlockHeader,
	transaction ControllerTransactionInterface,
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

// Deploy Deploy a contract instance
func (con *Controller) Deploy(contract *Contract) error {
	con.VM = NewTVM(con.Transaction.GetSource(), contract)
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	con.VM.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.GetValue()}
	err := con.VM.Deploy(msg)

	if err != nil {
		return err
	}
	err = con.VM.storeData()
	if err != nil {
		return err
	}
	return nil
}

// ExecuteABI Execute the contract with abi
func (con *Controller) ExecuteABI(sender *common.Address, contract *Contract, abiJSON string) (bool, []*types.Log, *types.TransactionError) {
	con.VM = NewTVM(sender, contract)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	if con.Transaction.GetValue() > 0 {
		amount := new(big.Int).SetUint64(con.Transaction.GetValue())
		if con.AccountDB.CanTransfer(*sender, amount) {
			con.AccountDB.Transfer(*sender, *con.Transaction.GetTarget(), amount)
		} else {
			return false, nil, types.TxErrorBalanceNotEnoughErr
		}
	}

	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue()}
	libLen, result, err := con.VM.CreateContractInstance(msg)
	if err != nil {
		return false, nil, types.NewTransactionError(types.TVMExecutedError, err.Error())
	}

	abi := ABI{}
	abiJSONError := json.Unmarshal([]byte(abiJSON), &abi)
	if abiJSONError != nil {
		return false, nil, types.TxErrorABIJSONErr
	}

	if !con.VM.VerifyABI(result.Abi, abi) {
		return false, nil, types.NewTransactionError(types.SysCheckABIError, fmt.Sprintf(`
			checkABI failed. abi:%s
		`, abi.FuncName))
	}

	con.VM.SetLibLine(libLen)
	err = con.VM.executABIVMSucceed(abi) //execute
	if err != nil {
		return false, nil, types.NewTransactionError(types.TVMExecutedError, err.Error())
	}
	err = con.VM.storeData() //store
	if err != nil {
		return false, nil, types.NewTransactionError(types.TVMExecutedError, err.Error())
	}
	return true, con.VM.Logs, nil
}

// ExecuteAbiEval Execute the contract with abi and returns result
func (con *Controller) ExecuteAbiEval(sender *common.Address, contract *Contract, abiJSON string) (*ExecuteResult, bool, []*types.Log, *types.TransactionError) {
	con.VM = NewTVM(sender, contract)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	if con.Transaction.GetValue() > 0 {
		amount := big.NewInt(int64(con.Transaction.GetValue()))
		if con.AccountDB.CanTransfer(*sender, amount) {
			con.AccountDB.Transfer(*sender, *con.Transaction.GetTarget(), amount)
		} else {
			return nil, false, nil, types.TxErrorBalanceNotEnoughErr
		}
	}
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue()}
	libLen, executeResult, err := con.VM.CreateContractInstance(msg)
	if err != nil {
		return nil, false, nil, types.NewTransactionError(types.TVMExecutedError, err.Error())
	}
	abi := ABI{}
	abiJSONError := json.Unmarshal([]byte(abiJSON), &abi)
	if abiJSONError != nil {
		return nil, false, nil, types.TxErrorABIJSONErr
	}

	if !con.VM.VerifyABI(executeResult.Abi, abi) {
		return nil, false, nil, types.NewTransactionError(types.SysCheckABIError, fmt.Sprintf(`
			checkABI failed. abi:%s
		`, abi.FuncName))
	}

	con.VM.SetLibLine(libLen)
	result := con.VM.executeABIKindEval(abi) //execute
	if result.ResultType == 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		return result, false, nil, types.NewTransactionError(types.TVMExecutedError, fmt.Errorf("C RETURN_TYPE_EXCEPTION").Error())
	}
	err = con.VM.storeData() //store
	if err != nil {
		return nil, false, nil, types.NewTransactionError(types.TVMExecutedError, err.Error())
	}
	return result, true, con.VM.Logs, nil
}

// GetGasLeft get gas left
func (con *Controller) GetGasLeft() uint64 {
	return con.GasLeft
}

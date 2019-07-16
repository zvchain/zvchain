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

/*
#include <stdlib.h>
*/
import "C"
import (
	"math/big"
	"strconv"
	"unsafe"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
	"github.com/zvchain/zvchain/taslog"
)

var logger = taslog.GetLoggerByIndex(taslog.TvmConfig, strconv.FormatInt(int64(common.InstanceIndex), 10))

//export Transfer
func Transfer(toAddressStr *C.char, value *C.char) bool {
	minerAddrString := C.GoString(toAddressStr)
	if !common.ValidateAddress(minerAddrString) {
		return false
	}
	transValue, ok := big.NewInt(0).SetString(C.GoString(value), 10)
	if !ok {
		return false
	}
	contractAddr := controller.VM.ContractAddress
	toAddress := common.HexToAddress(C.GoString(toAddressStr))

	if !controller.AccountDB.CanTransfer(*contractAddr, transValue) {
		return false
	}
	controller.AccountDB.Transfer(*contractAddr, toAddress, transValue)
	return true

}

//export GetBalance
func GetBalance(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	value := controller.AccountDB.GetBalance(address)
	return C.CString(value.String())
}

//export GetData
func GetData(key *C.char) *C.char {
	//hash := common.StringToHash(C.GoString(hashC))
	address := *controller.VM.ContractAddress
	state := controller.AccountDB.GetData(address, []byte(C.GoString(key)))
	return C.CString(string(state))
}

//export SetData
func SetData(keyC *C.char, data *C.char) {
	address := *controller.VM.ContractAddress
	key := []byte(C.GoString(keyC))
	state := []byte(C.GoString(data))
	controller.AccountDB.SetData(address, key, state)
}

//export BlockHash
func BlockHash(height C.ulonglong) *C.char {
	block := controller.Reader.QueryBlockHeaderByHeight(uint64(height))
	if block == nil {
		return C.CString("0x0000000000000000000000000000000000000000000000000000000000000000")
	}
	return C.CString(block.Hash.Hex())
}

//export Number
func Number() C.ulonglong {
	return C.ulonglong(controller.BlockHeader.Height)
}

//export Timestamp
func Timestamp() C.ulonglong {
	return C.ulonglong(uint64(controller.BlockHeader.CurTime.Unix()))
}

//export TxGasLimit
func TxGasLimit() C.ulonglong {
	return C.ulonglong(controller.Transaction.GetGasLimit())
}

//export ContractCall
func ContractCall(addressC *C.char, funName *C.char, jsonParms *C.char, cResult unsafe.Pointer) {
	goResult := CallContract(C.GoString(addressC), C.GoString(funName), C.GoString(jsonParms))
	ccResult := (*C.struct__tvm_execute_result_t)(cResult)
	ccResult.result_type = C.int(goResult.ResultType)
	ccResult.error_code = C.int(goResult.ErrorCode)
	if goResult.Content != "" {
		ccResult.content = C.CString(goResult.Content)
	}
	if goResult.Abi != "" {
		ccResult.abi = C.CString(goResult.Abi)
	}
}

//export EventCall
func EventCall(eventName *C.char, data *C.char) {

	var log types.Log
	log.Topics = append(log.Topics, common.BytesToHash(common.Sha256([]byte(C.GoString(eventName)))))
	log.Index = uint(len(controller.VM.Logs))
	log.Data = []byte(C.GoString(data))
	log.TxHash = controller.Transaction.GetHash()
	log.Address = *controller.VM.ContractAddress //*(controller.Transaction.Target)
	log.BlockNumber = controller.BlockHeader.Height
	//block is running ,no blockhash this time
	// log.BlockHash = controller.BlockHeader.Hash

	controller.VM.Logs = append(controller.VM.Logs, &log)
}

//export RemoveData
func RemoveData(key *C.char) {
	address := *controller.VM.ContractAddress
	controller.AccountDB.RemoveData(address, []byte(C.GoString(key)))
}

func executeMinerOperation(msg vm.MinerOperationMessage) bool {
	success, err := controller.mm.ExecuteOperation(controller.AccountDB, msg, controller.BlockHeader.Height)
	if err != nil {
		logger.Errorf("execute operation error:%v, source:%v", err, msg.Operator().Hex())
	}
	return success
}

//export MinerStake
func MinerStake(minerAddr *C.char, _type int, cvalue *C.char) bool {
	minerAddrString := C.GoString(minerAddr)
	if !common.ValidateAddress(minerAddrString) || _type != int(types.MinerTypeProposal) {
		return false
	}
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok || value.Sign() <= 0 || value.Cmp(common.MaxBigUint64) > 0 {
		return false
	}
	mPks := &types.MinerPks{
		MType: types.MinerType(byte(_type)),
	}
	payload, err := types.EncodePayload(mPks)
	if err != nil {
		logger.Errorf("encode payload error:%v", err)
		return false
	}
	target := common.HexToAddress(minerAddrString)
	msg := &minerOpMsg{
		source:  controller.VM.ContractAddress,
		target:  &target,
		value:   value,
		payload: payload,
		typ:     types.TransactionTypeStakeAdd,
	}

	return executeMinerOperation(msg)

}

//export MinerCancelStake
func MinerCancelStake(minerAddr *C.char, _type int, cvalue *C.char) bool {
	minerAddrString := C.GoString(minerAddr)
	if !common.ValidateAddress(minerAddrString) || _type != int(types.MinerTypeProposal) {
		return false
	}
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok || value.Sign() <= 0 || value.Cmp(common.MaxBigUint64) > 0 {
		return false
	}
	payload := []byte{byte(_type)}
	target := common.HexToAddress(minerAddrString)
	msg := &minerOpMsg{
		source:  controller.VM.ContractAddress,
		target:  &target,
		value:   value,
		payload: payload,
		typ:     types.TransactionTypeStakeReduce,
	}

	return executeMinerOperation(msg)
}

//export MinerRefundStake
func MinerRefundStake(minerAddr *C.char, _type int) bool {
	minerAddrString := C.GoString(minerAddr)
	if !common.ValidateAddress(minerAddrString) || _type != int(types.MinerTypeProposal) {
		return false
	}
	payload := []byte{byte(_type)}
	target := common.HexToAddress(minerAddrString)
	msg := &minerOpMsg{
		source:  controller.VM.ContractAddress,
		target:  &target,
		payload: payload,
		typ:     types.TransactionTypeStakeRefund,
	}

	return executeMinerOperation(msg)
}

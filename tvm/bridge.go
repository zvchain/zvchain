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
	"unsafe"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

//export Transfer
func Transfer(toAddress *C.char, value *C.char) bool {
	toAddressStr := C.GoString(toAddress)
	if !common.ValidateAddress(toAddressStr) {
		return false
	}
	transValue, ok := big.NewInt(0).SetString(C.GoString(value), 10)
	if !ok {
		return false
	}
	contractAddr := controller.VM.ContractAddress
	to := common.StringToAddress(toAddressStr)

	if !controller.AccountDB.CanTransfer(*contractAddr, transValue) {
		return false
	}
	controller.AccountDB.Transfer(*contractAddr, to, transValue)
	return true

}

//export GetBalance
func GetBalance(addressC *C.char) *C.char {
	toAddressStr := C.GoString(addressC)
	if !common.ValidateAddress(toAddressStr) {
		return C.CString("0")
	}
	address := common.StringToAddress(C.GoString(addressC))
	value := controller.AccountDB.GetBalance(address)
	return C.CString(value.String())
}

//export GetData
func GetData(key *C.char, keyLen C.int, value **C.char, valueLen *C.int) {
	//hash := common.StringToHash(C.GoString(hashC))
	address := *controller.VM.ContractAddress
	state := controller.AccountDB.GetData(address, C.GoBytes(unsafe.Pointer(key), keyLen))
	if state == nil {
		*value = nil
		*valueLen = -1
	} else {
		*value = (*C.char)(C.CBytes(state))
		*valueLen = C.int(len(state))
	}
}

//export SetData
func SetData(key *C.char, kenLen C.int, value *C.char, valueLen C.int) {
	address := *controller.VM.ContractAddress
	k := C.GoBytes(unsafe.Pointer(key), kenLen)
	v := C.GoBytes(unsafe.Pointer(value), valueLen)
	controller.AccountDB.SetData(address, k, v)
}

//export BlockHash
func BlockHash(height C.ulonglong) *C.char {
	block := controller.Reader.QueryBlockHeaderByHeight(uint64(height))
	if block == nil {
		return nil
	}
	return C.CString(block.Hash.Hex())
}

//export Number
func Number() C.ulonglong {
	return C.ulonglong(controller.BlockHeader.Height)
}

//export Timestamp
func Timestamp() C.ulonglong {
	return C.ulonglong(uint64(controller.BlockHeader.CurTime.UnixMilli()))
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
}

//export EventCall
func EventCall(eventName *C.char, data *C.char, dataLen C.int) {

	var log types.Log
	log.Topic = common.BytesToHash(common.Sha256([]byte(C.GoString(eventName))))
	log.Index = uint(len(controller.VM.Logs))
	log.Data = C.GoBytes(unsafe.Pointer(data), dataLen)
	log.TxHash = controller.Transaction.GetHash()
	log.Address = *controller.VM.ContractAddress //*(controller.Transaction.Target)
	log.BlockNumber = controller.BlockHeader.Height
	//block is running ,no blockhash this time
	// log.BlockHash = controller.BlockHeader.Hash

	controller.VM.Logs = append(controller.VM.Logs, &log)
}

//export RemoveData
func RemoveData(key *C.char, kenLen C.int) {
	address := *controller.VM.ContractAddress
	k := C.GoBytes(unsafe.Pointer(key), kenLen)
	controller.AccountDB.RemoveData(address, k)
}

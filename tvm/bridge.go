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
func Transfer(toAddressStr *C.char, value *C.char) {
	transValue, ok := big.NewInt(0).SetString(C.GoString(value), 10)
	if !ok {
		return
	}
	contractAddr := controller.VM.ContractAddress
	contractValue := controller.AccountDB.GetBalance(*contractAddr)
	if contractValue.Cmp(transValue) < 0 {
		return
	}
	toAddress := common.HexToAddress(C.GoString(toAddressStr))
	controller.AccountDB.AddBalance(toAddress, transValue)
	controller.AccountDB.SubBalance(*contractAddr, transValue)
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
	value := controller.AccountDB.GetData(address, C.GoString(key))
	return C.CString(string(value))
}

//export SetData
func SetData(keyC *C.char, data *C.char) {
	address := *controller.VM.ContractAddress
	key := C.GoString(keyC)
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
	controller.AccountDB.RemoveData(address, C.GoString(key))
}

//export MinerStake
func MinerStake(minerAddr *C.char, _type int, cvalue *C.char) bool {
	ss := controller.AccountDB.Snapshot()
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok {
		return false
	}
	source := controller.VM.ContractAddress
	miner := common.HexToAddress(C.GoString(minerAddr))
	if canTransfer(controller.AccountDB, *source, value) {
		mexist := controller.mm.GetMinerByID(miner.Bytes(), byte(_type), controller.AccountDB)
		if mexist != nil &&
			controller.mm.AddStake(mexist.ID, mexist, value.Uint64(), controller.AccountDB, controller.BlockHeader.Height) &&
			controller.mm.AddStakeDetail(source.Bytes(), mexist, value.Uint64(), controller.AccountDB) {
			controller.AccountDB.SubBalance(*source, value)
			return true
		}
	}
	controller.AccountDB.RevertToSnapshot(ss)
	return false
}

//export MinerCancelStake
func MinerCancelStake(minerAddr *C.char, _type int, cvalue *C.char) bool {
	ss := controller.AccountDB.Snapshot()
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok {
		return false
	}
	source := controller.VM.ContractAddress
	miner := common.HexToAddress(C.GoString(minerAddr))
	mexist := controller.mm.GetMinerByID(miner.Bytes(), byte(_type), controller.AccountDB)
	if mexist != nil &&
		controller.mm.CancelStake(source.Bytes(), mexist, value.Uint64(), controller.AccountDB, controller.BlockHeader.Height) &&
		controller.mm.ReduceStake(mexist.ID, mexist, value.Uint64(), controller.AccountDB, controller.BlockHeader.Height) {
		return true
	}
	controller.AccountDB.RevertToSnapshot(ss)
	return false
}

//export MinerRefundStake
func MinerRefundStake(minerAddr *C.char, _type int) bool {
	var success = false
	ss := controller.AccountDB.Snapshot()
	source := controller.VM.ContractAddress
	miner := common.HexToAddress(C.GoString(minerAddr))
	mexist := controller.mm.GetMinerByID(miner.Bytes(), byte(_type), controller.AccountDB)
	height := controller.BlockHeader.Height
	if mexist != nil {
		if mexist.Type == types.MinerTypeHeavy {
			latestCancelPledgeHeight := controller.mm.GetLatestCancelStakeHeight(source.Bytes(), mexist, controller.AccountDB)
			if height > latestCancelPledgeHeight+10 || (mexist.Status == types.MinerStatusAbort && height > mexist.AbortHeight+10) {
				value, ok := controller.mm.RefundStake(source.Bytes(), mexist, controller.AccountDB)
				if ok {
					refundValue := big.NewInt(0).SetUint64(value)
					controller.AccountDB.AddBalance(*source, refundValue)
					success = true
				}
			}
		} else {
			value, ok := controller.mm.RefundStake(source.Bytes(), mexist, controller.AccountDB)
			if ok {
				refundValue := big.NewInt(0).SetUint64(value)
				controller.AccountDB.AddBalance(*source, refundValue)
				success = true
			}
		}
	}
	if !success {
		controller.AccountDB.RevertToSnapshot(ss)
	}
	return success
}

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
#cgo LDFLAGS: -L ./ -ltvm

#include "tvm.h"
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>

void wrap_transfer(const char* p2, const char* value)
{
    void Transfer(const char*, const char* value);
    Transfer(p2, value);
}

char* wrap_get_balance(const char* address)
{
	char* GetBalance(const char*);
	return GetBalance(address);
}

void wrap_remove_data(const char* key, int key_len)
{
	void RemoveData(const char* , int);
	RemoveData(key, key_len);
}

void wrap_get_data(const char* key, int key_len, char** value, int* value_len)
{
	void GetData(const char*, int, char**, int*);
	return GetData(key, key_len, value, value_len);
}

void wrap_set_data(const char* key, int key_len, const char* value, int value_len)
{
	void SetData(const char*, int, const char*, int);
	SetData(key, key_len, value, value_len);
}

char* wrap_block_hash(unsigned long long height)
{
	char* BlockHash(unsigned long long);
	return BlockHash(height);
}

unsigned long long wrap_number()
{
	unsigned long long Number();
	return Number();
}

unsigned long long wrap_timestamp()
{
	unsigned long long Timestamp();
	return Timestamp();
}

unsigned long long wrap_tx_gas_limit()
{
	unsigned long long TxGasLimit();
	return TxGasLimit();
}

void wrap_contract_call(const char* address, const char* func_name, const char* json_parms, tvm_execute_result_t *result)
{
    char* ContractCall();
    ContractCall(address, func_name, json_parms, result);
}

void wrap_event_call(const char* event, const char* json_parms)
{
    void EventCall(const char*, const char*);
    EventCall(event, json_parms);
}

_Bool wrap_miner_stake(const char* minerAddr, int _type, const char* value) {
	_Bool MinerStake(const char*, int, const char*);
	return MinerStake(minerAddr, _type, value);
}

_Bool wrap_miner_cancel_stake(const char* minerAddr, int _type, const char* value) {
	_Bool MinerCancelStake(const char*, int, const char*);
	return MinerCancelStake(minerAddr, _type, value);
}

_Bool wrap_miner_refund_stake(const char* minerAddr, int _type) {
	_Bool MinerRefundStake(const char*, int);
	return MinerRefundStake(minerAddr, _type);
}
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type ExecuteResult struct {
	ResultType int
	ErrorCode  int
	Content    string
	Abi        string
}

// CallContract Execute the function of a contract which python code store in contractAddr
func CallContract(contractAddr string, funcName string, params string) *ExecuteResult {
	result := &ExecuteResult{}
	if !common.ValidateAddress(contractAddr) {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.TVMNoCodeError
		result.Content = fmt.Sprintf("invalid address %s!", contractAddr)
		return result
	}
	conAddr := common.StringToAddress(contractAddr)
	contract := LoadContract(conAddr)
	if contract.Code == "" {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.TVMNoCodeError
		result.Content = fmt.Sprintf("get code from address %s, but no code!", conAddr)
		return result
	}

	// prepare vm environment
	remainGas := controller.VM.Gas()
	oneVM := NewTVMForRetainContext(controller.VM.ContractAddress, contract, controller.VM.Logs)
	oneVM.SetGas(remainGas)
	finished := controller.StoreVMContext(oneVM)
	defer func() {
		// recover vm environment
		if finished {
			remainGas := oneVM.Gas()
			controller.RecoverVMContext()
			controller.VM.SetGas(remainGas)
		}
	}()
	if !finished {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.TVMCallMaxDeepError
		result.Content = fmt.Sprintf("call max deep cannot more than %d", MaxDepth)
		return result
	}

	msg := Msg{Data: []byte{}, Value: 0}
	executeResult, err := controller.VM.CreateContractInstance(msg)
	if err != nil {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.TVMExecutedError
		result.Content = err.Error()
		return result
	}

	abi := ABI{}
	abiJSON := fmt.Sprintf(`{"FuncName": "%s", "Args": %s}`, funcName, params)
	abiJSONError := json.Unmarshal([]byte(abiJSON), &abi)
	if abiJSONError != nil {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.TVMCheckABIError
		result.Content = abiJSONError.Error()
		return result
	}

	if !controller.VM.VerifyABI(executeResult.Abi, abi) {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.TVMCheckABIError
		result.Content = fmt.Sprintf("checkABI failed. abi:%s", abi.FuncName)
		return result
	}
	finalResult := controller.VM.executeABIKindEval(abi)
	return finalResult
}

func bridgeInit() {
	C.transfer_fn = (C.transfer_fn_t)(unsafe.Pointer(C.wrap_transfer))
	C.get_balance = (C.get_balance_fn_t)(unsafe.Pointer(C.wrap_get_balance))
	C.storage_get_data_fn = (C.storage_get_data_fn_t)(unsafe.Pointer(C.wrap_get_data))
	C.storage_set_data_fn = (C.storage_set_data_fn_t)(unsafe.Pointer(C.wrap_set_data))
	C.storage_remove_data_fn = (C.storage_remove_data_fn_t)(unsafe.Pointer(C.wrap_remove_data))
	// block
	C.block_hash_fn = (C.block_hash_fn_t)(unsafe.Pointer(C.wrap_block_hash))
	C.block_number_fn = (C.block_number_fn_t)(unsafe.Pointer(C.wrap_number))
	C.block_timestamp_fn = (C.block_timestamp_fn_t)(unsafe.Pointer(C.wrap_timestamp))
	C.gas_limit_fn = (C.gas_limit_fn_t)(unsafe.Pointer(C.wrap_tx_gas_limit))
	C.contract_call_fn = (C.contract_call_fn_t)(unsafe.Pointer(C.wrap_contract_call))
	C.event_call_fn = (C.event_call_fn_t)(unsafe.Pointer(C.wrap_event_call))
}

// Contract Contract contains the base message of a contract
type Contract struct {
	Code            string          `json:"code"`
	ContractName    string          `json:"contract_name"`
	ContractAddress *common.Address `json:"-"`
}

// LoadContract Load a contract-instance from a contract address
func LoadContract(address common.Address) *Contract {
	jsonString := controller.AccountDB.GetCode(address)
	con := &Contract{}
	_ = json.Unmarshal([]byte(jsonString), con)
	con.ContractAddress = &address
	return con
}

// TVM TVM is the role who execute contract code
type TVM struct {
	*Contract
	Sender *common.Address

	// xtm for log
	Logs []*types.Log
}

// NewTVM new a TVM instance
func NewTVM(sender *common.Address, contract *Contract) *TVM {
	C.tvm_start()
	return NewTVMForRetainContext(sender, contract, make([]*types.Log, 0))
}

func NewTVMForRetainContext(sender *common.Address, contract *Contract, logs []*types.Log) *TVM {
	tvm := &TVM{
		contract,
		sender,
		logs,
	}

	if !bridgeInited {
		bridgeInited = true
		bridgeInit()
	}
	C.tvm_set_gas(1000000)

	return tvm
}

// Gas Get the gas left of the TVM
func (tvm *TVM) Gas() int {
	return int(C.tvm_get_gas())
}

// SetGas Set the amount of gas that TVM can use
func (tvm *TVM) SetGas(gas int) {
	//fmt.Printf("SetGas: %d\n", gas);
	C.tvm_set_gas(C.int(gas))
}

// SetLibLine Correct the error line when python code is running
//func (tvm *TVM) SetLibLine(line int) {
//	C.tvm_set_lib_line(C.int(line))
//}

// DelTVM Run tvm gc to collect mem
func (tvm *TVM) DelTVM() {
	//C.tvm_gas_report()
	C.tvm_gc()
}

func (tvm *TVM) VerifyABI(standardABI string, callABI ABI) bool {
	//TODO:
	return true
	var standardABIStruct []ABIVerify
	err := json.Unmarshal([]byte(standardABI), &standardABIStruct)
	if err != nil {
		fmt.Println("abi unmarshal err:", err)
		return false
	}

	var argsType []string
	for i := 0; i < len(callABI.Args); i++ {

		switch callABI.Args[i].(type) {
		case float64:
			argsType = append(argsType, "int")
		case string:
			argsType = append(argsType, "str")
		case bool:
			argsType = append(argsType, "bool")
		case []interface{}:
			argsType = append(argsType, "list")
		case map[string]interface{}:
			argsType = append(argsType, "dict")
		default:
			argsType = append(argsType, "unknow")
		}
	}

	for _, value := range standardABIStruct {
		if value.FuncName == callABI.FuncName {
			if len(value.Args) == len(callABI.Args) {
				return compareSlice(value.Args, argsType)
			}
		}
	}
	return false
}

func compareSlice(a, b []string) bool {

	for key, value := range a {
		if value != b[key] {
			return false
		}
	}

	return true
}

// Msg Msg is msg instance which store running message when running a contract
type Msg struct {
	Data  []byte
	Value uint64
}

// CreateContractInstance Create contract instance
func (tvm *TVM) CreateContractInstance(msg Msg) (*ExecuteResult, error) {
	C.tvm_set_register()
	sender := C.CString(tvm.Sender.AddrPrefixString())
	value := C.ulonglong(msg.Value)
	C.tvm_set_msg(sender, value);
	C.free(unsafe.Pointer(sender))

	result, err := tvm.ExecuteScriptVMSucceedResults(tvm.Code)
	return result, err
}

//func (tvm *TVM) generateScript(res ABI) string {
//	var buf bytes.Buffer
//	buf.WriteString(fmt.Sprintf("tas_%s.", tvm.ContractName))
//	buf.WriteString(res.FuncName)
//	buf.WriteString("(")
//	for _, value := range res.Args {
//		tvm.jsonValueToBuf(&buf, value)
//		buf.WriteString(", ")
//	}
//	if len(res.Args) > 0 {
//		buf.Truncate(buf.Len() - 2)
//	}
//	buf.WriteString(")")
//	bufStr := buf.String()
//	return bufStr
//}

func (tvm *TVM) executeABIKindEval(res ABI) *ExecuteResult {
	args, _ := json.Marshal(res.Args)
	return tvm.funcCall(res.FuncName, string(args))
}

// ExecuteScriptVMSucceed Execute script and returns result
func (tvm *TVM) ExecuteScriptVMSucceed(script string) error {
	result := tvm.executePycode(script, C.PARSE_KIND_FILE)
	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
		fmt.Printf("execute error,code=%d,msg=%s \n", result.ErrorCode, result.Content)
		return errors.New(result.Content)
	}
	return nil
}

// ExecuteScriptVMSucceed Execute script and returns result
func (tvm *TVM) ExecuteScriptVMSucceedResults(script string) (result *ExecuteResult, err error) {
	result = tvm.executePycode(script, C.PARSE_KIND_FILE)
	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
		fmt.Printf("execute error,code=%d,msg=%s \n", result.ErrorCode, result.Content)
		return result, errors.New(result.Content)
	}
	return result, nil
}

func (tvm *TVM) executeScriptKindEval(script string) *ExecuteResult {
	return tvm.executePycode(script, C.PARSE_KIND_EVAL)
}

// ExecuteScriptKindFile Execute file and returns result
func (tvm *TVM) ExecuteScriptKindFile(script string) *ExecuteResult {
	return tvm.executePycode(script, C.PARSE_KIND_FILE)
}

func (tvm *TVM) ExportABI() (string, error) {
	C.tvm_set_register()
	result := tvm.executePycode(tvm.Code, C.PARSE_KIND_FILE)
	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
		return "", errors.New(result.Content)
	}
	return C.GoString(C.tvm_export_abi()), nil
}

func (tvm *TVM) executePycode(code string, parseKind C.tvm_parse_kind_t) *ExecuteResult {
	cResult := &C.tvm_execute_result_t{}
	C.tvm_init_result((*C.struct__tvm_execute_result_t)(unsafe.Pointer(cResult)))
	var param = C.CString(code)
	var contractName = C.CString(tvm.ContractName)

	//fmt.Println("-----------------code start-------------------")
	//fmt.Println(code)
	//fmt.Println("-----------------code end---------------------")
	C.tvm_execute(param, contractName, parseKind, (*C.tvm_execute_result_t)(unsafe.Pointer(cResult)))
	C.free(unsafe.Pointer(param))
	C.free(unsafe.Pointer(contractName))

	result := &ExecuteResult{}
	result.ResultType = int(cResult.result_type)
	result.ErrorCode = int(cResult.error_code)
	if cResult.content != nil {
		result.Content = C.GoString(cResult.content)
	}
	if cResult.abi != nil {
		result.Abi = C.GoString(cResult.abi)
	}
	//C.printResult((*C.ExecuteResult)(unsafe.Pointer(cResult)))
	C.tvm_deinit_result((*C.tvm_execute_result_t)(unsafe.Pointer(cResult)))
	return result
}

func (tvm *TVM) funcCall(funcName string, JSONArgs string) *ExecuteResult {
	cResult := &C.tvm_execute_result_t{}
	C.tvm_init_result((*C.struct__tvm_execute_result_t)(unsafe.Pointer(cResult)))
	var contract_name = C.CString(tvm.ContractName)
	var func_name = C.CString(funcName)
	var args = C.CString(JSONArgs)

	C.tvm_fun_call(contract_name, func_name, args, (*C.tvm_execute_result_t)(unsafe.Pointer(cResult)))
	C.free(unsafe.Pointer(contract_name))
	C.free(unsafe.Pointer(func_name))
	C.free(unsafe.Pointer(args))

	result := &ExecuteResult{}
	result.ResultType = int(cResult.result_type)
	result.ErrorCode = int(cResult.error_code)
	if cResult.content != nil {
		result.Content = C.GoString(cResult.content)
	}
	if cResult.abi != nil {
		result.Abi = C.GoString(cResult.abi)
	}
	//C.tvm_print_result((*C.tvm_execute_result_t)(unsafe.Pointer(cResult)))
	C.tvm_deinit_result((*C.tvm_execute_result_t)(unsafe.Pointer(cResult)))
	return result
}

//func (tvm *TVM) loadMsgWhenCall(msg Msg) error {
//	script := pycodeLoadWhenCall(tvm.Sender.AddrPrefixString(), msg.Value, tvm.ContractAddress.AddrPrefixString())
//	return tvm.ExecuteScriptVMSucceed(script)
//}

// Deploy TVM Deploy the contract code and load msg
func (tvm *TVM) Deploy(msg Msg) *ExecuteResult {
	C.tvm_set_register()

	sender := C.CString(tvm.Sender.AddrPrefixString())
	value := C.ulonglong(msg.Value)
	C.tvm_set_msg(sender, value);
	C.free(unsafe.Pointer(sender))

	result := tvm.executePycode(tvm.Code, C.PARSE_KIND_FILE)
	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
		return result
	}

	result = tvm.funcCall("__init__", "[]")
	return result
}

func (tvm *TVM) createContext() {
	C.tvm_create_context()
}

func (tvm *TVM) removeContext() {
	C.tvm_remove_context()
}

// ABI ABI stores the calling msg when execute contract
type ABI struct {
	FuncName string
	Args     []interface{}
}

// ABIVerify stores the contract function name and args types,
// in order to facilitate the abi verify
type ABIVerify struct {
	FuncName string
	Args     []string
}

func (tvm *TVM) jsonValueToBuf(buf *bytes.Buffer, value interface{}) {
	switch value.(type) {
	case float64:
		buf.WriteString(strconv.FormatFloat(value.(float64), 'f', 0, 64))
	case bool:
		x := value.(bool)
		if x {
			buf.WriteString("True")
		} else {
			buf.WriteString("False")
		}
	case string:
		buf.WriteString(`"`)
		buf.WriteString(value.(string))
		buf.WriteString(`"`)
	case []interface{}:
		buf.WriteString("[")
		for _, item := range value.([]interface{}) {
			tvm.jsonValueToBuf(buf, item)
			buf.WriteString(", ")
		}
		if len(value.([]interface{})) > 0 {
			buf.Truncate(buf.Len() - 2)
		}
		buf.WriteString("]")
	case map[string]interface{}:
		buf.WriteString("{")
		for key, item := range value.(map[string]interface{}) {
			tvm.jsonValueToBuf(buf, key)
			buf.WriteString(": ")
			tvm.jsonValueToBuf(buf, item)
			buf.WriteString(", ")
		}
		if len(value.(map[string]interface{})) > 0 {
			buf.Truncate(buf.Len() - 2)
		}
		buf.WriteString("}")
	default:
		fmt.Println(value)
		//panic("")
	}
}

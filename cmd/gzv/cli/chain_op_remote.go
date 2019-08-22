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

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/consensus/base"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
)

type RemoteChainOpImpl struct {
	host string
	port int
	base string
	aop  accountOp
	show bool
}

// InitRemoteChainOp connect node by ip and port
func InitRemoteChainOp(ip string, port int, show bool, op accountOp) *RemoteChainOpImpl {
	ca := &RemoteChainOpImpl{
		aop:  op,
		show: show,
	}
	ca.Connect(ip, port)
	return ca
}

// Connect connect node by ip and port
func (ca *RemoteChainOpImpl) Connect(ip string, port int) error {
	if ip == "" {
		return nil
	}
	ca.host = ip
	ca.port = port
	ca.base = fmt.Sprintf("http://%v:%v", ip, port)
	return nil
}

func (ca *RemoteChainOpImpl) request(method string, params ...interface{}) *Result {
	if ca.base == "" {
		return opError(ErrUnConnected)
	}

	param := RPCReqObj{
		Method:  "Gzv_" + method,
		Params:  params[:],
		ID:      1,
		Jsonrpc: "2.0",
	}

	if ca.show {
		fmt.Println("Request:")
		bs, _ := json.MarshalIndent(param, "", "\t")
		fmt.Println(string(bs))
		fmt.Println("==================================================================================")
	}

	paramBytes, err := json.Marshal(param)
	if err != nil {
		return opError(err)
	}

	resp, err := http.Post(ca.base, "application/json", bytes.NewReader(paramBytes))
	if err != nil {
		return opError(err)
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	ret := &RPCResObj{}
	if err := json.Unmarshal(responseBytes, ret); err != nil {
		return opError(err)
	}
	if ret.Error != nil {
		return opError(fmt.Errorf(ret.Error.Message))
	}
	return ret.Result
}

func (ca *RemoteChainOpImpl) nonce(addr string) (uint64, error) {
	ret := ca.request("nonce", addr)
	if !ret.IsSuccess() {
		return 0, fmt.Errorf(ret.Message)
	}
	return uint64(ret.Data.(float64)), nil
}

// Endpoint returns current connected ip and port
func (ca *RemoteChainOpImpl) Endpoint() string {
	return fmt.Sprintf("%v:%v", ca.host, ca.port)
}

// SendRaw send transaction to connected node
func (ca *RemoteChainOpImpl) SendRaw(tx *txRawData) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		return opError(fmt.Errorf("privatekey or pubkey error"))
	}
	source := pubkey.GetAddress()
	if source.AddrPrefixString() != aci.Address {
		return opError(fmt.Errorf("address error"))
	}

	if tx.Nonce == 0 {
		nonce, err := ca.nonce(aci.Address)
		if err != nil {
			return opError(err)
		}
		tx.Nonce = nonce
	}

	tranx := txRawToTransaction(tx)
	tranx.Hash = tranx.GenHash()
	sign, err := privateKey.Sign(tranx.Hash.Bytes())
	if err != nil {
		return opError(err)
	}
	tranx.Sign = sign.Bytes()
	tx.Sign = sign.Hex()

	jsonByte, err := json.Marshal(tx)
	if err != nil {
		return opError(err)
	}

	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	// Signature is required here
	return ca.request("tx", string(jsonByte))
}

// Balance query Balance by address
func (ca *RemoteChainOpImpl) Balance(addr string) *Result {
	return ca.request("balance", addr)
}

// MinerPoolInfo query miner pool info by address
func (ca *RemoteChainOpImpl) MinerPoolInfo(addr string) *Result {
	return ca.request("minerPoolInfo", addr,0)
}

// TicketsInfo query tickets by address
func (ca *RemoteChainOpImpl) TicketsInfo(addr string) *Result {
	return ca.request("ticketsInfo", addr)
}

// Nonce query Balance by address
func (ca *RemoteChainOpImpl) Nonce(addr string) *Result {
	return ca.request("nonce", addr)
}

// MinerInfo query miner info by address
func (ca *RemoteChainOpImpl) MinerInfo(addr string, detail string) *Result {
	return ca.request("minerInfo", addr, detail)
}

func (ca *RemoteChainOpImpl) BlockHeight() *Result {
	return ca.request("blockHeight")
}

func (ca *RemoteChainOpImpl) GroupHeight() *Result {
	return ca.request("groupHeight")
}

func (ca *RemoteChainOpImpl) TxInfo(hash string) *Result {
	return ca.request("transDetail", hash)
}

func (ca *RemoteChainOpImpl) BlockByHash(hash string) *Result {
	return ca.request("getBlockByHash", hash)
}

func (ca *RemoteChainOpImpl) BlockByHeight(h uint64) *Result {
	return ca.request("getBlockByHeight", h)
}

// StakeAdd adds stake for the given target account
func (ca *RemoteChainOpImpl) StakeAdd(target string, mType int, stake uint64, gas, gasPrice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)

	if target == "" {
		target = aci.Address
	}

	pks := &types.MinerPks{
		MType: types.MinerType(mType),
	}

	// When stakes for himself, pks will be required
	if aci.Address == target {
		if aci.Miner == nil {
			return opError(fmt.Errorf("the current account is not a miner account"))
		}
		var bpk groupsig.Pubkey
		bpk.SetHexString(aci.Miner.BPk)
		pks.Pk = bpk.Serialize()
		pks.VrfPk = base.Hex2VRFPublicKey(aci.Miner.VrfPk)
	} else {
		//if stake to Verify and target is not myself then return error
		if pks.MType == types.MinerTypeVerify {
			return opError(fmt.Errorf("you could not stake for other's verify node"))
		}
	}

	st := common.TAS2RA(stake)

	data, err := types.EncodePayload(pks)
	if err != nil {
		return opError(err)
	}
	tx := &txRawData{
		Target:   target,
		Value:    st,
		Gas:      gas,
		Gasprice: gasPrice,
		TxType:   types.TransactionTypeStakeAdd,
		Data:     data,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}


func (ca *RemoteChainOpImpl) ChangeFundGuardMode(mode int,gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	tx := &txRawData{
		Target:   aci.Address,
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeChangeFundGuardMode,
		Data: []byte{byte(mode)},
	}
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) VoteMinerPool(target string, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	target = strings.TrimSpace(target)
	if target == "" {
		return opError(fmt.Errorf("please input target address"))
	}
	if !common.ValidateAddress(target) {
		return opError(fmt.Errorf("Wrong address format"))
	}
	if aci.Address == target {
		return opError(fmt.Errorf("you could not vote to myself"))
	}
	tx := &txRawData{
		Target:   target,
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeVoteMinerPool,
	}
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) ApplyGuardMiner(gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	tx := &txRawData{
		Target:   aci.Address,
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeApplyGuardMiner,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

// MinerAbort send stop mining transaction
func (ca *RemoteChainOpImpl) MinerAbort(mtype int, gas, gasprice uint64, force bool) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if aci.Miner == nil {
		return opError(fmt.Errorf("the current account is not a miner account"))
	}
	if !force {
		checkResult := ca.GroupCheck(aci.Address)
		if !checkResult.IsSuccess() {
			return opError(fmt.Errorf(checkResult.Message))
		}
		m := checkResult.Data.(map[string]interface{})
		info := m["current_group_routine"]
		if info != nil {
			selected := info.(map[string]interface{})["selected"]
			if selected != nil && selected.(bool) {
				return opError(fmt.Errorf("You are selected to join a group currently, abort operation may result in frozen. And you can specify the '-f' if you insist"))
			}
		}
	}
	tx := &txRawData{
		Target:   aci.Address,
		Gas:      gas,
		Gasprice: gasprice,
		TxType:   types.TransactionTypeMinerAbort,
		Data:     []byte{byte(mtype)},
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

// StakeRefund send refund transaction. After the group is dissolved, the token will be refunded
func (ca *RemoteChainOpImpl) StakeRefund(target string, mType int, gas, gasPrice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if target == "" {
		target = aci.Address
	}
	tx := &txRawData{
		Target:   target,
		Gas:      gas,
		Gasprice: gasPrice,
		TxType:   types.TransactionTypeStakeRefund,
		Data:     []byte{byte(mType)},
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

// StakeReduce send reduce stake transaction
func (ca *RemoteChainOpImpl) StakeReduce(target string, mType int, value, gas, gasPrice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if target == "" {
		target = aci.Address
	}
	if value == 0 {
		return opError(fmt.Errorf("value must > 0"))
	}
	reduceValue := common.TAS2RA(value)
	tx := &txRawData{
		Target:   target,
		Gas:      gas,
		Gasprice: gasPrice,
		Value:    reduceValue,
		TxType:   types.TransactionTypeStakeReduce,
		Data:     []byte{byte(mType)},
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) ViewContract(addr string) *Result {
	return ca.request("viewAccount", addr)
}

func (ca *RemoteChainOpImpl) TxReceipt(hash string) *Result {
	return ca.request("txReceipt", hash)
}

func (ca *RemoteChainOpImpl) GroupCheck(addr string) *Result {
	return ca.request("groupCheck", addr)
}

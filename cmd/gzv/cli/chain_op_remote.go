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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"net/http"
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

func (ca *RemoteChainOpImpl) request(method string, params ...interface{}) *RPCResObjCmd {
	ret := &RPCResObjCmd{}
	if ca.base == "" {
		output(ErrUnConnected)
		return nil
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
		output(err)
		return nil
	}

	resp, err := http.Post(ca.base, "application/json", bytes.NewReader(paramBytes))
	if err != nil {
		ret.Error = opErrorRes(err)
		return ret
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(responseBytes, ret); err != nil {
		ret.Error = opErrorRes(err)
		return ret
	}
	return ret
}

func (ca *RemoteChainOpImpl) nonce(addr string) uint64 {
	var nonce uint64
	res := ca.request("nonce", addr)
	json.Unmarshal(res.Result, &nonce)
	return nonce
}

// Endpoint returns current connected ip and port
func (ca *RemoteChainOpImpl) Endpoint() string {
	return fmt.Sprintf("%v:%v", ca.host, ca.port)
}

// SendRaw send transaction to connected node
func (ca *RemoteChainOpImpl) SendRaw(tx *txRawData) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		output(err)
		return nil
	}
	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		output(fmt.Errorf("privatekey or pubkey error"))
		return nil
	}
	source := pubkey.GetAddress()
	if source.AddrPrefixString() != aci.Address {
		output(fmt.Errorf("privatekey or pubkey error"))
		return nil
	}

	if tx.Nonce == 0 {
		nonce := ca.nonce(aci.Address)
		tx.Nonce = nonce
	}

	tranx := txRawToTransaction(tx)
	tranx.Hash = tranx.GenHash()
	sign, err := privateKey.Sign(tranx.Hash.Bytes())
	if err != nil {
		output(err)
		return nil
	}
	tranx.Sign = sign.Bytes()
	tx.Sign = sign.Hex()

	jsonByte, err := json.Marshal(tx)
	if err != nil {
		output(err)
		return nil
	}

	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	// Signature is required here
	res = ca.request("tx", string(jsonByte))
	return res
}

// Balance query Balance by address
func (ca *RemoteChainOpImpl) Balance(addr string) *RPCResObjCmd {
	return ca.request("balance", addr)
}

// Nonce query Balance by address
func (ca *RemoteChainOpImpl) Nonce(addr string) *RPCResObjCmd {
	return ca.request("nonce", addr)
}

// MinerInfo query miner info by address
func (ca *RemoteChainOpImpl) MinerInfo(addr string, detail string) *RPCResObjCmd {
	return ca.request("minerInfo", addr, detail)
}

func (ca *RemoteChainOpImpl) BlockHeight() *RPCResObjCmd {
	res := ca.request("blockHeight")
	return res
}

func (ca *RemoteChainOpImpl) GroupHeight() *RPCResObjCmd {
	return ca.request("groupHeight")
}

func (ca *RemoteChainOpImpl) TxInfo(hash string) *RPCResObjCmd {
	return ca.request("transDetail", hash)
}

func (ca *RemoteChainOpImpl) BlockByHash(hash string) *RPCResObjCmd {
	return ca.request("getBlockByHash", hash)
}

func (ca *RemoteChainOpImpl) BlockByHeight(h uint64) *RPCResObjCmd {
	res := ca.request("getBlockByHeight")
	return res
}

// StakeAdd adds stake for the given target account
func (ca *RemoteChainOpImpl) StakeAdd(target string, mType int, stake uint64, gas, gasPrice uint64) *RPCResObjCmd {
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		output(err)
		return nil
	}

	if target == "" {
		target = aci.Address
	}

	pks := &types.MinerPks{
		MType: types.MinerType(mType),
	}

	// When stakes for himself, pks will be required
	if aci.Address == target {
		if aci.Miner == nil {
			output(fmt.Errorf("the current account is not a miner account"))
			return nil
		}
		var bpk groupsig.Pubkey
		bpk.SetHexString(aci.Miner.BPk)
		pks.Pk = bpk.Serialize()
		pks.VrfPk = base.Hex2VRFPublicKey(aci.Miner.VrfPk)
	} else {
		//if stake to Verify and target is not myself then return error
		if pks.MType == types.MinerTypeVerify {
			output(fmt.Errorf("you could not stake for other's verify node"))
			return nil
		}
	}

	st := common.TAS2RA(stake)

	data, err := types.EncodePayload(pks)
	if err != nil {
		output(err)
		return nil
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

// MinerAbort send stop mining transaction
func (ca *RemoteChainOpImpl) MinerAbort(mtype int, gas, gasprice uint64, force bool) *RPCResObjCmd {
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		output(err)
		return nil
	}

	if aci.Miner == nil {
		output(fmt.Errorf("the current account is not a miner account"))
		return nil
	}
	if !force {
		res := ca.GroupCheck(aci.Address)
		if res.Error != nil {
			output(res.Error)
			return nil
		}
		groupInfo := new(GroupCheckInfo)
		err := json.Unmarshal(res.Result, groupInfo)
		if err != nil {
			output(err)
			return nil
		}

		info := groupInfo.CurrentGroupRoutine
		if info != nil {
			selected := info.Selected
			if selected {
				output(fmt.Errorf("you are selected to join a group currently, abort operation may result in frozen. And you can specify the '-f' if you insist"))
				return nil
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
func (ca *RemoteChainOpImpl) StakeRefund(target string, mType int, gas, gasPrice uint64) *RPCResObjCmd {
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		output(err)
		return nil
	}

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
func (ca *RemoteChainOpImpl) StakeReduce(target string, mType int, value, gas, gasPrice uint64) *RPCResObjCmd {
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		output(err)
		return nil
	}
	if target == "" {
		target = aci.Address
	}
	if value == 0 {
		output(fmt.Errorf("value must > 0"))
		return nil
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

func (ca *RemoteChainOpImpl) ViewContract(addr string) *RPCResObjCmd {
	return ca.request("viewAccount", addr)
}

func (ca *RemoteChainOpImpl) TxReceipt(hash string) *RPCResObjCmd {
	return ca.request("txReceipt", hash)
}

func (ca *RemoteChainOpImpl) GroupCheck(addr string) *RPCResObjCmd {
	return ca.request("groupCheck", addr)
}

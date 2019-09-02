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

func (ca *RemoteChainOpImpl) request(method string, params ...interface{}) *RPCResObjCmd {
	ret := &RPCResObjCmd{}
	if ca.base == "" {
		ret.Error = opErrorRes(ErrUnConnected)
		return ret
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
		ret.Error = opErrorRes(err)
		return ret
	}

	resp, err := http.Post(ca.base, "application/json", bytes.NewReader(paramBytes))
	if err != nil {
		ret.Error = opErrorRes(err)
		return ret
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ret.Error = opErrorRes(err)
		return ret
	}
	if err := json.Unmarshal(responseBytes, ret); err != nil {
		ret.Error = opErrorRes(err)
		return ret
	}
	return ret
}

func (ca *RemoteChainOpImpl) nonce(addr string) (uint64, *ErrorResult) {
	var nonce uint64
	res := ca.request("nonce", addr)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.Result != nil {
		err := json.Unmarshal(res.Result, &nonce)
		if err != nil {
			return 0, opErrorRes(err)
		}
	}
	return nonce, nil
}

// Endpoint returns current connected ip and port
func (ca *RemoteChainOpImpl) Endpoint() string {
	return fmt.Sprintf("%v:%v", ca.host, ca.port)
}

// SendRaw send transaction to connected node
func (ca *RemoteChainOpImpl) SendRaw(tx *TxRawData) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	privateKey := common.HexToSecKey(aci.Sk)
	pubkey := common.HexToPubKey(aci.Pk)
	if privateKey.GetPubKey().Hex() != pubkey.Hex() {
		res.Error = opErrorRes(fmt.Errorf("privatekey or pubkey error"))
		return res
	}
	source := pubkey.GetAddress()
	if source.AddrPrefixString() != aci.Address {
		res.Error = opErrorRes(fmt.Errorf("privatekey or pubkey error"))
		return res
	}
	tx.Source = aci.Address

	if tx.Nonce == 0 {
		nonce, errRes := ca.nonce(aci.Address)
		if errRes == nil {
			tx.Nonce = nonce
		} else {
			res.Error = errRes
			return res
		}

	}
	tranx := txRawToTransaction(tx)
	sign, err := privateKey.Sign(tranx.Hash.Bytes())
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	tranx.Sign = sign.Bytes()
	tx.Sign = sign.Hex()

	jsonByte, err := json.Marshal(tx)
	if err != nil {
		res.Error = opErrorRes(err)
		return res
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

// MinerPoolInfo query miner pool info by address
func (ca *RemoteChainOpImpl) MinerPoolInfo(addr string) *RPCResObjCmd {
	return ca.request("minerPoolInfo", addr, 0)
}

// TicketsInfo query tickets by address
func (ca *RemoteChainOpImpl) TicketsInfo(addr string) *RPCResObjCmd {
	return ca.request("ticketsInfo", addr)
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
	res := ca.request("getBlockByHeight", h)
	return res
}

// StakeAdd adds value for the given target account
func (ca *RemoteChainOpImpl) StakeAdd(target string, mType int, stake uint64, gas, gasPrice uint64) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
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
			res.Error = opErrorRes(fmt.Errorf("the current account is not a miner account"))
			return res
		}
		var bpk groupsig.Pubkey
		bpk.SetHexString(aci.Miner.BPk)
		pks.Pk = bpk.Serialize()
		pks.VrfPk = base.Hex2VRFPublicKey(aci.Miner.VrfPk)
	} else {
		//if value to Verify and target is not myself then return error
		if pks.MType == types.MinerTypeVerify {
			res.Error = opErrorRes(fmt.Errorf("you could not value for other's verify node"))
			return res
		}
	}

	st := common.TAS2RA(stake)

	data, err := types.EncodePayload(pks)
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	tx := &TxRawData{
		Target:   target,
		Value:    st,
		GasLimit: gas,
		GasPrice: gasPrice,
		TxType:   types.TransactionTypeStakeAdd,
		Data:     data,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) ChangeFundGuardMode(mode int, gas, gasprice uint64) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	_, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	tx := &TxRawData{
		GasLimit: gas,
		GasPrice: gasprice,
		TxType:   types.TransactionTypeChangeFundGuardMode,
		Data:     []byte{byte(mode)},
	}
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) VoteMinerPool(target string, gas, gasprice uint64) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	target = strings.TrimSpace(target)
	if target == "" {
		res.Error = opErrorRes(fmt.Errorf("please input target address"))
		return res
	}
	if !common.ValidateAddress(target) {
		res.Error = opErrorRes(fmt.Errorf("wrong address format"))
		return res
	}
	if aci.Address == target {
		res.Error = opErrorRes(fmt.Errorf("you could not vote to myself"))
		return res
	}
	tx := &TxRawData{
		Target:   target,
		GasLimit: gas,
		GasPrice: gasprice,
		TxType:   types.TransactionTypeVoteMinerPool,
	}
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) ApplyGuardMiner(gas, gasprice uint64) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	tx := &TxRawData{
		GasLimit: gas,
		GasPrice: gasprice,
		TxType:   types.TransactionTypeApplyGuardMiner,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

// MinerAbort send stop mining transaction
func (ca *RemoteChainOpImpl) MinerAbort(mtype int, gas, gasprice uint64, force bool) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}

	if aci.Miner == nil {
		res.Error = opErrorRes(fmt.Errorf("the current account is not a miner account"))
		return res
	}
	if types.IsVerifyRole(types.MinerType(mtype)) && !force {
		groupCheckRes := ca.GroupCheck(aci.Address)
		if groupCheckRes.Error != nil {
			return groupCheckRes
		}
		groupInfo := new(GroupCheckInfo)
		err := json.Unmarshal(groupCheckRes.Result, groupInfo)
		if err != nil {
			res.Error = opErrorRes(err)
			return res
		}

		info := groupInfo.CurrentGroupRoutine
		if info != nil {
			selected := info.Selected
			if selected {
				res.Error = opErrorRes(fmt.Errorf("you are selected to join a group currently, abort operation may result in frozen. And you can specify the '-f' if you insist"))
				return res
			}
		}
	}
	tx := &TxRawData{
		GasLimit: gas,
		GasPrice: gasprice,
		TxType:   types.TransactionTypeMinerAbort,
		Data:     []byte{byte(mtype)},
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

// StakeRefund send refund transaction. After the group is dissolved, the token will be refunded
func (ca *RemoteChainOpImpl) StakeRefund(target string, mType int, gas, gasPrice uint64) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}

	if target == "" {
		target = aci.Address
	}
	tx := &TxRawData{
		Target:   target,
		GasLimit: gas,
		GasPrice: gasPrice,
		TxType:   types.TransactionTypeStakeRefund,
		Data:     []byte{byte(mType)},
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

// StakeReduce send reduce value transaction
func (ca *RemoteChainOpImpl) StakeReduce(target string, mType int, value, gas, gasPrice uint64) *RPCResObjCmd {
	res := new(RPCResObjCmd)
	aci, err := ca.aop.AccountInfo()
	if err != nil {
		res.Error = opErrorRes(err)
		return res
	}
	if target == "" {
		target = aci.Address
	}
	if value == 0 {
		res.Error = opErrorRes(fmt.Errorf("value must > 0"))
		return res
	}
	reduceValue := common.TAS2RA(value)
	tx := &TxRawData{
		Target:   target,
		GasLimit: gas,
		GasPrice: gasPrice,
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

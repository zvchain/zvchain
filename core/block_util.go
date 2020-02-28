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

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/serialize"
	"github.com/zvchain/zvchain/storage/trie"
	"github.com/zvchain/zvchain/tvm"
)

const (
	teamFoundationToken     = 750000000 * common.ZVC // amount of tokens that belong to team
	businessFoundationToken = 250000000 * common.ZVC // amount of tokens that belongs to business
	stakePlatformToken      = 400000000 * common.ZVC // amount of tokens that belongs to mining pool
	circulatesToken         = 100000000 * common.ZVC // amount of tokens that belongs to circulates
)

type txSlice []*types.Transaction

func (ts txSlice) txsToRaw() []*types.RawTransaction {
	raws := make([]*types.RawTransaction, len(ts))
	for i, tx := range ts {
		raws[i] = tx.RawTransaction
	}
	return raws
}

func (ts txSlice) txsHashes() []common.Hash {
	hs := make([]common.Hash, len(ts))
	for i, tx := range ts {
		hs[i] = tx.Hash
	}
	return hs
}

func (ts txSlice) calcTxTree() common.Hash {
	if nil == ts || 0 == len(ts) {
		return common.EmptyHash
	}

	buf := new(bytes.Buffer)

	for _, tx := range ts {
		buf.Write(tx.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return common.EmptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(receipts); i++ {
		if receipts[i] != nil {
			keybuf.Reset()
			serialize.Encode(keybuf, uint(i))
			encode, _ := serialize.EncodeToBytes(receipts[i])
			trie.Update(keybuf.Bytes(), encode)
		}
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
}

func setupGenesisStateDB(stateDB *account.AccountDB, genesisInfo *types.GenesisInfo) {
	// FoundationContract
	businessFoundationAddr := setupFoundationContract(stateDB, types.GetAdminAddr(), businessFoundationToken, 1)
	stateDB.SetBalance(*businessFoundationAddr, big.NewInt(0).SetUint64(businessFoundationToken))
	teamFoundationAddr := setupFoundationContract(stateDB, types.GetAdminAddr(), teamFoundationToken, 2)
	stateDB.SetBalance(*teamFoundationAddr, big.NewInt(0).SetUint64(teamFoundationToken))
	stateDB.SetNonce(types.GetAdminAddr(), 2)

	// permission contract

	SetupPermissionContracts(stateDB, genesisInfo)

	// mining pool and circulates
	stateDB.SetBalance(types.GetStakePlatformAddr(), big.NewInt(0).SetUint64(stakePlatformToken))
	stateDB.SetBalance(types.GetCirculatesAddr(), big.NewInt(0).SetUint64(circulatesToken))

	// genesis balance: just for stakes two roles with minimum required value
	genesisBalance := big.NewInt(0).SetUint64(4 * minimumStake())
	for _, mem := range genesisInfo.Group.Members() {
		addr := common.BytesToAddress(mem.ID())
		stateDB.SetBalance(addr, genesisBalance)
	}
}

func setupFoundationContract(stateDB *account.AccountDB, adminAddr common.Address, totalToken, nonce uint64) *common.Address {
	code := fmt.Sprintf(foundationContract, adminAddr.AddrPrefixString(), totalToken)
	txRaw := &types.RawTransaction{}
	addr := adminAddr
	txRaw.Source = &addr
	txRaw.Value = &types.BigInt{Int: *big.NewInt(0)}
	txRaw.GasLimit = &types.BigInt{Int: *big.NewInt(300000)}
	controller := tvm.NewController(stateDB, nil, nil, types.NewTransaction(txRaw, txRaw.GenHash()), 0, nil)
	contract := tvm.Contract{
		Code:         code,
		ContractName: "Foundation",
	}
	jsonBytes, err := json.Marshal(contract)
	if err != nil {
		panic(fmt.Sprintf("deploy FoundationContract error: %s", err.Error()))
	}
	contractAddress := common.BytesToAddress(common.Sha256(common.BytesCombine(txRaw.GetSource()[:], common.Uint64ToByte(nonce))))
	stateDB.CreateAccount(contractAddress)
	stateDB.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	_, _, transactionError := controller.Deploy(&contract)
	if transactionError != nil {
		panic(fmt.Sprintf("deploy FoundationContract error: %s", transactionError.Message))
	}
	return contract.ContractAddress
}

func MakeABIString(function string, args ...interface{}) string {
	abi := tvm.ABI{}
	abi.FuncName = function
	if args == nil {
		args = make([]interface{}, 0)
	}
	abi.Args = args
	js, _ := json.Marshal(abi)
	return string(js)
}

func SetupPermissionContracts(stateDB *account.AccountDB, genesisInfo *types.GenesisInfo) {

	if len(genesisInfo.Group.Members()) == 0 {
		panic(fmt.Sprintf("deploy permission contract error: genesis group len = 0"))
	}
	adminAddr := common.BytesToAddress(genesisInfo.Group.Members()[0].ID())
	nonce := stateDB.GetNonce(adminAddr)
	if nonce != 0 {
		panic(fmt.Sprintf("deploy permission contract error: adminAddr nonce not 0"))
	}
	nonce += 1

	UpgradableAddress, UpgradableVC := setupPermissionContract(stateDB, adminAddr, nonce, "PermissionsUpgradable", "./contracts/PermissionsUpgradable.py")

	nonce += 1
	AccountAddress, _ := setupPermissionContract(stateDB, adminAddr, nonce, "AccountManager", "./contracts/AccountMgr.py")

	nonce += 1
	NodeAddress, _ := setupPermissionContract(stateDB, adminAddr, nonce, "NodeManager", "./contracts/NodeMgr.py")

	nonce += 1
	OrgAddress, _ := setupPermissionContract(stateDB, adminAddr, nonce, "OrgManager", "./contracts/OrgMgr.py")

	nonce += 1
	VoterAddress, _ := setupPermissionContract(stateDB, adminAddr, nonce, "VoteManager", "./contracts/VoteMgr.py")

	nonce += 1
	InterfaceAddress, InterfaceVC := setupPermissionContract(stateDB, adminAddr, nonce, "PermissionInterface", "./contracts/PermissionsInterface.py")

	nonce += 1
	ImplAddress, _ := setupPermissionContract(stateDB, adminAddr, nonce, "PermissionsImplementation", "./contracts/PermissionsImplementation.py")

	UpgradableContent := tvm.LoadContract(*UpgradableAddress)
	initAbi := MakeABIString("init",
		InterfaceAddress,
		ImplAddress,
		UpgradableAddress,
		AccountAddress,
		OrgAddress,
		VoterAddress,
		NodeAddress)
	UpgradableVC.VM.SetGas(50000000)

	_, _, transactionError := UpgradableVC.ExecuteAbiEval(&adminAddr, UpgradableContent, initAbi)

	if transactionError != nil {
		panic(transactionError.Message)
	}

	nwAdminOrg := types.NETWORK_ADMIN_ORG
	config, err := types.ParsePermissionConfig("./")
	if err == nil {
		nwAdminOrg = config.NwAdminOrg
	}

	InterfaceContent := tvm.LoadContract(*InterfaceAddress)
	InterfaceVC.GasLeft = 50000000
	InterfaceVC.VM.SetGas(50000000)

	_, _, transactionError = InterfaceVC.ExecuteAbiEval(&adminAddr, InterfaceContent, MakeABIString("set_policy", nwAdminOrg))
	if transactionError != nil {
		panic(transactionError.Message)
	}

	_, _, transactionError = InterfaceVC.ExecuteAbiEval(&adminAddr, InterfaceContent, MakeABIString("init"))
	if transactionError != nil {
		panic(transactionError.Message)
	}

	for i, mem := range genesisInfo.Group.Members() {
		addr := common.BytesToAddress(mem.ID()).AddrPrefixString()
		fmt.Printf("add_alliance_node : %v\n", addr)
		blsPk := common.ToHex(genesisInfo.Pks[i])
		vrfPk := common.ToHex(genesisInfo.VrfPKs[i])

		_, _, transactionError = InterfaceVC.ExecuteAbiEval(&adminAddr, InterfaceContent, MakeABIString("add_alliance_node", addr, 2, vrfPk, blsPk, 0))
		if transactionError != nil {
			panic(transactionError.Message)
		}
		if addr == adminAddr.AddrPrefixString() {
			fmt.Printf("add_alliance_account : %v\n", addr)
			_, _, transactionError = InterfaceVC.ExecuteAbiEval(&adminAddr, InterfaceContent, MakeABIString("add_alliance_account", addr))
			if transactionError != nil {
				panic(transactionError.Message)
			}
		}
	}
	_, _, transactionError = InterfaceVC.ExecuteAbiEval(&adminAddr, InterfaceContent, MakeABIString("update_network_boot_status"))
	if transactionError != nil {
		panic(transactionError.Message)
	}
}

func setupPermissionContract(stateDB *account.AccountDB, adminAddr common.Address, nonce uint64, name, filePath string) (*common.Address, *tvm.Controller) {
	code, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("setupPermissionContract read file %v,error=\"%s\"\n", filePath, err)
		return nil, nil
	}
	txRaw := &types.RawTransaction{}
	addr := adminAddr
	txRaw.Source = &addr
	txRaw.Value = &types.BigInt{Int: *big.NewInt(0)}
	txRaw.GasLimit = &types.BigInt{Int: *big.NewInt(300000)}
	bh := types.BlockHeader{}
	bh.Height = 0
	controller := tvm.NewController(stateDB, nil, &bh, types.NewTransaction(txRaw, txRaw.GenHash()), 0, nil)
	contract := tvm.Contract{
		Code:         string(code),
		ContractName: name,
	}
	jsonBytes, err := json.Marshal(contract)
	if err != nil {
		panic(fmt.Sprintf("deploy permission contract %s error: %s", name, err.Error()))
	}
	contractAddress := common.BytesToAddress(common.Sha256(common.BytesCombine(txRaw.GetSource()[:], common.Uint64ToByte(nonce))))
	stateDB.CreateAccount(contractAddress)
	stateDB.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	_, _, transactionError := controller.Deploy(&contract)
	if transactionError != nil {
		panic(fmt.Sprintf("deploy permission contract %s error: %s", name, transactionError.Message))
	}
	return contract.ContractAddress, controller
}

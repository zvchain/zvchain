package core

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/params"
	"github.com/zvchain/zvchain/tvm"
	"math/big"
	"strings"
	"sync"
)

const AddressManagerContract = "zv1388dee8f36656bff16a886570ed8967e1d378d73309317e9b70bfb6ed3c1be8"
const AddressManagerContractTest = "zv07ef5f16247334a2076f8348db719a0115c7c2904403377ae93c7b8fd2827fa2"
const Nonce = 5

func isUseContract() bool {
	chain := BlockChainImpl
	isZip5 := false
	if chain != nil {
		isZip5 = params.GetChainConfig().IsZIP005(chain.QueryTopBlock().Height)
		if isZip5 {
			addressManager.CheckAndUpdate()
		}
	}
	return isZip5
}

func DaemonNodeAddress() common.Address {
	if isUseContract() && addressManager.daemonNodeAddr != nil {
		fmt.Println("DaemonNodeAddress,", addressManager.daemonNodeAddr.AddrPrefixString())
		return *addressManager.daemonNodeAddr
	}
	return types.DaemonNodeAddress()
}

func UserNodeAddress() common.Address {
	if isUseContract() && addressManager.userNodeAddr != nil {
		fmt.Println("UserNodeAddress,", addressManager.userNodeAddr.AddrPrefixString())
		return *addressManager.userNodeAddr
	}
	return types.UserNodeAddress()
}

func CirculatesAddr() common.Address {
	if isUseContract() && addressManager.circulatesAddr != nil {
		fmt.Println("circulatesAddr,", addressManager.circulatesAddr.AddrPrefixString())
		return *addressManager.circulatesAddr
	}
	return types.CirculatesAddr()
}

func StakePlatformAddr() common.Address {
	if isUseContract() && addressManager.stakePlatformAddr != nil {
		fmt.Println("stakePlatformAddr,", addressManager.stakePlatformAddr.AddrPrefixString())
		return *addressManager.stakePlatformAddr
	}
	return types.StakePlatformAddr()
}

func AdminAddr() common.Address {
	if isUseContract() && addressManager.adminAddr != nil {
		fmt.Println("AdminAddr,", addressManager.adminAddr.AddrPrefixString())
		return *addressManager.adminAddr
	}
	return types.AdminAddr()
}

func GuardAddress() []common.Address {
	if isUseContract() && addressManager.GuardNodes != nil {
		fmt.Println("GuardNodes[0],", addressManager.GuardNodes[0])
		return addressManager.GuardNodes
	}
	return types.GuardAddress()
}

type AddressManager struct {
	adminAddr         *common.Address
	stakePlatformAddr *common.Address
	circulatesAddr    *common.Address
	userNodeAddr      *common.Address
	daemonNodeAddr    *common.Address
	GuardNodes        []common.Address
	ContractAddr      *common.Address
	//TestContractAddr  common.Address
	cacheBlockHash common.Hash
	mu             sync.Mutex
}

var addressManager AddressManager

func AddressContractAddr() string {
	if types.IsNormalChain() {
		return AddressManagerContract
	}
	return AddressManagerContractTest
}

func (am *AddressManager) CheckAndUpdate() {
	am.mu.Lock()
	defer am.mu.Unlock()
	chain := BlockChainImpl
	if am.cacheBlockHash == chain.QueryTopBlock().Hash {
		return
	}
	am.cacheBlockHash = chain.QueryTopBlock().Hash
	if am.ContractAddr == nil {
		if params.GetChainConfig().EqualZIP005(chain.QueryTopBlock().Height) {
			am.deployAddressManagerContract()
		} else {
			genesisInfo := chain.consensusHelper.GenerateGenesisInfo()
			adminAddr := common.BytesToAddress(genesisInfo.Group.Members()[0].ID())
			addr := am.getContractAddr(adminAddr, Nonce)
			fmt.Println("get_adminAddraddress,", addr.AddrPrefixString())
			am.ContractAddr = &addr
		}
	}
	am.loadAllAddress()
}

func (am *AddressManager) deployAddressManagerContract() {
	codeStr := addressManagerContract
	contractName := "AddressManager"
	if !types.IsNormalChain() {
		codeStr = addressManagerContractTest
	}
	am.autoCreateContract(contractName, codeStr)
}

func (am *AddressManager) getContractAddr(source common.Address, nonce uint64) common.Address {
	contractAddress := common.BytesToAddress(common.Sha256(common.BytesCombine(source[:], common.Uint64ToByte(nonce))))
	return contractAddress
}

func (am *AddressManager) autoCreateContract(contractName string, contractCode string) *common.Address {
	chain := BlockChainImpl
	stateDB, _ := chain.LatestAccountDB()
	txRaw := &types.RawTransaction{}
	genesisInfo := chain.consensusHelper.GenerateGenesisInfo()
	adminAddr := common.BytesToAddress(genesisInfo.Group.Members()[0].ID())
	txRaw.Source = &adminAddr
	txRaw.Value = &types.BigInt{Int: *big.NewInt(0)}
	txRaw.GasLimit = &types.BigInt{Int: *big.NewInt(300000)}
	txRaw.Nonce = Nonce
	controller := tvm.NewController(stateDB, nil, nil, types.NewTransaction(txRaw, txRaw.GenHash()), 0, nil)
	contract := tvm.Contract{
		Code:         contractCode,
		ContractName: contractName,
	}
	jsonBytes, err := json.Marshal(contract)
	if err != nil {
		panic(fmt.Sprintf("jsonMarshal Addressmanagercontract error: %s", err.Error()))
	}
	contractAddress := am.getContractAddr(adminAddr, Nonce)
	fmt.Println("get_create_adminAddraddress,", contractAddress.AddrPrefixString())
	stateDB.CreateAccount(contractAddress)
	stateDB.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	_, _, transactionError := controller.Deploy(&contract)
	if transactionError != nil {
		panic(fmt.Sprintf("deploy Addressmanagercontract error: %s", transactionError.Message))
	}
	am.ContractAddr = &contractAddress
	return contract.ContractAddress
}

func (am *AddressManager) loadAllAddress() bool {

	am.GuardNodes = make([]common.Address, 0, 0)
	chain := BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		fmt.Errorf("address manager loadAllAddress err:%v ", err)
		return false
	}
	genesisInfo := chain.consensusHelper.GenerateGenesisInfo()
	adminAddr := common.BytesToAddress(genesisInfo.Group.Members()[0].ID())
	nonce := db.GetNonce(adminAddr)
	if nonce != 0 {
		nonce += 1
	}
	contractAddress := am.ContractAddr

	iter := db.DataIterator(*contractAddress, []byte{})
	if iter == nil {
		fmt.Errorf("address manager loadAllAddress err,iter is nil ")
		return false
	}

	for iter.Next() {
		k := string(iter.Key[:])
		if strings.HasPrefix(k, "pool_lists@") {
			addr := strings.TrimLeft(k, "pool_lists@")
			am.GuardNodes = append(am.GuardNodes, common.StringToAddress(addr))
		} else {
			v := tvm.VmDataConvert(iter.Value[:])
			switch k {
			case "adminAddr":
				if addr, ok := v.(string); ok {
					adminAddr := common.StringToAddress(addr)
					am.adminAddr = &adminAddr
				}
			case "stakePlatformAddr":
				if addr, ok := v.(string); ok {
					stakePlatformAddr := common.StringToAddress(addr)
					am.stakePlatformAddr = &stakePlatformAddr
				}
			case "circulatesAddr":
				if addr, ok := v.(string); ok {
					circulatesAddr := common.StringToAddress(addr)
					am.circulatesAddr = &circulatesAddr
				}
			case "userNodeAddr":
				if addr, ok := v.(string); ok {
					userNodeAddr := common.StringToAddress(addr)
					am.userNodeAddr = &userNodeAddr
				}
			case "daemonNodeAddr":
				if addr, ok := v.(string); ok {
					daemonNodeAddr := common.StringToAddress(addr)
					am.daemonNodeAddr = &daemonNodeAddr
				}
			}
		}
	}
	return true
}

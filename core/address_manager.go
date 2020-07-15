package core

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/params"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/tvm"
	"math/big"
	"strings"
	"sync"
)

const AddressContract = "zv0000000000000000000000000000000000000000000000000000000000000001"
const AddressSource = "0x0001"

func isUseContract() bool {
	chain := BlockChainImpl
	isZip5 := false
	if chain != nil {
		isZip5 = params.GetChainConfig().IsZIP005(chain.QueryTopBlock().Height)
	}
	return isZip5
}

func DaemonNodeAddress() common.Address {
	if isUseContract() {
		daemonNodeAddr := loadNormalAddress("daemonNodeAddr")
		if daemonNodeAddr != nil {
			fmt.Println("daemonNodeAddr,", daemonNodeAddr)
			return *daemonNodeAddr
		}
	}
	return types.DaemonNodeAddress()
}

func UserNodeAddress() common.Address {
	if isUseContract() {
		userNodeAddr := loadNormalAddress("userNodeAddr")
		if userNodeAddr != nil {
			fmt.Println("userNodeAddr,", userNodeAddr)
			return *userNodeAddr
		}
	}
	return types.UserNodeAddress()
}

func CirculatesAddr() common.Address {
	if isUseContract() {
		circulatesAddr := loadNormalAddress("circulatesAddr")
		if circulatesAddr != nil {
			fmt.Println("circulatesAddr,", circulatesAddr)
			return *circulatesAddr
		}
	}
	return types.CirculatesAddr()
}

func StakePlatformAddr() common.Address {
	if isUseContract() {
		stakePlatformAddr := loadNormalAddress("stakePlatformAddr")
		if stakePlatformAddr != nil {
			fmt.Println("stakePlatformAddr,", stakePlatformAddr)
			return *stakePlatformAddr
		}
	}
	return types.StakePlatformAddr()
}

func AdminAddr() common.Address {
	if isUseContract() {

		adminAddr := loadNormalAddress("adminAddr")
		if adminAddr != nil {
			fmt.Println("AdminAddr,", adminAddr)
			return *adminAddr
		}
	}
	return types.AdminAddr()
}

func GuardAddress() []common.Address {
	if isUseContract() {
		guardAddress := loadGuardAddress()
		if guardAddress != nil && len(guardAddress) > 0 {
			fmt.Println("GuardNodes[0],", guardAddress[0])
			return guardAddress

		}
	}
	return types.GuardAddress()
}

type AddressManager struct {
	deployContract bool
	mu             sync.Mutex
}

var addressManager AddressManager

func (am *AddressManager) CheckAndUpdate(accountdb *account.AccountDB, bh *types.BlockHeader) {
	am.mu.Lock()
	defer am.mu.Unlock()
	if am.deployContract {
		return
	}
	am.deployContract = true
	if params.GetChainConfig().EqualZIP005(bh.Height) {
		am.deployAddressManagerContract(accountdb)
	}
}

func (am *AddressManager) deployAddressManagerContract(stateDB *account.AccountDB) {
	contractCode := addressManagerContract
	contractName := "AddressManager"
	if !types.IsNormalChain() {
		contractCode = addressManagerContractTest
	}
	txRaw := &types.RawTransaction{}
	adminAddr := common.StringToAddress(AddressSource)
	txRaw.Source = &adminAddr
	txRaw.Value = &types.BigInt{Int: *big.NewInt(0)}
	txRaw.GasLimit = &types.BigInt{Int: *big.NewInt(300000)}
	controller := tvm.NewController(stateDB, nil, nil, types.NewTransaction(txRaw, txRaw.GenHash()), 0, nil)
	contract := tvm.Contract{
		Code:         contractCode,
		ContractName: contractName,
	}
	jsonBytes, err := json.Marshal(contract)
	if err != nil {
		panic(fmt.Sprintf("jsonMarshal Addressmanagercontract error: %s", err.Error()))
	}
	contractAddress := common.StringToAddress(AddressContract)
	fmt.Println("get_create_adminAddraddress,", contractAddress.AddrPrefixString())
	stateDB.CreateAccount(contractAddress)
	stateDB.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	result, _, transactionError := controller.Deploy(&contract)
	re, _ := json.Marshal(&result)
	fmt.Println("controller.Deploy,", string(re))
	if transactionError != nil {
		panic(fmt.Sprintf("deploy Addressmanagercontract error: %s", transactionError.Message))
	}
	//am.ContractAddr = &contractAddress
}

func loadGuardAddress() []common.Address {

	guardNodes := make([]common.Address, 0, 0)
	chain := BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		fmt.Errorf("address manager loadAllAddress err:%v ", err)
		return nil
	}
	contractAddress := common.StringToAddress(AddressContract)
	iter := db.DataIterator(contractAddress, []byte{})
	if iter == nil {
		fmt.Errorf("address manager loadAllAddress err,iter is nil ")
		return nil
	}

	for iter.Next() {
		k := string(iter.Key[:])
		if strings.HasPrefix(k, "pool_lists@") {
			addr := strings.TrimLeft(k, "pool_lists@")
			guardNodes = append(guardNodes, common.StringToAddress(addr))
		}
	}
	return guardNodes
}

func loadNormalAddress(key string) *common.Address {
	chain := BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		fmt.Errorf("address manager loadAllAddress err:%v ", err)
		return nil
	}
	contractAddress := common.StringToAddress(AddressContract)
	iter := db.DataIterator(contractAddress, []byte{})
	if iter == nil {
		fmt.Errorf("address manager loadAllAddress err,iter is nil ")
		return nil
	}
	for iter.Next() {
		k := string(iter.Key[:])
		if !strings.HasPrefix(k, "pool_lists@") {
			v := tvm.VmDataConvert(iter.Value[:])
			resultAddr := &common.Address{}
			switch k {
			case "adminAddr":
				if addr, ok := v.(string); ok {
					adminAddr := common.StringToAddress(addr)
					resultAddr = &adminAddr
				}

			case "stakePlatformAddr":
				if addr, ok := v.(string); ok {
					stakePlatformAddr := common.StringToAddress(addr)
					//am.stakePlatformAddr = &stakePlatformAddr
					resultAddr = &stakePlatformAddr

				}
			case "circulatesAddr":
				if addr, ok := v.(string); ok {
					circulatesAddr := common.StringToAddress(addr)
					resultAddr = &circulatesAddr

					//am.circulatesAddr = &circulatesAddr
				}
			case "userNodeAddr":
				if addr, ok := v.(string); ok {
					userNodeAddr := common.StringToAddress(addr)
					resultAddr = &userNodeAddr

					//am.userNodeAddr = &userNodeAddr
				}
			case "daemonNodeAddr":
				if addr, ok := v.(string); ok {
					daemonNodeAddr := common.StringToAddress(addr)
					resultAddr = &daemonNodeAddr
					//am.daemonNodeAddr = &daemonNodeAddr
				}
			}
			if key == k {
				return resultAddr
			}
		}
	}
	return nil
}

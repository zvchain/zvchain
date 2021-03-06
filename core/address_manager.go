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
)

const AddressContract = "zv0000000000000000000000000000000000000000000000000000000000000006"

//address manager contract creator
const AddressSource = "zv0007"

func isUseContract() bool {
	chain := BlockChainImpl
	isZip6 := false
	if chain != nil {
		topBlock := chain.QueryTopBlock()
		if topBlock != nil {
			isZip6 = params.GetChainConfig().IsZIP006(topBlock.Height)
		}
	}
	return isZip6
}

func DaemonNodeAddress() common.Address {
	if isUseContract() {
		daemonNodeAddr := loadNormalAddress("daemonNodeAddr")
		if daemonNodeAddr != nil {
			return *daemonNodeAddr
		}
	}
	return types.DaemonNodeAddress()
}

func UserNodeAddress() common.Address {
	if isUseContract() {
		userNodeAddr := loadNormalAddress("userNodeAddr")
		if userNodeAddr != nil {
			return *userNodeAddr
		}
	}
	return types.UserNodeAddress()
}

func CirculatesAddr() common.Address {
	if isUseContract() {
		circulatesAddr := loadNormalAddress("circulatesAddr")
		if circulatesAddr != nil {
			return *circulatesAddr
		}
	}
	return types.CirculatesAddr()
}

func StakePlatformAddr() common.Address {
	if isUseContract() {
		stakePlatformAddr := loadNormalAddress("stakePlatformAddr")
		if stakePlatformAddr != nil {
			return *stakePlatformAddr
		}
	}
	return types.StakePlatformAddr()
}

func AdminAddr() common.Address {
	if isUseContract() {

		adminAddr := loadNormalAddress("adminAddr")
		if adminAddr != nil {
			return *adminAddr
		}
	}
	return types.AdminAddr()
}

func GuardAddress() []common.Address {
	return types.GuardAddress()
}

func IsInExtractGuardNodes(addr common.Address) bool {
	addresses := GuardAddress()
	for _, addrStr := range addresses {
		if addrStr == addr {
			return true
		}
	}
	return false
}

type AddressManager struct {
}

var addressManager AddressManager

func (am *AddressManager) DeployAddressManagerContract(stateDB *account.AccountDB) {
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
	stateDB.CreateAccount(contractAddress)
	stateDB.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	_, _, transactionError := controller.Deploy(&contract)
	if transactionError != nil {
		panic(fmt.Sprintf("deploy Addressmanagercontract error: %s", transactionError.Message))
	}
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
		if strings.HasPrefix(k, "guard_lists@") {
			addr := strings.TrimLeft(k, "guard_lists@")
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
	iter := db.DataIterator(contractAddress, []byte(key))
	if iter == nil {
		fmt.Errorf("address manager loadAllAddress err,iter is nil ")
		return nil
	}
	for iter.Next() {
		k := string(iter.Key[:])
		v := tvm.VmDataConvert(iter.Value[:])
		if key == k {
			if addr, ok := v.(string); ok {
				resultAddr := common.StringToAddress(addr)
				return &resultAddr

			}
		}
	}
	return nil
}

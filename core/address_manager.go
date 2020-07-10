package core

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/params"
	"github.com/zvchain/zvchain/tvm"
	"strings"
	"sync"
)

const AddressManagerContract = "zv1388dee8f36656bff16a886570ed8967e1d378d73309317e9b70bfb6ed3c1be8"
const AddressManagerContractTest = "zv07ef5f16247334a2076f8348db719a0115c7c2904403377ae93c7b8fd2827fa2"

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
	if isUseContract() {
		return addressManager.daemonNodeAddr
	}
	return types.DaemonNodeAddress()
}

func UserNodeAddress() common.Address {
	if isUseContract() {
		return addressManager.userNodeAddr
	}
	return types.UserNodeAddress()
}

func CirculatesAddr() common.Address {
	if isUseContract() {
		return addressManager.circulatesAddr
	}
	return types.CirculatesAddr()
}

func StakePlatformAddr() common.Address {
	if isUseContract() {
		return addressManager.stakePlatformAddr
	}
	return types.StakePlatformAddr()
}

func AdminAddr() common.Address {
	if isUseContract() {
		return addressManager.adminAddr
	}
	return types.AdminAddr()
}

func GuardAddress() []common.Address {
	if isUseContract() {
		return addressManager.GuardNodes
	}
	return types.GuardAddress()
}

type AddressManager struct {
	adminAddr         common.Address
	stakePlatformAddr common.Address
	circulatesAddr    common.Address
	userNodeAddr      common.Address
	daemonNodeAddr    common.Address
	GuardNodes        []common.Address

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
	am.loadAllAddress()
}

func (am *AddressManager) loadAllAddress() bool {

	am.GuardNodes = make([]common.Address, 0, 0)
	chain := BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		fmt.Errorf("address manager loadAllAddress err:%v ", err)
		return false
	}

	iter := db.DataIterator(common.StringToAddress(AddressContractAddr()), []byte{})
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
					am.adminAddr = common.StringToAddress(addr)
				}
			case "stakePlatformAddr":
				if addr, ok := v.(string); ok {
					am.stakePlatformAddr = common.StringToAddress(addr)
				}
			case "circulatesAddr":
				if addr, ok := v.(string); ok {
					am.circulatesAddr = common.StringToAddress(addr)
				}
			case "userNodeAddr":
				if addr, ok := v.(string); ok {
					am.userNodeAddr = common.StringToAddress(addr)
				}
			case "daemonNodeAddr":
				if addr, ok := v.(string); ok {
					am.daemonNodeAddr = common.StringToAddress(addr)
				}
			}
		}
	}
	return true
}

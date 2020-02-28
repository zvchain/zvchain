package permission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

const (
	RSSuccess          uint64 = 0
	RSFail             uint64 = 1
	RSBalanceNotEnough uint64 = 2
	RSAbiError         uint64 = 3
	RSTvmError         uint64 = 4
	RSGASnotEnough     uint64 = 5
	RSNoCode           uint64 = 6
)

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

type ContractManager struct {
	config *types.PermissionConfig
	Sk     *common.PrivateKey
}

func NewContractManager(pconfig *types.PermissionConfig) *ContractManager {
	p := &ContractManager{config: pconfig}
	return p
}

func (t *ContractManager) NewTxFromTxArgs(txArgs SendTxArgs) *TxRawData {
	tx := TxRawData{}

	tx.Value = txArgs.Value
	tx.GasLimit = txArgs.Gas
	tx.GasPrice = txArgs.GasPrice
	tx.TxType = txArgs.TxType
	tx.Target = txArgs.Target
	tx.Data = []byte(txArgs.Data)
	return &tx
}

func (t *ContractManager) DeployContract(contract tvm.Contract, TxArgs SendTxArgs, sk common.PrivateKey) (txHash string, status uint64) {
	address := sk.GetPubKey().GetAddress().AddrPrefixString()

	//fmt.Printf("Balance: %v \n", GetBalance(address))
	tx := t.NewTxFromTxArgs(TxArgs)
	tx.Nonce = GetNonce(address)
	data, err := json.Marshal(contract)
	if err != nil {
		return "", 1
	}
	tx.Data = data
	tx.TxType = types.TransactionTypeContractCreate
	tx.Source = address

	txHash, err = t.SendTx(tx, sk)
	if err != nil {
		return "", 1
	}
	timeout, res := t.WaitExecuted(txHash)
	if timeout {
		return "", 1
	}
	return gjson.Get(res, "contractAddress").String(), gjson.Get(res, "status").Uint()
}

func (t *ContractManager) CallContract(address string, txArgs SendTxArgs) (txHash string, e error) {
	caller := t.Sk.GetPubKey().GetAddress().AddrPrefixString()
	tx := t.NewTxFromTxArgs(txArgs)
	tx.Nonce = GetNonce(caller)
	tx.TxType = types.TransactionTypeContractCall
	tx.Source = caller
	tx.Target = address
	txHash, err := t.SendTx(tx, *t.Sk)
	if err != nil {
		Logger.Errorf("send tx error ,hash:%v,err:%v", txHash, err)
		return "", err
	}
	/*timeout, res := t.WaitExecuted(txHash)
	if timeout {
		return "", 1
	}*/
	return txHash, nil
}

func (t *ContractManager) SendTx(tx *TxRawData, sk common.PrivateKey) (string, error) {
	tranx := TxRawToTransaction(tx)
	sign, err := sk.Sign(tranx.GenHash().Bytes())
	if err != nil {
		return "", fmt.Errorf("sign error")
	}
	tx.Sign = sign.Hex()
	tranx.Sign = sign.Bytes()
	if DataSourceType == DataSourceLocalChain {
		Logger.Debugf("sent tx:%v", tranx)
		_, err := core.BlockChainImpl.AddTransactionToPool(tranx)

		return tranx.GenHash().String(), err
	}

	param := RPCReqObj{
		Method:  PublicRpcNamespace + "tx",
		Params:  []interface{}{tx},
		ID:      1,
		Jsonrpc: "2.0",
	}
	paramBytes, err := json.Marshal(param)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(RpcAddr, "application/json", bytes.NewReader(paramBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBytes, _ := ioutil.ReadAll(resp.Body)
	e := gjson.Get(string(responseBytes), "error").Raw
	if e != "" {
		return "", fmt.Errorf("%s", e)
	}
	return gjson.Get(string(responseBytes), "result").String(), nil
}

func (t *ContractManager) WaitExecuted(hash string) (timeout bool, result string) {
	outTicker := time.After(time.Minute)
	for {
		select {
		case <-outTicker:
			return true, ""
		case <-time.After(time.Second):
			receipt := GetReceipt(hash)
			if receipt != "" && receipt != "null" {
				return false, receipt
			}
		}
	}
}

func (t *ContractManager) Init() (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("init"))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) SetPolicy(nwAdminOrg string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("set_policy", nwAdminOrg))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) AddOrg(orgId string, enodeId string, account string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("add_org", orgId, enodeId, account))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) ApproveOrg(orgId string, url string, account string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("approve_org", orgId, url, account))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) UpdateOrgStatus(orgId string, _action *big.Int) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("update_org_status", orgId, _action.Uint64()))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) AddNode(orgId string, url string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("add_node", orgId, url))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) UpdateNodeStatus(orgId string, url string, _action *big.Int) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("update_node_status", orgId, url, _action.Uint64()))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) ApproveOrgStatus(orgId string, _action *big.Int) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("approve_org_status", orgId, _action.Uint64()))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) AssignAdmin(orgId string, acctId string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("assign_alliance_admin", orgId, acctId))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) UpdateAccountAccess(acctId string, orgId string, access *big.Int) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("update_account_access", orgId, acctId, access))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) ApproveAdmin(orgId string, acctId string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("approve_alliance_admin", orgId, acctId))
	return t.CallContract(t.config.InterfAddress, txArgs)
}
func (t *ContractManager) AddAccount(acctId string, orgId string, access *big.Int, isAdmin bool) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("add_account", acctId, orgId, access, isAdmin))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) UpdateAccountStatus(orgId string, acctId string, _action *big.Int) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("update_account_status", orgId, acctId, _action.Uint64()))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) AddAllianceNode(nodeUrl string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("add_alliance_node", nodeUrl))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) AddAllianceAccount(account string) (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("add_alliance_account", account))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) UpdateNetworkBootStatus() (string, error) {
	txArgs := CallTxArgsSetData(MakeABIString("update_network_boot_status"))
	return t.CallContract(t.config.InterfAddress, txArgs)
}

func (t *ContractManager) GetNetworkBootStatus() (bool, error) {
	result := GetAccountData(t.config.ImplAddress, "network_boot")
	networkBoot := gjson.Get(result, "0.value").Bool()
	return networkBoot, nil
}

func (t *ContractManager) GetPendingOp(orgId string) (string, string, string, int64, error) {
	result := GetAccountData(t.config.VoterAddress, "voter_list")

	voteListStr := gjson.Get(result, "0.value").String()
	var voteList []types.VoteInfo
	json.Unmarshal([]byte(voteListStr), &voteList)
	for index := range voteList {
		vote := voteList[index]
		if vote.OrgId == orgId {
			return vote.OrgId, vote.NodeId, vote.Account, int64(vote.OpType), nil
		}
	}
	return "", "", "", 0, nil
}

package permission

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	PERMISSIONED_CONFIG = "permissioned-nodes.json"
	BLACKLIST_CONFIG    = "disallowed-nodes.json"
)

type NodeOperation uint8

var Logger *logrus.Logger

const (
	NodeAdd NodeOperation = iota
	NodeDelete
)

type PermissionCtrl struct {
	contractMgr *ContractManager

	dataDir string

	permConfig *types.PermissionConfig

	startWaitGroup *sync.WaitGroup // waitgroup to make sure all dependenies are ready before we start the service
	//stopFeed       event.Feed      // broadcasting stopEvent when service is being stopped
	errorChan       chan error // channel to capture error when starting aysnc
	selfAddr        string
	mux             sync.Mutex
	genesisAccounts []string
}

// Create a service instance for permissioning
//
// Permission Service depends on the following:
// 1. EthService to be ready
// 2. Downloader to sync up blocks
// 3. InProc RPC server to be ready
func NewPermissionCtrl(pconfig *types.PermissionConfig) (*PermissionCtrl, error) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	p := &PermissionCtrl{

		//key:            common.HexToSecKey(core.BlockChainImpl.Account.MinerSk()),
		permConfig:     pconfig,
		startWaitGroup: wg,
		errorChan:      make(chan error),
		contractMgr:    NewContractManager(pconfig),
	}

	return p, nil
}

var permissionCtrl *PermissionCtrl
var permissionCtrlAPI *PermissionCtrlAPI

func Init(Sk string, genesisAccounts []string) error {
	Logger = log.PermissionLogger
	Logger.Debugf("Permission Init")

	if permissionCtrl != nil {
		return nil
	}
	config, err := types.ParsePermissionConfig("./")
	if err != nil {
		return err
	}
	permissionCtrl, err = NewPermissionCtrl(config)
	if err != nil {
		return err
	}
	privateKey := common.HexToSecKey(Sk)
	Logger.Debugf("guardianKey.ImportKe;%v", Sk)
	if privateKey == nil {
		return fmt.Errorf(("HexToSecKey error"))
	}

	permissionCtrl.contractMgr.Sk = privateKey
	permissionCtrl.selfAddr = privateKey.GetPubKey().GetAddress().AddrPrefixString()
	permissionCtrl.genesisAccounts = genesisAccounts
	permissionCtrlAPI = NewPermissionCtrlAPI(permissionCtrl)
	notify.BUS.Subscribe(notify.BlockAddSucc, permissionCtrl.onBlockAddSuccess)

	return nil
}

func PermissionCtrlInstance() *PermissionCtrl {
	return permissionCtrl
}

func PermissionCtrlAPIInstance() *PermissionCtrlAPI {
	return permissionCtrlAPI
}

// This is to make sure all contract instances are ready and initialized
//
// Required to be call after standard service start lifecycle
func (p *PermissionCtrl) AfterStart() error {
	Logger.Debugf("Permission AfterStart \n")
	p.LoadConfigFormContract()
	// populate the initial list of permissioned nodes and account accesses
	if err := p.populateInitPermissions(); err != nil {
		return fmt.Errorf("populateInitPermissions failed: %v", err)
	}

	Logger.Debugf("permission service: is now ready")

	return nil
}

// start service asynchronously due to dependencies
func (p *PermissionCtrl) asyncStart() {
	p.AfterStart()
}

func (p *PermissionCtrl) LoadConfigFormContract() {
	adminAddr := common.StringToAddress(p.genesisAccounts[0])

	if len(p.permConfig.UpgrdAddress) == 0 {
		p.permConfig.UpgrdAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(1)))).AddrPrefixString()
	}
	if len(p.permConfig.AccountAddress) == 0 {
		p.permConfig.AccountAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(2)))).AddrPrefixString()
	}
	if len(p.permConfig.NodeAddress) == 0 {
		p.permConfig.NodeAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(3)))).AddrPrefixString()
	}
	if len(p.permConfig.OrgAddress) == 0 {
		p.permConfig.OrgAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(4)))).AddrPrefixString()
	}
	if len(p.permConfig.VoterAddress) == 0 {
		p.permConfig.VoterAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(5)))).AddrPrefixString()
	}
	if len(p.permConfig.InterfAddress) == 0 {
		p.permConfig.InterfAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(6)))).AddrPrefixString()
	}
	if len(p.permConfig.ImplAddress) == 0 {
		p.permConfig.ImplAddress = common.BytesToAddress(common.Sha256(common.BytesCombine(adminAddr[:], common.Uint64ToByte(7)))).AddrPrefixString()
	}
}

func (p *PermissionCtrl) Start() error {
	Logger.Debugf("permission service: starting")
	go func() {
		Logger.Debugf("permission service: starting async")
		p.asyncStart()
	}()
	return nil
}

func (p *PermissionCtrl) Stop() error {
	Logger.Debugf("permission service: stopping")

	Logger.Debugf("permission service: stopped")
	return nil
}

// adds or deletes and entry from a given file
func (p *PermissionCtrl) updateFile(fileName, enodeId string, operation NodeOperation, createFile bool) {
	// Load the nodes from the config file
	var nodeList []string
	index := 0
	// if createFile is false means the file is already existing. read the file
	if !createFile {
		blob, err := ioutil.ReadFile(fileName)
		if err != nil && !createFile {
			Logger.Debug("Failed to access the file", "fileName", fileName, "err", err)
			return
		}

		if err := json.Unmarshal(blob, &nodeList); err != nil {
			Logger.Debug("Failed to load nodes list from file", "fileName", fileName, "err", err)
			return
		}

		// logic to update the permissioned-nodes.json file based on action

		recExists := false
		for i, eid := range nodeList {
			if eid == enodeId {
				index = i
				recExists = true
				break
			}
		}
		if (operation == NodeAdd && recExists) || (operation == NodeDelete && !recExists) {
			return
		}
	}
	if operation == NodeAdd {
		nodeList = append(nodeList, enodeId)
	} else {
		nodeList = append(nodeList[:index], nodeList[index+1:]...)
	}
	blob, _ := json.Marshal(nodeList)

	p.mux.Lock()
	defer p.mux.Unlock()

	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		Logger.Debug("Error writing new node info to file", "fileName", fileName, "err", err)
	}
}

// updates node information in the permissioned-nodes.json file based on node
// management activities in smart contract
func (p *PermissionCtrl) updatePermissionedNodes(enodeId string, operation NodeOperation) {
	Logger.Debug("updatePermissionedNodes", "DataDir", p.dataDir, "file", PERMISSIONED_CONFIG)

	path := filepath.Join(p.dataDir, PERMISSIONED_CONFIG)
	if _, err := os.Stat(path); err != nil {
		Logger.Debug("Read Error for permissioned-nodes.json file. This is because 'permissioned' flag is specified but no permissioned-nodes.json file is present", "err", err)
		return
	}

	p.updateFile(path, enodeId, operation, false)

}

//this function populates the black listed node information into the disallowed-nodes.json file
func (p *PermissionCtrl) updateDisallowedNodes(url string, operation NodeOperation) {
	Logger.Debug("updateDisallowedNodes", "DataDir", p.dataDir, "file", BLACKLIST_CONFIG)

	fileExists := true
	path := filepath.Join(p.dataDir, BLACKLIST_CONFIG)
	// Check if the file is existing. If the file is not existing create the file
	if _, err := os.Stat(path); err != nil {
		Logger.Debug("Read Error for disallowed-nodes.json file", "err", err)
		if _, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644); err != nil {
			Logger.Debug("Failed to create disallowed-nodes.json file", "err", err)
			return
		}
		fileExists = false
	}

	if fileExists {
		p.updateFile(path, url, operation, false)
	} else {
		p.updateFile(path, url, operation, true)
	}
}

// Thus function checks if the initial network boot up status and if no
// populates permissions model with details from permission-config.json
func (p *PermissionCtrl) populateInitPermissions() error {

	networkInitialized, err := p.contractMgr.GetNetworkBootStatus()
	if err != nil {
		// handle the scenario of no contract code.
		Logger.Debugln("Failed to retrieve network boot status ", "err", err)
		return err
	}

	if !networkInitialized {
		Logger.Errorf(" permission  network status error")
	}
	//populate orgs, nodes, roles and accounts from contract
	for _, f := range []func() error{
		p.populateOrgsFromContract,
		p.populateNodesFromContract,
		p.populateAccountsFromContract,
		p.populateVotesFromContract,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

// initialize the permissions model and populate initial values
func (p *PermissionCtrl) bootupNetwork() error {
	if _, err := p.contractMgr.SetPolicy(p.permConfig.NwAdminOrg); err != nil {
		Logger.Debugln("bootupNetwork SetPolicy failed", "err", err)
		return err
	}
	if _, err := p.contractMgr.Init(); err != nil {
		Logger.Debugln("bootupNetwork init failed", "err", err)
		return err
	}

	types.OrgInfoMap.UpsertOrg(p.permConfig.NwAdminOrg, types.OrgApproved)
	// populate the initial node list from static-nodes.json
	if err := p.populateStaticNodesToContract(); err != nil {
		return err
	}
	// populate initial account access to full access
	if err := p.populateInitAccountAccess(); err != nil {
		return err
	}

	// update network status to boot completed
	if err := p.updateNetworkStatus(); err != nil {
		Logger.Debug("failed to updated network boot status", "error", err)
		return err
	}
	return nil
}

// populates the account access details from contract into cache
func (p *PermissionCtrl) populateAccountsFromContract() error {
	//populate accounts

	result := GetAccountData(p.permConfig.AccountAddress, "account_list")
	Logger.Debugf(result)
	listStr := gjson.Get(result, "0.value").String()
	var accounts []types.AccountInfo
	json.Unmarshal([]byte(listStr), &accounts)
	for index := range accounts {
		account := accounts[index]
		types.AcctInfoMap.UpsertAccount(account.OrgId, account.Account, account.IsAdmin, account.IsVoter, account.Access, types.AcctStatus(account.Status))
	}
	return nil
}

// populates the node details from contract into cache
func (p *PermissionCtrl) populateNodesFromContract() error {

	result := GetAccountData(p.permConfig.NodeAddress, "node_list")
	Logger.Debugf(result)
	listStr := gjson.Get(result, "0.value").String()

	var nodes []types.NodeInfo
	json.Unmarshal([]byte(listStr), &nodes)
	for index := range nodes {
		node := nodes[index]
		types.NodeInfoMap.UpsertNode(node.OrgId, node.NodeId, types.NodeStatus(node.Status))
	}
	return nil
}

// populates the org details from contract into cache
func (p *PermissionCtrl) populateOrgsFromContract() error {

	result := GetAccountData(p.permConfig.OrgAddress, "org_list")
	Logger.Debugf(result)

	orgsListStr := gjson.Get(result, "0.value").String()
	var orgsList []types.OrgInfo
	json.Unmarshal([]byte(orgsListStr), &orgsList)
	for index := range orgsList {
		org := orgsList[index]
		types.OrgInfoMap.UpsertOrg(org.OrgId, types.OrgStatus(org.Status))
	}
	return nil
}

// populates the vote details from contract into cache
func (p *PermissionCtrl) populateVotesFromContract() error {

	result := GetAccountData(p.permConfig.VoterAddress, "pending_op_list")
	Logger.Debugf(result)
	listStr := gjson.Get(result, "0.value").String()
	var list []types.VoteInfo
	json.Unmarshal([]byte(listStr), &list)
	types.VoteInfoMap.Clear()
	for index := range list {
		vote := list[index]
		types.VoteInfoMap.AddVote(&vote)
	}

	result = GetAccountData(p.permConfig.VoterAddress, "passed_op_list")
	listStr = gjson.Get(result, "0.value").String()
	var passedlist []types.VoteInfo
	json.Unmarshal([]byte(listStr), &passedlist)
	for index := range passedlist {
		vote := list[index]
		types.VoteInfoMap.AddVote(&vote)
	}
	return nil
}

// Reads the node list from static-nodes.json and populates into the contract
func (p *PermissionCtrl) populateStaticNodesToContract() error {
	nodes := p.genesisAccounts
	for index := range nodes {
		node := nodes[index]
		_, err := p.contractMgr.AddAllianceNode(node)
		if err != nil {
			Logger.Debug("Failed to propose node", "err", err, "node", node)
			return err
		}
		types.NodeInfoMap.UpsertNode(p.permConfig.NwAdminOrg, node, 2)
	}
	return nil
}

// Invokes the initAccounts function of smart contract to set the initial
// set of accounts access to full access
func (p *PermissionCtrl) populateInitAccountAccess() error {
	for _, a := range p.permConfig.Accounts {
		_, er := p.contractMgr.AddAllianceAccount(a.String())
		if er != nil {
			Logger.Debug("Error adding permission initial account list", "err", er, "account", a)
			return er
		}
		types.AcctInfoMap.UpsertAccount(p.permConfig.NwAdminOrg, a, true, true, types.FullAccess, 2)
	}
	return nil
}

// updates network boot status to true
func (p *PermissionCtrl) updateNetworkStatus() error {
	_, err := p.contractMgr.UpdateNetworkBootStatus()
	if err != nil {
		Logger.Debug("Failed to update network boot status ", "err", err)
		return err
	}
	return nil
}

func (p *PermissionCtrl) onBlockAddSuccess(message notify.Message) error {

	b := message.GetData().(*types.Block)
	for _, tx := range b.Transactions {
		if tx.Type != types.TransactionTypeContractCall {
			continue
		}
		receipt := core.BlockChainImpl.GetTransactionPool().GetReceipt(tx.GenHash())
		if receipt == nil || receipt.Logs == nil {
			continue
		}
		isLoadNode := false
		isLoadAccount := false
		isLoadOrg := false
		isLoadVote := false
		for _, log := range receipt.Logs {
			Logger.Debug("PermissionCtrl onBlockAddSuccess logs:%v\n", log.String())

			if log.Address.AddrPrefixString() == p.permConfig.NodeAddress && !isLoadNode {
				isLoadNode = true
				Logger.Debug("PermissionCtrl load NodeAddress\n")
				p.populateNodesFromContract()
			} else if log.Address.AddrPrefixString() == p.permConfig.AccountAddress && !isLoadAccount {
				isLoadAccount = true
				Logger.Debug("PermissionCtrl load AccountAddress\n")
				p.populateAccountsFromContract()
			} else if log.Address.AddrPrefixString() == p.permConfig.OrgAddress && !isLoadOrg {
				isLoadOrg = true
				Logger.Debug("PermissionCtrl load OrgAddress\n")
				p.populateOrgsFromContract()
			} else if log.Address.AddrPrefixString() == p.permConfig.VoterAddress && !isLoadVote {
				isLoadVote = true
				Logger.Debug("PermissionCtrl load VoterAddress\n")
				p.populateVotesFromContract()
			}
			if isLoadOrg && isLoadAccount && isLoadNode && isLoadVote {
				break
			}
		}
	}
	return nil
}

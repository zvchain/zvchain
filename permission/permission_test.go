package permission

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"os"
	"testing"
)

const (
	arbitraryNetworkAdminOrg  = "NETWORK_ADMIN"
	arbitraryNetworkAdminRole = "NETWORK_ADMIN_ROLE"
	arbitraryOrgAdminRole     = "ORG_ADMIN_ROLE"
	arbitraryNode1            = "enode://ac6b1096ca56b9f6d004b779ae3728bf83f8e22453404cc3cef16a3d9b96608bc67c4b30db88e0a5a6c6390213f7acbe1153ff6d23ce57380104288ae19373ef@127.0.0.1:21000?discport=0&raftport=50401"
	arbitraryNode2            = "enode://0ba6b9f606a43a95edc6247cdb1c1e105145817be7bcafd6b2c0ba15d58145f0dc1a194f70ba73cd6f4cdd6864edc7687f311254c7555cc32e4d45aeb1b80416@127.0.0.1:21001?discport=0&raftport=50402"
	arbitraryOrgToAdd         = "ORG1"
	arbitrarySubOrg           = "SUB1"
	arbitrartNewRole1         = "NEW_ROLE_1"
	arbitrartNewRole2         = "NEW_ROLE_2"
)

var ErrAccountsLinked = errors.New("Accounts linked to the role. Cannot be removed")
var ErrPendingApproval = errors.New("Pending approvals for the organization. Approve first")
var ErrAcctBlacklisted = errors.New("Blacklisted account. Operation not allowed")
var ErrNodeBlacklisted = errors.New("Blacklisted node. Operation not allowed")

const guardianKeyHex = "0xe01cfb4e1c156a795fa6d525fb481727dabcbf066b3a153daf835f5570599a79"

var (
	guardianKey     *common.PrivateKey
	guardianAccount common.PublicKey
	//backend         bind.ContractBackend
	permUpgrAddress, permInterfaceAddress, permImplAddress, voterManagerAddress,
	nodeManagerAddress, roleManagerAddress, accountManagerAddress, orgManagerAddress common.Address
	//ethereum        *eth.Ethereum
	//stack           *node.Node
	guardianAddress common.Address
)

//func TestMain(m *testing.M) {
//	setup(nil)
//	ret := m.Run()
//	teardown()
//	os.Exit(ret)
//}

func TestDeployAllContract(t *testing.T) {
	var err error

	guardianKey = &common.PrivateKey{}
	guardianKey.ImportKey(common.FromHex(guardianKeyHex))
	if guardianKey == nil {
		return
	}
	guardianAccount = guardianKey.GetPubKey()
	guardianAddress = guardianAccount.GetAddress()

	DataSourceType = DataSourceRPC
	config := new(types.PermissionConfig)
	config.NwAdminOrg = arbitraryNetworkAdminOrg
	pc, err := NewPermissionCtrl(config)
	if err != nil {
		return
	}
	//AccountMgr.py
	//NodeMgr.py
	//OrgMgr.py
	//PermissionsImplementation.py
	//PermissionsInterface.py
	//PermissionsUpgradable.py
	//VoteMgr.py
	pc.contractMgr.Sk = guardianKey
	config.AccountAddress = DeployContract(t, pc.contractMgr, "AccountManager", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/AccountMgr.py")
	fmt.Printf("config.AccountAddress=\"%s\"\n", config.AccountAddress)

	config.NodeAddress = DeployContract(t, pc.contractMgr, "NodeManager", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/NodeMgr.py")
	fmt.Printf("config.NodeAddress=\"%s\"\n", config.NodeAddress)

	config.OrgAddress = DeployContract(t, pc.contractMgr, "OrgManager", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/OrgMgr.py")
	fmt.Printf("config.OrgAddress=\"%s\"\n", config.OrgAddress)

	config.VoterAddress = DeployContract(t, pc.contractMgr, "VoteManager", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/VoteMgr.py")
	fmt.Printf("config.VoterAddress=\"%s\"\n", config.VoterAddress)

	config.InterfAddress = DeployContract(t, pc.contractMgr, "PermissionInterface", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/PermissionsInterface.py")
	fmt.Printf("config.InterfAddress=\"%s\"\n", config.InterfAddress)

	config.ImplAddress = DeployContract(t, pc.contractMgr, "PermissionsImplementation", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/PermissionsImplementation.py")
	fmt.Printf("config.ImplAddress=\"%s\"\n", config.ImplAddress)

	config.UpgrdAddress = DeployContract(t, pc.contractMgr, "PermissionsUpgradable", "/Users/lavrock/go/src/github.com/zvchain/zvchain/permission/py/PermissionsUpgradable.py")
	fmt.Printf("config.UpgrdAddress=\"%s\"\n", config.UpgrdAddress)

	//def init(self, _perm_interface,_perm_impl,_perm_upgradeable_addr,_account_mgr_addr, _org_mgr_addr, _vote_mgr_addr, _node_mgr_addr):

	hash := CallContract(t, config.UpgrdAddress, pc.contractMgr, MakeABIString("init",
		config.InterfAddress,
		config.ImplAddress,
		config.UpgrdAddress,
		config.AccountAddress,
		config.OrgAddress,
		config.VoterAddress,
		config.NodeAddress))
	t.Logf("hash:%s", hash)

}

func TestContractsDeployed(t *testing.T) {
	var err error

	guardianKey = &common.PrivateKey{}
	guardianKey.ImportKey(common.FromHex(guardianKeyHex))
	if guardianKey == nil {
		return
	}
	guardianAccount = guardianKey.GetPubKey()
	guardianAddress = guardianAccount.GetAddress()

	config := new(types.PermissionConfig)
	config.NwAdminOrg = arbitraryNetworkAdminOrg
	pc, err := NewPermissionCtrl(config)
	if err != nil {
		return
	}
	initConfig(pc)
	pc.AfterStart()
}

func initConfig(pc *PermissionCtrl) {
	DataSourceType = DataSourceRPC
	pc.contractMgr.Sk = guardianKey
	config := pc.permConfig
	config.AccountAddress = "zvb616e04225e6e8c8bedd38d47048e3086cfa7d20c2d7cfaa928a94aefa71c804"
	config.NodeAddress = "zv19470b4ac4d2066020232ed8fdc5d4f36509200278326d2978d03e7ae02f5e25"
	config.OrgAddress = "zv3bb1431c6ccf163f01aeddcff347679bc44dc9758071958dea30059cc269498f"
	config.VoterAddress = "zvc1a334a8760f13d7e4b59ed795e9af2d8d862b41f22dad8d34251cbf8f8bf81e"
	config.InterfAddress = "zv996d0220cefcf60ca001914a40ae24e21d1932768a443d9168a0a7ec5906bed9"
	config.ImplAddress = "zv72af71d312e039d04072ca18af1a91b4e2de647c8574c2e423f97fed96909a99"
	config.UpgrdAddress = "zv4c8972216a637684280fa24b41f9ba4d93264f123f84820f8c082015dbd67364"
	pc.permConfig.Accounts = make([]common.Address, 0)
	pc.permConfig.Accounts = append(pc.permConfig.Accounts, common.StringToAddress("zve75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"))

}

func TestContractsData(t *testing.T) {
	var err error

	guardianKey = &common.PrivateKey{}
	guardianKey.ImportKey(common.FromHex(guardianKeyHex))
	if guardianKey == nil {
		return
	}
	guardianAccount = guardianKey.GetPubKey()
	guardianAddress = guardianAccount.GetAddress()

	config := new(types.PermissionConfig)
	config.NwAdminOrg = arbitraryNetworkAdminOrg
	pc, err := NewPermissionCtrl(config)
	if err != nil {
		return
	}

	initConfig(pc)

	fmt.Printf("AccountContractAddr data:%s\n", GetAccountData(config.AccountAddress, ""))
	fmt.Printf("NodeContractAddr data:%s\n", GetAccountData(config.NodeAddress, ""))
	fmt.Printf("OrgContractAddr data:%s\n", GetAccountData(config.OrgAddress, ""))
	fmt.Printf("VoteContractAddr data:%s\n", GetAccountData(config.VoterAddress, ""))
	fmt.Printf("InterfaceContractAddr data:%s\n", GetAccountData(config.InterfAddress, ""))
	fmt.Printf("ImplContractAddr data:%s\n", GetAccountData(config.ImplAddress, ""))

	fmt.Printf("UpgradableContractAddr data:%s\n", GetAccountData(config.UpgrdAddress, ""))

}

func DeployContract(t *testing.T, contractMgr *ContractManager, name, file string) string {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("cannot read contract %v file! error:%v", name, err)
	}
	contractData := Contract{string(content), name}
	if err != nil {
		t.Errorf("cannot read contract file! error:%v", err)
	}
	address, status := contractMgr.DeployContract(contractData, DefaultDeployTxArgs, *guardianKey)

	if status != RSSuccess {
		t.Errorf("deploy function contract error,name:%v,state:%v", address, status)
	}

	return address
}

func CallContract(t *testing.T, addr string, contractMgr *ContractManager, abi string) string {
	txArgs := CallTxArgsSetData(abi)
	address, status := contractMgr.CallContract(addr, txArgs)

	if status != RSSuccess {
		t.Errorf("call function contract error")
	}

	return address
}

func TestPermissionCtrl_PopulateInitPermissions_AfterNetworkIsInitialized(t *testing.T) {
	testObject := typicalPermissionCtrl(t)
	assert.NoError(t, testObject.AfterStart())

	err := testObject.populateInitPermissions()

	assert.NoError(t, err)

	// assert cache
	assert.Equal(t, 1, len(types.OrgInfoMap.GetOrgList()))
	cachedOrg := types.OrgInfoMap.GetOrgList()[0]
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedOrg.OrgId)

	assert.Equal(t, types.OrgApproved, cachedOrg.Status)

	assert.Equal(t, 0, len(types.NodeInfoMap.GetNodeList()))

	assert.Equal(t, 1, len(types.AcctInfoMap.GetAcctList()))
	cachedAccount := types.AcctInfoMap.GetAcctList()[0]
	assert.Equal(t, arbitraryNetworkAdminOrg, cachedAccount.OrgId)

	assert.Equal(t, guardianAddress, cachedAccount.AcctId)
}

func typicalPermissionControlsAPI(t *testing.T) *PermissionCtrlAPI {
	pc := typicalPermissionCtrl(t)
	if !assert.NoError(t, pc.AfterStart()) {
		t.Fail()
	}
	if !assert.NoError(t, pc.populateInitPermissions()) {
		t.Fail()
	}
	return NewPermissionCtrlAPI(pc)
}

func TestPermissionCtrlsAPI_ListAPIs(t *testing.T) {
	testObject := typicalPermissionControlsAPI(t)

	orgDetails, err := testObject.GetOrgDetails(arbitraryNetworkAdminOrg)
	assert.NoError(t, err)
	assert.Equal(t, orgDetails.AcctList[0].AcctId, guardianAddress)

	orgDetails, err = testObject.GetOrgDetails("XYZ")
	assert.Equal(t, err, errors.New("org does not exist"))

	// test NodeList
	assert.Equal(t, len(testObject.NodeList()), 0)
	// test AcctList
	assert.True(t, len(testObject.AcctList()) > 0, fmt.Sprintf("expected non zero account list"))
	// test OrgList
	assert.True(t, len(testObject.OrgList()) > 0, fmt.Sprintf("expected non zero org list"))
}

//func TestQuorumControlsAPI_OrgAPIs(t *testing.T) {
//	testObject := typicalQuorumControlsAPI(t)
//	invalidTxa := ethapi.SendTxArgs{From: getArbitraryAccount()}
//	txa := ethapi.SendTxArgs{From: guardianAddress}
//
//	// test AddOrg
//	orgAdminKey, _ := crypto.GenerateKey()
//	orgAdminAddress := crypto.PubkeyToAddress(orgAdminKey.PublicKey)
//
//	_, err := testObject.AddOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.AddOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, txa)
//	assert.NoError(t, err)
//
//	_, err = testObject.AddOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, txa)
//	assert.Equal(t, err, ErrPendingApproval)
//
//	_, err = testObject.ApproveOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.ApproveOrg("XYZ", arbitraryNode1, orgAdminAddress, txa)
//	assert.Equal(t, err, errors.New("Nothing to approve"))
//
//	_, err = testObject.ApproveOrg(arbitraryOrgToAdd, arbitraryNode1, orgAdminAddress, txa)
//	assert.NoError(t, err)
//
//	types.OrgInfoMap.UpsertOrg(arbitraryOrgToAdd, "", arbitraryOrgToAdd, big.NewInt(1), types.OrgApproved)
//	_, err = testObject.UpdateOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.UpdateOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), txa)
//	assert.NoError(t, err)
//
//	types.OrgInfoMap.UpsertOrg(arbitraryOrgToAdd, "", arbitraryOrgToAdd, big.NewInt(1), types.OrgSuspended)
//	_, err = testObject.ApproveOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.ApproveOrgStatus(arbitraryOrgToAdd, uint8(SuspendOrg), txa)
//	assert.NoError(t, err)
//
//	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, arbitrarySubOrg, "", invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, arbitrarySubOrg, "", txa)
//	assert.NoError(t, err)
//	types.OrgInfoMap.UpsertOrg(arbitrarySubOrg, arbitraryNetworkAdminOrg, arbitraryNetworkAdminOrg, big.NewInt(2), types.OrgApproved)
//
//	suborg := "ABC.12345"
//	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, suborg, "", txa)
//	assert.Equal(t, err, errors.New("Org id cannot contain special characters"))
//
//	_, err = testObject.AddSubOrg(arbitraryNetworkAdminOrg, "", "", txa)
//	assert.Equal(t, err, errors.New("Invalid input"))
//
//	_, err = testObject.GetOrgDetails(arbitraryOrgToAdd)
//	assert.NoError(t, err)
//
//}
//
//func TestQuorumControlsAPI_NodeAPIs(t *testing.T) {
//	testObject := typicalQuorumControlsAPI(t)
//	invalidTxa := ethapi.SendTxArgs{From: getArbitraryAccount()}
//	txa := ethapi.SendTxArgs{From: guardianAddress}
//
//	_, err := testObject.AddNode(arbitraryNetworkAdminOrg, arbitraryNode2, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.AddNode(arbitraryNetworkAdminOrg, arbitraryNode2, txa)
//	assert.NoError(t, err)
//	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeApproved)
//
//	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(SuspendNode), invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(SuspendNode), txa)
//	assert.NoError(t, err)
//	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeDeactivated)
//
//	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(ActivateSuspendedNode), txa)
//	assert.NoError(t, err)
//	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeApproved)
//
//	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(BlacklistNode), txa)
//	assert.NoError(t, err)
//	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeBlackListed)
//
//	_, err = testObject.UpdateNodeStatus(arbitraryNetworkAdminOrg, arbitraryNode2, uint8(ActivateSuspendedNode), txa)
//	assert.Equal(t, err, ErrNodeBlacklisted)
//
//	_, err = testObject.RecoverBlackListedNode(arbitraryNetworkAdminOrg, arbitraryNode2, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.RecoverBlackListedNode(arbitraryNetworkAdminOrg, arbitraryNode2, txa)
//	assert.NoError(t, err)
//	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeRecoveryInitiated)
//
//	_, err = testObject.ApproveBlackListedNodeRecovery(arbitraryNetworkAdminOrg, arbitraryNode2, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.ApproveBlackListedNodeRecovery(arbitraryNetworkAdminOrg, arbitraryNode2, txa)
//	assert.NoError(t, err)
//	types.NodeInfoMap.UpsertNode(arbitraryNetworkAdminOrg, arbitraryNode2, types.NodeApproved)
//}
//
//func TestQuorumControlsAPI_RoleAndAccountsAPIs(t *testing.T) {
//	testObject := typicalQuorumControlsAPI(t)
//	invalidTxa := ethapi.SendTxArgs{From: getArbitraryAccount()}
//	txa := ethapi.SendTxArgs{From: guardianAddress}
//	acct := getArbitraryAccount()
//
//	_, err := testObject.AssignAdminRole(arbitraryNetworkAdminOrg, acct, arbitraryNetworkAdminRole, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.AssignAdminRole(arbitraryNetworkAdminOrg, acct, arbitraryNetworkAdminRole, txa)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitraryNetworkAdminRole, acct, true, types.AcctPendingApproval)
//
//	_, err = testObject.ApproveAdminRole(arbitraryNetworkAdminOrg, acct, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.ApproveAdminRole(arbitraryNetworkAdminOrg, acct, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.ApproveAdminRole(arbitraryNetworkAdminOrg, acct, txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitraryNetworkAdminRole, acct, true, types.AcctActive)
//
//	_, err = testObject.AddNewRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, uint8(types.FullAccess), false, false, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.AddNewRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, uint8(types.FullAccess), false, false, txa)
//	assert.NoError(t, err)
//	types.RoleInfoMap.UpsertRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, false, false, types.FullAccess, true)
//
//	acct = getArbitraryAccount()
//	_, err = testObject.AddAccountToOrg(acct, arbitraryNetworkAdminOrg, arbitrartNewRole1, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.AddAccountToOrg(acct, arbitraryNetworkAdminOrg, arbitrartNewRole1, txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole1, acct, true, types.AcctActive)
//
//	_, err = testObject.RemoveRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.RemoveRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, txa)
//	assert.Equal(t, err, ErrAccountsLinked)
//
//	_, err = testObject.AddNewRole(arbitraryNetworkAdminOrg, arbitrartNewRole2, uint8(types.FullAccess), false, false, txa)
//	assert.NoError(t, err)
//	types.RoleInfoMap.UpsertRole(arbitraryNetworkAdminOrg, arbitrartNewRole2, false, false, types.FullAccess, true)
//
//	_, err = testObject.ChangeAccountRole(acct, arbitraryNetworkAdminOrg, arbitrartNewRole2, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.ChangeAccountRole(acct, arbitraryNetworkAdminOrg, arbitrartNewRole2, txa)
//	assert.NoError(t, err)
//
//	_, err = testObject.RemoveRole(arbitraryNetworkAdminOrg, arbitrartNewRole1, txa)
//	assert.Equal(t, err, ErrAccountsLinked)
//
//	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(SuspendAccount), invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(SuspendAccount), txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctSuspended)
//
//	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(ActivateSuspendedAccount), txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctActive)
//
//	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(BlacklistAccount), txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctBlacklisted)
//
//	_, err = testObject.UpdateAccountStatus(arbitraryNetworkAdminOrg, acct, uint8(ActivateSuspendedAccount), txa)
//	assert.Equal(t, err, ErrAcctBlacklisted)
//
//	_, err = testObject.RecoverBlackListedAccount(arbitraryNetworkAdminOrg, acct, invalidTxa)
//	assert.Equal(t, err, errors.New("Invalid account id"))
//
//	_, err = testObject.RecoverBlackListedAccount(arbitraryNetworkAdminOrg, acct, txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctRecoveryInitiated)
//	_, err = testObject.ApproveBlackListedAccountRecovery(arbitraryNetworkAdminOrg, acct, txa)
//	assert.NoError(t, err)
//	types.AcctInfoMap.UpsertAccount(arbitraryNetworkAdminOrg, arbitrartNewRole2, acct, true, types.AcctActive)
//
//}
//
func getArbitraryAccount() common.Address {
	acctKey, _ := common.GenerateKey("")
	return acctKey.GetPubKey().GetAddress()
}

func typicalPermissionCtrl(t *testing.T) *PermissionCtrl {
	guardianKey = &common.PrivateKey{}
	guardianKey.ImportKey(common.FromHex(guardianKeyHex))
	if guardianKey == nil {
		return nil
	}
	guardianAccount = guardianKey.GetPubKey()
	guardianAddress = guardianAccount.GetAddress()

	config := new(types.PermissionConfig)
	config.NwAdminOrg = arbitraryNetworkAdminOrg
	pc, err := NewPermissionCtrl(config)
	if err != nil {
		return nil
	}

	initConfig(pc)
	return pc
}

//func tmpKeyStore(encrypted bool) (string, *keystore.KeyStore, error) {
//	d, err := ioutil.TempDir("", "eth-keystore-test")
//	if err != nil {
//		return "", nil, err
//	}
//	new := keystore.NewPlaintextKeyStore
//	if encrypted {
//		new = func(kd string) *keystore.KeyStore {
//			return keystore.NewKeyStore(kd, keystore.LightScryptN, keystore.LightScryptP)
//		}
//	}
//	return d, new(d), err
//}
//
//func TestPermissionCtrl_whenUpdateFile(t *testing.T) {
//	testObject := typicalPermissionCtrl(t)
//	assert.NoError(t, testObject.AfterStart())
//
//	err := testObject.populateInitPermissions()
//	assert.NoError(t, err)
//
//	d, _ := ioutil.TempDir("", "qdata")
//	defer os.RemoveAll(d)
//
//	testObject.dataDir = d
//	testObject.updatePermissionedNodes(arbitraryNode1, NodeAdd)
//
//	permFile, _ := os.Create(d + "/" + "permissioned-nodes.json")
//
//	testObject.updateFile("testFile", arbitraryNode2, NodeAdd, false)
//	testObject.updateFile(permFile.Name(), arbitraryNode2, NodeAdd, false)
//	testObject.updateFile(permFile.Name(), arbitraryNode2, NodeAdd, true)
//	testObject.updateFile(permFile.Name(), arbitraryNode2, NodeAdd, true)
//	testObject.updateFile(permFile.Name(), arbitraryNode1, NodeAdd, false)
//	testObject.updateFile(permFile.Name(), arbitraryNode1, NodeDelete, false)
//	testObject.updateFile(permFile.Name(), arbitraryNode1, NodeDelete, false)
//
//	blob, err := ioutil.ReadFile(permFile.Name())
//	var nodeList []string
//	if err := json.Unmarshal(blob, &nodeList); err != nil {
//		t.Fatal("Failed to load nodes list from file", "fileName", permFile, "err", err)
//		return
//	}
//	assert.Equal(t, len(nodeList), 1)
//	testObject.updatePermissionedNodes(arbitraryNode1, NodeAdd)
//	testObject.updatePermissionedNodes(arbitraryNode1, NodeDelete)
//
//	blob, err = ioutil.ReadFile(permFile.Name())
//	if err := json.Unmarshal(blob, &nodeList); err != nil {
//		t.Fatal("Failed to load nodes list from file", "fileName", permFile, "err", err)
//		return
//	}
//	assert.Equal(t, len(nodeList), 1)
//
//	testObject.updateDisallowedNodes(arbitraryNode2, NodeAdd)
//	testObject.updateDisallowedNodes(arbitraryNode2, NodeDelete)
//	blob, err = ioutil.ReadFile(d + "/" + "disallowed-nodes.json")
//	if err := json.Unmarshal(blob, &nodeList); err != nil {
//		t.Fatal("Failed to load nodes list from file", "fileName", permFile, "err", err)
//		return
//	}
//	assert.Equal(t, len(nodeList), 0)
//
//}
//

func TestPermissionConfig(t *testing.T) {

	_, err := ParsePermissionConfig("./perm_config.json")
	assert.True(t, err != nil, "expected file not there error")
	pc := typicalPermissionCtrl(t)
	initConfig(pc)

}

func TestParsePermissionConfig(t *testing.T) {
	d, _ := ioutil.TempDir("", "./qdata")
	defer os.RemoveAll(d)

	_, err := ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected file not there error")

	fileName := d + "/permission-config.json"
	_, err = os.Create(fileName)
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected unmarshalling error")

	// write permission-config.json into the temp dir
	var tmpPermCofig types.PermissionConfig
	tmpPermCofig.NwAdminOrg = arbitraryNetworkAdminOrg

	pc := typicalPermissionCtrl(t)
	pc.permConfig = &tmpPermCofig
	initConfig(pc)
	blob, err := json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	_, err = ParsePermissionConfig(d)

	assert.True(t, err != nil, "expected sub org depth not set error")

	_ = os.Remove(fileName)
	blob, _ = json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected account not given  error")

	_ = os.Remove(fileName)
	tmpPermCofig.Accounts = append(tmpPermCofig.Accounts, common.StringToAddress("zve75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"))
	blob, err = json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	_, err = ParsePermissionConfig(d)
	assert.True(t, err != nil, "expected contract address error")

	_ = os.Remove(fileName)
	tmpPermCofig.InterfAddress = "zve75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"
	blob, err = json.Marshal(tmpPermCofig)
	if err := ioutil.WriteFile(fileName, blob, 0644); err != nil {
		t.Fatal("Error writing new node info to file", "fileName", fileName, "err", err)
	}
	permConfig, err := ParsePermissionConfig(d)
	assert.False(t, permConfig.IsEmpty(), "expected non empty object")
}

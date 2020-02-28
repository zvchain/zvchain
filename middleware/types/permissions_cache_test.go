package types

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"strconv"
	"testing"

	testifyAssert "github.com/stretchr/testify/assert"
)

var (
	NETWORKADMIN = "NWADMIN"
	ORGADMIN     = "OADMIN"
	NODE1        = "zv996d0220cefcf60ca001914a40ae24e21d1932768a443d9168a0a7ec5906bed9"
	NODE2        = "zv3bb1431c6ccf163f01aeddcff347679bc44dc9758071958dea30059cc269498f"
)

var Acct1 = common.StringToAddress("zv996d0220cefcf60ca001914a40ae24e21d1932768a443d9168a0a7ec5906bed9")
var Acct2 = common.StringToAddress("zv3bb1431c6ccf163f01aeddcff347679bc44dc9758071958dea30059cc269498f")

func TestSetSyncStatus(t *testing.T) {
	assert := testifyAssert.New(t)

	SetSyncStatus()

	// check if the value is set properly by calling Get
	syncStatus := GetSyncStatus()
	assert.True(syncStatus == true, fmt.Sprintf("Expected syncstatus %v . Got %v ", true, syncStatus))
}

func TestSetDefaults(t *testing.T) {
	assert := testifyAssert.New(t)

	// get the default values and confirm the same
	defaultAccess := GetDefaults()

	assert.True(defaultAccess == FullAccess, fmt.Sprintf("Expected network admin role %v, got %v", FullAccess, defaultAccess))

	SetDefaultAccess()
	defaultAccess = GetDefaults()
	assert.True(defaultAccess == ReadOnly, fmt.Sprintf("Expected network admin role %v, got %v", ReadOnly, defaultAccess))
}

func TestOrgCache_UpsertOrg(t *testing.T) {
	assert := testifyAssert.New(t)

	//add a org and get the org details
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgApproved)
	orgInfo := OrgInfoMap.GetOrg(NETWORKADMIN)

	assert.False(orgInfo == nil, fmt.Sprintf("Expected org details, got nil"))
	assert.True(orgInfo.OrgId == NETWORKADMIN, fmt.Sprintf("Expected org id %v, got %v", NETWORKADMIN, orgInfo.OrgId))

	// update org status to suspended
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgSuspended)
	orgInfo = OrgInfoMap.GetOrg(NETWORKADMIN)

	assert.True(orgInfo.Status == OrgSuspended, fmt.Sprintf("Expected org status %v, got %v", OrgSuspended, orgInfo.Status))

	//add another org and check get org list
	OrgInfoMap.UpsertOrg(ORGADMIN, OrgApproved)
	orgList := OrgInfoMap.GetOrgList()
	assert.True(len(orgList) == 2, fmt.Sprintf("Expected 2 entries, got %v", len(orgList)))

	//add sub org and check get orglist
	OrgInfoMap.UpsertOrg("SUB1", OrgApproved)
	orgList = OrgInfoMap.GetOrgList()
	assert.True(len(orgList) == 3, fmt.Sprintf("Expected 3 entries, got %v", len(orgList)))

	//suspend the sub org and check get orglist
	OrgInfoMap.UpsertOrg("SUB1", OrgSuspended)
	orgList = OrgInfoMap.GetOrgList()
	assert.True(len(orgList) == 3, fmt.Sprintf("Expected 3 entries, got %v", len(orgList)))
}

func TestNodeCache_UpsertNode(t *testing.T) {
	assert := testifyAssert.New(t)

	// add a node into the cache and validate
	NodeInfoMap.UpsertNode(NETWORKADMIN, NODE1, NodeApproved)
	nodeInfo := NodeInfoMap.GetNodeById(NODE1)
	assert.False(nodeInfo == nil, fmt.Sprintf("Expected node details, got nil"))
	assert.True(nodeInfo.OrgId == NETWORKADMIN, fmt.Sprintf("Expected org id for node %v, got %v", NETWORKADMIN, nodeInfo.OrgId))
	assert.True(nodeInfo.NodeId == NODE1, fmt.Sprintf("Expected node id %v, got %v", NODE1, nodeInfo.NodeId))

	// add another node and validate the list function
	NodeInfoMap.UpsertNode(ORGADMIN, NODE2, NodeApproved)
	nodeList := NodeInfoMap.GetNodeList()
	assert.True(len(nodeList) == 2, fmt.Sprintf("Expected 2 entries, got %v", len(nodeList)))

	// check node details update by updating node status
	NodeInfoMap.UpsertNode(ORGADMIN, NODE2, NodeDeactivated)
	nodeInfo = NodeInfoMap.GetNodeById(NODE2)
	assert.True(nodeInfo.Status == NodeDeactivated, fmt.Sprintf("Expected node status %v, got %v", NodeDeactivated, nodeInfo.Status))
}

func TestAcctCache_UpsertAccount(t *testing.T) {
	assert := testifyAssert.New(t)

	// add an account into the cache and validate
	AcctInfoMap.UpsertAccount(NETWORKADMIN, Acct1, true, true, FullAccess, AcctActive)
	acctInfo := AcctInfoMap.GetAccount(Acct1)
	assert.False(acctInfo == nil, fmt.Sprintf("Expected account details, got nil"))
	assert.True(acctInfo.OrgId == NETWORKADMIN, fmt.Sprintf("Expected org id for the account to be %v, got %v", NETWORKADMIN, acctInfo.OrgId))
	assert.True(acctInfo.AcctId == Acct1, fmt.Sprintf("Expected account id %x, got %x", Acct1, acctInfo.AcctId))

	// add a second account and validate the list function
	AcctInfoMap.UpsertAccount(ORGADMIN, Acct2, true, true, FullAccess, AcctActive)
	acctList := AcctInfoMap.GetAcctList()
	assert.True(len(acctList) == 2, fmt.Sprintf("Expected 2 entries, got %v", len(acctList)))

	// update account status and validate
	AcctInfoMap.UpsertAccount(ORGADMIN, Acct2, true, true, FullAccess, AcctBlacklisted)
	acctInfo = AcctInfoMap.GetAccount(Acct2)
	assert.True(acctInfo.Status == AcctBlacklisted, fmt.Sprintf("Expected account status to be %v, got %v", AcctBlacklisted, acctInfo.Status))

	// validate the list for org and role functions
	acctList = AcctInfoMap.GetAcctListOrg(NETWORKADMIN)
	assert.True(len(acctList) == 1, fmt.Sprintf("Expected number of accounts for the org to be 1, got %v", len(acctList)))

}

func TestGetAcctAccess(t *testing.T) {
	assert := testifyAssert.New(t)

	// default access when the cache is not populated, should return default access
	SetDefaultAccess()
	access := GetAcctAccess(Acct1)
	assert.True(access == ReadOnly, fmt.Sprintf("Expected account access to be %v, got %v", ReadOnly, access))

	// Create an org with two roles and two accounts linked to different roles. Validate account access
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgApproved)
	AcctInfoMap.UpsertAccount(NETWORKADMIN, Acct1, true, true, FullAccess, AcctActive)
	AcctInfoMap.UpsertAccount(NETWORKADMIN, Acct2, true, true, ReadOnly, AcctActive)

	access = GetAcctAccess(Acct1)
	assert.True(access == FullAccess, fmt.Sprintf("Expected account access to be %v, got %v", FullAccess, access))

	// mark the org as pending suspension. The account access should not change
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgPendingSuspension)
	access = GetAcctAccess(Acct1)
	assert.True(access == FullAccess, fmt.Sprintf("Expected account access to be %v, got %v", FullAccess, access))

	// suspend the org and the account access should be readonly now
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgSuspended)
	access = GetAcctAccess(Acct1)
	assert.True(access == ReadOnly, fmt.Sprintf("Expected account access to be %v, got %v", ReadOnly, access))

}

func TestValidateNodeForTxn(t *testing.T) {
	assert := testifyAssert.New(t)
	// pass the enode as null and the response should be true
	txnAllowed := ValidateNodeForTxn("", Acct1)
	assert.True(txnAllowed == true, "Expected access %v, got %v", true, txnAllowed)

	SetDefaultAccess()

	// if a proper enode id is not passed, return should be false
	txnAllowed = ValidateNodeForTxn("ABCDE", Acct1)
	assert.True(txnAllowed == false, "Expected access %v, got %v", true, txnAllowed)

	// if cache is not populated but the enode and account details are proper,
	// should return true
	txnAllowed = ValidateNodeForTxn(NODE1, Acct1)
	assert.True(txnAllowed == true, "Expected access %v, got %v", true, txnAllowed)

	// populate an org, account and node. validate access
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgApproved)
	NodeInfoMap.UpsertNode(NETWORKADMIN, NODE1, NodeApproved)
	AcctInfoMap.UpsertAccount(NETWORKADMIN, Acct1, true, true, FullAccess, AcctActive)
	txnAllowed = ValidateNodeForTxn(NODE1, Acct1)
	assert.True(txnAllowed == true, "Expected access %v, got %v", true, txnAllowed)

	// test access from a node not linked to the org. should return false
	OrgInfoMap.UpsertOrg(ORGADMIN, OrgApproved)
	NodeInfoMap.UpsertNode(ORGADMIN, NODE2, NodeApproved)
	AcctInfoMap.UpsertAccount(ORGADMIN, Acct2, true, true, ReadOnly, AcctActive)
	txnAllowed = ValidateNodeForTxn(NODE1, Acct2)
	assert.True(txnAllowed == false, "Expected access %v, got %v", true, txnAllowed)
}

// This is to make sure enode.ParseV4() honors single hexNodeId value eventhough it does follow enode URI scheme
func TestValidateNodeForTxn_whenUsingOnlyHexNodeId(t *testing.T) {
	OrgInfoMap.UpsertOrg(NETWORKADMIN, OrgApproved)
	NodeInfoMap.UpsertNode(NETWORKADMIN, NODE1, NodeApproved)
	AcctInfoMap.UpsertAccount(NETWORKADMIN, Acct1, true, true, ReadOnly, AcctActive)
	arbitraryPrivateKey, _ := common.GenerateKey("")
	hexNodeId := arbitraryPrivateKey.GetPubKey().GetAddress().String()

	SetDefaultAccess()

	txnAllowed := ValidateNodeForTxn(hexNodeId, Acct1)

	testifyAssert.False(t, txnAllowed)
}

// test the cache limit
func TestLRUCacheLimit(t *testing.T) {
	for i := 0; i < defaultOrgMapLimit; i++ {
		orgName := "ORG" + strconv.Itoa(i)
		OrgInfoMap.UpsertOrg(orgName, OrgApproved)
	}

	o := OrgInfoMap.GetOrg("ORG1")
	testifyAssert.True(t, o != nil)
}

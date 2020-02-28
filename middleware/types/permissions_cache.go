package types

import (
	"encoding/json"
	"github.com/zvchain/zvchain/common"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

const (
	MODEL_CONFIG      = "permission-config.json"
	NETWORK_ADMIN_ORG = "NETWORK_ADMIN"
)

type AccessType uint8

const (
	ReadOnly AccessType = iota
	Transact
	ContractDeploy
	FullAccess
)

type OrgStatus uint8

const (
	OrgPendingApproval OrgStatus = iota + 1
	OrgApproved
	OrgPendingSuspension
	OrgSuspended
)

type NodeStatus uint8

const (
	NodePendingApproval NodeStatus = iota + 1
	NodeApproved
	NodeDeactivated
	NodeBlackListed
	NodeRecoveryInitiated
)

type AcctStatus uint8

const (
	AcctPendingApproval AcctStatus = iota + 1
	AcctActive
	AcctInactive
	AcctSuspended
	AcctBlacklisted
	AdminRevoked
	AcctRecoveryInitiated
	AcctRecoveryCompleted
)

type OrgInfo struct {
	OrgId string `json:"org_id"`

	Status OrgStatus `json:"status"`
}

type OrgInfoList struct {
	orgs []OrgInfo `json:"org_list"`
}

type NodeInfo struct {
	OrgId  string     `json:"org_id"`
	NodeId string     `json:"enode_id"`
	Status NodeStatus `json:"status"`
}

type VoteInfo struct {
	OrgId         string `json:"org_id"`
	NodeId        string `json:"node_id"`
	Account       string `json:"account"`
	OpType        int    `json:"op_type"`
	VotedAccounts string `json:"voted_accounts"`
	VotedCounts   int    `json:"voted_counts"`
	Passed        bool   `json:"passed"`
}

type AccountInfo struct {
	OrgId   string         `json:"org_id"`
	Account common.Address `json:"account"`
	IsAdmin bool           `json:"is_org_admin"`
	IsVoter bool           `json:"is_voter"`
	Access  AccessType     `json:"access"`
	Status  AcctStatus     `json:"status"`
}

type OrgDetailInfo struct {
	NodeList []NodeInfo    `json:"nodeList"`
	AcctList []AccountInfo `json:"acctList"`
}

// permission config for bootstrapping
type PermissionConfig struct {
	UpgrdAddress   string `json:"upgrdableAddress"`
	InterfAddress  string `json:"interfaceAddress"`
	ImplAddress    string `json:"implAddress"`
	NodeAddress    string `json:"nodeMgrAddress"`
	AccountAddress string `json:"accountMgrAddress"`
	VoterAddress   string `json:"voterMgrAddress"`
	OrgAddress     string `json:"orgMgrAddress"`

	NwAdminOrg string `json:"nwAdminOrg"`

	Accounts []common.Address `json:"accounts"` //initial list of account that need full access
}

type OrgKey struct {
	OrgId string
}

type NodeKey struct {
	OrgId  string
	NodeId string
}

type AccountKey struct {
	AcctId common.Address
}

type OrgCache struct {
	c   *lru.Cache
	mux sync.Mutex
}

type NodeCache struct {
	c *lru.Cache
}

type AcctCache struct {
	c *lru.Cache
}

type VoteCache struct {
	c *lru.Cache
}

func NewOrgCache() *OrgCache {
	c, _ := lru.New(defaultOrgMapLimit)
	return &OrgCache{c, sync.Mutex{}}
}

func NewNodeCache() *NodeCache {
	c, _ := lru.New(defaultNodeMapLimit)
	return &NodeCache{c}
}

func NewAcctCache() *AcctCache {
	c, _ := lru.New(defaultAccountMapLimit)
	return &AcctCache{c}
}

func NewVoteCache() *VoteCache {
	c, _ := lru.New(defaultAccountMapLimit)
	return &VoteCache{c}
}

var syncStarted = false

var DefaultAccess = FullAccess

const defaultOrgMapLimit = 2000
const defaultNodeMapLimit = 1000
const defaultAccountMapLimit = 6000
const defaultVoteMapLimit = 6000

var OrgInfoMap = NewOrgCache()
var NodeInfoMap = NewNodeCache()
var AcctInfoMap = NewAcctCache()
var VoteInfoMap = NewVoteCache()

func (pc *PermissionConfig) IsEmpty() bool {
	return pc.InterfAddress == ""
}

func SetSyncStatus() {
	syncStarted = true
}

func GetSyncStatus() bool {
	return syncStarted
}

// sets the default access to Readonly
func SetDefaultAccess() {
	DefaultAccess = ReadOnly
}

func GetDefaults() AccessType {
	return DefaultAccess
}

func (o *OrgCache) UpsertOrg(orgId string, status OrgStatus) {
	defer o.mux.Unlock()
	o.mux.Lock()
	var key OrgKey

	key = OrgKey{orgId}

	norg := &OrgInfo{orgId, status}
	o.c.Add(key, norg)
}

func containsKey(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (o *OrgCache) GetOrg(orgId string) *OrgInfo {
	defer o.mux.Unlock()
	o.mux.Lock()
	key := OrgKey{OrgId: orgId}
	if ent, ok := o.c.Get(key); ok {
		return ent.(*OrgInfo)
	}
	return nil
}

func (o *OrgCache) GetOrgList() []OrgInfo {
	olist := make([]OrgInfo, len(o.c.Keys()))
	for i, k := range o.c.Keys() {
		v, _ := o.c.Get(k)
		vp := v.(*OrgInfo)
		olist[i] = *vp
	}
	return olist
}

func (n *NodeCache) UpsertNode(orgId string, NodeId string, status NodeStatus) {
	key := NodeKey{OrgId: orgId, NodeId: NodeId}
	n.c.Add(key, &NodeInfo{orgId, NodeId, status})
}

func (n *NodeCache) GetNodeById(id string) *NodeInfo {
	for _, k := range n.c.Keys() {
		ent := k.(NodeKey)
		if ent.NodeId == id {
			v, _ := n.c.Get(ent)
			return v.(*NodeInfo)
		}
	}
	return nil
}

func (n *NodeCache) GetNodeList() []NodeInfo {
	olist := make([]NodeInfo, len(n.c.Keys()))
	for i, k := range n.c.Keys() {
		v, _ := n.c.Get(k)
		vp := v.(*NodeInfo)
		olist[i] = *vp
	}
	return olist
}

func (v *VoteCache) AddVote(vote *VoteInfo) {
	v.c.Add(vote, vote)
}

func (v *VoteCache) Clear() {
	c, _ := lru.New(defaultVoteMapLimit)

	v.c = c
}

func (v *VoteCache) GetVoteList() []VoteInfo {
	vlist := make([]VoteInfo, len(v.c.Keys()))
	for i, k := range v.c.Keys() {
		v, _ := v.c.Get(k)
		vp := v.(*VoteInfo)
		vlist[i] = *vp
	}
	return vlist
}

func (a *AcctCache) UpsertAccount(orgId string, acct common.Address, isAdmin bool, isVoter bool, access AccessType, status AcctStatus) {
	key := AccountKey{acct}
	a.c.Add(key, &AccountInfo{orgId, acct, isAdmin, isVoter, access, status})
}

func (a *AcctCache) GetAccount(acct common.Address) *AccountInfo {
	if v, ok := a.c.Get(AccountKey{acct}); ok {
		return v.(*AccountInfo)
	}
	return nil
}

func (a *AcctCache) GetAcctList() []AccountInfo {
	alist := make([]AccountInfo, len(a.c.Keys()))
	for i, k := range a.c.Keys() {
		v, _ := a.c.Get(k)
		vp := v.(*AccountInfo)
		alist[i] = *vp
	}
	return alist
}

func (a *AcctCache) GetAcctListOrg(orgId string) []AccountInfo {
	var alist []AccountInfo
	for _, k := range a.c.Keys() {
		v, _ := a.c.Get(k)
		vp := v.(*AccountInfo)
		if vp.OrgId == orgId {
			alist = append(alist, *vp)
		}
	}
	return alist
}

// Returns the access type for an account. If not found returns
// default access
func GetAcctAccess(acctId common.Address) AccessType {

	// check if the org status is fine to do the transaction
	a := AcctInfoMap.GetAccount(acctId)
	if a != nil && a.Status == AcctActive {
		// get the org details and ultimate org details. check org status
		// if the org is not approved or pending suspension
		o := OrgInfoMap.GetOrg(a.OrgId)
		if o != nil && (o.Status == OrgApproved || o.Status == OrgPendingSuspension) {
			if a.IsAdmin {
				return FullAccess
			}
			return a.Access
		}
	}
	return DefaultAccess
}

func ValidateNodeForTxn(nodeId string, from common.Address) bool {

	ac := AcctInfoMap.GetAccount(from)
	if ac == nil {
		return true
	}

	// scan through the node list and validate
	for _, n := range NodeInfoMap.GetNodeList() {
		if n.OrgId == ac.OrgId {
			if nodeId == n.NodeId {
				return true
			}
		}
	}
	return false
}

// function reads the permissions config file passed and populates the
// config structure accordingly
func ParsePermissionConfig(dir string) (*PermissionConfig, error) {
	fullPath := filepath.Join(dir, MODEL_CONFIG)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	permConfig := &PermissionConfig{}
	blob, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(blob, &permConfig)
	if err != nil {
		return nil, err
	}

	return permConfig, nil
}

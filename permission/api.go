package permission

import (
	"errors"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"regexp"
)

var isStringAlphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`).MatchString

//default gas limit to use if not passed in sendTxArgs
var defaultGasLimit = uint64(4712384)

//default gas price to use if not passed in sendTxArgs
var defaultGasPrice = big.NewInt(0)

// PermAction represents actions in permission contract
type PermAction int

const (
	AddOrg PermAction = iota
	ApproveOrg
	UpdateOrgStatus
	ApproveOrgStatus
	AddNode
	UpdateNodeStatus
	AssignAdminRole
	ApproveAdminRole
	AddAccountToOrg
	ChangeAccountAccess
	UpdateAccountStatus
	InitiateNodeRecovery
	InitiateAccountRecovery
	ApproveNodeRecovery
	ApproveAccountRecovery
)

type AccountUpdateAction int

const (
	SuspendAccount AccountUpdateAction = iota + 1
	ActivateSuspendedAccount
	BlacklistAccount
	RecoverBlacklistedAccount
	ApproveBlacklistedAccountRecovery
)

type NodeUpdateAction int

const (
	SuspendNode NodeUpdateAction = iota + 1
	ActivateSuspendedNode
	BlacklistNode
	RecoverBlacklistedNode
	ApproveBlacklistedNodeRecovery
)

type OrgUpdateAction int

const (
	SuspendOrg OrgUpdateAction = iota + 1
	ActivateSuspendedOrg
)

// PermissionCtrlAPI provides an API to access Quorum's node permission and org key management related services
type PermissionCtrlAPI struct {
	permCtrl *PermissionCtrl
}

// txArgs holds arguments required for execute functions
type txArgs struct {
	orgId      string
	porgId     string
	url        string
	roleId     string
	isVoter    bool
	isAdmin    bool
	acctId     string
	accessType uint8
	action     uint8
	voter      string
	morgId     string
	tmKey      string
}

type PendingOpInfo struct {
	PendingKey string `json:"pendingKey"`
	PendingOp  string `json:"pendingOp"`
}

type ExecStatus struct {
	Status bool   `json:"status"`
	Msg    string `json:"msg"`
}

func (e ExecStatus) OpStatus() (string, error) {
	if e.Status {
		return e.Msg, nil
	}
	return "", fmt.Errorf("%s", e.Msg)
}

var (
	ErrNotNetworkAdmin    = ExecStatus{false, "Operation can be performed by network admin only. Account not a network admin."}
	ErrNotOrgAdmin        = ExecStatus{false, "Operation can be performed by org admin only. Account not a org admin."}
	ErrNodePresent        = ExecStatus{false, "EnodeId already part of network."}
	ErrInvalidNode        = ExecStatus{false, "Invalid enode id"}
	ErrInvalidAccount     = ExecStatus{false, "Invalid account id"}
	ErrOrgExists          = ExecStatus{false, "Org already exists"}
	ErrPendingApprovals   = ExecStatus{false, "Pending approvals for the organization. Approve first"}
	ErrNothingToApprove   = ExecStatus{false, "Nothing to approve"}
	ErrOpNotAllowed       = ExecStatus{false, "Operation not allowed"}
	ErrNodeOrgMismatch    = ExecStatus{false, "Enode id passed does not belong to the organization."}
	ErrBlacklistedNode    = ExecStatus{false, "Blacklisted node. Operation not allowed"}
	ErrBlacklistedAccount = ExecStatus{false, "Blacklisted account. Operation not allowed"}
	ErrAccountOrgAdmin    = ExecStatus{false, "Account already org admin for the org"}
	ErrOrgAdminExists     = ExecStatus{false, "Org admin exists for the org"}
	ErrAccountInUse       = ExecStatus{false, "Account already in use in another organization"}
	ErrRoleExists         = ExecStatus{false, "Role exists for the org"}
	ErrRoleActive         = ExecStatus{false, "Accounts linked to the role. Cannot be removed"}
	ErrAdminRoles         = ExecStatus{false, "Admin role cannot be removed"}
	ErrInvalidOrgName     = ExecStatus{false, "Org id cannot contain special characters"}
	ErrInvalidParentOrg   = ExecStatus{false, "Invalid parent org id"}
	ErrAccountNotThere    = ExecStatus{false, "Account does not exists"}
	ErrOrgNotOwner        = ExecStatus{false, "Account does not belong to this org"}
	ErrMaxDepth           = ExecStatus{false, "Max depth for sub orgs reached"}
	ErrMaxBreadth         = ExecStatus{false, "Max breadth for sub orgs reached"}
	ErrNodeDoesNotExists  = ExecStatus{false, "Node does not exists"}
	ErrOrgDoesNotExists   = ExecStatus{false, "Org does not exists"}
	ErrInactiveRole       = ExecStatus{false, "Role is already inactive"}
	ErrInvalidRole        = ExecStatus{false, "Invalid role"}
	ErrInvalidInput       = ExecStatus{false, "Invalid input"}
	ErrNotMasterOrg       = ExecStatus{false, "Org is not a master org"}

	ExecSuccess = ExecStatus{true, "Action completed successfully"}
)

// NewPermissionCtrlAPI creates a new PermissionCtrlAPI to access quorum services
func NewPermissionCtrlAPI(p *PermissionCtrl) *PermissionCtrlAPI {
	return &PermissionCtrlAPI{p}
}

func (q *PermissionCtrlAPI) Namespace() string {
	return "Perm"
}

func (q *PermissionCtrlAPI) Version() string {
	return "1"
}

func (q *PermissionCtrlAPI) OrgList() []types.OrgInfo {
	return types.OrgInfoMap.GetOrgList()
}

func (q *PermissionCtrlAPI) NodeList() []types.NodeInfo {
	return types.NodeInfoMap.GetNodeList()
}

func (q *PermissionCtrlAPI) AccountList() []types.AccountInfo {
	return types.AcctInfoMap.GetAcctList()
}

func (q *PermissionCtrlAPI) VoteList() []types.VoteInfo {
	return types.VoteInfoMap.GetVoteList()
}

func (q *PermissionCtrlAPI) GetOrgDetails(orgId string) (types.OrgDetailInfo, error) {
	if o := types.OrgInfoMap.GetOrg(orgId); o == nil {
		return types.OrgDetailInfo{}, errors.New("org does not exist")
	}
	var acctList []types.AccountInfo
	var nodeList []types.NodeInfo
	for _, a := range q.AccountList() {
		if a.OrgId == orgId {
			acctList = append(acctList, a)
		}
	}

	for _, a := range q.NodeList() {
		if a.OrgId == orgId {
			nodeList = append(nodeList, a)
		}
	}
	return types.OrgDetailInfo{NodeList: nodeList, AcctList: acctList}, nil
}

func reportExecError(action PermAction, err error) (string, error) {
	Logger.Debug("Failed to execute permission action", "action", action, "err", err)
	msg := fmt.Sprintf("failed to execute permissions action: %v", err)
	return ExecStatus{false, msg}.OpStatus()
}

func (q *PermissionCtrlAPI) AddOrg(orgId string, url string, acct string) (string, error) {
	args := txArgs{orgId: orgId, url: url, acctId: acct}

	if execStatus := q.valAddOrg(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	tx, err := q.permCtrl.contractMgr.AddOrg(args.orgId, args.url, args.acctId)
	if err != nil {
		return reportExecError(AddOrg, err)
	}
	Logger.Debug("executed permission action", "action", AddOrg, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) ApproveOrg(orgId string, url string, acct string) (string, error) {

	args := txArgs{orgId: orgId, url: url, acctId: acct}
	if execStatus := q.valApproveOrg(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	tx, err := q.permCtrl.contractMgr.ApproveOrg(args.orgId, args.url, args.acctId)
	if err != nil {
		return reportExecError(ApproveOrg, err)
	}
	Logger.Debug("executed permission action", "action", ApproveOrg, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) UpdateOrgStatus(oId string, status uint8) (string, error) {
	args := txArgs{orgId: oId, action: status}
	if execStatus := q.valUpdateOrgStatus(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	// and in suspended state for suspension revoke
	tx, err := q.permCtrl.contractMgr.UpdateOrgStatus(args.orgId, big.NewInt(int64(args.action)))
	if err != nil {
		return reportExecError(UpdateOrgStatus, err)
	}
	Logger.Debug("executed permission action", "action", UpdateOrgStatus, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) AddNode(orgId string, url string) (string, error) {

	args := txArgs{orgId: orgId, url: url}
	if execStatus := q.valAddNode(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	// check if node is already there
	tx, err := q.permCtrl.contractMgr.AddNode(args.orgId, args.url)
	if err != nil {
		return reportExecError(AddNode, err)
	}
	Logger.Debug("executed permission action", "action", AddNode, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) UpdateNodeStatus(orgId string, url string, action uint8) (string, error) {

	args := txArgs{orgId: orgId, url: url, action: action}
	if execStatus := q.valUpdateNodeStatus(args, UpdateNodeStatus); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	// check node status for operation
	tx, err := q.permCtrl.contractMgr.UpdateNodeStatus(args.orgId, args.url, big.NewInt(int64(args.action)))
	if err != nil {
		return reportExecError(UpdateNodeStatus, err)
	}
	Logger.Debug("executed permission action", "action", UpdateNodeStatus, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) ApproveOrgStatus(orgId string, status uint8) (string, error) {

	args := txArgs{orgId: orgId, action: status}
	if execStatus := q.valApproveOrgStatus(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	// validate that status change is pending approval
	tx, err := q.permCtrl.contractMgr.ApproveOrgStatus(args.orgId, big.NewInt(int64(args.action)))
	if err != nil {
		return reportExecError(ApproveOrgStatus, err)
	}
	Logger.Debug("executed permission action", "action", ApproveOrgStatus, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) AssignAdmin(orgId string, acct string, roleId string) (string, error) {

	args := txArgs{orgId: orgId, acctId: acct, roleId: roleId}
	if execStatus := q.valAssignAdmin(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	// check if account is already in use in another org
	tx, err := q.permCtrl.contractMgr.AssignAdmin(args.orgId, args.acctId)
	if err != nil {
		return reportExecError(AssignAdminRole, err)
	}
	Logger.Debug("executed permission action", "action", AssignAdminRole, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) ApproveAdmin(orgId string, acct string) (string, error) {

	args := txArgs{orgId: orgId, acctId: acct}
	if execStatus := q.valApproveAdmin(args); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	// check if anything is pending approval
	tx, err := q.permCtrl.contractMgr.ApproveAdmin(args.orgId, args.acctId)
	if err != nil {
		return reportExecError(ApproveAdminRole, err)
	}
	Logger.Debug("executed permission action", "action", ApproveAdminRole, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) AddAccount(acct string, orgId string, access uint8, is_admin bool) (string, error) {

	args := txArgs{orgId: orgId, accessType: access, acctId: acct, isAdmin: is_admin}

	tx, err := q.permCtrl.contractMgr.AddAccount(args.acctId, args.orgId, big.NewInt(int64(args.accessType)), is_admin)
	if err != nil {
		return reportExecError(AddAccountToOrg, err)
	}
	Logger.Debug("executed permission action", "action", AddAccountToOrg, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) UpdateAccountAccess(acct string, orgId string, access uint8) (string, error) {

	tx, err := q.permCtrl.contractMgr.UpdateAccountAccess(acct, orgId, big.NewInt(int64(access)))
	if err != nil {
		return reportExecError(ChangeAccountAccess, err)
	}
	Logger.Debug("executed permission action", "action", ChangeAccountAccess, "tx", tx)
	return ExecSuccess.OpStatus()
}

func (q *PermissionCtrlAPI) UpdateAccountStatus(orgId string, acct string, status uint8) (string, error) {

	args := txArgs{orgId: orgId, acctId: acct, action: status}

	if execStatus := q.valUpdateAccountStatus(args, UpdateAccountStatus); execStatus != ExecSuccess {
		return execStatus.OpStatus()
	}
	tx, err := q.permCtrl.contractMgr.UpdateAccountStatus(args.orgId, args.acctId, big.NewInt(int64(args.action)))
	if err != nil {
		return reportExecError(UpdateAccountStatus, err)
	}
	Logger.Debug("executed permission action", "action", UpdateAccountStatus, "tx", tx)
	return ExecSuccess.OpStatus()
}

// check if the account is network admin
func (q *PermissionCtrlAPI) isNetworkAdmin(account string) bool {
	ac := types.AcctInfoMap.GetAccount(common.StringToAddress(account))

	return ac != nil && ac.OrgId == q.permCtrl.permConfig.NwAdminOrg
}

func (q *PermissionCtrlAPI) isOrgAdmin(account string, orgId string) (ExecStatus, error) {
	org := types.OrgInfoMap.GetOrg(orgId)
	if org == nil {
		return ErrOrgDoesNotExists, errors.New("invalid org")
	}
	ac := types.AcctInfoMap.GetAccount(common.StringToAddress(account))
	if ac == nil {
		return ErrNotOrgAdmin, errors.New("not org admin")
	}
	// check if the account is network admin
	if !(ac.IsAdmin && (ac.OrgId == orgId)) {
		return ErrNotOrgAdmin, errors.New("not org admin")
	}
	return ExecSuccess, nil
}

func (q *PermissionCtrlAPI) validateOrg(orgId, pOrgId string) (ExecStatus, error) {
	// validate Parent org id
	if pOrgId != "" {
		if types.OrgInfoMap.GetOrg(pOrgId) == nil {
			return ErrInvalidParentOrg, errors.New("invalid parent org")
		}
		locOrgId := pOrgId + "." + orgId
		if types.OrgInfoMap.GetOrg(locOrgId) != nil {
			return ErrOrgExists, errors.New("org exists")
		}
	} else if types.OrgInfoMap.GetOrg(orgId) != nil {
		return ErrOrgExists, errors.New("org exists")
	}
	return ExecSuccess, nil
}

func (q *PermissionCtrlAPI) validatePendingOp(authOrg, orgId, url string, account string, pendingOp int64) bool {
	pOrg, pUrl, pAcct, op, err := q.permCtrl.contractMgr.GetPendingOp(authOrg)
	return err == nil && (op == pendingOp && pOrg == orgId && pUrl == url && pAcct == account)
}

func (q *PermissionCtrlAPI) checkPendingOp(orgId string) bool {
	_, _, _, op, err := q.permCtrl.contractMgr.GetPendingOp(orgId)
	return err == nil && op != 0
}

func (q *PermissionCtrlAPI) checkOrgStatus(orgId string, op uint8) (ExecStatus, error) {
	org := types.OrgInfoMap.GetOrg(orgId)

	if org == nil {
		return ErrOrgDoesNotExists, errors.New("org does not exist")
	}

	if !((op == 1 && org.Status == types.OrgApproved) || (op == 2 && org.Status == types.OrgSuspended)) {
		return ErrOpNotAllowed, errors.New("operation not allowed for current status")
	}
	return ExecSuccess, nil
}

func (q *PermissionCtrlAPI) valNodeStatusChange(orgId, nodeId string, op NodeUpdateAction, permAction PermAction) (ExecStatus, error) {
	// validates if the enode is linked the passed organization
	// validate node id and
	if len(nodeId) == 0 {
		return ErrInvalidNode, errors.New("invalid node id")
	}
	if execStatus, err := q.valNodeDetails(nodeId); err != nil && execStatus != ErrNodePresent {
		return execStatus, errors.New("node not found")
	}

	node := types.NodeInfoMap.GetNodeById(nodeId)
	if node != nil {
		if node.OrgId != orgId {
			return ErrNodeOrgMismatch, errors.New("node does not belong to the organization passed")
		}

		if node.Status == types.NodeBlackListed && op != RecoverBlacklistedNode {
			return ErrBlacklistedNode, errors.New("blacklisted node. operation not allowed")
		}

		// validate the op and node status and check if the op can be performed
		if (permAction == UpdateNodeStatus && (op != SuspendNode && op != ActivateSuspendedNode && op != BlacklistNode)) ||
			(permAction == InitiateNodeRecovery && op != RecoverBlacklistedNode) ||
			(permAction == ApproveNodeRecovery && op != ApproveBlacklistedNodeRecovery) {
			return ErrOpNotAllowed, errors.New("invalid node status change operation")
		}

		if (op == SuspendNode && node.Status != types.NodeApproved) ||
			(op == ActivateSuspendedNode && node.Status != types.NodeDeactivated) ||
			(op == BlacklistNode && node.Status == types.NodeRecoveryInitiated) ||
			(op == RecoverBlacklistedNode && node.Status != types.NodeBlackListed) ||
			(op == ApproveBlacklistedNodeRecovery && node.Status != types.NodeRecoveryInitiated) {
			return ErrOpNotAllowed, errors.New("node status change cannot be performed")
		}
	} else {
		return ErrNodeDoesNotExists, errors.New("node does not exist")
	}

	return ExecSuccess, nil
}

func (q *PermissionCtrlAPI) valAccountStatusChange(orgId string, account string, permAction PermAction, op AccountUpdateAction) (ExecStatus, error) {
	// validates if the enode is linked the passed organization
	ac := types.AcctInfoMap.GetAccount(common.StringToAddress(account))

	if ac == nil {
		return ErrAccountNotThere, errors.New("account not there")
	}

	if ac.IsAdmin && (op == 1 || op == 3) {
		return ErrOpNotAllowed, errors.New("operation not allowed on org admin account")
	}

	if ac.OrgId != orgId {
		return ErrOrgNotOwner, errors.New("account does not belong to the organization passed")
	}
	if (permAction == UpdateAccountStatus && (op != SuspendAccount && op != ActivateSuspendedAccount && op != BlacklistAccount)) ||
		(permAction == InitiateAccountRecovery && op != RecoverBlacklistedAccount) ||
		(permAction == ApproveAccountRecovery && op != ApproveBlacklistedAccountRecovery) {
		return ErrOpNotAllowed, errors.New("invalid account status change operation")
	}

	if ac.Status == types.AcctBlacklisted && op != RecoverBlacklistedAccount {
		return ErrBlacklistedAccount, errors.New("blacklisted account. operation not allowed")
	}

	if (op == SuspendAccount && ac.Status != types.AcctActive) ||
		(op == ActivateSuspendedAccount && ac.Status != types.AcctSuspended) ||
		(op == BlacklistAccount && ac.Status == types.AcctRecoveryInitiated) ||
		(op == RecoverBlacklistedAccount && ac.Status != types.AcctBlacklisted) ||
		(op == ApproveBlacklistedAccountRecovery && ac.Status != types.AcctRecoveryInitiated) {
		return ErrOpNotAllowed, errors.New("account status change cannot be performed")
	}
	return ExecSuccess, nil
}

func (q *PermissionCtrlAPI) checkOrgAdminExists(orgId string, account string) (ExecStatus, error) {
	ac := types.AcctInfoMap.GetAccount(common.StringToAddress(account))

	if ac != nil {
		if ac.OrgId != orgId {
			return ErrAccountInUse, errors.New("account part of another org")
		}
		if ac.IsAdmin {
			return ErrAccountOrgAdmin, errors.New("account already org admin for the org")
		}
	}
	return ExecSuccess, nil
}

func (q *PermissionCtrlAPI) checkNodeExists(nodeId string) bool {
	node := types.NodeInfoMap.GetNodeById(nodeId)
	if node != nil {
		return true
	}

	return false
}

func (q *PermissionCtrlAPI) valNodeDetails(url string) (ExecStatus, error) {
	// validate node id and
	if len(url) != 0 {

		// check if node already there
		if q.checkNodeExists(url) {
			return ErrNodePresent, errors.New("duplicate node")
		}
	}
	return ExecSuccess, nil
}

// all validations for add org operation
func (q *PermissionCtrlAPI) valAddOrg(args txArgs) ExecStatus {
	// check if the org id contains "."
	if args.orgId == "" || args.url == "" || args.acctId == "" {
		return ErrInvalidInput
	}
	if !isStringAlphaNumeric(args.orgId) {
		return ErrInvalidOrgName
	}

	// check if caller is network admin
	if !q.isNetworkAdmin(q.permCtrl.selfAddr) {
		return ErrNotNetworkAdmin
	}

	// check if any previous op is pending approval for network admin
	if q.checkPendingOp(q.permCtrl.permConfig.NwAdminOrg) {
		return ErrPendingApprovals
	}
	// check if org already exists
	if execStatus, er := q.validateOrg(args.orgId, ""); er != nil {
		return execStatus
	}

	// validate node id and
	if execStatus, er := q.valNodeDetails(args.url); er != nil {
		return execStatus
	}

	// check if account is already part of another org
	if execStatus, er := q.checkOrgAdminExists(args.orgId, args.acctId); er != nil {
		return execStatus
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valApproveOrg(args txArgs) ExecStatus {
	// check caller is network admin
	if !q.isNetworkAdmin(q.permCtrl.selfAddr) {
		return ErrNotNetworkAdmin
	}
	// check if anything pending approval
	if !q.validatePendingOp(q.permCtrl.permConfig.NwAdminOrg, args.orgId, args.url, args.acctId, 1) {
		return ErrNothingToApprove
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valAddSubOrg(args txArgs) ExecStatus {
	// check if the org id contains "."
	if args.orgId == "" {
		return ErrInvalidInput
	}
	if !isStringAlphaNumeric(args.orgId) {
		return ErrInvalidOrgName
	}

	// check if caller is network admin
	if execStatus, er := q.isOrgAdmin(q.permCtrl.selfAddr, args.porgId); er != nil {
		return execStatus
	}

	// check if org already exists
	if execStatus, er := q.validateOrg(args.orgId, args.porgId); er != nil {
		return execStatus
	}

	if execStatus, er := q.valNodeDetails(args.url); er != nil {
		return execStatus
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valUpdateOrgStatus(args txArgs) ExecStatus {
	// check if called is network admin
	if !q.isNetworkAdmin(q.permCtrl.selfAddr) {
		Logger.Debugf("UpdateOrgStatus failed :ErrNotNetworkAdmin")
		return ErrNotNetworkAdmin
	}
	if OrgUpdateAction(args.action) != SuspendOrg &&
		OrgUpdateAction(args.action) != ActivateSuspendedOrg {
		Logger.Debugf("UpdateOrgStatus failed action :ErrOpNotAllowed")
		return ErrOpNotAllowed
	}

	//check if passed org id is network admin org. update should not be allowed
	if args.orgId == q.permCtrl.permConfig.NwAdminOrg {
		Logger.Debugf("UpdateOrgStatus failed orgId :ErrOpNotAllowed")
		return ErrOpNotAllowed
	}
	// check if status update can be performed. Org should be approved for suspension
	if execStatus, er := q.checkOrgStatus(args.orgId, args.action); er != nil {
		Logger.Debugf("UpdateOrgStatus checkOrgStatus orgId :ErrOpNotAllowed")
		return execStatus
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valApproveOrgStatus(args txArgs) ExecStatus {
	// check if called is network admin
	if !q.isNetworkAdmin(q.permCtrl.selfAddr) {
		return ErrNotNetworkAdmin
	}
	// check if anything is pending approval
	var pendingOp int64
	if args.action == 1 {
		pendingOp = 2
	} else if args.action == 2 {
		pendingOp = 3
	} else {
		return ErrOpNotAllowed
	}
	if !q.validatePendingOp(q.permCtrl.permConfig.NwAdminOrg, args.orgId, "", "", pendingOp) {
		return ErrNothingToApprove
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valAddNode(args txArgs) ExecStatus {
	if args.url == "" {
		return ErrInvalidInput
	}
	// check if caller is network admin
	//if execStatus, er := q.isOrgAdmin(q.permCtrl.selfAddr, args.orgId); er != nil {
	//	return execStatus
	//}

	if execStatus, er := q.valNodeDetails(args.url); er != nil {
		return execStatus
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valUpdateNodeStatus(args txArgs, permAction PermAction) ExecStatus {
	// check if org admin
	// check if caller is network admin
	if execStatus, er := q.isOrgAdmin(q.permCtrl.selfAddr, args.orgId); er != nil {
		return execStatus
	}

	// validation status change is with in allowed set
	if execStatus, er := q.valNodeStatusChange(args.orgId, args.url, NodeUpdateAction(args.action), permAction); er != nil {
		return execStatus
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valAssignAdmin(args txArgs) ExecStatus {
	if args.acctId == "" {
		return ErrInvalidInput
	}
	// check if caller is network admin
	if args.isAdmin && args.orgId != q.permCtrl.permConfig.NwAdminOrg {
		return ErrOpNotAllowed
	}

	if !q.isNetworkAdmin(q.permCtrl.selfAddr) {
		return ErrNotNetworkAdmin
	}

	if _, err := q.validateOrg(args.orgId, ""); err == nil {
		return ErrOrgDoesNotExists
	}

	// check if account is already part of another org
	if execStatus, er := q.checkOrgAdminExists(args.orgId, args.acctId); er != nil && execStatus != ErrOrgAdminExists {
		return execStatus
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valApproveAdmin(args txArgs) ExecStatus {
	// check if caller is network admin
	if !q.isNetworkAdmin(q.permCtrl.selfAddr) {
		return ErrNotNetworkAdmin
	}
	// check if the org exists

	// check if account is valid
	ac := types.AcctInfoMap.GetAccount(common.StringToAddress(args.acctId))
	if ac == nil {
		return ErrInvalidAccount
	}
	// validate pending op
	if !q.validatePendingOp(q.permCtrl.permConfig.NwAdminOrg, ac.OrgId, "", args.acctId, 4) {
		return ErrNothingToApprove
	}
	return ExecSuccess
}

func (q *PermissionCtrlAPI) valUpdateAccountStatus(args txArgs, permAction PermAction) ExecStatus {
	// check if the caller is org admin
	if execStatus, er := q.isOrgAdmin(q.permCtrl.selfAddr, args.orgId); er != nil {
		return execStatus
	}
	// validation status change is with in allowed set
	if execStatus, er := q.valAccountStatusChange(args.orgId, args.acctId, permAction, AccountUpdateAction(args.action)); er != nil {
		return execStatus
	}
	return ExecSuccess
}

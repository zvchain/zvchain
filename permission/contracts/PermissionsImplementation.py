import ujson


class PermissionsImplementation:
    # account status
    ACCOUNT_NOT_IN_LIST = 0
    PENDING_APPROVAL = 1
    ACTIVE = 2
    SUSPENDED = 3

    ACCESS_READONLY = 0
    ACCESS_TRANSACT = 1
    ACCESS_CONTRACT_DEPLOY = 2
    ACCESS_FULL_ACCESS = 3

    # org status
    ORG_NOT_IN_LIST = 0
    PROPOSED = 1
    APPROVED = 2
    PENDING_SUSPENSION = 3
    SUSNPENDED = 4
    PENDING_SUSPENSION_REVOKE = 5

    # vote option
    VOTE_OP_ADD_ACTIVITY_ORG = 1
    VOTE_OP_SUSPEND_ORG = 2
    VOTE_OP_REVOKE_SUSPEND_ORG = 3
    VOTE_OP_ASSIGN_ALLIANCE_ADMIN = 4
    VOTE_OP_REMOVE_ALLIANCE_ADMIN = 5

    VOTE_OP_ADD_MINER_NODE = 6
    VOTE_OP_ASSIGN_NODE_TO_MINER = 7
    VOTE_OP_REMOVE_MINER = 8

    # miner role
    NOT_A_MINER = 0
    PROPOSAL_MINER = 1
    VERIFY_MINER = 2
    PROPOSAL_AND_VERIFY_MINER = 3

    # weight
    NO_WEIGHT = 0
    DEFAULT_WEIGHT = 10

    # status op
    OP_SUSPEND = 1
    OP_REVOKE_SUSPEND = 2

    def __init__(self):
        self.account_mgr_addr = ''
        self.org_mgr_addr = ''
        self.vote_mgr_addr = ''
        self.node_mgr_addr = ''
        self.perm_upgradeable_addr = ''

        self.alliance_admin_org = ''
        self.network_boot = False

    def _only_from_interface(self):
        return Contract(self.perm_upgradeable_addr).get_perm_interface() == msg.sender

    def _only_from_upgradable(self):
        return self.perm_upgradeable_addr == msg.sender

    def _is_alliance_admin_account(self, _account):
        return Contract(self.account_mgr_addr).is_alliance_admin_account(_account)

    def _check_org_status(self, _org_id, _status):
        return Contract(self.org_mgr_addr).check_org_status(_org_id, _status)

    def _is_alliance_admin_org(self, _org_id):
        return _org_id == self.alliance_admin_org

    def _is_org_admin(self, _account, _org_id):
        return Contract(self.account_mgr_addr).is_org_admin(_account, _org_id)

    def _org_approved(self, _org_id):
        return Contract(self.org_mgr_addr).check_org_status(_org_id, self.APPROVED)

    def _account_not_exists(self, _account):
        return not Contract(self.account_mgr_addr).account_is_exists(_account)

    def _check_network_boot_status(self, _status):
        return _status == self.network_boot

    def _check_miner_status(self, _node_id, _status):
        return Contract(self.node_mgr_addr).check_miner_status(_node_id, _status)

    def _update_voter_list(self, _account, _action):
        if _action:
            Contract(self.vote_mgr_addr).add_voter(self.alliance_admin_org, _account)
        else:
            Contract(self.vote_mgr_addr).delete_voter(self.alliance_admin_org, _account)

    def _handle_miner_info(self, _node_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        return ujson.dumps([_node_id, _miner_role, _vrf_pk, _bls_pk, _weight])

    @register.public(str, str, str, str, str)
    def set_default_contract_addr(self, _account_mgr_addr, _org_mgr_addr, _vote_mgr_addr, _node_mgr_addr,
                                  _perm_upgradeable_addr):
        assert self.account_mgr_addr == '' and self.org_mgr_addr == '' and self.vote_mgr_addr == '' and self.node_mgr_addr == '' and self.perm_upgradeable_addr == '', "contract addresses can only be set once"
        self.account_mgr_addr = _account_mgr_addr
        self.org_mgr_addr = _org_mgr_addr
        self.vote_mgr_addr = _vote_mgr_addr
        self.node_mgr_addr = _node_mgr_addr
        self.perm_upgradeable_addr = _perm_upgradeable_addr

    @register.public(str)
    def set_policy(self, _alliance_admin_org):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._check_network_boot_status(False), "to ensure network boot status is False"
        self.alliance_admin_org = _alliance_admin_org
        Contract(self.account_mgr_addr).set_alliance_admin_org(_alliance_admin_org)
        Contract(self.vote_mgr_addr).set_org_id(_alliance_admin_org)

    @register.public()
    def init(self):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._check_network_boot_status(False), "to ensure network boot status is False"
        # add to org mgr
        Contract(self.org_mgr_addr).add_alliance_admin_org(self.alliance_admin_org)

    @register.public(str, int, str, str, int)
    def add_alliance_node(self, _node_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._check_network_boot_status(False), "to ensure network boot status is False"
        Contract(self.node_mgr_addr).add_admin_node(_node_id, self.alliance_admin_org, _miner_role, _vrf_pk, _bls_pk,
                                                    _weight)

    @register.public(str)
    def add_alliance_account(self, _account):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._check_network_boot_status(False), "to ensure network boot status is False"
        self._update_voter_list(_account, True)
        Contract(self.account_mgr_addr).assign_alliance_admin(_account, self.alliance_admin_org,
                                                              self.ACCESS_FULL_ACCESS,
                                                              self.ACTIVE)

    @register.public()
    def update_network_boot_status(self):
        self.network_boot = True

    @register.public(str, bool)
    def set_migration_policy(self, _alliance_admin_org, _network_boot):
        assert self._only_from_upgradable(), "only can invoke from permission upgradeable contract"
        assert self._check_network_boot_status(False), "to ensure network boot status is False"
        self.alliance_admin_org = _alliance_admin_org
        self.network_boot = _network_boot

    @register.public(str, str, str, str, str)
    def set_upgradable_impl(self, _perm_upgradable_addr, _account_mgr_addr, _org_mgr_addr, _vote_mgr_addr,
                            _node_mgr_addr):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        Contract(self.account_mgr_addr).set_upgradable_impl(_perm_upgradable_addr)
        Contract(self.vote_mgr_addr).set_upgradable_impl(_perm_upgradable_addr)
        Contract(self.org_mgr_addr).set_upgradable_impl(_perm_upgradable_addr)
        Contract(self.node_mgr_addr).set_upgradable_impl(_perm_upgradable_addr)

    @register.public(str, str, str, str)
    def add_org(self, _org_id, _account, _node_id, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"

        Contract(self.vote_mgr_addr).add_item(_org_id, _node_id, _account, self.VOTE_OP_ADD_ACTIVITY_ORG, '')
        Contract(self.org_mgr_addr).add_org(_org_id)
        Contract(self.node_mgr_addr).add_node(_node_id, _org_id)
        Contract(self.account_mgr_addr).add_account(_account, _org_id, self.ACCESS_CONTRACT_DEPLOY, True,
                                                    self.PENDING_APPROVAL)

    @register.public(str, str, str, str)
    def approve_org(self, _org_id, _account, _node_id, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        assert self._check_org_status(_org_id,
                                      self.PROPOSED), "to approve an org should ensure the org status is proposed status"
        if Contract(self.vote_mgr_addr).approve_item(_org_id, _node_id, _account, self.VOTE_OP_ADD_ACTIVITY_ORG, '',
                                                     _caller):
            Contract(self.org_mgr_addr).approve_org(_org_id)
            Contract(self.node_mgr_addr).approve_node(_node_id, _org_id)
            Contract(self.account_mgr_addr).approve_admin(_account, _org_id)

    @register.public(str, int, str)
    def update_org_status(self, _org_id, _action, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"

        vote_type = 0
        if _action == self.OP_SUSPEND:
            vote_type = self.VOTE_OP_SUSPEND_ORG
        elif _action == self.OP_REVOKE_SUSPEND:
            vote_type = self.VOTE_OP_REVOKE_SUSPEND_ORG

        Contract(self.org_mgr_addr).update_org_status(_org_id, _action)
        Contract(self.vote_mgr_addr).add_item(_org_id, '', '', vote_type, '')

    @register.public(str, int, str)
    def approve_org_status(self, _org_id, _action, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"

        vote_type = 0
        org_status = 0
        if _action == self.OP_SUSPEND:
            vote_type = self.VOTE_OP_SUSPEND_ORG
            org_status = self.PENDING_SUSPENSION
        elif _action == self.OP_REVOKE_SUSPEND:
            vote_type = self.VOTE_OP_REVOKE_SUSPEND_ORG
            org_status = self.PENDING_SUSPENSION_REVOKE

        assert self._check_org_status(_org_id, org_status), "the current organization's status is not expected"
        if Contract(self.vote_mgr_addr).approve_item(_org_id, '', '', vote_type, '', _caller):
            Contract(self.org_mgr_addr).approve_org_status(_org_id, _action)

    @register.public(str, str, str)
    def assign_alliance_admin(self, _org_id, _account, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        assert self._is_alliance_admin_org(_org_id), "org id must equal alliance admin org"

        Contract(self.account_mgr_addr).assign_alliance_admin(_account, _org_id, self.ACCESS_FULL_ACCESS,
                                                              self.PENDING_APPROVAL)
        Contract(self.vote_mgr_addr).add_item(_org_id, '', _account, self.VOTE_OP_ASSIGN_ALLIANCE_ADMIN, '')

    @register.public(str, str, str)
    def approve_alliance_admin(self, _org_id, _account, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        assert self._is_alliance_admin_org(_org_id), "org id must equal alliance admin org"

        if Contract(self.vote_mgr_addr).approve_item(_org_id, '', _account, self.VOTE_OP_ASSIGN_ALLIANCE_ADMIN, '',
                                                     _caller):
            result = Contract(self.account_mgr_addr).approve_admin(_account, _org_id)
            if result:
                self._update_voter_list(_account, True)

    @register.public(str, str, int, str, str, int, str)
    def add_miner_node(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        Contract(self.node_mgr_addr).add_org_node(_node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight, True)
        _miner_info = self._handle_miner_info(_node_id, _miner_role, _vrf_pk, _bls_pk, _weight)
        Contract(self.vote_mgr_addr).add_item(_org_id, _node_id, '', self.VOTE_OP_ADD_MINER_NODE, _miner_info)

    @register.public(str, str, int, str, str, int, str)
    def approve_miner_node(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        _miner_info = self._handle_miner_info(_node_id, _miner_role, _vrf_pk, _bls_pk, _weight)
        if Contract(self.vote_mgr_addr).approve_item(_org_id, _node_id, '', self.VOTE_OP_ADD_MINER_NODE, _miner_info,
                                                     _caller):
            Contract(self.node_mgr_addr).approve_org_node(_node_id, _org_id, _miner_role)
            # todo 通过投票后，矿工数据更新到矿工池子
            pass

    @register.public(str, str, int, str, str, int, str)
    def assign_node_to_miner(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        _miner_info = self._handle_miner_info(_node_id, _miner_role, _vrf_pk, _bls_pk, _weight)
        Contract(self.node_mgr_addr).assign_node_to_miner(_node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight)
        Contract(self.vote_mgr_addr).add_item(_org_id, _node_id, '', self.VOTE_OP_ASSIGN_NODE_TO_MINER, _miner_info)

    @register.public(str, str, int, str, str, int, str)
    def approve_node_to_miner(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"
        _miner_info = self._handle_miner_info(_node_id, _miner_role, _vrf_pk, _bls_pk, _weight)
        if Contract(self.vote_mgr_addr).approve_item(_org_id, _node_id, '', self.VOTE_OP_ASSIGN_NODE_TO_MINER,
                                                     _miner_info, _caller):
            Contract(self.node_mgr_addr).approve_org_node(_node_id, _org_id, _miner_role)
            # todo 通过投票后，矿工数据更新到矿工池子
            pass

    @register.public(str, str, bool, str)
    def remove_miner(self, _node_id, _org_id, _disable_node, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"

        Contract(self.vote_mgr_addr).add_item(_org_id, _node_id, '', self.VOTE_OP_REMOVE_MINER, '')
        Contract(self.node_mgr_addr).remove_miner(_node_id, _org_id, _disable_node)

    @register.public(str, str, bool, str)
    def approve_remove_miner(self, _node_id, _org_id, _disable_node, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_alliance_admin_account(
            _caller), "caller should be affiliated with the alliance admin organization"

        if Contract(self.vote_mgr_addr).approve_item(_org_id, _node_id, '', self.VOTE_OP_REMOVE_MINER, '',
                                                     _caller):
            Contract(self.node_mgr_addr).approve_remove_miner(_node_id, _org_id, _disable_node)
            # todo 通过投票后，矿工数据更新到矿工池子
            pass

    # add a new account
    @register.public(str, str, int, bool, str)
    def add_account(self, _account, _org_id, _access, _is_admin, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_org_admin(_caller, _org_id), "the caller must be the admin of the organization"
        assert self._org_approved(_org_id), "the organization must be approved status"
        assert self._account_not_exists(_account), "the new account is guaranteed not to exist in the organization"
        Contract(self.account_mgr_addr).add_account(_account, _org_id, _access, _is_admin, self.ACTIVE)

    @register.public(str, str, int, str)
    def update_account_status(self, _account, _org_id, _action, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_org_admin(_caller, _org_id), "the caller must be the admin of the organization"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"
        Contract(self.account_mgr_addr).update_account_status(_account, _org_id, _action)

    @register.public(str, str, int, str)
    def update_account_access(self, _account, _org_id, _authority, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_org_admin(_caller, _org_id), "the caller must be the admin of the organization"
        Contract(self.account_mgr_addr).update_account_access(_account, _org_id, _authority)

    @register.public(str, str, str)
    def add_node(self, _node_id, _org_id, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_org_admin(_caller, _org_id), "the caller must be the admin of the organization"
        assert self._org_approved(_org_id), "the organization must be approved status"
        Contract(self.node_mgr_addr).add_org_node(_node_id, _org_id, self.NOT_A_MINER, '', '', 0, False)

    @register.public(str, str, int, str)
    def update_node_status(self, _node_id, _org_id, _action, _caller):
        assert self._only_from_interface(), "only can invoke from permission interface contract"
        assert self._is_org_admin(_caller, _org_id), "the caller must be the admin of the organization"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"
        Contract(self.node_mgr_addr).update_node_status(_node_id, _org_id, _action, False)

    @register.public()
    def get_policy(self):
        return ujson.dumps([self.alliance_admin_org, self.network_boot])

    @register.public()
    def get_contracts_addr(self):
        return ujson.dumps([self.account_mgr_addr, self.org_mgr_addr, self.vote_mgr_addr, self.node_mgr_addr,
                            self.perm_upgradeable_addr])

    @register.public()
    def get_org_list(self):
        return Contract(self.org_mgr_addr).get_org_list()

    @register.public()
    def get_node_list(self):
        return Contract(self.node_mgr_addr).get_node_list()

    @register.public()
    def get_account_list(self):
        return Contract(self.account_mgr_addr).get_account_list()

    @register.public(str)
    def get_account_access(self, _account):
        return Contract(self.account_mgr_addr).get_account_access(_account)

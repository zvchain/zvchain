class PermissionInterface:
    def __init__(self):
        self.perm_upgradeable_addr = ''
        self.perm_permission_impl = ''

    # only can invoke from upgradeable
    def invoke_from_upgradeable(self):
        return msg.sender == self.perm_upgradeable_addr

    # ↓↓↓↓↓↓The following methods are operated by the alliance administrator↓↓↓↓↓↓
    # add a new org
    @register.public(str, str, str)
    def add_org(self, _org_id, _account, _node_id):
        Contract(self.perm_permission_impl).add_org(_org_id, _account, _node_id, msg.sender)

    # approve a added org.It must be approved by most of the alliance administrators
    # before the organization from PROPOSED status changes to APPROVED status
    @register.public(str, str, str)
    def approve_org(self, _org_id, _account, _node_id):
        Contract(self.perm_permission_impl).approve_org(_org_id, _account, _node_id, msg.sender)

    # update a existed org status,attention alliance can't change status
    @register.public(str, int)
    def update_org_status(self, _org_id, _action):
        Contract(self.perm_permission_impl).update_org_status(_org_id, _action, msg.sender)

    # when a alliance admin proposed a update org status option,It must be approved
    # by most of the alliance administrators before the organization status changed
    @register.public(str, int)
    def approve_org_status(self, _org_id, _action):
        Contract(self.perm_permission_impl).approve_org_status(_org_id, _action, msg.sender)

    # this method can use when someone belongs to alliance admin org want to assign
    # a new account to alliance admin
    @register.public(str, str)
    def assign_alliance_admin(self, _org_id, _account):
        Contract(self.perm_permission_impl).assign_alliance_admin(_org_id, _account, msg.sender)

    # when an alliance admin subject a assign alliance admin request,it will be work after most
    # of admins belong to the alliance admin org invoke this method
    @register.public(str, str)
    def approve_alliance_admin(self, _org_id, _account):
        Contract(self.perm_permission_impl).approve_alliance_admin(_org_id, _account, msg.sender)

    # only can invoke from alliance admin,when add a miner node can use this method
    @register.public(str, str, int, str, str, int)
    def add_miner_node(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        Contract(self.perm_permission_impl).add_miner_node(_node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight,
                                                           msg.sender)

    # a miner node can be successfully added only after a alliance admin invoked add_miner_node method
    # and most union administrators are required to invoke approve_miner_node method
    @register.public(str, str, int, str, str, int)
    def approve_miner_node(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        Contract(self.perm_permission_impl).approve_miner_node(_node_id, _org_id, _miner_role, _vrf_pk, _bls_pk,
                                                               _weight, msg.sender)

    # when some alliance want to assign a common sync node to a miner node, can invoke this method
    @register.public(str, str, int, str, str, int)
    def assign_node_to_miner(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        Contract(self.perm_permission_impl).assign_node_to_miner(_node_id, _org_id, _miner_role, _vrf_pk, _bls_pk,
                                                                 _weight, msg.sender)

    # after some alliance admin invoke assign_node_to_miner method,it will be changed successfully after
    # most of the alliance admin invoke this method
    @register.public(str, str, int, str, str, int)
    def approve_node_to_miner(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        Contract(self.perm_permission_impl).approve_node_to_miner(_node_id, _org_id, _miner_role, _vrf_pk, _bls_pk,
                                                                  _weight, msg.sender)

    # when a alliance want to remove a miner,some can invoke this method.
    # _disable_node param means if you want to remove miner meanwhile suspending node synchronization,
    # it's value is True, else if you want to remove the miner function and keep the node's synchronization
    # function,the value is False
    @register.public(str, str, bool)
    def remove_miner(self, _node_id, _org_id, _disable_node):
        Contract(self.perm_permission_impl).remove_miner(_node_id, _org_id, _disable_node, msg.sender)

    # when a alliance invoked remove_miner before,it'll be successfully removed after
    # most of alliance admin approve remove miner
    @register.public(str, str, bool)
    def approve_remove_miner(self, _node_id, _org_id, _disable_node):
        Contract(self.perm_permission_impl).approve_remove_miner(_node_id, _org_id, _disable_node, msg.sender)

    # ↓↓↓↓↓↓The following methods are operated by the organization administrator↓↓↓↓↓↓
    # some organization admin want to add a new account to org
    @register.public(str, str, int, bool)
    def add_account(self, _account, _org_id, _access, _is_admin):
        Contract(self.perm_permission_impl).add_account(_account, _org_id, _access, _is_admin, msg.sender)

    # some organization admin want to change account status
    @register.public(str, str, int)
    def update_account_status(self, _account, _org_id, _action):
        Contract(self.perm_permission_impl).update_account_status(_account, _org_id, _action, msg.sender)

    # means when a organization want to change the account access belongs to the org.
    # _authority contains account's property like is_admin, is_voter and access level(
    # There are usually four access levels: 0-read only, 1-contract deploy, 2-contract invoke,
    # 3-full access-only alliance admin). and you can only use the first three access:0,1,2
    @register.public(str, str, int)
    def update_account_access(self, _account, _org_id, _access):
        Contract(self.perm_permission_impl).update_account_access(_account, _org_id, _access, msg.sender)

    # add a new node to org,org admin can invoke this method
    @register.public(str, str)
    def add_node(self, _node_id, _org_id):
        Contract(self.perm_permission_impl).add_node(_node_id, _org_id, msg.sender)

    # a org admin want to update a node status that belongs to the org
    @register.public(str, str, int)
    def update_node_status(self, _node_id, _org_id, _action):
        Contract(self.perm_permission_impl).update_node_status(_node_id, _org_id, _action, msg.sender)

    # call by upgradable when call init
    @register.public(str, str, str, str, str, str)
    def set_perm_permission_impl(self, _perm_impl, _perm_upgradable_addr, _account_mgr_addr, _org_mgr_addr,
                                 _vote_mgr_addr, _node_mgr_addr):
        if self.perm_upgradeable_addr == '':
            self.perm_upgradeable_addr = _perm_upgradable_addr

        assert self.invoke_from_upgradeable(), "ensure invoke from upgradeable contract"
        self.perm_permission_impl = _perm_impl
        Contract(self.perm_permission_impl).set_default_contract_addr(_account_mgr_addr, _org_mgr_addr, _vote_mgr_addr,
                                                                      _node_mgr_addr, _perm_upgradable_addr)
        Contract(self.perm_permission_impl).set_upgradable_impl(_perm_upgradable_addr, _account_mgr_addr, _org_mgr_addr,
                                                                _vote_mgr_addr, _node_mgr_addr)

    # set_policy method is automatic invoked by system when network boot up
    @register.public(str)
    def set_policy(self, _alliance_admin_org):
        Contract(self.perm_permission_impl).set_policy(_alliance_admin_org)

    # init method is automatic invoked by system when network boot up
    @register.public()
    def init(self):
        Contract(self.perm_permission_impl).init()

    # add_alliance_node method is automatic invoked by system when network boot up
    @register.public(str, int, str, str, int)
    def add_alliance_node(self, _node_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        Contract(self.perm_permission_impl).add_alliance_node(_node_id, _miner_role, _vrf_pk, _bls_pk, _weight)

    # add_alliance_node method is automatic invoked by system when network boot up
    @register.public(str)
    def add_alliance_account(self, _account):
        Contract(self.perm_permission_impl).add_alliance_account(_account)

    # update_network_boot_status method is automatic invoked by system when network boot up
    @register.public()
    def update_network_boot_status(self):
        Contract(self.perm_permission_impl).update_network_boot_status()

    @register.public()
    def get_org_list(self):
        return Contract(self.perm_permission_impl).get_org_list()

    @register.public()
    def get_node_list(self):
        return Contract(self.perm_permission_impl).get_node_list()

    @register.public()
    def get_account_list(self):
        return Contract(self.perm_permission_impl).get_account_list()

    @register.public(str)
    def get_account_access(self, _account):
        return Contract(self.perm_permission_impl).get_account_access(_account)

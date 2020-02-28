import ujson


class PermissionsUpgradable:

    def __init__(self):
        self.guardian = msg.sender
        self.initDone = False
        self.perm_impl = ''
        self.perm_interface = ''

    def is_guardian(self, address):
        return address == self.guardian

    @register.public(str, str, str, str, str, str, str)
    def init(self, _perm_interface, _perm_impl, _perm_upgradeable_addr, _account_mgr_addr, _org_mgr_addr,
             _vote_mgr_addr, _node_mgr_addr):
        assert self.is_guardian(msg.sender) and not self.initDone, "caller must guardian and not init done"
        self.perm_impl = _perm_impl
        self.perm_interface = _perm_interface
        # 绑定合约
        Contract(self.perm_interface).set_perm_permission_impl(_perm_impl, _perm_upgradeable_addr, _account_mgr_addr,
                                                               _org_mgr_addr, _vote_mgr_addr, _node_mgr_addr)
        self.initDone = True

    @register.public(str)
    def change_impl(self, _new_perm_impl):
        assert self.is_guardian(msg.sender), "caller must guardian"
        policy = Contract(self.perm_impl).get_policy()
        _policy = ujson.loads(policy)
        _alliance_admin_org, _network_boot = _policy[0], _policy[1]

        contract_addrs = Contract(self.perm_impl).get_contracts_addr()
        _contract_addrs = ujson.loads(contract_addrs)
        _account_mgr_addr, _org_mgr_addr, _vote_mgr_addr, _node_mgr_addr, _perm_upgradeable_addr = _contract_addrs[0], \
                                                                                                   _contract_addrs[1], \
                                                                                                   _contract_addrs[2], \
                                                                                                   _contract_addrs[3], \
                                                                                                   _contract_addrs[4]

        self.perm_impl = _new_perm_impl
        Contract(self.perm_interface).set_perm_permission_impl(_new_perm_impl, _perm_upgradeable_addr,
                                                               _account_mgr_addr, _org_mgr_addr, _vote_mgr_addr,
                                                               _node_mgr_addr)
        Contract(_new_perm_impl).set_migration_policy(_alliance_admin_org, _network_boot)

    @register.public()
    def get_guardian(self):
        return self.guardian

    @register.public()
    def get_perm_impl(self):
        return self.perm_impl

    @register.public()
    def get_msg_sender(self):
        return msg.sender

    @register.public()
    def get_perm_interface(self):
        return self.perm_interface

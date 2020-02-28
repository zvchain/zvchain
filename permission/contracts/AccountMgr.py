import ujson

add_account_event = Event("add_account")

account_status_changed_event = Event("account_status_changed")

account_access_changed_event = Event("account_access_changed")


class AccountManager:
    NOT_IN_LIST = 0
    PENDING_APPROVAL = 1
    ACTIVE = 2
    SUSPENDED = 3

    ACCESS_READONLY = 0
    ACCESS_TRANSACT = 1
    ACCESS_CONTRACT_DEPLOY = 2
    ACCESS_FULL_ACCESS = 3

    # status op
    OP_SUSPEND = 1
    OP_REVOKE_SUSPEND = 2

    class AccountDetails:
        def __init__(self, _account, _org_id, _access, _is_org_admin, _status, _is_voter):
            self.account = _account
            self.is_org_admin = _is_org_admin
            self.org_id = _org_id
            self.access = _access
            self.status = _status
            self.is_voter = _is_voter

    def __init__(self):
        self.perm_upgradable_addr = ''
        self.account_list = ''
        self.account_index = zdict()
        self.alliance_admin_org = ''
        self.account_count = 0

    @register.public(str)
    def set_upgradable_impl(self, _perm_upgradable_addr):
        if self.perm_upgradable_addr == '':
            self.perm_upgradable_addr = _perm_upgradable_addr

    # only from permission impl contract
    def _invoke_from_permission_impl(self):
        return msg.sender == Contract(self.perm_upgradable_addr).get_perm_impl()

    @register.public(str)
    def set_alliance_admin_org(self, _alliance_admin_org):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        self.alliance_admin_org = _alliance_admin_org

    @register.public(str)
    def account_is_exists(self, _account):
        return _account in self.account_index

    def _get_account_index(self, _account):
        assert self.account_is_exists(_account), "account must exist"
        return self.account_index[_account]

    def _account_belongs_to_org(self, _account, _org_id):
        assert self.account_is_exists(_account), "account must exist"
        _account_list = ujson.loads(self.account_list)
        return _account_list[self._get_account_index(_account)]['org_id'] == _org_id

    def _check_account_status(self, _account, _status):
        idx = self._get_account_index(_account)
        account_list = ujson.loads(self.account_list)
        return account_list[idx]['status'] == _status

    @register.public(str)
    def is_alliance_admin_account(self, _account):
        _account_list = ujson.loads(self.account_list)
        return _account_list[self._get_account_index(_account)]['org_id'] == self.alliance_admin_org

    @register.public(str, str)
    def is_org_admin(self, _account, _org_id):
        assert self.account_is_exists(_account), "account must exist"
        idx = self._get_account_index(_account)
        _account_list = ujson.loads(self.account_list)
        return _account_list[idx]['is_org_admin'] and _account_list[idx]['org_id'] == _org_id

    def _add_account(self, _account, _org_id, _access, _is_org_admin, _status):
        assert not self.account_is_exists(_account), "the account already exists and cannot be added repeatedly"
        _is_voter = False
        if _access > self.ACCESS_READONLY:
            _is_voter = True

        self.account_index[_account] = self.account_count
        new_account_details = self.AccountDetails(_account, _org_id, _access, _is_org_admin, _status, _is_voter)
        if self.account_list == '':
            self.account_list = ujson.dumps([new_account_details.__dict__])
        else:
            account_list = ujson.loads(self.account_list)
            account_list.append(new_account_details.__dict__)
            self.account_list = ujson.dumps(account_list)
        self.account_count += 1
        add_account_event.emit(_account, _org_id, _access, _is_org_admin)

    # 调此函数的方法要判断是否为组织管理员调用的
    @register.public(str, str, int, bool, int)
    def add_account(self, _account, _org_id, _access, _is_org_admin, _status):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert _access == self.ACCESS_READONLY or _access == self.ACCESS_TRANSACT or _access == self.ACCESS_CONTRACT_DEPLOY, "access must be one of 0,1,2"
        # 新添加账户所属组织不能为联盟管理员组织
        assert _org_id != self.alliance_admin_org, "the organization to which the new alliance administrator belongs mustn't be the alliance_admin_org"
        self._add_account(_account, _org_id, _access, _is_org_admin, _status)

    @register.public(str, str, int, int)
    def assign_alliance_admin(self, _account, _org_id, _access, _status):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        # 新添加联盟管理员所属组织必须为联盟管理员组织
        assert _org_id == self.alliance_admin_org, "the organization to which the new alliance administrator belongs must be the alliance_admin_org"
        self._add_account(_account, _org_id, _access, True, _status)

    def _change_status(self, _account, _status):
        idx = self._get_account_index(_account)
        _account_list = ujson.loads(self.account_list)
        _account_list[idx]['status'] = _status
        self.account_list = ujson.dumps(_account_list)

    def _change_access(self, _account, _access):
        idx = self._get_account_index(_account)
        _account_list = ujson.loads(self.account_list)
        _account_list[idx]['access'] = _access
        self.account_list = ujson.dumps(_account_list)

    @register.public(str, str)
    def approve_admin(self, _account, _org_id):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self.account_is_exists(_account), "account doesn't exist"
        self._change_status(_account, self.ACTIVE)
        account_status_changed_event.emit(_org_id, _account, self.ACTIVE)
        return self.alliance_admin_org == _org_id

    # _action=1,suspend the account; _action=2,active a suspended account
    @register.public(str, str, int)
    def update_account_status(self, _account, _org_id, _action):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._account_belongs_to_org(_account, _org_id), "account must exist and belongs to an exist org"
        assert not self.is_org_admin(_account, _org_id), "status change not possible for org admin accounts"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"

        cur_status = 0
        dest_status = 0
        if _action == self.OP_SUSPEND:
            cur_status = self.ACTIVE
            dest_status = self.SUSPENDED
        elif _action == self.OP_REVOKE_SUSPEND:
            cur_status = self.SUSPENDED
            dest_status = self.ACTIVE
        assert self._check_account_status(_account, cur_status), "current account's status is not expected "
        self._change_status(_account, dest_status)
        account_status_changed_event.emit(_org_id, _account, dest_status)

    @register.public(str, str, int)
    def update_account_access(self, _account, _org_id, _access):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._account_belongs_to_org(_account, _org_id), "account must exist and belongs to an exist org"
        assert _access == self.ACCESS_READONLY or _access == self.ACCESS_TRANSACT or \
               _access == self.ACCESS_CONTRACT_DEPLOY, "access must be one of 0,1,2"
        assert not self.is_org_admin(_account, _org_id), "access change not possible for org admin accounts"
        self._change_access(_account, _access)
        account_access_changed_event.emit(_org_id, _account, _access)

    @register.public()
    def get_account_list(self):
        return self.account_list

    @register.public(str)
    def get_account_access(self, _account):
        assert self.account_is_exists(_account), "account doesn't exist"
        _idx = self.account_index[_account]
        _account_list = ujson.loads(self.account_list)
        return _account_list[_idx]['access']

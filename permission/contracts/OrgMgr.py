import ujson

org_pending_approval_event = Event("org_pending_approval")
org_approved_event = Event("org_approved")
org_suspended_event = Event("org_suspended")
org_suspension_revoked_event = Event("org_suspension_revoked")


class OrgManager:
    NOT_IN_LIST = 0
    PROPOSED = 1
    APPROVED = 2
    PENDING_SUSPENSION = 3
    SUSPENDED = 4
    PENDING_SUSPENSION_REVOKE = 5

    # status op
    OP_SUSPEND = 1
    OP_REVOKE_SUSPEND = 2

    def __init__(self):
        self.perm_upgradable_addr = ''
        self.alliance_admin_org = ''

        # org_list save orgs details,
        self.org_list = ''
        # org_index save org index
        self.org_index = zdict()
        self.tmp = zdict()
        self.org_count = 0

    ''' 
    status means the org status,the meaning is as follows:
    0 - Not in list
    1 - Org proposed for approval by network admins
    2 - Org in Approved status
    3 - Org proposed for suspension and pending approval by network admins
    4 - Org in Suspended
    '''

    class OrgDetails:
        def __init__(self, _org_id, _status):
            self.org_id = _org_id
            self.status = _status

    # only from permission impl contract
    def _invoke_from_permission_impl(self):
        return msg.sender == Contract(self.perm_upgradable_addr).get_perm_impl()

    def _org_is_exists(self, _org_id):
        return _org_id in self.org_index

    def _get_org_index(self, _org_id):
        return self.org_index[_org_id]

    # todo
    def _add_org(self, _org_id, _status):
        # add org to org list
        self.org_index[_org_id] = self.org_count
        new_org = self.OrgDetails(_org_id, _status)

        if self.org_list == '':
            self.org_list = ujson.dumps([new_org.__dict__])
        else:
            _org_list = ujson.loads(self.org_list)
            _org_list.append(new_org.__dict__)
            self.org_list = ujson.dumps(_org_list)

        self.org_count += 1
        org_pending_approval_event.emit(_org_id, _status)

    def _change_status(self, _org_id, _status):
        _org_list = ujson.loads(self.org_list)
        _org_list[self._get_org_index(_org_id)]['status'] = _status
        self.org_list = ujson.dumps(_org_list)

    def _suspend_org(self, _org_id):
        self._change_status(_org_id, self.PENDING_SUSPENSION)
        org_pending_approval_event.emit(_org_id, self.PENDING_SUSPENSION)

    def _revoke_suspended_org(self, _org_id):
        self._change_status(_org_id, self.PENDING_SUSPENSION_REVOKE)
        org_pending_approval_event.emit(_org_id, self.PENDING_SUSPENSION_REVOKE)

    def _approve_suspend_org(self, _org_id):
        self._change_status(_org_id, self.SUSPENDED)
        org_suspended_event.emit(_org_id, self.SUSPENDED)

    def _approve_revoke_suspended_org(self, _org_id):
        self._change_status(_org_id, self.APPROVED)
        org_suspension_revoked_event.emit(_org_id, self.APPROVED)

    @register.public(str)
    def set_upgradable_impl(self, _perm_upgradable_addr):
        if self.perm_upgradable_addr == '':
            self.perm_upgradable_addr = _perm_upgradable_addr

    @register.public(str, int)
    def check_org_status(self, _org_id, _status):
        _org_list = ujson.loads(self.org_list)
        return _org_list[self._get_org_index(_org_id)]['status'] == _status

    @register.public(str)
    def add_alliance_admin_org(self, _org_id):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert not self._org_is_exists(
            _org_id), "adding alliance admin org first ensures that alliance admin org does not exist"
        self.alliance_admin_org = _org_id
        self._add_org(_org_id, self.APPROVED)

    @register.public(str)
    def add_org(self, _org_id):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert not self._org_is_exists(_org_id), "adding a new org first ensures that the org does not exist"
        self._add_org(_org_id, self.PROPOSED)

    @register.public(str)
    def approve_org(self, _org_id):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._org_is_exists(_org_id) and self.check_org_status(_org_id,
                                                                      self.PROPOSED), "to approve a org,the org must exists and the status is proposed "
        self._change_status(_org_id, self.APPROVED)
        org_approved_event.emit(_org_id, self.APPROVED)

    '''update_org_status when _action=1,means to suspend the org,else when
    _action=2,means to revoke the suspended org'''

    @register.public(str, int)
    def update_org_status(self, _org_id, _action):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._org_is_exists(_org_id), "update org status first ensures the org already existed"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"
        assert _org_id != self.alliance_admin_org, "can't update alliance admin org status"

        cur_status = self.NOT_IN_LIST
        if _action == self.OP_SUSPEND:
            cur_status = self.APPROVED
        elif _action == self.OP_REVOKE_SUSPEND:
            cur_status = self.SUSPENDED
        assert self.check_org_status(_org_id, cur_status), "current status is not wanted status"
        if _action == self.OP_SUSPEND:
            self._suspend_org(_org_id)
        elif _action == self.OP_REVOKE_SUSPEND:
            self._revoke_suspended_org(_org_id)

    @register.public(str, int)
    def approve_org_status(self, _org_id, _action):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._org_is_exists(_org_id), "approve org status first ensures the org already existed"
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND, "action must be one of 1 or 2"
        assert _org_id != self.alliance_admin_org, "can't update alliance admin org status"

        cur_status = self.NOT_IN_LIST
        if _action == self.OP_SUSPEND:
            cur_status = self.PENDING_SUSPENSION
        elif _action == self.OP_REVOKE_SUSPEND:
            cur_status = self.PENDING_SUSPENSION_REVOKE
        assert self.check_org_status(_org_id, cur_status), "current status is not wanted status"
        if _action == self.OP_SUSPEND:
            self._approve_suspend_org(_org_id)
        elif _action == self.OP_REVOKE_SUSPEND:
            self._approve_revoke_suspended_org(_org_id)

    @register.public()
    def get_org_list(self):
        return self.org_list

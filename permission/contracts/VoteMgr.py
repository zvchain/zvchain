import ujson

voter_added_event = Event("voter_added")
voter_deleted_event = Event("voter_deleted_event")
voting_item_added_event = Event("voting_item_added")
voting_processed_event = Event("voting_processed")


class VoteManager:
    class PendingOpDetails:
        def __init__(self, _org_id, _node_id, _account, _op_type, _miner_info):
            self.org_id = _org_id
            self.node_id = _node_id
            self.account = _account
            self.op_type = _op_type
            self.miner_info = _miner_info
            self.passed = False

            # json array, marshal str
            self.voted_accounts = ''
            self.voted_counts = 0

    class VoterDetails:
        def __init__(self, _account, _active):
            self.account = _account
            self.active = _active

    def __init__(self):
        self.perm_upgradable_addr = ''
        self.org_id = 'alliance_admin_org'
        self.total_voter_count = 0
        self.valid_voter_count = 0

        # json array,marshal pending PendingOpDetails obj
        self.pending_op_list = ''
        self.pending_op_index = zdict()
        self.pending_op_count = 0

        # json array,marshal passed PendingOpDetails obj
        self.passed_op_list = ''
        self.passed_op_count = 0

        # json array,marshal VoterDetails obj
        self.voter_list = ''
        self.voter_index = zdict()

    @register.public(str)
    def set_org_id(self, _alliance_admin_org):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        self.org_id = _alliance_admin_org

    @register.public(str)
    def set_upgradable_impl(self, _perm_upgradable_addr):
        if self.perm_upgradable_addr == '':
            self.perm_upgradable_addr = _perm_upgradable_addr

    # only from permission impl contract
    def _invoke_from_permission_impl(self):
        return msg.sender == Contract(self.perm_upgradable_addr).get_perm_impl()

    def _is_alliance_admin_org(self, _org_id):
        return _org_id == self.org_id

    def _voter_is_exist(self, _account):
        return _account in self.voter_index

    def _get_voter_id(self, _account):
        assert self._voter_is_exist(_account), "voter isn't exist"
        return self.voter_index[_account]

    def _pending_op_exist(self, _key):
        return _key in self.pending_op_index

    def _never_voted(self, _key, _caller):
        _pending_op_list = ujson.loads(self.pending_op_list)
        _voted_accounts = _pending_op_list[self.pending_op_index[_key]]['voted_accounts']
        if _voted_accounts == '':
            return True
        else:
            return _caller not in ujson.loads(_voted_accounts)

    def _voter_is_valid(self, _caller):
        idx = self._get_voter_id(_caller)
        _voter_list = ujson.loads(self.voter_list)
        return self._voter_is_exist(_caller) and _voter_list[idx]['active']

    def _gen_key(self, _org_id, _node_id, _account, _op_type, _miner_info):
        _key = str(_org_id) + str(_node_id) + str(_account) + str(_op_type) + str(_miner_info)
        return _key

    @register.public(str)
    def set_permission_impl(self, _permission_impl):
        self.permission_impl = _permission_impl

    @register.public(str, str)
    def add_voter(self, _org_id, _account):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._is_alliance_admin_org(_org_id), "org id must be the alliance admin org"
        assert not self._voter_is_exist(_account), "for adding the voter shouldn't be added before"

        voter_details = self.VoterDetails(_account, True)
        if self.voter_list == '':
            self.voter_list = ujson.dumps([voter_details.__dict__])
        else:
            _voter_list = ujson.loads(self.voter_list)
            _voter_list.append(voter_details.__dict__)
            self.voter_list = ujson.dumps(_voter_list)

        self.voter_index[_account] = self.total_voter_count

        self.total_voter_count += 1
        self.valid_voter_count += 1

        voter_added_event.emit(_org_id, _account)

    # deleting a voter is a logical deletion
    @register.public(str, str)
    def delete_voter(self, _org_id, _account):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._is_alliance_admin_org(_org_id), "org id must be the alliance admin org"
        assert self._voter_is_exist(_account), "for delete the voter,it must be added to voter list before"

        self.valid_voter_count -= 1

        _voter_list = ujson.loads(self.voter_list)
        _voter_list[self._get_voter_id(_account)]['active'] = False
        self.voter_list = ujson.dumps(_voter_list)

        voter_deleted_event.emit(_org_id, _account)

    @register.public(str, str, str, int, str)
    def add_item(self, _org_id, _node_id, _account, _op_type, _miner_info):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        _key = self._gen_key(_org_id, _node_id, _account, _op_type, _miner_info)
        self.pending_op_index[_key] = self.pending_op_count

        pending_op_details = self.PendingOpDetails(_org_id, _node_id, _account, _op_type, _miner_info)
        if self.pending_op_list == '' or self.pending_op_count == 0:
            self.pending_op_list = ujson.dumps([pending_op_details.__dict__])
        else:
            _pending_op_list = ujson.loads(self.pending_op_list)
            _pending_op_list.append(pending_op_details.__dict__)
            self.pending_op_list = ujson.dumps(_pending_op_list)

        self.pending_op_count += 1
        voting_item_added_event.emit(_org_id, _node_id, _account, _op_type)

    @register.public(str, str, str, int, str, str)
    def approve_item(self, _org_id, _node_id, _account, _op_type, _miner_info, _caller):
        assert self._invoke_from_permission_impl(), "this func must invoke from permission implementation contract"
        assert self._voter_is_valid(_caller), "caller must be a valid voter"
        _key = self._gen_key(_org_id, _node_id, _account, _op_type, _miner_info)
        assert self._pending_op_exist(_key), "pending option must existed"
        assert self._never_voted(_key, _caller), "caller should ensure that never voted this pending item before"

        _pending_op_list = ujson.loads(self.pending_op_list)
        voted_accounts = _pending_op_list[self.pending_op_index[_key]]['voted_accounts']
        voted_counts = _pending_op_list[self.pending_op_index[_key]]['voted_counts']

        if voted_accounts == '' or voted_counts == 0:
            _pending_op_list[self.pending_op_index[_key]]['voted_accounts'] = ujson.dumps([_caller])
        else:
            _voted_accounts = ujson.loads(voted_accounts)
            _voted_accounts.append(_caller)
            _pending_op_list[self.pending_op_index[_key]]['voted_accounts'] = ujson.dumps(_voted_accounts)

        _pending_op_list[self.pending_op_index[_key]]['voted_counts'] += 1

        voting_processed_event.emit(_org_id, _node_id, _account, _op_type, _caller)

        if _pending_op_list[self.pending_op_index[_key]]['voted_counts'] > self.valid_voter_count // 2:
            _pending_op_list[self.pending_op_index[_key]]['passed'] = True

            if self.passed_op_list == '':
                self.passed_op_list = ujson.dumps([_pending_op_list[self.pending_op_index[_key]]])
            else:
                _passed_op_list = ujson.loads(self.passed_op_list)
                _passed_op_list.append(_pending_op_list[self.pending_op_index[_key]])
                self.passed_op_list = ujson.dumps(_passed_op_list)

            del _pending_op_list[self.pending_op_index[_key]]
            del self.pending_op_index[_key]

            # due to delete item from pending_op_list, pending_op_index need to change
            for i in range(len(_pending_op_list)):
                #  recover key for change pending_op_index
                _key = self._gen_key(_pending_op_list[i]['org_id'],
                                     _pending_op_list[i]['node_id'],
                                     _pending_op_list[i]['account'],
                                     _pending_op_list[i]['op_type'],
                                     _pending_op_list[i]['miner_info'])
                self.pending_op_index[_key] = i

            self.pending_op_list = ujson.dumps(_pending_op_list)
            self.pending_op_count -= 1
            self.passed_op_count += 1
            return True
        else:
            self.pending_op_list = ujson.dumps(_pending_op_list)
            return False

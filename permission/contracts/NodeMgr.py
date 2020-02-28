import ujson

node_proposed_event = Event("node_proposed")
node_approved_event = Event("node_approved")
node_pending_deactivated_event = Event("node_pending_deactivated")
node_deactivated_event = Event("node_deactivated")
node_activated_event = Event("node_activated")
miner_proposed_event = Event("miner_proposed")
miner_approved_event = Event("miner_approved")
miner_pending_removed_event = Event("miner_pending_removed")
miner_removed_event = Event("miner_removed")


class NodeManager:
    # node status
    NOT_IN_LIST = 0
    PENDING_APPROVAL = 1
    ACTIVE = 2
    PENDING_SUSPENDED = 3
    SUSPENDED = 4

    # miner role
    NOT_A_MINER = 0
    PROPOSAL_MINER = 1
    VERIFY_MINER = 2
    PROPOSAL_AND_VERIFY_MINER = 3

    # miner status
    MINER_NOT_VALIDATED = 0
    MINER_PENDING_VALIDATE = 1
    MINER_VALIDATED = 2
    MINER_PENDING_ABOLISH = 3

    # weight
    NO_WEIGHT = 0
    DEFAULT_WEIGHT = 10

    # status op
    OP_SUSPEND = 1
    OP_REVOKE_SUSPEND = 2
    OP_PENDING_SUSPEND = 3

    class NodeDetails:
        def __init__(self, _node_id, _org_id, _status, _miner_role, _vrf_pk, _bls_pk, _weight, _miner_status):
            self.node_id = _node_id
            self.org_id = _org_id
            self.status = _status

            # miner message
            self.miner_role = _miner_role
            self.vrf_pk = _vrf_pk
            self.bls_pk = _bls_pk
            self.weight = _weight
            self.miner_status = _miner_status

    def __init__(self):
        self.perm_upgradable_addr = ''

        # use an array to store node details
        self.node_list = ''
        # mapping of nodeid to array index to track node
        self.node_id_to_index = zdict()
        # tracking total number of nodes in network
        self.number_of_nodes = 0

    # checks if node id is linked to the org id passed
    def _check_org(self, _node_id, _org_id):
        assert not self.node_list == '', "node list is empty"
        list_of_nodes = ujson.loads(self.node_list)
        return list_of_nodes[self._get_node_index(_node_id)]['org_id'] == _org_id

    def _get_node_index(self, _node_id):
        return self.node_id_to_index[_node_id]

    def _get_node_status(self, _node_id):
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert not self.node_list == '', "node list is empty"
        list_of_nodes = ujson.loads(self.node_list)
        return list_of_nodes[self._get_node_index(_node_id)]['status']

    def _check_miner_role(self, _node_id, _miner_role):
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert not self.node_list == '', "node list is empty"
        list_of_nodes = ujson.loads(self.node_list)
        return list_of_nodes[self._get_node_index(_node_id)]['miner_role'] == _miner_role

    def _get_miner_status(self, _node_id):
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert not self.node_list == '', "node list is empty"
        list_of_nodes = ujson.loads(self.node_list)
        return list_of_nodes[self._get_node_index(_node_id)]['miner_status']

    def check_miner_status(self, _node_id, _miner_status):
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert not self.node_list == '', "node list is empty"
        list_of_nodes = ujson.loads(self.node_list)
        return list_of_nodes[self._get_node_index(_node_id)]['miner_status'] == _miner_status

    # confirms that the caller is the address of implementation
    def only_implementation(self):
        return msg.sender == Contract(self.perm_upgradable_addr).get_perm_impl()

    # checks if the node exists in the network
    def node_exists(self, _node_id):
        return _node_id in self.node_id_to_index

    @register.public(str)
    def set_upgradable_impl(self, _perm_upgradable_addr):
        if self.perm_upgradable_addr == '':
            self.perm_upgradable_addr = _perm_upgradable_addr

    # fetches the node details given an node id
    @register.public(str)
    def get_node_details(self, _node_id):
        assert self.node_exists(_node_id), "passed node id does not exist"
        node_index = self._get_node_index(_node_id)
        list_of_nodes = ujson.loads(self.node_list)
        assert len(list_of_nodes) > 0 and node_index <= len(list_of_nodes) - 1, "node index does not exist"
        return ujson.dumps(list_of_nodes[node_index])

    # fetches the node details given the index of the node
    @register.public(int)
    def get_node_details_form_index(self, _node_index):
        assert not self.node_list == '', "node list is empty"
        list_of_nodes = ujson.loads(self.node_list)
        assert len(list_of_nodes) > 0 and _node_index <= len(list_of_nodes) - 1, "node index does not exist"
        return ujson.dumps(list_of_nodes[_node_index])

    # returns the total number of nodes in the network
    @register.public()
    def get_number_of_nodes(self):
        return self.number_of_nodes

    # called at the time of network initialization for adding
    @register.public(str, str, int, str, str, int)
    def add_admin_node(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        assert self.only_implementation(), "invalid caller"
        assert not self.node_exists(_node_id), "passed node id exist"
        assert _miner_role == self.NOT_A_MINER or _miner_role == self.PROPOSAL_MINER or \
               _miner_role == self.VERIFY_MINER or _miner_role == self.PROPOSAL_AND_VERIFY_MINER, "miner role must be one of 0,1,2,3"
        self.node_id_to_index[_node_id] = self.number_of_nodes
        self.number_of_nodes += 1

        _miner_status = self.MINER_NOT_VALIDATED
        if _miner_role == self.PROPOSAL_MINER or _miner_role == self.VERIFY_MINER or _miner_role == self.PROPOSAL_AND_VERIFY_MINER:
            _miner_status = self.MINER_VALIDATED

        if self.node_list == '':
            node = self.NodeDetails(_node_id, _org_id, self.ACTIVE, _miner_role, _vrf_pk, _bls_pk, _weight,
                                    _miner_status)
            self.node_list = ujson.dumps([node.__dict__])
        else:
            list_of_nodes = ujson.loads(self.node_list)
            node = self.NodeDetails(_node_id, _org_id, self.ACTIVE, _miner_role, _vrf_pk, _bls_pk, _weight,
                                    _miner_status)
            list_of_nodes.append(node.__dict__)
            self.node_list = ujson.dumps(list_of_nodes)
        miner_approved_event.emit(_node_id, _org_id, _miner_role, _weight, _miner_status)

    # called at the time of new org creation to add node to org
    @register.public(str, str)
    def add_node(self, _node_id, _org_id):
        assert self.only_implementation(), "invalid caller"
        assert not self.node_exists(_node_id), "passed node id exist"
        self.node_id_to_index[_node_id] = self.number_of_nodes
        self.number_of_nodes += 1
        node = self.NodeDetails(_node_id, _org_id, self.PENDING_APPROVAL, self.NOT_A_MINER, "", "",
                                self.NO_WEIGHT, self.MINER_NOT_VALIDATED)
        if self.node_list == '':
            self.node_list = ujson.dumps([node.__dict__])
        else:
            list_of_nodes = ujson.loads(self.node_list)
            list_of_nodes.append(node.__dict__)
            self.node_list = ujson.dumps(list_of_nodes)
        node_proposed_event.emit(_node_id, _org_id, self.PENDING_APPROVAL)

    # function to approve the node addition. only called at the time
    @register.public(str, str)
    def approve_node(self, _node_id, _org_id):
        assert self.only_implementation(), "invalid caller"
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert self._check_org(_node_id, _org_id), "node id does not belong to the passed org id"
        assert self._get_node_status(_node_id) == self.PENDING_APPROVAL, "nothing pending for approval"
        node_index = self._get_node_index(_node_id)
        list_of_nodes = ujson.loads(self.node_list)
        list_of_nodes[node_index]['status'] = self.ACTIVE
        self.node_list = ujson.dumps(list_of_nodes)
        node_approved_event.emit(list_of_nodes[node_index]['node_id'], list_of_nodes[node_index]['org_id'],
                                 self.ACTIVE)

    # called org admins to add new nodes to the org or called alliance admins to add new miner node
    @register.public(str, str, int, str, str, int, bool)
    def add_org_node(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight, _add_miner):
        assert self.only_implementation(), "invalid caller"
        assert not self.node_exists(_node_id), "passed node id exist"
        assert _miner_role == self.NOT_A_MINER or _miner_role == self.PROPOSAL_MINER or \
               _miner_role == self.VERIFY_MINER or _miner_role == self.PROPOSAL_AND_VERIFY_MINER, "miner role must be one of 0,1,2,3"
        self.node_id_to_index[_node_id] = self.number_of_nodes
        self.number_of_nodes += 1

        # don't need vote by voter
        miner_status = self.MINER_NOT_VALIDATED
        if _add_miner:
            miner_status = self.MINER_PENDING_VALIDATE

        if self.node_list == '':
            node = self.NodeDetails(_node_id, _org_id, self.ACTIVE, _miner_role, _vrf_pk, _bls_pk, _weight,
                                    miner_status)
            self.node_list = ujson.dumps([node.__dict__])
        else:
            list_of_nodes = ujson.loads(self.node_list)
            node = self.NodeDetails(_node_id, _org_id, self.ACTIVE, _miner_role, _vrf_pk, _bls_pk, _weight,
                                    miner_status)
            list_of_nodes.append(node.__dict__)
            self.node_list = ujson.dumps(list_of_nodes)

        if _add_miner:
            miner_proposed_event.emit(_node_id, _org_id, _miner_role, _weight, miner_status)
        else:
            node_proposed_event.emit(_node_id, _org_id, self.ACTIVE)

    @register.public(str, str, int)
    def approve_org_node(self, _node_id, _org_id, _miner_role):
        assert self.only_implementation(), "invalid caller"
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert self._check_org(_node_id, _org_id), "node id does not belong to the passed org id"
        assert self._check_miner_role(_node_id, _miner_role), "miner role does not meet the requirements"
        assert self.check_miner_status(_node_id,
                                       self.MINER_PENDING_VALIDATE), "miner status does not meet the requirements"

        node_index = self._get_node_index(_node_id)
        list_of_nodes = ujson.loads(self.node_list)
        list_of_nodes[node_index]['miner_status'] = self.MINER_VALIDATED
        self.node_list = ujson.dumps(list_of_nodes)
        miner_approved_event.emit(list_of_nodes[node_index]['node_id'], list_of_nodes[node_index]['org_id'],
                                  list_of_nodes[node_index]['miner_role'], list_of_nodes[node_index]['weight'],
                                  self.MINER_VALIDATED)

    @register.public(str, str, int, str, str, int)
    def assign_node_to_miner(self, _node_id, _org_id, _miner_role, _vrf_pk, _bls_pk, _weight):
        assert self.only_implementation(), "invalid caller"
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert self._check_org(_node_id, _org_id), "node id does not belong to the passed org id"

        assert self.check_miner_status(_node_id,
                                       self.MINER_NOT_VALIDATED), "miner status does not meet the requirements"

        _idx = self._get_node_index(_node_id)
        list_of_nodes = ujson.loads(self.node_list)

        list_of_nodes[_idx]['miner_role'] = _miner_role
        list_of_nodes[_idx]['vrf_pk'] = _vrf_pk
        list_of_nodes[_idx]['bls_pk'] = _bls_pk
        list_of_nodes[_idx]['weight'] = _weight
        list_of_nodes[_idx]['miner_status'] = self.MINER_PENDING_VALIDATE
        self.node_list = ujson.dumps(list_of_nodes)
        miner_proposed_event.emit(list_of_nodes[_idx]['node_id'], list_of_nodes[_idx]['org_id'],
                                  list_of_nodes[_idx]['miner_role'], list_of_nodes[_idx]['weight'],
                                  self.MINER_PENDING_VALIDATE)

    @register.public(str, str, bool)
    def remove_miner(self, _node_id, _org_id, _disable_node):

        assert self.only_implementation(), "invalid caller"
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert self._check_org(_node_id, _org_id), "node id does not belong to the passed org id"
        assert self.check_miner_status(_node_id, self.MINER_VALIDATED), "miner status does not meet the requirements"

        _idx = self._get_node_index(_node_id)
        list_of_nodes = ujson.loads(self.node_list)

        list_of_nodes[_idx]['miner_status'] = self.MINER_PENDING_ABOLISH
        self.node_list = ujson.dumps(list_of_nodes)
        miner_pending_removed_event.emit(_node_id, _org_id)
        if _disable_node:
            self.update_node_status(_node_id, _org_id, self.OP_PENDING_SUSPEND, True)

    @register.public(str, str, bool)
    def approve_remove_miner(self, _node_id, _org_id, _disable_node):

        assert self.only_implementation(), "invalid caller"
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert self._check_org(_node_id, _org_id), "node id does not belong to the passed org id"

        _idx = self._get_node_index(_node_id)
        list_of_nodes = ujson.loads(self.node_list)

        list_of_nodes[_idx]['miner_role'] = self.NOT_A_MINER
        list_of_nodes[_idx]['vrf_pk'] = ''
        list_of_nodes[_idx]['bls_pk'] = ''
        list_of_nodes[_idx]['weight'] = self.NO_WEIGHT
        list_of_nodes[_idx]['miner_status'] = self.MINER_NOT_VALIDATED
        self.node_list = ujson.dumps(list_of_nodes)
        miner_removed_event.emit(_node_id, _org_id)
        if _disable_node:
            self.update_node_status(_node_id, _org_id, self.OP_SUSPEND, True)

    #  updates the node status
    @register.public(str, str, int, bool)
    def update_node_status(self, _node_id, _org_id, _action, _need_vote):
        assert self.only_implementation(), "invalid caller"
        assert self.node_exists(_node_id), "passed node id does not exists"
        assert self._check_org(_node_id, _org_id), "node id does not belong to the passed org id"
        assert self._get_miner_status(
            _node_id) == self.MINER_NOT_VALIDATED or (_need_vote and self._get_miner_status(
            _node_id) == self.MINER_PENDING_ABOLISH), 'miner status is not wanted status or it\'s a miner node,' \
                                                      'your identity can\'t change the node status'
        assert _action == self.OP_SUSPEND or _action == self.OP_REVOKE_SUSPEND or _action == self.OP_PENDING_SUSPEND, \
            "invalid operation. wrong action passed"

        if _action == self.OP_SUSPEND:
            assert self._get_node_status(_node_id) == self.ACTIVE or self._get_node_status(
                _node_id) == self.PENDING_SUSPENDED, "operation cannot be performed"
            list_of_nodes = ujson.loads(self.node_list)
            list_of_nodes[self._get_node_index(_node_id)]['status'] = self.SUSPENDED
            self.node_list = ujson.dumps(list_of_nodes)
            node_deactivated_event.emit(_node_id, _org_id)
        elif _action == self.OP_REVOKE_SUSPEND:
            assert self._get_node_status(_node_id) == self.SUSPENDED, "operation cannot be performed"
            list_of_nodes = ujson.loads(self.node_list)
            list_of_nodes[self._get_node_index(_node_id)]['status'] = self.ACTIVE
            self.node_list = ujson.dumps(list_of_nodes)
            node_activated_event.emit(_node_id, _org_id)

        elif _action == self.OP_PENDING_SUSPEND:
            assert self._get_node_status(_node_id) == self.ACTIVE, "operation cannot be performed"
            list_of_nodes = ujson.loads(self.node_list)
            list_of_nodes[self._get_node_index(_node_id)]['status'] = self.PENDING_SUSPENDED
            self.node_list = ujson.dumps(list_of_nodes)
            node_pending_deactivated_event.emit(_node_id, _org_id)

    @register.public()
    def get_node_list(self):
        return self.node_list

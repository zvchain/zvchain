import account
event = Event("send")
class Router(object):
    def __init__(self):
        self.name = "router"

    @register.public(str, str, str, str)
    def call_contract(self, addr, contract_name, value):
        self.name = 'tt'
        event.emit(addr=addr)
        return account.contract_call(addr, contract_name, value)

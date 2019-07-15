import account
event = Event("send")
class Router(object):
    def __init__(self):
        self.name = "router"

    @register.public(str, str, str, str)
    def call_contract(self, addr, func_name, value):
        self.name = 'tt'
        event.emit(addr=addr)
        if func_name == "set_name":
            return Contract(addr).set_name(value)
        else:
            return Contract(addr).private_set_name(value)

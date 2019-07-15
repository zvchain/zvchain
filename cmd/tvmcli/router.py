import account
event = Event("send")
class Router(object):
    def __init__(self):
        self.name = "router"

    @register.public(str, str, str, str)
    def call_contract(self, addr, func_name, value):
        self.name = 'router1'
        event.emit(addr=addr)
        if func_name == "set_name":
            print("py print", Contract(addr).set_name(value))
        else:
            print("py print", Contract(addr).private_set_name(value))

    @register.public(str, int)
    def call_contract2(self, addr, times):
        if times == 0:
            return
        event.emit(times)
        Contract(addr).call_contract2(addr, times-1)

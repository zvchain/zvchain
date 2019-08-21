import account
# event = Event("send")
class Router(object):
    def __init__(self):
        self.name = "router"

    @register.public(str, str, str)
    def call_contract(self, addr, func_name, value):
        # event.emit(addr=addr)
        print("py print", getattr(Contract(addr), func_name)(value))

    @register.public(str, int)
    def call_contract2(self, addr, times):
        if times == 0:
            return
        # event.emit(times)
        Contract(addr).call_contract2(addr, times-1)

    @register.public(str, int)
    def call_contract3(self, addr, times):
        if times == 0:
            return
        # event.emit(times)
        Contract(addr).call_contract3(addr, times-1)

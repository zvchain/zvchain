# event = Event("receiver")
class Receiver():
    def __init__(self):
        print('__init__', msg)
        self.name = "receiver"

    @register.public(str)
    def set_name(self, name):
        # event.emit(name=name)
        print('set_name', msg)
        print('set_name', name)
        self.name = name
        return name

    def private_set_name(self, name):
        self.name = name

    @register.public(str, int)
    def call_contract2(self, addr, times):
        if times == 0:
            return
        # event.emit(times)
        # error
        Contract(addr).contract_call2(addr, times-1)

    @register.public(str, int)
    def call_contract3(self, addr, times):
        if times == 0:
            return
        # event.emit(times)
        Contract(addr).call_contract3(addr, times-1)
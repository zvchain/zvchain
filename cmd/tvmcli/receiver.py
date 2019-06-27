
class Receiver():
    def __init__(self):
        print('__init__', msg)
        self.name = "receiver"

    @register.public(str)
    def set_name(self, name):
        print('set_name', msg)
        self.name = name

    def private_set_name(self, name):
        self.name = name
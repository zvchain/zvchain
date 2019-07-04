import account
class Setandget(object):
    def __init__(self):
        pass

    @register.public(str, str)
    def setdata(self, key,value):
        return account.set_data(key,value)


    @register.public(str)
    def getdata(self, addr):
        return account.get_data(addr)

    @register.public(str)
    def removedata(self, key):
        return account.remove_data(key)
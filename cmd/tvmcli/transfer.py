import account
class Transfer(object):
    def __init__(self):
        pass

    @register.public(str, int)
    def transfer(self, addr,amount):
        print ("PY>>>>")
        return account.transfer(addr,amount)


    @register.public(str)
    def ckeckbalance(self, addr):
        return account.get_balance(addr)
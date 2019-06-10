import account
class Stake(object):
    def __init__(self):
        pass
 
    @register.public(str, int, int)
    def stake(self, addr, _type, value):
        account.stake(addr, _type, value)
 
    @register.public(str, int, int)
    def cancel_stake(self, addr, _type, value):
        account.cancel_stake(addr, _type, value)
 
    @register.public(str, int)
    def refund_stake(self, addr, _type):
        account.refund_stake(addr, _type)
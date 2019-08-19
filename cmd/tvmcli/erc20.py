class Token(object):
    def __init__(self):
        self.name = 'ZVC Token'
        self.symbol = "TAS"
        self.decimal = 3

        self.totalSupply = 100000

        self.balanceOf = zdict()
        self.allowance = zdict()

        self.balanceOf['0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd'] = self.totalSupply

        # self.owner = msg.sender

    # @register.public()
    # def symbol(self):
    #     return self.symbol

    @register.public(str)
    def balance_of(self, address):
        return self.balanceOf[address]

    def _transfer(self, _from, _to, _value):
        if self.balanceOf[_to] is None:
            self.balanceOf[_to] = 0
        if self.balanceOf[_from] is None:
            self.balanceOf[_from] = 0
        # 接收账户地址是否合法
        # require(Address(_to).invalid())
        # 账户余额是否满足转账金额
        if self.balanceOf[_from] < _value:
            raise Exception('账户余额小于转账金额')
        # 检查转账金额是否合法
        if _value <= 0:
            raise Exception('转账金额必须大于等于0')
        # 转账
        self.balanceOf[_from] -= _value
        self.balanceOf[_to] += _value
        # Event.emit("Transfer", _from, _to, _value)

    @register.public(str, int)
    def transfer(self, _to, _value):
        self._transfer(msg.sender, _to, _value)

    @register.public(str, int)
    def approve(self, _spender, _valuexj):
        if _value <= 0:
            raise Exception('授权金额必须大于等于0')
        if self.allowance[msg.sender] is None:
            self.allowance[msg.sender] = TasCollectionStorage()
        self.allowance[msg.sender][_spender] = _value
        # account.eventCall('Approval', 'index', 'data')
        # Event.emit("Approval", msg.sender, _spender, _value)

    @register.public(str, str, int)
    def transfer_from(self, _from, _to, _value):
        if _value > self.allowance[_from][msg.sender]:
            raise Exception('超过授权转账额度')
        self.allowance[_from][msg.sender] -= _value
        self._transfer(_from, _to, _value)

    # def approveAndCall(self, _spender, _value, _extraData):
    #         spender = Address(_spender)
    #     if self.approve(spender, _value):
    #         spender.call("receive_approval", msg.sender, _value, this, _extraData)
    #         return True
    #     else:
    #         return False

    @register.public(int)
    def burn(self, _value):
        if _value <= 0:
            raise Exception('燃烧金额必须大于等于0')
        if self.balanceOf[msg.sender] < _value:
            raise Exception('账户余额不足')
        self.balanceOf[msg.sender] -= _value
        self.totalSupply -= _value
        # Event.emit("Burn", msg.sender, _value)

    # def burn_from(self, _from, _value):
    #     # if _from not in self.balanceOf:
    #     #     self.balanceOf[_from] = 0
    #     #检查账户余额
    #     require(self.balanceOf[_from] >= _value)
    #     require(_value <= self.allowance[_from][msg.sender])
    #     self.balanceOf[_from] -= _value
    #     self.allowance[_from][msg.sender] -= _value
    #     self.totalSupply -= _value
    #     Event.emit("Burn", _from, _value)
    #     return True

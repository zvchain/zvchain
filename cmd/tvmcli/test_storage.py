
class Token():
    def __init__(self):
        # print(self.foo)

        # self.foo = 'a'
        # print(self.foo)
        # del self.foo
        # print(self.foo)

        self.int = 2147483647
        print(self.int)
        self.bigint = 10000000000000000000000000000000
        print(self.bigint)
        self.str = 'hello'
        print(self.str)
        self.str = ''
        print(self.str)
        self.bool = True
        print(self.bool)
        self.bool = False
        print(self.bool)
        self.none = None
        print(self.none)

        self.bytes = b'hello world'
        print(self.bytes)

        self.zdict = zdict()
        print(self.zdict)
        # print(self.zdict['a'])

        self.zdict['a'] = 'b'
        print(self.zdict['a'])
        print('a' in self.zdict)
        print('b' in self.zdict)

        self.zdict['c'] = 'b'
        # del self.zdict['c']
        print(self.zdict['c'])

        self.zdict['b'] = zdict()
        print(self.zdict['b'])
        self.zdict['b']['c'] = 'd'
        # del self.zdict['b']['c']
        print(self.zdict['b']['c'])

        self.zdict['b']['d'] = zdict()
        print(self.zdict['b']['d'])
        self.zdict['b']['d']['e'] = 'f'
        print(self.zdict['b']['d']['e'])

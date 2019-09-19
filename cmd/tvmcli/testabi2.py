class TestABI():
    def __init__(self):
        self.b = True

    @register.public(int)
    def testint(self, count):
        self.count = count

    @register.public(str)
    def teststr(self, string):
        self.string = string

    @register.public(bool)
    def testbool(self, b):
        self.b = b
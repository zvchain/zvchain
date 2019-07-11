class TestABI():
    def __init__(self):
        pass

    @register.public(int)
    def testint(self, count):
        self.count = count

    @register.public(str)
    def teststr(self, string):
        self.string = string
class TestABI():
    def __init__(self):
        pass

    @register.public(int)
    def testint(self, count):
        self.count = count

    @register.public(str)
    def teststr(self, string):
        self.string = string

    @register.public()
    def testutf8(self):
        self.utf8 = '你好，世界'

    @register.public(str)
    def testutf82(self, s):
        self.utf82 = s
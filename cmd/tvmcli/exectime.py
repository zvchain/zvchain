

class Max():
    def __init__(self):
        pass

    @register.public(int)
    def exec1(self, max):
        counter = ""
        while 0 <= max:
            counter += str(max)
            max -= 1

    @register.public(int)
    def exec2(self, max):
        counter = 0
        while counter < max:
            counter += 1

    @register.public(int)
    def exec3(self, max):
        self.counter = 0
        while self.counter < max:
            self.counter += 1
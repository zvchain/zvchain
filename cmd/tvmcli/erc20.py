import account
import ujson

class Distribution(object):
    def __init__(self):
        self.distribution_lists = zdict()
        self.admin = msg.sender
        self.total_percentage_dict = zdict()

    @register.public(str, dict)
    def init_distribution_list(self, group, distribution_list):
        if msg.sender != self.admin:
            return
        self.distribution_lists[group] = '''{"zvede238caaaaca16c1473bb66fcd605cb9c3c88992c7d1053130e2de65fa5fd7a": 100, "zvede238caaaaca16c1473bb66fcd605cb9c3c88992c7d1053130e2de65fa5fd7b": 100}'''
        #ujson.dumps(distribution_list)
        total_percentage_dict = 0
        for k in distribution_list:
            if type(distribution_list[k]) != int:
                return
            total_percentage_dict += distribution_list[k]
        self.total_percentage_dict[group] = total_percentage_dict

    @register.public(str)
    def deposit(self, group):
        if msg.value == 0:
            return
        if group not in self.distribution_lists:
            return
        all_user = ujson.loads(self.distribution_lists[group])
        total = self.total_percentage_dict[group]
        for user in all_user:
            can_withdraw = msg.value * all_user[user] // total
            account.transfer(user, can_withdraw)
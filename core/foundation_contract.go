//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

const foundationContract = `
import block
import account
class Foundation(object):
    def __init__(self):
        self.admin = "%s"
        self.total_token = %d
        self.withdrawed = 0
        self.first_year_weight = 64
        self.total_weight = 360
        self.height_per_period = 10000000

    def calculate_weight(self, period):
        return self.first_year_weight // (2 ** (period // 3))

    def calculate_released(self):
        n = block.number()
        period = n // self.height_per_period
        remainder = n %% self.height_per_period
        if period >= 12:
            return self.total_token
        weight = 0
        for i in range(period):
            weight = weight + self.calculate_weight(i)
        tokens = self.total_token * weight // self.total_weight
        tokens += self.total_token * self.calculate_weight(period) * remainder // self.total_weight // self.height_per_period
        return tokens

    @register.public(int)
    def withdraw(self, amount):
        if msg.sender != self.admin:
            return
        can_withdraw = self.calculate_released() - self.withdrawed
        if amount > can_withdraw:
            return
        if account.get_balance(this) < amount:
            return
        self.withdrawed += amount
        account.transfer(self.admin, amount)

    @register.public(str)
    def change_admin(self, admin):
        if msg.sender != self.admin:
            return
        self.admin = admin

`

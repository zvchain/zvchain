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

const addressManagerContract = `
def checkAddr(addr):
    if len(addr) != 66:
        raise Exception('address length is wrong !')
    if not addr.startswith('zv'):
        raise Exception('address prefix is wrong!')


class AddressManager(object):
    """docstring fos AddressManager"""

    def __init__(self):
        self.adminAddr = "zv556dca04a59808f1598f90fabb1fa8a061ed1a636d270ff1a0c809e8aeb000ed"
        self.stakePlatformAddr = "zv88200d8e51a63301911c19f72439cac224afc7076ee705391c16f203109c0ccf"
        self.circulatesAddr = "zv1d676136438ef8badbc59c89bae08ea3cdfccbbe8f4b22ac8d47361d6a3d510d"
        self.userNodeAddr = "zv9f03cdec76c6617a151c65d393e9f6149cec59f10df00bb8918be4873b314cf4"
        self.daemonNodeAddr = "zvb5344ed02ff6e01239c13e9e4a27b3c5caf28c1d9d2f1fa15b809607a33cb12d"

    @register.public(str)
    def set_admin_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.adminAddr:
            raise Exception('msg sender is not admin!')
        self.adminAddr = _addr

    @register.public(str)
    def set_stake_platform_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.stakePlatformAddr:
            raise Exception('msg sender is not stake platform addr!')
        self.stakePlatformAddr = _addr

    @register.public(str)
    def set_circulates_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.circulatesAddr:
            raise Exception('msg sender is not circulates addr!')
        self.circulatesAddr = _addr

    @register.public(str)
    def set_user_node_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.userNodeAddr:
            raise Exception('msg sender is not node addr!')
        self.userNodeAddr = _addr

    @register.public(str)
    def set_daemon_node_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.daemonNodeAddr:
            raise Exception('msg sender is not daemon node addr!')
        self.daemonNodeAddr = _addr

`
const addressManagerContractTest = `
def checkAddr(addr):
    if len(addr) != 66:
        raise Exception('address length is wrong !')
    if not (addr.startswith('zv')):
        raise Exception('address prefix is wrong!')


class AddressManager(object):
    """docstring fos AddressManager"""

    def __init__(self):
        self.adminAddr = "zv28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031b"
        self.stakePlatformAddr = "zv01cf40d3a25d0a00bb6876de356e702ae5a2a379c95e77c5fd04f4cc6bb680c0"
        self.circulatesAddr = "zvebb50bcade66df3fcb8df1eeeebad6c76332f2aee43c9c11b5cd30187b45f6d3"
        self.userNodeAddr = "zve30c75b3fd8888f410ac38ec0a07d82dcc613053513855fb4dd6d75bc69e8139"
        self.daemonNodeAddr = "zvae1889182874d8dad3c3e033cde3229a3320755692e37cbe1caab687bf6a1122"

    @register.public(str)
    def set_admin_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.adminAddr:
            raise Exception('msg sender is not admin!')
        self.adminAddr = _addr

    @register.public(str)
    def set_stake_platform_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.stakePlatformAddr:
            raise Exception('msg sender is not stake platform addr!')
        self.stakePlatformAddr = _addr

    @register.public(str)
    def set_circulates_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.circulatesAddr:
            raise Exception('msg sender is not circulates addr!')
        self.circulatesAddr = _addr

    @register.public(str)
    def set_user_node_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.userNodeAddr:
            raise Exception('msg sender is not node addr!')
        self.userNodeAddr = _addr

    @register.public(str)
    def set_daemon_node_addr(self, _addr):
        checkAddr(_addr)
        if msg.sender != self.daemonNodeAddr:
            raise Exception('msg sender is not daemon node addr!')
        self.daemonNodeAddr = _addr

`

//   Copyright (C) 2019 ZVChain
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

package account

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/trie"
)

func (adb *AccountDB) GetDataSafe(address common.Address, key []byte) []byte {
	adb.lock.RLock()
	defer adb.lock.RUnlock()
	return adb.GetData(address, key)
}

func (adb *AccountDB) SetDataSafe(address common.Address, key, val []byte) {
	adb.lock.Lock()
	defer adb.lock.Unlock()
	adb.SetData(address, key, val)
}

func (adb *AccountDB) RemoveDataSafe(address common.Address, key []byte) {
	adb.lock.RLock()
	defer adb.lock.RUnlock()
	adb.RemoveData(address, key)
}

func (adb *AccountDB) DataIteratorSafe(address common.Address, prefix []byte) *trie.Iterator {
	adb.lock.RLock()
	defer adb.lock.RUnlock()
	return adb.DataIterator(address, prefix)
}

func (adb *AccountDB) AsAccountDBTS() types.AccountDBTS {
	return adb
}

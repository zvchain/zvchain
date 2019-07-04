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

package group

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"sync"
)

type skStorage struct {
	file    string
	seckeys map[string]groupsig.Seckey
	lock    sync.RWMutex
}

func (store *skStorage) storeSeckey(prefix string, hash common.Hash, sk groupsig.Seckey) {
	store.lock.Lock()
	defer store.lock.Unlock()
	store.seckeys[hash.Hex()+prefix] = sk
}

func (store *skStorage) getSeckey(prefix string, hash common.Hash) (groupsig.Seckey, bool) {
	store.lock.RLock()
	defer store.lock.RUnlock()
	if v, ok := store.seckeys[hash.Hex()+prefix]; ok {
		return v, true
	}
	return groupsig.Seckey{}, false
}

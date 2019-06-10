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

package account

import (
	"strings"

	"github.com/zvchain/zvchain/storage/trie"
)

type DataIterator struct {
	*trie.Iterator
	object *accountObject
	prefix string
}

func (di *DataIterator) Next() bool {
	if len(di.prefix) == 0 {
		return di.Iterator.Next()
	}
	for di.Iterator.Next() {
		if strings.HasPrefix(string(di.Key), di.prefix) {
			return true
		}
	}
	return false
}

func (di *DataIterator) GetValue() []byte {
	if v, ok := di.object.dirtyStorage[string(di.Key)]; ok {
		return v
	}
	return di.Value
}

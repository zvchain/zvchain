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

package common

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"
)

/*
**  Creator: pxf
**  Date: 2019/5/8 上午11:37
**  Description:
 */

// MustNewLRUCache creates a new lru cache.
// Caution: if fail, the function will cause panic
func MustNewLRUCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}

// MustNewLRUCacheWithEvictCB creates a new lru cache with buffer eviction
// Caution: if fail, the function will cause panic.
func MustNewLRUCacheWithEvictCB(size int, cb func(k, v interface{})) *lru.Cache {
	cache, err := lru.NewWithEvict(size, cb)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}

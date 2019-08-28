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
	"crypto/rand"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/crypto/chacha20poly1305"
	"strings"
)

/*
**  Creator: pxf
**  Date: 2019/5/8 上午11:37
**  Description:
 */

// MustNewLRUCache creates a new lru cache.
// Caution: if fail, the function will cause panic
// developer should promise size > 0 when use this function
func MustNewLRUCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		// this err will only happens if size is native
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

// EncryptWithKey implements symmetric encryption with the specified key
// all data in encrypted within the storage using this function
func EncryptWithKey(Key []byte, Data []byte) (result []byte, err error) {
	nonce := make([]byte, chacha20poly1305.NonceSize, chacha20poly1305.NonceSize)
	cipher, err := chacha20poly1305.New(Key)
	if err != nil {
		return
	}

	_, err = rand.Read(nonce)
	if err != nil {
		return
	}
	Data = cipher.Seal(Data[:0], nonce, Data, nil) // is this okay

	result = append(Data, nonce...) // append nonce
	return
}

// DecryptWithKey implements symmetric decryption with the specified key
// extract 12 byte nonce from the data and deseal the data
// if key is incorrect, err return not nil
func DecryptWithKey(Key []byte, Data []byte) (result []byte, err error) {

	// make sure data is at least 28 bytes(includes 16 bytes of AEAD cipher and 12 bytes of nonce)
	if len(Data) < 28 {
		err = fmt.Errorf("Invalid data")
		return
	}

	dataWithoutNonce := Data[0 : len(Data)-chacha20poly1305.NonceSize]

	nonce := Data[len(Data)-chacha20poly1305.NonceSize:]

	cipher, err := chacha20poly1305.New(Key)
	if err != nil {
		return
	}

	return cipher.Open(result[:0], nonce, dataWithoutNonce, nil)

}


func CheckWeakPassword(password string)bool{
	password = strings.TrimSpace(password)
	if password == ""{
		return true
	}
	return false
}
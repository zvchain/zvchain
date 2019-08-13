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
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/zvchain/zvchain/common/ecies"
	"github.com/zvchain/zvchain/common/secp256k1"
)

// PrivateKey data struct
type PrivateKey struct {
	PrivKey ecdsa.PrivateKey
}

// Sign returns the message signature using the private key
func (pk PrivateKey) Sign(hash []byte) (Sign, error) {
	var sign Sign

	pribytes := pk.PrivKey.D.Bytes()
	seckbytes := pribytes
	if len(pribytes) < 32 {
		seckbytes = make([]byte, 32)
		copy(seckbytes[32-len(pribytes):32], pribytes) //make sure that the length of seckey is 32 bytes
	}

	sig, err := secp256k1.Sign(hash, seckbytes)
	if err == nil {
		signData := BytesToSign(sig)
		if signData == nil {
			err = fmt.Errorf("BytesToSign fail, sign=%x", sig)
		} else {
			sign = *signData
		}
	} else {
		err = fmt.Errorf("Sign Failed, reason : %v.\n", err.Error())
	}

	return sign, err
}

// GenerateKey creates a Private key by the specified string
func GenerateKey(s string) (PrivateKey, error) {
	var r io.Reader
	if len(s) > 0 {
		r = strings.NewReader(s)
	} else {
		r = rand.Reader
	}
	var pk PrivateKey
	_pk, err := ecdsa.GenerateKey(getDefaultCurve(), r)
	if err == nil {
		pk.PrivKey = *_pk
	} else {
		err = fmt.Errorf("GenKey Failed, reason : %v.\n", err.Error())
		return pk, err
	}
	return pk, nil
}

// GetPubKey returns the public key mapped to the private key
func (pk *PrivateKey) GetPubKey() PublicKey {
	var pubk PublicKey
	pubk.PubKey = pk.PrivKey.PublicKey
	return pubk
}

// Hex converts the private key to a hex string
func (pk *PrivateKey) Hex() string {
	return ToHex(pk.Bytes())
}

// HexToSecKey returns a private key with the hex string imported.
func HexToSecKey(s string) (sk *PrivateKey) {
	if len(s) < len(HexPrefix) || s[:len(HexPrefix)] != HexPrefix {
		return
	}
	sk = BytesToSecKey(FromHex(s))
	return
}

// Bytes converts the private key to a byte array
func (pk *PrivateKey) Bytes() []byte {
	buf := make([]byte, SecKeyLength)
	copy(buf[:PubKeyLength], pk.GetPubKey().Bytes())
	d := pk.PrivKey.D.Bytes()
	if len(d) > 32 {
		// this case must not happen
		panic("privateKey data length error: D length is more than 32!")
	}
	copy(buf[SecKeyLength-len(d):SecKeyLength], d)
	return buf
}

// ExportKey returns a byte array representation of the secure key of private key
func (pk *PrivateKey) ExportKey() []byte {
	return pk.PrivKey.D.Bytes()
}

// ImportKey constructs the private key from the input secure key.
func (pk *PrivateKey) ImportKey(key []byte) bool {
	var one = new(big.Int).SetInt64(1)

	params := getDefaultCurve().Params()
	d := new(big.Int).SetBytes(key)
	if d.Cmp(params.N) >= 0 || d.Cmp(one) < 0 {
		return false
	}

	pk.PrivKey.Curve = getDefaultCurve()
	pk.PrivKey.D = d
	pk.PrivKey.PublicKey.X, pk.PrivKey.PublicKey.Y = pk.PrivKey.Curve.ScalarBaseMult(key)
	return true
}

// BytesToSecKey returns a private key with the byte array imported
func BytesToSecKey(data []byte) (sk *PrivateKey) {
	//fmt.Printf("begin bytesToSecKey, len=%v, data=%v.\n", len(data), data)
	if len(data) < SecKeyLength {
		return nil
	}
	sk = new(PrivateKey)
	bufPub := data[:PubKeyLength]
	bufD := data[PubKeyLength:]
	sk.PrivKey.PublicKey = BytesToPublicKey(bufPub).PubKey
	sk.PrivKey.D = new(big.Int).SetBytes(bufD)
	if sk.PrivKey.X != nil && sk.PrivKey.Y != nil && sk.PrivKey.D != nil {
		return sk
	}
	return nil
}

// Decrypt returns the plain message
func (pk *PrivateKey) Decrypt(rand io.Reader, ct []byte) (m []byte, err error) {
	prv := ecies.ImportECDSA(&pk.PrivKey)
	return prv.Decrypt(rand, ct, nil, nil)
}

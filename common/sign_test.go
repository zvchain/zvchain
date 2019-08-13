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
	"testing"

	"bytes"

	"golang.org/x/crypto/sha3"
)

func TestPrivateKey(test *testing.T) {
	fmt.Printf("\nbegin TestPrivateKey...\n")
	sk, _ := GenerateKey("")
	str := sk.Hex()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)
	new_sk := HexToSecKey(str)
	new_str := new_sk.Hex()
	fmt.Printf("import sec key and export again, len=%v, data=%v.\n", len(new_str), new_str)
	fmt.Printf("end TestPrivateKey.\n")
}

func TestPublickKey(test *testing.T) {
	fmt.Printf("\nbegin TestPublicKey...\n")
	sk, _ := GenerateKey("")
	pk := sk.GetPubKey()
	//buf := pub_k.toBytes()
	//fmt.Printf("byte buf len of public key = %v.\n", len(buf))
	str := pk.Hex()
	fmt.Printf("pub key export, len=%v, data=%v.\n", len(str), str)
	new_pk := HexToPubKey(str)
	new_str := new_pk.Hex()
	fmt.Printf("import pub key and export again, len=%v, data=%v.\n", len(new_str), new_str)

	fmt.Printf("\nbegin test address...\n")
	a := pk.GetAddress()
	str = a.Hex()
	fmt.Printf("address export, len=%v, data=%v.\n", len(str), str)
	new_a := HexToAddress(str)
	new_str = new_a.Hex()
	fmt.Printf("import address and export again, len=%v, data=%v.\n", len(new_str), new_str)

	fmt.Printf("end TestPublicKey.\n")
}

func TestSign(test *testing.T) {
	fmt.Printf("\nbegin TestSign...\n")
	plain_txt := "My name is thiefox."
	buf := []byte(plain_txt)
	sha3_hash := sha3.Sum256(buf)
	pri_k, _ := GenerateKey("")
	pub_k := pri_k.GetPubKey()

	pub_buf := pub_k.Bytes()
	pub_k = *BytesToPublicKey(pub_buf)

	sha3_si, _ := pri_k.Sign(sha3_hash[:])
	{
		buf_r := sha3_si.r.Bytes()
		buf_s := sha3_si.s.Bytes()
		fmt.Printf("sha3 sign, r len = %v, s len = %v.\n", len(buf_r), len(buf_s))
	}
	success := pub_k.Verify(sha3_hash[:], &sha3_si)
	fmt.Printf("sha3 sign verify result=%v.\n", success)
	fmt.Printf("end TestSign.\n")
}

func TestEncryptDecrypt(t *testing.T) {
	fmt.Printf("\nbegin TestEncryptDecrypt...\n")
	sk1, _ := GenerateKey("")
	pk1 := sk1.GetPubKey()

	t.Log(sk1.Hex())
	t.Log(pk1.Hex())

	sk2, _ := GenerateKey("")

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &pk1, message)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := sk1.Decrypt(rand.Reader, ct)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	fmt.Println(message)
	fmt.Println(ct)
	fmt.Println(pt)

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = sk2.Decrypt(rand.Reader, ct)
	if err == nil {
		fmt.Println("ecies: encryption should not have succeeded")
		t.FailNow()
	}
	fmt.Printf("end TestEncryptDecrypt.\n")
}
func TestSignBytes(test *testing.T) {
	plain_txt := "Sign bytes convert."
	buf := []byte(plain_txt)

	pri_k, _ := GenerateKey("")

	sha3_hash := sha3.Sum256(buf)
	sign, _ := pri_k.Sign(sha3_hash[:])

	h := sign.Hex()
	fmt.Println(h)

	sign_bytes := sign.Bytes()
	sign_r := BytesToSign(sign_bytes)
	fmt.Println(sign_r.Hex())
	if h != sign_r.Hex() {
		fmt.Println("sign dismatch!", h, sign_r.Hex())
	}

}

func TestRecoverPubkey(test *testing.T) {
	fmt.Printf("\nbegin TestRecoverPubkey...\n")
	plain_txt := "Sign Recover Pubkey tesing."
	buf := []byte(plain_txt)
	sha3_hash := sha3.Sum256(buf)

	sk, _ := GenerateKey("")
	sign, _ := sk.Sign(sha3_hash[:])

	pk, err := sign.RecoverPubkey(sha3_hash[:])
	if err == nil {
		if !bytes.Equal(pk.Bytes(), sk.GetPubKey().Bytes()) {
			fmt.Printf("original pk = %v\n", sk.GetPubKey().Bytes())
			fmt.Printf("recover pk = %v\n", pk)
		}
	}
	fmt.Printf("end TestRecoverPubkey.\n")
}

func TestHash(test *testing.T) {
	h1 := Hash{1, 2, 3, 4}
	h2 := Hash{1, 2, 3, 4}
	fmt.Printf("%v\n", h1 == h2)
}

func BenchmarkSign(b *testing.B) {
	msg := []byte("This is TASchain achates' testing message")
	sk, _ := GenerateKey("")
	sha3_hash := sha3.Sum256(msg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Sign(sha3_hash[:])
	}
}

func BenchmarkVerify(b *testing.B) {
	msg := []byte("This is TASchain achates' testing message")
	sk, _ := GenerateKey("")
	pk := sk.GetPubKey()
	sha3_hash := sha3.Sum256(msg)
	sign, _ := sk.Sign(sha3_hash[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Verify(sha3_hash[:], &sign)
	}
}

func BenchmarkRecover(b *testing.B) {
	msg := []byte("This is TASchain achates' testing message")
	sk, _ := GenerateKey("")
	sha3_hash := sha3.Sum256(msg)
	sign, _ := sk.Sign(sha3_hash[:])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sign.RecoverPubkey(sha3_hash[:])
	}
}

func TestAccount(test *testing.T) {
	privateKey, _ := GenerateKey("")
	pubkey := privateKey.GetPubKey()
	address := pubkey.GetAddress()
	fmt.Printf("sk:%s\n", privateKey.Hex())
	fmt.Printf("address:%s\n", address.Hex())
}

func TestGenerateKey(t *testing.T) {
	s := "1111345111111111111111111111111111111111"
	sk, _ := GenerateKey(s)
	t.Logf(sk.Hex())

	sk2, _ := GenerateKey(s)
	t.Logf(sk2.Hex())

	sk3, _ := GenerateKey(s)
	t.Logf(sk3.Hex())
}

func TestSignSeckey(t *testing.T) {
	seck := HexToSecKey("0x0477cc7bad86a3c6e4a37ed7dd29820d2ed7cba4b1acef7e00b2b0824eed90590c1a6d5c8d4c09a9b3efcb867a1e9eed3991c95a6b958cbd3a1544d2153cb4a6e40061a70ab47c4bed82877ebd399e696cc079f87943e4b95b78fb8b62bfe74cf6")
	if seck == nil {
		t.Fatal("seck key error")
	}

	sign, _ := seck.Sign(HexToHash("0x123").Bytes())
	t.Logf(sign.Hex())
}

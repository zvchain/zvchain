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

package logical

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"io"
	"testing"
)

func getEncryptKey(sk *groupsig.Seckey, pk *groupsig.Pubkey) ([]byte, error) {
	if !sk.IsValid() || !pk.IsValid() {
		return nil, errors.New("invalid input parameter in getEncryptKey")
	}
	dh := groupsig.DH(sk, pk)
	key := sha256.Sum256(dh.Serialize())
	return key[:], nil
}

func encryptAESCTR(key []byte, iv []byte, plainText []byte) ([]byte, error) {
	if key == nil || iv == nil || plainText == nil {
		return nil, errors.New("invalid input parameter in encryptAESCTR")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != block.BlockSize() {
		return nil, errors.New("cipher.NewCTR: IV length must equal block size")
	}
	ctr := cipher.NewCTR(block, iv)
	cipherText := make([]byte, len(plainText))
	ctr.XORKeyStream(cipherText, plainText)
	return cipherText, nil
}

func batchEncryptPieces(iv []byte, pieces []groupsig.Seckey, selfSK groupsig.Seckey, peerPKs []groupsig.Pubkey) ([]byte, error) {
	n := len(pieces)
	buff := make([]byte, n*32)
	for i := 0; i < len(pieces); i++ {
		key, err := getEncryptKey(&selfSK, &peerPKs[i])
		if err != nil {
			return nil, err
		}

		piece := pieces[i].Serialize() // len(piece) <= 32
		pt := make([]byte, 32)
		copy(pt[32-len(piece):32], piece) //make sure 32-byte-alignment
		ct, err := encryptAESCTR(key, iv, pt)
		if err != nil {
			return nil, err
		}
		copy(buff[i*32:], ct)
	}
	return buff, nil
}

func encryptSharePieces(pieces []groupsig.Seckey, selfSK groupsig.Seckey, peerPKs []groupsig.Pubkey) ([]byte, error) {
	if !selfSK.IsValid() || pieces == nil || peerPKs == nil {
		return nil, errors.New("invalid input parameter in encryptSharePieces")
	}
	if len(pieces) != len(peerPKs) {
		return nil, errors.New("pieces and peerPks are not same length ")
	}
	n := len(pieces)
	buff := make([]byte, aes.BlockSize+n*32+128)

	iv := buff[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	cps, err := batchEncryptPieces(iv, pieces, selfSK, peerPKs)
	if err != nil {
		return nil, err
	}
	copy(buff[aes.BlockSize:], cps)

	selfPk := groupsig.NewPubkeyFromSeckey(selfSK)
	copy(buff[aes.BlockSize+n*32:], selfPk.Serialize())
	return buff, nil
}

func decryptSharePieces(bs [][]byte, selfSK groupsig.Seckey, selfIndex int) ([]groupsig.Seckey, error) {
	if bs == nil || !selfSK.IsValid() {
		return nil, errors.New("invalid input parameters in decryptSharePieces")
	}
	m := len(bs)
	n := (len(bs[0]) - aes.BlockSize - 128) / 32

	if selfIndex >= n {
		return nil, errors.New("invalid input parameter selfIndex in decryptSharePieces")
	}

	pieces := make([]groupsig.Seckey, m)
	for j := 0; j < m; j++ {
		nj := (len(bs[j]) - aes.BlockSize - 128) / 32
		if nj != n {
			return nil, errors.New("encrypted piece buffers are not same size")
		}
		iv := bs[j][:aes.BlockSize]
		pk := groupsig.DeserializePubkeyBytes(bs[j][aes.BlockSize+n*32:])

		key, err := getEncryptKey(&selfSK, &pk)
		if err != nil {
			return nil, err
		}

		ct := bs[j][aes.BlockSize+selfIndex*32 : aes.BlockSize+(selfIndex+1)*32]
		pt, err := encryptAESCTR(key, iv, ct) // encrypt and decrypt are same in AES CTR method
		_ = pieces[j].Deserialize(pt)
	}

	return pieces, nil
}

// checkEvil returns the check result: true if the data is fake. otherwise return false.
func checkEvil(encryptedPieces []byte, ids []groupsig.ID, originPieces []groupsig.Seckey, encSk groupsig.Seckey, peerPKs []groupsig.Pubkey) (bool, error) {
	if !encSk.IsValid() || encryptedPieces == nil || originPieces == nil {
		return false, errors.New("invalid input parameters in checkEvil")
	}
	n := len(originPieces)
	if len(encryptedPieces) != (aes.BlockSize + n*32 + 128) {
		return true, nil
	}
	pk := groupsig.NewPubkeyFromSeckey(encSk)
	bytePk := pk.Serialize()
	if !bytes.Equal(bytePk, encryptedPieces[aes.BlockSize+n*32:]) {
		return true, nil
	}
	iv := encryptedPieces[:aes.BlockSize]

	cps, err := batchEncryptPieces(iv, originPieces, encSk, peerPKs)
	if err != nil {
		return false, err
	}

	if !bytes.Equal(encryptedPieces[aes.BlockSize:aes.BlockSize+n*32], cps) {
		return true, nil
	}

	return false, nil
}

func TestSharePiecesCryptogram(t *testing.T) {
	fmt.Printf("TestSharePiecesCryptogram begin \n")

	n := 9
	k := 5

	sks := make([]groupsig.Seckey, n)
	pks := make([]groupsig.Pubkey, n)
	ids := make([]groupsig.ID, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		sks[i] = *groupsig.NewSeckeyFromRand(r.Deri(i))
		pks[i] = *groupsig.NewPubkeyFromSeckey(sks[i])
		err := ids[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			t.Error(err)
		}
	}

	shares := make([][]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		shares[j] = make([]groupsig.Seckey, n)
		msec := sks[j].GetMasterSecretKey(k)
		for i := 0; i < n; i++ {
			err := shares[j][i].Set(msec, &ids[i])
			if err != nil {
				t.Error(err)
			}
		}
	}

	cs := make([][]byte, n)
	for j := 0; j < n; j++ {
		ct, err := encryptSharePieces(shares[j][:], sks[j], pks[:])
		if err != nil {
			fmt.Printf("fail to encryptSharePieces \n")
			return
		}
		cs[j] = make([]byte, len(ct))
		copy(cs[j], ct)
	}

	ps := make([][]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		ps[j] = make([]groupsig.Seckey, n)
	}

	for i := 0; i < n; i++ {
		pts, err := decryptSharePieces(cs, sks[i], i)
		if err != nil {
			fmt.Printf("fail to decryptSharePieces \n")
			return
		}
		for j := 0; j < n; j++ {
			ps[j][i] = pts[j]
		}
	}

	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			if !ps[j][i].IsEqual(shares[j][i]) {
				fmt.Printf("share piece doesn't match!!!\n")
			}
		}
	}

	for j := 0; j < n; j++ {
		b, err := groupsig.CheckSharePiecesValid(shares[j], ids, k)
		if err != nil {
			t.Error(err)
		}
		if !b {
			fmt.Printf("fail to check share pieces valid. i= %v \n", j)
		}
		b, err = checkEvil(cs[j], ids, shares[j], sks[j], pks)
		if err != nil {
			t.Error(err)
		}
		if b {
			fmt.Printf("i= %v is evil \n", j)
		}
	}

	msk := make([]groupsig.Seckey, n)
	shareVec := make([]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			shareVec[i] = shares[i][j]
		}
		msk[j] = *groupsig.AggregateSeckeys(shareVec)
	}

	msg := []byte("this is test message")
	sigs := make([]groupsig.Signature, n)
	for i := 0; i < n; i++ {
		sigs[i] = groupsig.Sign(msk[i], msg)
	}

	gpk := groupsig.AggregatePubkeys(pks)
	for m := k; m <= n; m++ {
		sigVec := make([]groupsig.Signature, m)
		idVec := make([]groupsig.ID, m)

		for i := 0; i < m; i++ {
			sigVec[i] = sigs[i]
			idVec[i] = ids[i]
		}
		gsig := groupsig.RecoverSignature(sigVec, idVec)

		fmt.Printf("m = %v, sig = %v\n", m, gsig.Serialize())

		if !groupsig.VerifySig(*gpk, msg, *gsig) {
			fmt.Printf("fail to VerifySig when m= %v \n", m)
		}
	}
	fmt.Printf("TestSharePiecesCryptogram end \n")
}

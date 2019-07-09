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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"io"
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

func decryptSharePiecesWithMySK(bs [][]byte, selfSK groupsig.Seckey, index int) ([]groupsig.Seckey, error) {
	if bs == nil || !selfSK.IsValid() {
		return nil, errors.New("invalid parameters in decryptSharePiecesWithMySK")
	}
	m := len(bs)
	n := (len(bs[0]) - aes.BlockSize - 128) / 32

	if index >= n || index < 0 {
		return nil, errors.New("invalid index in decryptSharePiecesWithMySK")
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

		ct := bs[j][aes.BlockSize+index*32 : aes.BlockSize+(index+1)*32]
		pt, err := encryptAESCTR(key, iv, ct) // encrypt and decrypt are same in AES CTR method
		_ = pieces[j].Deserialize(pt)
	}
	return pieces, nil
}

func decryptSharePiecesWithMyPK(bs [][]byte, encSks []groupsig.Seckey, selfPK groupsig.Pubkey, index int) ([]groupsig.Seckey, error) {
	if bs == nil || encSks == nil || !selfPK.IsValid() {
		return nil, errors.New("invalid parameters in decryptSharePiecesWithMyPK")
	}
	if len(bs) != len(encSks) {
		return nil, errors.New("bs and encSks are not same size")
	}
	m := len(bs)
	n := (len(bs[0]) - aes.BlockSize - 128) / 32

	if index >= n || index < 0 {
		return nil, errors.New("invalid index in decryptSharePiecesWithMyPK")
	}

	pieces := make([]groupsig.Seckey, m)
	for j := 0; j < m; j++ {
		nj := (len(bs[j]) - aes.BlockSize - 128) / 32
		if nj != n {
			return nil, errors.New("encrypted piece buffers are not same size")
		}
		iv := bs[j][:aes.BlockSize]

		key, err := getEncryptKey(&encSks[j], &selfPK)
		if err != nil {
			return nil, err
		}

		ct := bs[j][aes.BlockSize+index*32 : aes.BlockSize+(index+1)*32]
		pt, err := encryptAESCTR(key, iv, ct) // encrypt and decrypt are same in AES CTR method
		_ = pieces[j].Deserialize(pt)
	}
	return pieces, nil
}

// checkEvil returns true if the cipher data is fake. otherwise return false.
func checkEvil(encryptedPieces []byte, originPieces []groupsig.Seckey, encSk groupsig.Seckey, peerPKs []groupsig.Pubkey) (bool, error) {
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

// generateSharePiecePacket takes the input and generates share piece
func generateSharePiecePacket(miner *model.SelfMinerDO, encSeckey groupsig.Seckey, seed common.Hash, cands candidates) *sharePiecePacket {
	rand := miner.GenSecretForGroup(seed)

	secs := make([]groupsig.Seckey, cands.threshold())
	for i := 0; i < len(secs); i++ {
		secs[i] = *groupsig.NewSeckeyFromRand(rand.Deri(i))
	}

	pieces := make([]groupsig.Seckey, 0)
	for _, mem := range cands {
		pieces = append(pieces, *groupsig.ShareSeckey(secs, mem.ID))
	}
	return &sharePiecePacket{
		seed:      seed,
		sender:    miner.ID,
		encSeckey: encSeckey,
		pieces:    pieces,
	}
}

// generateEncryptedSharePiecePacket takes the input and generates encrypted share piece packet handled by core
func generateEncryptedSharePiecePacket(miner *model.SelfMinerDO, encSeckey groupsig.Seckey, seed common.Hash, cands candidates) types.EncryptedSharePiecePacket {
	rand := miner.GenSecretForGroup(seed)
	sec0 := *groupsig.NewSeckeyFromRand(rand.Deri(0))
	pk := *groupsig.NewPubkeyFromSeckey(sec0)

	oriPieces := generateSharePiecePacket(miner, encSeckey, seed, cands)

	packet := &encryptedSharePiecePacket{
		pubkey0:          pk,
		memberPubkeys:    cands.pubkeys(),
		sharePiecePacket: oriPieces,
	}

	return packet

}

func deserializeSharePieces(pieceData []byte) []groupsig.Seckey {
	secks := make([]groupsig.Seckey, 0)
	reader := bytes.NewReader(pieceData)

	bs := make([]byte, groupsig.SkLength)

	for n, _ := reader.Read(bs); n == groupsig.SkLength; n, _ = reader.Read(bs) {
		secks = append(secks, *groupsig.DeserializeSeckey(bs))
	}
	return secks
}

func generateEncryptedSeckey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(base.NewRand())
}

// aggrSignSecKeyWithMySK generate miner signature private key with my sk and encrypted pk
func aggrSignSecKeyWithMySK(packets []types.EncryptedSharePiecePacket, idx int, mySK groupsig.Seckey) (*groupsig.Seckey, error) {
	bs := make([][]byte, 0)
	for _, packet := range packets {
		bs = append(bs, packet.Pieces())
	}
	shares, err := decryptSharePiecesWithMySK(bs, mySK, idx)
	if err != nil {
		return nil, err
	}
	sk := groupsig.AggregateSeckeys(shares)
	return sk, nil
}

// aggrSignSecKeyWithMyPK generate miner signature private key with encrypted sk and my pk
func aggrSignSecKeyWithMyPK(packets []types.EncryptedSharePiecePacket, idx int, encSKs []groupsig.Seckey, myPK groupsig.Pubkey) (*groupsig.Seckey, error) {
	bs := make([][]byte, 0)
	for _, packet := range packets {
		bs = append(bs, packet.Pieces())
	}
	shares, err := decryptSharePiecesWithMyPK(bs, encSKs, myPK, idx)
	if err != nil {
		return nil, err
	}
	sk := groupsig.AggregateSeckeys(shares)
	return sk, nil
}

// aggrGroupPubKey generate group public key
func aggrGroupPubKey(packets []types.EncryptedSharePiecePacket) *groupsig.Pubkey {
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range packets {
		pk := groupsig.DeserializePubkeyBytes(v.Pubkey0())
		pubs = append(pubs, pk)
	}
	gpk := groupsig.AggregatePubkeys(pubs)
	return gpk
}

func aggrGroupSign(packets []types.MpkPacket) *groupsig.Signature {
	sigs := make([]groupsig.Signature, 0)
	ids := make([]groupsig.ID, 0)
	for _, pkt := range packets {
		sigs = append(sigs, *groupsig.DeserializeSign(pkt.Sign()))
		ids = append(ids, groupsig.DeserializeID(pkt.Sender()))
	}
	return groupsig.RecoverSignature(sigs, ids)
}

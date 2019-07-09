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
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"testing"
)

func TestSharePiecesCryptogram(t *testing.T) {
	t.Log("TestSharePiecesCryptogram begin \n")

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
			t.Errorf("fail to encryptSharePieces \n")
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
		pts, err := decryptSharePiecesWithMySK(cs, sks[i], i)
		if err != nil {
			t.Errorf("fail to decryptSharePieces \n")
			return
		}
		for j := 0; j < n; j++ {
			ps[j][i] = pts[j]
		}
	}

	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			if !ps[j][i].IsEqual(shares[j][i]) {
				t.Errorf("share piece doesn't match!!!\n")
			}
		}
	}

	for j := 0; j < n; j++ {
		b, err := groupsig.CheckSharePiecesValid(shares[j], ids, k)
		if err != nil {
			t.Error(err)
		}
		if !b {
			t.Errorf("fail to check share pieces valid. i= %v \n", j)
		}
		b, err = checkEvil(cs[j], shares[j], sks[j], pks)
		if err != nil {
			t.Error(err)
		}
		if b {
			t.Errorf("i= %v is evil \n", j)
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

		t.Logf("m = %v, sig = %v\n", m, gsig.Serialize())

		if !groupsig.VerifySig(*gpk, msg, *gsig) {
			t.Errorf("fail to VerifySig when m= %v \n", m)
		}
	}
	t.Log("TestSharePiecesCryptogram end \n")
}

func TestGenerateSharePiecePacket(t *testing.T) {
	selfs := createMinerDOs("key_file_test")
	cands := newCandidates(selfs)
	seed := common.BigToHash(new(big.Int).SetUint64(1030000))
	encSks := make([]groupsig.Seckey, len(selfs))
	for i := range selfs {
		encSks[i] = generateEncryptedSeckey()
	}

	for a := 0; a < 10; a++ {
		t.Logf("round==================%v================", a)
		for i, self := range selfs {
			sp := generateSharePiecePacket(self, encSks[i], seed, cands)
			t.Logf("share piece generated from:%v", self.ID.GetHexString())
			for j, piece := range sp.pieces {
				t.Logf("\t for %v %v", j, piece.GetHexString())
			}
		}
	}
}

func TestGenerateSharePieceAndDeserialize(t *testing.T) {
	selfs := createMinerDOs("key_file_test")
	cands := newCandidates(selfs)
	seed := common.BigToHash(new(big.Int).SetUint64(13123))
	encSks := make([]groupsig.Seckey, len(selfs))
	for i := range selfs {
		encSks[i] = generateEncryptedSeckey()
	}
	sp := generateSharePiecePacket(selfs[0], encSks[0], seed, cands)
	t.Logf("share piece generated from:%v", selfs[0].ID.GetHexString())
	for j, piece := range sp.pieces {
		t.Logf("\t for %v %v", j, piece.GetHexString())
	}

	oriPacket := &originSharePiecePacket{sharePiecePacket: sp}
	deserializePieces := deserializeSharePieces(oriPacket.Pieces())

	for i, p1 := range sp.pieces {
		if !p1.IsEqual(deserializePieces[i]) {
			t.Errorf("deserialize share piece error:%v %v", p1.GetHexString(), deserializePieces[i].GetHexString())
		}
	}

}

func TestEncryptAndDecryptSharePiece(t *testing.T) {
	selfs := createMinerDOs("key_file_test")
	cands := newCandidates(selfs)
	seed := common.BigToHash(new(big.Int).SetUint64(13123))
	encSks := make([]groupsig.Seckey, len(selfs))
	for i := range selfs {
		encSks[i] = generateEncryptedSeckey()
	}

	encPieces := make([]types.EncryptedSharePiecePacket, len(cands))
	for i, self := range selfs {
		encPieces[i] = generateEncryptedSharePiecePacket(self, encSks[i], seed, cands)
		t.Logf("encrypted pieces data from %v :%v", self.ID.GetHexString(), encPieces[i].Pieces())
		oriPs := &originSharePiecePacket{sharePiecePacket: encPieces[i].(*encryptedSharePiecePacket).sharePiecePacket}
		t.Logf("origin pieces data from %v: %v", self.ID.GetHexString(), oriPs.Pieces())
	}

	psBytes := make([][]byte, 0)
	for _, p := range encPieces {
		psBytes = append(psBytes, p.Pieces())
	}

	for i, self := range selfs {
		sps, err := decryptSharePiecesWithMySK(psBytes, self.SK, i)
		if err != nil {
			t.Error(err)
		}
		for j, sp := range sps {
			oriSP := encPieces[j].(*encryptedSharePiecePacket).sharePiecePacket.pieces[i]
			if !sp.IsEqual(oriSP) {
				t.Errorf("decrypt by sk result in piece diff %v %v", sp.GetHexString(), oriSP.GetHexString())
			}
		}
	}
	for i, self := range selfs {
		sps, err := decryptSharePiecesWithMyPK(psBytes, encSks, self.PK, i)
		if err != nil {
			t.Error(err)
		}
		for j, sp := range sps {
			oriSP := encPieces[j].(*encryptedSharePiecePacket).sharePiecePacket.pieces[i]
			if !sp.IsEqual(oriSP) {
				t.Errorf("decrypt by pk result in piece diff %v %v", sp.GetHexString(), oriSP.GetHexString())
			}
		}
	}
}

func TestAggregateGroupAndVerify(t *testing.T) {
	selfs := createMinerDOs("key_file_test")
	cands := newCandidates(selfs)
	seed := common.BigToHash(new(big.Int).SetUint64(13123))
	encSks := make([]groupsig.Seckey, len(selfs))
	for i := range selfs {
		encSks[i] = generateEncryptedSeckey()
	}

	// Generate encrypted share piece
	encPieces := make([]types.EncryptedSharePiecePacket, len(cands))
	for i, self := range selfs {
		encPieces[i] = generateEncryptedSharePiecePacket(self, encSks[i], seed, cands)
	}

	mpks := make([]types.MpkPacket, len(selfs))

	// Generate mpks
	for i, self := range selfs {
		msk, err := aggrSignSecKeyWithMySK(encPieces, i, self.SK)
		if err != nil {
			t.Error(err)
		}
		mpk := *groupsig.NewPubkeyFromSeckey(*msk)
		mSign := groupsig.Sign(*msk, seed.Bytes())
		if !groupsig.VerifySig(mpk, seed.Bytes(), mSign) {
			t.Errorf("verify member sign fail")
		}
		mpks[i] = &mpkPacket{
			sender: self.ID,
			seed:   seed,
			mPk:    mpk,
			sign:   mSign,
		}
	}
	// Aggregate group signature
	gSign := aggrGroupSign(mpks)

	// Aggregate group pubkey
	gPk := aggrGroupPubKey(encPieces)
	if !groupsig.VerifySig(*gPk, seed.Bytes(), *gSign) {
		t.Fatal("verify group sig fail")
	}
}

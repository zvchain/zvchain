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

func TestAggrGroupPubKey(t *testing.T) {
	var pks [5]groupsig.Pubkey
	pks[0].Deserialize(common.FromHex("0x0050d203007d74edd73a9de19d138d636c9cc4b0808176319c57ecbcfedbd16ff23cfb857e866dd06eec6f335a5b6222efc7af8b40fd7f08ad1a6443a324289e3cda65a762ff1ce590ac7e9e27f9175ddec9dad90d4c6b21dea7f9926eb1dcdcd3392f93e3d7752528f6c7203a12cabc63863b8a7d00e2ea691fa6fc01cbf6263bb1d97b7a526fdd5b85010dd9ab12280d25942f8181bf2014c2f337f6f85173f665856ba3f5262daff1c6fcd544973f96e2dc54bac642ae2fd4fef040a5dad4d9aeeade5faf011cf1769f6e36c895a28f783957db5ea8b44d64044a0ac3ae0661a5307a79f61fb17622e6abcb1e6b742392f9d93ebd2375e9da37e34293dfa4efe7892eb2fc88b192f21aa2211a7aa513f0383f9da3d820833eb77236b48ffb42dfdbe8875c4a913434f9a914df4c502a8fb343976fde1961731855f2b193a04696f408a2c95522fa561bdc557d01ab26c33ec8ba72bebd1b05218aff42557370b796122656cb80b43d65ed7fcbe610"))
	pks[1].Deserialize(common.FromHex("0x975c3169ad5d946ec5d04230fd774799e61fd61b0d057355e623fb374cf30e4751197d91593d53e7fdaf12692d0ad6a9b37297422ec2558dfae3cfb10f53e646abbe81f195183beab80b0324c4969afdb0b399b63e1de73ffdc46e8ae5ac40994c155be67e223d6ecd9c08005adb64b8479fcf6bb31310477af0ad9a90d958ed91e1ee0c3d75af0d73c14bfd84b6c5afcc6539c376646610d353566eb2daf3409a75d2df60a3fcb6f1bb4d284a8f1a49df92c8de0695b15fbe27e2dbd36eb3c2f95d2d8020b16cf32d8c5be9b6a3ca848c1b7af31e51bb41eb4a527ebc3f53ededec0b1933119f8fda071445d3e4d7c82b1ddbead0125c3e140a3e0c044836b1a02c9565104b3d46f1389d657ade3203159f8163b7227dbfbd51df6709b8fb5c400a642aa38bec493fde5e60dcb864d1101a8f44cc9479ed3b28eb0e5c7d35aff9c2843b1ab420afcced9b87b869f3c82c9bacb4069f6371cf215e0d649dd3b6c68067811eba111c9c3ffdfe5ddc426c"))
	pks[2].Deserialize(common.FromHex("0x05d082448d523d78764e82cebfb7f9b9b3c45b71f615e5342ae79c56b1b16e2ca6115e288a89796c018afd5e97504c97e98bc160e1c443918d720248cb19bb6e77ac7e31825c3c09cba99432f2d197e6f22608f6729b15083bd9783d79cccfcea7f90402bd19abc477ca7bdc4b71976d62c77f047cedffd0db166aaa4bf6fc4371174df66b59e3f614e14ca928e4801a7667cbaf0cb6dd3bcd3366fbcfb3235c87b2e0cda93f1cd61e28ab95f155a5ecdef2f291a239a1c5f6f6ac318ed7bb6de416002697dd23ff16e472e51ff0f9444efcc0b268608b6a2c1737513405740f7bcd04325d0ca30e77a7d4e45db0485304af855296109a15f38a88289314c439d63631ff20af5c457065db2516a9b4072b7dc917a16a552c7cd462e46b6575cebe9761c688a3941cb4984d11381fb9fe16cd04684bc8dcb37064849d72f130932cbc2912cdb829b2d409800262c9a6ad0f79a680639616b1c75a1a9d943458ce7dcbcb8a5eef75041360ed180ae118e1"))
	pks[3].Deserialize(common.FromHex("0xaf661722eae72ce5535604c8fb3a9e21fbf0328d32ee1f0f81d8986fddf13dac56bd12fac37dba985c7f10a105ad0d1f083832fa72ab90f3e37c08d626d570f2a12a0aed1aa1c7deaf8bed8fa593d7a07a75df2940bc43c2d588f95fd90d7a9b45d0287428170e90db2422e0a00c06cc3701a859e77f54ae2ba9cd5d2f5e4e2bdf3bf7c841dfdf4079c98d4b720f783dd694fb8c62ad15891d82b44d15b78d31eccacd89eddcfe1915ef2e89ecc531d0b16be69725daeae87b10133709c7716f45ec030ba49f92c91428ad60b947a73eaf5fac31310a0e0a4b59c68f69a5abe3bef2154c2e114a98060bda838df0ebf920e1dd7ccf726d67a8df5a6c82632a2247b29efc772206b2abc6a3085781bee02ed93eb674d8bf711a4c25cde086183b3fddc735b79c9c93fa5dd4c4bd0f1e7d1666967751b0a18813f639bf5db1605fa1f83bcc74387ded063c9ec3f7936bc20771e4645b43036d9da86525063725ca9d86760594b37bc9ad0d9277b9ed61c5"))
	pks[4].Deserialize(common.FromHex("0x5a5b54d690d62eba85974845fbdec22b5aeaf01cdd18bb6805ecc02a13c3c10783634514d581ad8eb86f231ccd9ebd6b80aaa3c1fda87903edb118f9f8617a5427ff2775c871883d1597762dbc44254a2f8752a8eba59c47d029ae1f0305484f0751637cb002a22edcc534631267614018984dfddcc9ec35f42327b190db240b40681952df0f1eb6f1651edcff2e9640e6fbbdaae41888c42015682e997f6380f4d695e5936beeaf0bd8d3c5d8e40cd39e5c52390ff16d6ab829fd1dfb59779917d8aca56d08359a3992391f1aa5a13c4718e90af349cf49341a574cc5bbe2dca846bb1b3e6bcb9fcb57fed1f2eb4be424f673fd4a01f9077ec2e39394577ace1f43f3ce90c1a51b494050a3b37b3013145e35165314625fa9c175fceb5295baad30e5f041dd7ab413f41e58ee62213406d14b6c28831e044b684869b496783144bcc642cb7c070aa2a26e45bbc4592c27fe592aa24d29d441adbd5b29f4066cdfca45a05ebb8dc388f4563b3974f9ad"))
	//pks[5].SetHexString("0x793ea6a4d47bbcdb0e86381480e998faa33beed56da5b72525a696711499197fadb556dfc6fc0465c8917d7cb6d4fd44e692775f4ae5ba48199ed52e8623886841cc452133b31fbeec5c515c52fe636121071e9a4a8f36b02eccb9da756a2f0d7070c1a38311f30f9e6fc06ae7172d4971aee58d986b086befe135ec021c004b0296cde897ee1c1baa39e6d08c6a3d7f6c496a1f9fb2a6666cb7243e3daa39f2b4f81f879663e4fb0ec88b878fd505d87c331200f2f33acdd4870d7ff72554a64b5a9fec20c6ef223d17f4f32593bfb3d35c276a529386618475fbc32bc85ff91e4b019e5fc8f52a0be828996b2d137b16a65066b5636f4ba439fca129589d4ef15d2f294a679d4a64515e9b19927ca11ecf6e79933155749ecd633446676b6386d6654a9e14605c88f51b0c6a08cdbd1fc191d1fe1c24aacd830b0ebb795575da6954cc0b3a0af9c41e3a8e4ac5a45209b769a8dffd5e5502b8bd83e5491503697044502089900ccd4523b196e6e5f6"))

	gpk := groupsig.AggregatePubkeys(pks[:])
	t.Log(gpk)
}

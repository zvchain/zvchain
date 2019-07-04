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
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

func encryptSharePieces(pieces []groupsig.Seckey, selfSK groupsig.Seckey, peerPKs []groupsig.Pubkey) ([]byte, error) {
	return nil, nil
}

func decryptSharePieces(bs [][]byte, selfSK groupsig.Seckey, selfIndex int) ([]groupsig.Seckey, error) {
	return []groupsig.Seckey{}, nil
}

func checkEvil(originPieces []groupsig.Seckey, ids []groupsig.ID) (bool, error) {
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

	memPks := make([]groupsig.Pubkey, 0)
	for _, mem := range cands {
		memPks = append(memPks, mem.PK)
	}

	packet := &encryptedSharePiecePacket{
		pubkey0:          pk,
		memberPubkeys:    memPks,
		sharePiecePacket: oriPieces,
	}

	return packet

}

func generateEncryptedSeckey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(base.NewRand())
}

func decryptPackets(packets []types.EncryptedSharePiecePacket, sk groupsig.Seckey, idx int) ([]groupsig.Seckey, error) {
	bs := make([][]byte, 0)
	for _, packet := range packets {
		bs = append(bs, packet.Pieces())
	}
	return decryptSharePieces(bs, sk, idx)
}

// aggrSignSecKey generate miner signature private key
func aggrSignSecKey(packets []types.EncryptedSharePiecePacket, cand candidates, mInfo *model.SelfMinerDO) (*groupsig.Seckey, error) {
	shares, err := decryptPackets(packets, mInfo.SK, cand.find(mInfo.ID))
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

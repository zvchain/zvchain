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

package model

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
)

// MinerDO defines the important infos for one miner
type MinerDO struct {
	PK          groupsig.Pubkey
	VrfPK       base.VRFPublicKey
	ID          groupsig.ID
	Stake       uint64
	NType       byte
	ApplyHeight uint64
	AbortHeight uint64
}

func (md *MinerDO) IsAbort(h uint64) bool {
	return md.AbortHeight > 0 && h >= md.AbortHeight
}

func (md *MinerDO) EffectAt(h uint64) bool {
	return h >= md.ApplyHeight
}

// CanCastAt means whether it can be cast block at this height
func (md *MinerDO) CanCastAt(h uint64) bool {
	return md.IsWeight() && !md.IsAbort(h) && md.EffectAt(h)
}

// CanJoinGroupAt means whether it can join the group at this height
func (md *MinerDO) CanJoinGroupAt(h uint64) bool {
	return md.IsLight() && !md.IsAbort(h) && md.EffectAt(h)
}

func (md *MinerDO) IsLight() bool {
	return md.NType == types.MinerTypeLight
}

func (md *MinerDO) IsWeight() bool {
	return md.NType == types.MinerTypeHeavy
}

// SelfMinerDO inherited from MinerDO.
// And some private key included
type SelfMinerDO struct {
	MinerDO
	SecretSeed base.Rand // Private random number
	SK         groupsig.Seckey
	VrfSK      base.VRFPrivateKey
}

func (mi *SelfMinerDO) Read(p []byte) (n int, err error) {
	bs := mi.SecretSeed.Bytes()
	if p == nil || len(p) < len(bs) {
		p = make([]byte, len(bs))
	}
	copy(p, bs)
	return len(bs), nil
}

func NewSelfMinerDO(prk *common.PrivateKey) (SelfMinerDO, error) {
	var mi SelfMinerDO

	keyBytes := prk.ExportKey()
	tempBuf := make([]byte, 32)
	if len(keyBytes) < 32 {
		copy(tempBuf[32-len(keyBytes):32], keyBytes[:])
	} else {
		copy(tempBuf[:], keyBytes[len(keyBytes)-32:])
	}
	mi.SecretSeed = base.RandFromBytes(tempBuf[:])
	mi.SK = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
	mi.PK = *groupsig.NewPubkeyFromSeckey(mi.SK)
	mi.ID = groupsig.DeserializeID(prk.GetPubKey().GetAddress().Bytes())

	var err error
	mi.VrfPK, mi.VrfSK, err = base.VRFGenerateKey(&mi)
	return mi, err
}

func (mi SelfMinerDO) GetMinerID() groupsig.ID {
	return mi.ID
}

func (mi SelfMinerDO) GetSecret() base.Rand {
	return mi.SecretSeed
}

func (mi SelfMinerDO) GetDefaultSecKey() groupsig.Seckey {
	return mi.SK
}

func (mi SelfMinerDO) GetDefaultPubKey() groupsig.Pubkey {
	return mi.PK
}

func (mi SelfMinerDO) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}

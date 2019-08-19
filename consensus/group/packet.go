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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
)

type sharePiecePacket struct {
	seed      common.Hash
	sender    groupsig.ID
	pieces    []groupsig.Seckey
	encSeckey groupsig.Seckey
}

func (sp *sharePiecePacket) Seed() common.Hash {
	return sp.seed
}

func (sp *sharePiecePacket) Sender() []byte {
	return sp.sender.Serialize()
}

type encryptedSharePiecePacket struct {
	*sharePiecePacket
	memberPubkeys []groupsig.Pubkey
	pubkey0       groupsig.Pubkey
}

func (sp *encryptedSharePiecePacket) Pieces() []byte {
	bs, err := encryptSharePieces(sp.pieces, sp.encSeckey, sp.memberPubkeys)
	if err != nil {
		return nil
	}
	return bs
}

func (sp *encryptedSharePiecePacket) Pubkey0() []byte {
	return sp.pubkey0.Serialize()
}

type originSharePiecePacket struct {
	*sharePiecePacket
}

func (sp *originSharePiecePacket) Pieces() []byte {
	buf := bytes.Buffer{}
	for _, p := range sp.pieces {
		buf.Write(p.Serialize())
	}
	return buf.Bytes()
}

func (sp *originSharePiecePacket) EncSeckey() []byte {
	return sp.encSeckey.Serialize()
}

type mpkPacket struct {
	sender groupsig.ID
	seed   common.Hash
	mPk    groupsig.Pubkey
	sign   groupsig.Signature
}

func (pkt *mpkPacket) Seed() common.Hash {
	return pkt.seed
}

func (pkt *mpkPacket) Sender() []byte {
	return pkt.sender.Serialize()
}

func (pkt *mpkPacket) Mpk() []byte {
	return pkt.mPk.Serialize()
}

func (pkt *mpkPacket) Sign() []byte {
	return pkt.sign.Serialize()
}

type member struct {
	id []byte
	pk []byte
}

func (m *member) ID() []byte {
	return m.id
}

func (m *member) PK() []byte {
	return m.pk
}

type groupHeader struct {
	seed          common.Hash
	workHeight    uint64
	dismissHeight uint64
	gpk           groupsig.Pubkey
	threshold     uint32
	groupHeight   uint64
}

func (gh *groupHeader) Seed() common.Hash {
	return gh.seed
}

func (gh *groupHeader) WorkHeight() uint64 {
	return gh.workHeight
}

func (gh *groupHeader) DismissHeight() uint64 {
	return gh.dismissHeight
}

func (gh *groupHeader) PublicKey() []byte {
	return gh.gpk.Serialize()
}

func (gh *groupHeader) Threshold() uint32 {
	return gh.threshold
}

func (gh *groupHeader) GroupHeight() uint64 {
	return gh.groupHeight
}

type group struct {
	header  types.GroupHeaderI
	members []types.MemberI
}

func (g *group) Header() types.GroupHeaderI {
	return g.header
}

func (g *group) Members() []types.MemberI {
	return g.members
}

type createResult struct {
	code         types.CreateResultCode
	seed         common.Hash
	gInfo        *group
	frozenMiners []groupsig.ID
	err          error
}

func (cr *createResult) Seed() common.Hash {
	return cr.seed
}

func (cr *createResult) Code() types.CreateResultCode {
	return cr.code
}

func (cr *createResult) GroupInfo() types.GroupI {
	if cr.gInfo == nil {
		return nil
	}
	return cr.gInfo
}

func (cr *createResult) FrozenMiners() [][]byte {
	bs := make([][]byte, 0)
	if len(cr.frozenMiners) > 0 {
		for _, mem := range cr.frozenMiners {
			bs = append(bs, mem.Serialize())
		}
	}
	return bs
}

func (cr *createResult) Err() error {
	return cr.err
}

func errCreateResult(err error) *createResult {
	return &createResult{code: types.CreateResultFail, err: err}
}
func idleCreateResult(err error) *createResult {
	return &createResult{code: types.CreateResultIdle, err: err}
}

func generateGroupInfo(packets []types.MpkPacket, era *era, gpk groupsig.Pubkey, threshold int) *group {
	members := make([]types.MemberI, 0)
	for _, pkt := range packets {
		members = append(members, &member{id: pkt.Sender(), pk: pkt.Mpk()})
	}
	gHeader := &groupHeader{
		seed:          era.Seed(),
		workHeight:    types.ActivateEpochOfGroupsCreatedAt(era.seedHeight).Start(),
		dismissHeight: types.DismissEpochOfGroupsCreatedAt(era.seedHeight).Start(),
		gpk:           gpk,
		threshold:     uint32(threshold),
	}
	return &group{
		header:  gHeader,
		members: members,
	}
}

type punishment struct {
	penaltyTargets [][]byte
	rewardTargets  [][]byte
}

func (pm *punishment) PenaltyTarget() [][]byte {
	return pm.penaltyTargets
}

func (pm *punishment) RewardTarget() [][]byte {
	return pm.rewardTargets
}

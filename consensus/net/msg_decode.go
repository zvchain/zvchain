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

package net

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
)

func baseMessage(sign *tas_middleware_pb.SignData) *model.BaseSignedMessage {
	return &model.BaseSignedMessage{SI: *pbToSignData(sign)}
}

func pbToGroupInfo(gi *tas_middleware_pb.ConsensusGroupInitInfo) *model.ConsensusGroupInitInfo {
	gis := pbToConsensusGroupInitSummary(gi.GI)
	mems := make([]groupsig.ID, len(gi.Mems))
	for idx, mem := range gi.Mems {
		mems[idx] = groupsig.DeserializeID(mem)
	}
	return &model.ConsensusGroupInitInfo{
		GI:   *gis,
		Mems: mems,
	}
}

func unMarshalConsensusGroupRawMessage(b []byte) (*model.ConsensusGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	m := model.ConsensusGroupRawMessage{
		GInfo:             *pbToGroupInfo(message.GInfo),
		BaseSignedMessage: *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalConsensusSharePieceMessage(b []byte) (*model.ConsensusSharePieceMessage, error) {
	m := new(tas_middleware_pb.ConsensusSharePieceMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	gHash := common.BytesToHash(m.GHash)

	dest := groupsig.DeserializeID(m.Dest)

	share := pbToSharePiece(m.SharePiece)
	message := model.ConsensusSharePieceMessage{
		GHash:             gHash,
		Dest:              dest,
		Share:             *share,
		BaseSignedMessage: *baseMessage(m.Sign),
		MemCnt:            *m.MemCnt,
	}
	return &message, nil
}

func unMarshalConsensusSignPubKeyMessage(b []byte) (*model.ConsensusSignPubKeyMessage, error) {
	m := new(tas_middleware_pb.ConsensusSignPubKeyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPubKeyMessage error:%s", e.Error())
		return nil, e
	}
	gisHash := common.BytesToHash(m.GHash)

	pk := groupsig.DeserializePubkeyBytes(m.SignPK)

	base := baseMessage(m.SignData)
	message := model.ConsensusSignPubKeyMessage{
		GHash:             gisHash,
		SignPK:            pk,
		GroupID:           groupsig.DeserializeID(m.GroupID),
		BaseSignedMessage: *base,
		MemCnt:            *m.MemCnt,
	}
	return &message, nil
}

func unMarshalConsensusGroupInitedMessage(b []byte) (*model.ConsensusGroupInitedMessage, error) {
	m := new(tas_middleware_pb.ConsensusGroupInitedMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusGroupInitedMessage error:%s", e.Error())
		return nil, e
	}

	ch := uint64(0)
	if m.CreateHeight != nil {
		ch = *m.CreateHeight
	}
	var sign groupsig.Signature
	if len(m.ParentSign) > 0 {
		sign.Deserialize(m.ParentSign)
	}
	message := model.ConsensusGroupInitedMessage{
		GHash:             common.BytesToHash(m.GHash),
		GroupID:           groupsig.DeserializeID(m.GroupID),
		GroupPK:           groupsig.DeserializePubkeyBytes(m.GroupPK),
		CreateHeight:      ch,
		ParentSign:        sign,
		BaseSignedMessage: *baseMessage(m.Sign),
		MemCnt:            *m.MemCnt,
		MemMask:           m.MemMask,
	}
	return &message, nil
}

func unMarshalConsensusSignPKReqMessage(b []byte) (*model.ConsensusSignPubkeyReqMessage, error) {
	m := new(tas_middleware_pb.ConsensusSignPubkeyReqMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusSignPKReqMessage error: %v", e.Error())
		return nil, e
	}
	message := &model.ConsensusSignPubkeyReqMessage{
		GroupID:           groupsig.DeserializeID(m.GroupID),
		BaseSignedMessage: *baseMessage(m.SignData),
	}
	return message, nil
}

/*
Group coinage
*/

func unMarshalConsensusCurrentMessage(b []byte) (*model.ConsensusCurrentMessage, error) {
	m := new(tas_middleware_pb.ConsensusCurrentMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCurrentMessage error:%s", e.Error())
		return nil, e
	}

	GroupID := m.GroupID
	PreHash := common.BytesToHash(m.PreHash)

	var PreTime time.Time
	PreTime.UnmarshalBinary(m.PreTime)

	BlockHeight := m.BlockHeight
	si := pbToSignData(m.Sign)
	base := model.BaseSignedMessage{SI: *si}
	message := model.ConsensusCurrentMessage{GroupID: GroupID, PreHash: PreHash, PreTime: PreTime, BlockHeight: *BlockHeight, BaseSignedMessage: base}
	return &message, nil
}

func unMarshalConsensusCastMessage(b []byte) (*model.ConsensusCastMessage, error) {
	m := new(tas_middleware_pb.ConsensusCastMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusCastMessage error:%s", e.Error())
		return nil, e
	}

	bh := types.PbToBlockHeader(m.Bh)

	return &model.ConsensusCastMessage{
		BH:                *bh,
		ProveHash:         common.BytesToHash(m.ProveHash),
		BaseSignedMessage: *baseMessage(m.Sign),
	}, nil
}

func unMarshalConsensusVerifyMessage(b []byte) (*model.ConsensusVerifyMessage, error) {
	m := new(tas_middleware_pb.ConsensusVerifyMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("unMarshalConsensusVerifyMessage error:%v", e.Error())
		return nil, e
	}
	return &model.ConsensusVerifyMessage{
		BlockHash:         common.BytesToHash(m.BlockHash),
		RandomSign:        *groupsig.DeserializeSign(m.RandomSign),
		BaseSignedMessage: *baseMessage(m.Sign),
	}, nil
}

func unMarshalConsensusBlockMessage(b []byte) (*model.ConsensusBlockMessage, error) {
	m := new(tas_middleware_pb.ConsensusBlockMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		logger.Errorf("[handler]unMarshalConsensusBlockMessage error:%s", e.Error())
		return nil, e
	}
	block := types.PbToBlock(m.Block)
	message := model.ConsensusBlockMessage{Block: *block}
	return &message, nil
}

func pbToConsensusGroupInitSummary(m *tas_middleware_pb.ConsensusGroupInitSummary) *model.ConsensusGroupInitSummary {
	gh := types.PbToGroupHeader(m.Header)
	return &model.ConsensusGroupInitSummary{
		GHeader:   gh,
		Signature: *groupsig.DeserializeSign(m.Signature),
	}
}

func pbToSignData(s *tas_middleware_pb.SignData) *model.SignData {

	var sig groupsig.Signature
	e := sig.Deserialize(s.DataSign)
	if e != nil {
		logger.Errorf("[handler]groupsig.Signature Deserialize error:%s", e.Error())
		return nil
	}

	id := groupsig.ID{}
	e1 := id.Deserialize(s.SignMember)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.ID Deserialize error:%s", e1.Error())
		return nil
	}

	v := int32(0)
	if s.Version != nil {
		v = *s.Version
	}
	sign := model.SignData{DataHash: common.BytesToHash(s.DataHash), DataSign: sig, SignMember: id, Version: v}
	return &sign
}

func pbToSharePiece(s *tas_middleware_pb.SharePiece) *model.SharePiece {
	var share groupsig.Seckey
	var pub groupsig.Pubkey

	e1 := share.Deserialize(s.Seckey)
	if e1 != nil {
		logger.Errorf("[handler]groupsig.Seckey Deserialize error:%s", e1.Error())
		return nil
	}

	e2 := pub.Deserialize(s.Pubkey)
	if e2 != nil {
		logger.Errorf("[handler]groupsig.Pubkey Deserialize error:%s", e2.Error())
		return nil
	}

	sp := model.SharePiece{Share: share, Pub: pub}
	return &sp
}

func unMarshalConsensusCreateGroupRawMessage(b []byte) (*model.ConsensusCreateGroupRawMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupRawMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupRawMessage error:%s", e.Error())
		return nil, e
	}

	gi := pbToGroupInfo(message.GInfo)

	m := model.ConsensusCreateGroupRawMessage{
		GInfo:             *gi,
		BaseSignedMessage: *baseMessage(message.Sign),
	}
	return &m, nil
}

func unMarshalConsensusCreateGroupSignMessage(b []byte) (*model.ConsensusCreateGroupSignMessage, error) {
	message := new(tas_middleware_pb.ConsensusCreateGroupSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		logger.Errorf("[handler]UnMarshalConsensusCreateGroupSignMessage error:%s", e.Error())
		return nil, e
	}

	m := model.ConsensusCreateGroupSignMessage{
		GHash:             common.BytesToHash(message.GHash),
		BaseSignedMessage: *baseMessage(message.Sign),
	}
	return &m, nil
}

func pbToReward(b *tas_middleware_pb.Reward) *types.Reward {
	return &types.Reward{
		TxHash:     common.BytesToHash(b.TxHash),
		TargetIds:  b.TargetIds,
		BlockHash:  common.BytesToHash(b.BlockHash),
		GroupID:    b.GroupId,
		Sign:       b.Sign,
		TotalValue: *b.TotalValue,
	}
}

func unMarshalCastRewardReqMessage(b []byte) (*model.CastRewardTransSignReqMessage, error) {
	message := new(tas_middleware_pb.CastRewardTransSignReqMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalCastRewardReqMessage error:%s", e.Error())
		return nil, e
	}

	rw := pbToReward(message.Reward)
	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	signPieces := make([]groupsig.Signature, len(message.SignedPieces))
	for idx, sp := range message.SignedPieces {
		signPieces[idx] = *groupsig.DeserializeSign(sp)
	}

	m := &model.CastRewardTransSignReqMessage{
		BaseSignedMessage: base,
		Reward:            *rw,
		SignedPieces:      signPieces,
	}
	return m, nil
}

func unMarshalCastRewardSignMessage(b []byte) (*model.CastRewardTransSignMessage, error) {
	message := new(tas_middleware_pb.CastRewardTransSignMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalCastRewardSignMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	m := &model.CastRewardTransSignMessage{
		BaseSignedMessage: base,
		ReqHash:           common.BytesToHash(message.ReqHash),
		BlockHash:         common.BytesToHash(message.BlockHash),
	}
	return m, nil
}

func unMarshalCreateGroupPingMessage(b []byte) (*model.CreateGroupPingMessage, error) {
	message := new(tas_middleware_pb.CreateGroupPingMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalCreateGroupPingMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	m := &model.CreateGroupPingMessage{
		BaseSignedMessage: base,
		FromGroupID:       groupsig.DeserializeID(message.FromGroupID),
		PingID:            *message.PingID,
		BaseHeight:        *message.BaseHeight,
	}
	return m, nil
}

func unMarshalCreateGroupPongMessage(b []byte) (*model.CreateGroupPongMessage, error) {
	message := new(tas_middleware_pb.CreateGroupPongMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalCreateGroupPongMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	var ts time.Time
	ts.UnmarshalBinary(message.Ts)

	m := &model.CreateGroupPongMessage{
		BaseSignedMessage: base,
		PingID:            *message.PingID,
		Ts:                ts,
	}
	return m, nil
}

func unMarshalSharePieceReqMessage(b []byte) (*model.ReqSharePieceMessage, error) {
	message := new(tas_middleware_pb.ReqSharePieceMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalSharePieceReqMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	m := &model.ReqSharePieceMessage{
		BaseSignedMessage: base,
		GHash:             common.BytesToHash(message.GHash),
	}
	return m, nil
}

func unMarshalSharePieceResponseMessage(b []byte) (*model.ResponseSharePieceMessage, error) {
	message := new(tas_middleware_pb.ResponseSharePieceMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]unMarshalResponseSharePieceMessage error:%s", e.Error())
		return nil, e
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	m := &model.ResponseSharePieceMessage{
		BaseSignedMessage: base,
		GHash:             common.BytesToHash(message.GHash),
		Share:             *pbToSharePiece(message.SharePiece),
	}
	return m, nil
}

func unmarshalReqProposalBlockMessage(b []byte) (*model.ReqProposalBlock, error) {
	message := &tas_middleware_pb.ReqProposalBlockMessage{}
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	m := &model.ReqProposalBlock{
		Hash: common.BytesToHash(message.Hash),
	}
	return m, nil
}

func unmarshalResponseProposalBlockMessage(b []byte) (*model.ResponseProposalBlock, error) {
	message := &tas_middleware_pb.ResponseProposalBlockMessage{}
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	transactions := types.PbToTransactions(message.Transactions)

	m := &model.ResponseProposalBlock{
		Hash:         common.BytesToHash(message.Hash),
		Transactions: transactions,
	}
	return m, nil
}

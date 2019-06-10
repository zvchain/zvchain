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
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/consensus/model"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
)

func marshalGroupInfo(gInfo *model.ConsensusGroupInitInfo) *tas_middleware_pb.ConsensusGroupInitInfo {
	mems := make([][]byte, gInfo.MemberSize())
	for i, mem := range gInfo.Mems {
		mems[i] = mem.Serialize()
	}

	return &tas_middleware_pb.ConsensusGroupInitInfo{
		GI:   consensusGroupInitSummaryToPb(&gInfo.GI),
		Mems: mems,
	}
}

func marshalConsensusGroupRawMessage(m *model.ConsensusGroupRawMessage) ([]byte, error) {
	gi := marshalGroupInfo(&m.GInfo)

	sign := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusGroupRawMessage{
		GInfo: gi,
		Sign:  sign,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSharePieceMessage(m *model.ConsensusSharePieceMessage) ([]byte, error) {
	share := sharePieceToPb(&m.Share)
	sign := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSharePieceMessage{
		GHash:      m.GHash.Bytes(),
		Dest:       m.Dest.Serialize(),
		SharePiece: share,
		Sign:       sign,
		MemCnt:     &m.MemCnt,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyMessage(m *model.ConsensusSignPubKeyMessage) ([]byte, error) {
	signData := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSignPubKeyMessage{
		GHash:    m.GHash.Bytes(),
		SignPK:   m.SignPK.Serialize(),
		SignData: signData,
		GroupID:  m.GroupID.Serialize(),
		MemCnt:   &m.MemCnt,
	}
	return proto.Marshal(&message)
}
func marshalConsensusGroupInitedMessage(m *model.ConsensusGroupInitedMessage) ([]byte, error) {
	si := signDataToPb(&m.SI)
	message := tas_middleware_pb.ConsensusGroupInitedMessage{
		GHash:        m.GHash.Bytes(),
		GroupID:      m.GroupID.Serialize(),
		GroupPK:      m.GroupPK.Serialize(),
		CreateHeight: &m.CreateHeight,
		ParentSign:   m.ParentSign.Serialize(),
		Sign:         si,
		MemCnt:       &m.MemCnt,
		MemMask:      m.MemMask,
	}
	return proto.Marshal(&message)
}

func marshalConsensusSignPubKeyReqMessage(m *model.ConsensusSignPubkeyReqMessage) ([]byte, error) {
	signData := signDataToPb(&m.SI)

	message := tas_middleware_pb.ConsensusSignPubkeyReqMessage{
		GroupID:  m.GroupID.Serialize(),
		SignData: signData,
	}
	return proto.Marshal(&message)
}

/*
Group coinage
*/

func marshalConsensusVerifyMessage(m *model.ConsensusVerifyMessage) ([]byte, error) {
	message := &tas_middleware_pb.ConsensusVerifyMessage{
		BlockHash:  m.BlockHash.Bytes(),
		RandomSign: m.RandomSign.Serialize(),
		Sign:       signDataToPb(&m.SI),
	}
	return proto.Marshal(message)
}

func marshalConsensusBlockMessage(m *model.ConsensusBlockMessage) ([]byte, error) {
	block := types.BlockToPb(&m.Block)
	if block == nil {
		logger.Errorf("[peer]Block is nil while marshalConsensusBlockMessage")
	}
	message := tas_middleware_pb.ConsensusBlockMessage{Block: block}
	return proto.Marshal(&message)
}

func consensusGroupInitSummaryToPb(m *model.ConsensusGroupInitSummary) *tas_middleware_pb.ConsensusGroupInitSummary {
	message := tas_middleware_pb.ConsensusGroupInitSummary{
		Header:    types.GroupToPbHeader(m.GHeader),
		Signature: m.Signature.Serialize(),
	}
	return &message
}

func signDataToPb(s *model.SignData) *tas_middleware_pb.SignData {
	sign := tas_middleware_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Serialize(), SignMember: s.SignMember.Serialize(), Version: &s.Version}
	return &sign
}

func sharePieceToPb(s *model.SharePiece) *tas_middleware_pb.SharePiece {
	share := tas_middleware_pb.SharePiece{Seckey: s.Share.Serialize(), Pubkey: s.Pub.Serialize()}
	return &share
}

func marshalConsensusCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) ([]byte, error) {
	gi := marshalGroupInfo(&msg.GInfo)

	sign := signDataToPb(&msg.SI)

	message := tas_middleware_pb.ConsensusCreateGroupRawMessage{GInfo: gi, Sign: sign}
	return proto.Marshal(&message)
}

func marshalConsensusCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage) ([]byte, error) {
	sign := signDataToPb(&msg.SI)

	message := tas_middleware_pb.ConsensusCreateGroupSignMessage{GHash: msg.GHash.Bytes(), Sign: sign}
	return proto.Marshal(&message)
}

func bonusToPB(bonus *types.Bonus) *tas_middleware_pb.Bonus {
	return &tas_middleware_pb.Bonus{
		TxHash:     bonus.TxHash.Bytes(),
		TargetIds:  bonus.TargetIds,
		BlockHash:  bonus.BlockHash.Bytes(),
		GroupId:    bonus.GroupID,
		Sign:       bonus.Sign,
		TotalValue: &bonus.TotalValue,
	}
}

func marshalCastRewardTransSignReqMessage(msg *model.CastRewardTransSignReqMessage) ([]byte, error) {
	b := bonusToPB(&msg.Reward)
	si := signDataToPb(&msg.SI)
	pieces := make([][]byte, 0)
	for _, sp := range msg.SignedPieces {
		pieces = append(pieces, sp.Serialize())
	}
	message := &tas_middleware_pb.CastRewardTransSignReqMessage{
		Sign:         si,
		Reward:       b,
		SignedPieces: pieces,
	}
	return proto.Marshal(message)
}

func marshalCastRewardTransSignMessage(msg *model.CastRewardTransSignMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	message := &tas_middleware_pb.CastRewardTransSignMessage{
		Sign:      si,
		ReqHash:   msg.ReqHash.Bytes(),
		BlockHash: msg.BlockHash.Bytes(),
	}
	return proto.Marshal(message)
}

func marshalCreateGroupPingMessage(msg *model.CreateGroupPingMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	message := &tas_middleware_pb.CreateGroupPingMessage{
		Sign:        si,
		PingID:      &msg.PingID,
		FromGroupID: msg.FromGroupID.Serialize(),
		BaseHeight:  &msg.BaseHeight,
	}
	return proto.Marshal(message)
}

func marshalCreateGroupPongMessage(msg *model.CreateGroupPongMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	tb, _ := msg.Ts.MarshalBinary()
	message := &tas_middleware_pb.CreateGroupPongMessage{
		Sign:   si,
		PingID: &msg.PingID,
		Ts:     tb,
	}
	return proto.Marshal(message)
}

func marshalSharePieceReqMessage(msg *model.ReqSharePieceMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	message := &tas_middleware_pb.ReqSharePieceMessage{
		Sign:  si,
		GHash: msg.GHash.Bytes(),
	}
	return proto.Marshal(message)
}

func marshalSharePieceResponseMessage(msg *model.ResponseSharePieceMessage) ([]byte, error) {
	si := signDataToPb(&msg.SI)
	message := &tas_middleware_pb.ResponseSharePieceMessage{
		Sign:       si,
		GHash:      msg.GHash.Bytes(),
		SharePiece: sharePieceToPb(&msg.Share),
	}
	return proto.Marshal(message)
}

func marshalReqProposalBlockMessage(msg *model.ReqProposalBlock) ([]byte, error) {
	m := &tas_middleware_pb.ReqProposalBlockMessage{
		Hash: msg.Hash.Bytes(),
	}
	return proto.Marshal(m)
}

func marshalResponseProposalBlockMessage(msg *model.ResponseProposalBlock) ([]byte, error) {
	transactions := types.TransactionsToPb(msg.Transactions)
	m := &tas_middleware_pb.ResponseProposalBlockMessage{Hash: msg.Hash.Bytes(), Transactions: transactions}
	return proto.Marshal(m)
}

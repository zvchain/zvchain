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

func marshalConsensusVerifyMessage(m *model.ConsensusVerifyMessage) ([]byte, error) {
	message := &tas_middleware_pb.ConsensusVerifyMessage{
		BlockHash:  m.BlockHash.Bytes(),
		RandomSign: m.RandomSign.Serialize(),
		Sign:       signDataToPb(&m.SI),
	}
	return proto.Marshal(message)
}

func signDataToPb(s *model.SignData) *tas_middleware_pb.SignData {
	sign := tas_middleware_pb.SignData{DataHash: s.DataHash.Bytes(), DataSign: s.DataSign.Serialize(), SignMember: s.SignMember.Serialize(), Version: &s.Version}
	return &sign
}

func rewardToPB(reward *types.Reward) *tas_middleware_pb.Reward {
	return &tas_middleware_pb.Reward{
		TxHash:     reward.TxHash.Bytes(),
		TargetIds:  reward.TargetIds,
		BlockHash:  reward.BlockHash.Bytes(),
		GroupId:    reward.Group.Bytes(),
		Sign:       reward.Sign,
		TotalValue: &reward.TotalValue,
	}
}

func marshalCastRewardTransSignReqMessage(msg *model.CastRewardTransSignReqMessage) ([]byte, error) {
	b := rewardToPB(&msg.Reward)
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
		BlockHash: msg.BlockHash.Bytes(),
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

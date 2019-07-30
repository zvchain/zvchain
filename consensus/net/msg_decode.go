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
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
)

func baseMessage(sign *tas_middleware_pb.SignData) *model.BaseSignedMessage {
	return &model.BaseSignedMessage{SI: *pbToSignData(sign)}
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
		return nil, fmt.Errorf("unMarshalConsensusVerifyMessage:%v", e)
	}
	return &model.ConsensusVerifyMessage{
		BlockHash:         common.BytesToHash(m.BlockHash),
		RandomSign:        *groupsig.DeserializeSign(m.RandomSign),
		BaseSignedMessage: *baseMessage(m.Sign),
	}, nil
}

func pbToSignData(s *tas_middleware_pb.SignData) *model.SignData {
	v := int32(0)
	if s.Version != nil {
		v = *s.Version
	}
	sign := model.SignData{
		DataHash:   common.BytesToHash(s.DataHash),
		DataSign:   *groupsig.DeserializeSign(s.DataSign),
		SignMember: groupsig.DeserializeID(s.SignMember),
		Version:    v,
	}
	return &sign
}

func pbToReward(b *tas_middleware_pb.Reward) *types.Reward {
	return &types.Reward{
		TxHash:     common.BytesToHash(b.TxHash),
		TargetIds:  b.TargetIds,
		BlockHash:  common.BytesToHash(b.BlockHash),
		Group:      common.BytesToHash(b.GroupId),
		Sign:       b.Sign,
		TotalValue: *b.TotalValue,
	}
}

func unMarshalCastRewardReqMessage(b []byte) (*model.CastRewardTransSignReqMessage, error) {
	message := new(tas_middleware_pb.CastRewardTransSignReqMessage)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, fmt.Errorf("unMarshalCastRewardReqMessage:%v", e)
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
		return nil, fmt.Errorf("unMarshalCastRewardSignMessage:%v", e)
	}

	sign := pbToSignData(message.Sign)
	base := model.BaseSignedMessage{SI: *sign}

	m := &model.CastRewardTransSignMessage{
		BaseSignedMessage: base,
		BlockHash:         common.BytesToHash(message.BlockHash),
	}
	return m, nil
}

func unmarshalReqProposalBlockMessage(b []byte) (*model.ReqProposalBlock, error) {
	message := &tas_middleware_pb.ReqProposalBlockMessage{}
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, fmt.Errorf("unmarshalReqProposalBlockMessage:%v", e)
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
		return nil, fmt.Errorf("unmarshalResponseProposalBlockMessage:%v", e)
	}
	transactions := types.PbToTransactions(message.Transactions)

	m := &model.ResponseProposalBlock{
		Hash:         common.BytesToHash(message.Hash),
		Transactions: transactions,
	}
	return m, nil
}

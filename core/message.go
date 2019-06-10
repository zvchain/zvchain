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

package core

import (
	"github.com/gogo/protobuf/proto"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
)

type MessageBase struct {
}

type blockResponseMessage struct {
	Blocks []*types.Block
}

type SyncCandidateInfo struct {
	Candidate       string // Candidate's ID
	CandidateHeight uint64 // Candidate's current block/group height
	ReqHeight       uint64 // Candidate's current request block/group height
}

type syncMessage struct {
	CandidateInfo *SyncCandidateInfo
}

func (msg *syncMessage) GetRaw() []byte {
	panic("implement me")
}

func (msg *syncMessage) GetData() interface{} {
	return msg.CandidateInfo
}

type syncRequest struct {
	ReqHeight uint64
	ReqSize   int32
}

func marshalSyncRequest(r *syncRequest) ([]byte, error) {
	pbr := &tas_middleware_pb.SyncRequest{
		ReqSize:   &r.ReqSize,
		ReqHeight: &r.ReqHeight,
	}
	return proto.Marshal(pbr)
}

func unmarshalSyncRequest(b []byte) (*syncRequest, error) {
	m := new(tas_middleware_pb.SyncRequest)
	e := proto.Unmarshal(b, m)
	if e != nil {
		return nil, e
	}
	return &syncRequest{ReqHeight: *m.ReqHeight, ReqSize: *m.ReqSize}, nil
}

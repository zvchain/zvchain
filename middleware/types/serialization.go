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

package types

import (
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/taslog"
)

// logger is middleware module system
var logger taslog.Logger

func InitMiddleware() {
	logger = taslog.GetLoggerByIndex(taslog.MiddlewareLogConfig, common.GlobalConf.GetString("instance", "index", ""))
}

// UnMarshalTransactions deserialize from []byte to *Transaction
func UnMarshalTransactions(b []byte) ([]*Transaction, error) {
	ts := new(tas_middleware_pb.TransactionSlice)
	error := proto.Unmarshal(b, ts)
	if error != nil {
		logger.Errorf("[handler]Unmarshal transactions error:%s", error.Error())
		return nil, error
	}

	result := PbToTransactions(ts.Transactions)
	return result, nil
}

// UnMarshalBlock deserialize from []byte to *Block
func UnMarshalBlock(bytes []byte) (*Block, error) {
	b := new(tas_middleware_pb.Block)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("[handler]Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	block := PbToBlock(b)
	return block, nil
}

// UnMarshalBlockHeader deserialize from []byte to *BlockHeader
func UnMarshalBlockHeader(bytes []byte) (*BlockHeader, error) {
	b := new(tas_middleware_pb.BlockHeader)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("[handler]Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	header := PbToBlockHeader(b)
	return header, nil
}

// UnMarshalMember deserialize from []byte to *Member
func UnMarshalMember(b []byte) (*Member, error) {
	member := new(tas_middleware_pb.Member)
	e := proto.Unmarshal(b, member)
	if e != nil {
		logger.Errorf("UnMarshalMember error:%s\n", e.Error())
		return nil, e
	}
	m := pbToMember(member)
	return m, nil
}

// UnMarshalGroup deserialize from []byte to *Group
func UnMarshalGroup(b []byte) (*Group, error) {
	group := new(tas_middleware_pb.Group)
	e := proto.Unmarshal(b, group)
	if e != nil {
		logger.Errorf("UnMarshalGroup error:%s\n", e.Error())
		return nil, e
	}
	g := PbToGroup(group)
	return g, nil
}

// MarshalTransaction serialize *Transaction
func MarshalTransaction(t *Transaction) ([]byte, error) {
	transaction := transactionToPb(t)
	return proto.Marshal(transaction)
}

// MarshalTransactions serialize []*Transaction
func MarshalTransactions(txs []*Transaction) ([]byte, error) {
	transactions := TransactionsToPb(txs)
	transactionSlice := tas_middleware_pb.TransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

// MarshalBlock serialize *Block
func MarshalBlock(b *Block) ([]byte, error) {
	block := BlockToPb(b)
	if block == nil {
		return nil, nil
	}
	return proto.Marshal(block)
}

// MarshalBlockHeader Serialize *BlockHeader
func MarshalBlockHeader(b *BlockHeader) ([]byte, error) {
	block := BlockHeaderToPb(b)
	if block == nil {
		return nil, nil
	}
	return proto.Marshal(block)
}

// MarshalMember serialize *Member
func MarshalMember(m *Member) ([]byte, error) {
	member := memberToPb(m)
	return proto.Marshal(member)
}

// MarshalGroup serialize *Group
func MarshalGroup(g *Group) ([]byte, error) {
	group := GroupToPb(g)
	return proto.Marshal(group)
}

func pbToTransaction(t *tas_middleware_pb.Transaction) *Transaction {
	if t == nil {
		return &Transaction{}
	}

	var target *common.Address
	if t.Target != nil {
		t := common.BytesToAddress(t.Target)
		target = &t
	}
	value := new(BigInt).SetBytesWithSign(t.Value)
	gasLimit := new(BigInt).SetBytesWithSign(t.GasLimit)
	gasPrice := new(BigInt).SetBytesWithSign(t.GasPrice)

	transaction := &Transaction{Data: t.Data, Value: value, Nonce: *t.Nonce,
		Target: target, GasLimit: gasLimit, GasPrice: gasPrice, Hash: common.BytesToHash(t.Hash),
		ExtraData: t.ExtraData, ExtraDataType: int8(*t.ExtraDataType), Type: int8(*t.Type), Sign: t.Sign}
	return transaction
}

func PbToTransactions(txs []*tas_middleware_pb.Transaction) []*Transaction {
	result := make([]*Transaction, 0)
	if txs == nil {
		return result
	}
	for _, t := range txs {
		transaction := pbToTransaction(t)
		result = append(result, transaction)
	}
	return result
}

func PbToBlockHeader(h *tas_middleware_pb.BlockHeader) *BlockHeader {
	if h == nil {
		return nil
	}
	header := BlockHeader{Hash: common.BytesToHash(h.Hash), Height: *h.Height, PreHash: common.BytesToHash(h.PreHash), Elapsed: *h.Elapsed,
		ProveValue: h.ProveValue, CurTime: time2.Int64ToTimeStamp(*h.CurTime), Castor: h.Castor, GroupID: h.GroupId, Signature: h.Signature,
		Nonce: *h.Nonce, TxTree: common.BytesToHash(h.TxTree), ReceiptTree: common.BytesToHash(h.ReceiptTree), StateTree: common.BytesToHash(h.StateTree),
		ExtraData: h.ExtraData, TotalQN: *h.TotalQN, Random: h.Random}
	return &header
}

func PbToBlock(b *tas_middleware_pb.Block) *Block {
	if b == nil {
		return nil
	}
	h := PbToBlockHeader(b.Header)
	txs := PbToTransactions(b.Transactions)
	block := Block{Header: h, Transactions: txs}
	return &block
}

func PbToGroupHeader(g *tas_middleware_pb.GroupHeader) *GroupHeader {
	header := GroupHeader{
		Hash:          common.BytesToHash(g.Hash),
		Parent:        g.Parent,
		PreGroup:      g.PreGroup,
		Authority:     *g.Authority,
		Name:          *g.Name,
		BeginTime:     time2.Int64ToTimeStamp(*g.BeginTime),
		MemberRoot:    common.BytesToHash(g.MemberRoot),
		CreateHeight:  *g.CreateHeight,
		ReadyHeight:   *g.ReadyHeight,
		WorkHeight:    *g.WorkHeight,
		DismissHeight: *g.DismissHeight,
		Extends:       *g.Extends,
	}
	return &header
}

func PbToGroup(g *tas_middleware_pb.Group) *Group {
	group := Group{
		Header:      PbToGroupHeader(g.Header),
		ID:          g.Id,
		Members:     g.Members,
		PubKey:      g.PubKey,
		Signature:   g.Signature,
		GroupHeight: *g.GroupHeight,
	}
	return &group
}

func PbToGroups(g *tas_middleware_pb.GroupSlice) []*Group {
	result := make([]*Group, 0)
	for _, group := range g.Groups {
		result = append(result, PbToGroup(group))
	}
	return result
}

func pbToMember(m *tas_middleware_pb.Member) *Member {
	member := Member{ID: m.Id, PubKey: m.PubKey}
	return &member
}

func transactionToPb(t *Transaction) *tas_middleware_pb.Transaction {
	if t == nil {
		return nil
	}
	var (
		target []byte
	)
	if t.Target != nil {
		target = t.Target.Bytes()
	}
	et := int32(t.ExtraDataType)
	tp := int32(t.Type)
	transaction := tas_middleware_pb.Transaction{Data: t.Data, Value: t.Value.GetBytesWithSign(), Nonce: &t.Nonce,
		Target: target, GasLimit: t.GasLimit.GetBytesWithSign(), GasPrice: t.GasPrice.GetBytesWithSign(), Hash: t.Hash.Bytes(),
		ExtraData: t.ExtraData, ExtraDataType: &et, Type: &tp, Sign: t.Sign}
	return &transaction
}

func TransactionsToPb(txs []*Transaction) []*tas_middleware_pb.Transaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*tas_middleware_pb.Transaction, 0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func BlockHeaderToPb(h *BlockHeader) *tas_middleware_pb.BlockHeader {
	ts := h.CurTime.Unix()
	header := tas_middleware_pb.BlockHeader{Hash: h.Hash.Bytes(), Height: &h.Height, PreHash: h.PreHash.Bytes(), Elapsed: &h.Elapsed,
		ProveValue: h.ProveValue, CurTime: &ts, Castor: h.Castor, GroupId: h.GroupID, Signature: h.Signature,
		Nonce: &h.Nonce, TxTree: h.TxTree.Bytes(), ReceiptTree: h.ReceiptTree.Bytes(), StateTree: h.StateTree.Bytes(),
		ExtraData: h.ExtraData, TotalQN: &h.TotalQN, Random: h.Random}
	return &header
}

func BlockToPb(b *Block) *tas_middleware_pb.Block {
	if b == nil {
		return nil
	}
	header := BlockHeaderToPb(b.Header)
	transactions := TransactionsToPb(b.Transactions)
	block := tas_middleware_pb.Block{Header: header, Transactions: transactions}
	return &block
}

func GroupToPbHeader(g *GroupHeader) *tas_middleware_pb.GroupHeader {
	t := g.BeginTime.Unix()
	header := tas_middleware_pb.GroupHeader{
		Hash:          g.Hash.Bytes(),
		Parent:        g.Parent,
		PreGroup:      g.PreGroup,
		Authority:     &g.Authority,
		Name:          &g.Name,
		BeginTime:     &t,
		MemberRoot:    g.MemberRoot.Bytes(),
		CreateHeight:  &g.CreateHeight,
		ReadyHeight:   &g.ReadyHeight,
		WorkHeight:    &g.WorkHeight,
		DismissHeight: &g.DismissHeight,
		Extends:       &g.Extends,
	}
	return &header
}

func GroupToPb(g *Group) *tas_middleware_pb.Group {
	group := tas_middleware_pb.Group{
		Header:      GroupToPbHeader(g.Header),
		Id:          g.ID,
		Members:     g.Members,
		PubKey:      g.PubKey,
		Signature:   g.Signature,
		GroupHeight: &g.GroupHeight,
	}
	return &group
}

func memberToPb(m *Member) *tas_middleware_pb.Member {
	member := tas_middleware_pb.Member{Id: m.ID, PubKey: m.PubKey}
	return &member
}

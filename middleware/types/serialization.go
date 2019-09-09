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
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	time2 "github.com/zvchain/zvchain/middleware/time"
)

// MiddleWareLogger is middleware module system
var MiddleWareLogger *logrus.Logger

func InitMiddleware() {
	MiddleWareLogger = log.MiddlewareLogger
}

// UnMarshalTransactions deserialize from []byte to *Transaction
func UnMarshalTransactions(b []byte) ([]*RawTransaction, error) {
	ts := new(tas_middleware_pb.RawTransactionSlice)
	error := proto.Unmarshal(b, ts)
	if error != nil {
		MiddleWareLogger.Errorf("[handler]Unmarshal transactions error:%s", error.Error())
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
		MiddleWareLogger.Errorf("[handler]Unmarshal Block error:%s", error.Error())
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
		MiddleWareLogger.Errorf("[handler]Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	header := PbToBlockHeader(b)
	return header, nil
}

// MarshalTransactions serialize []*Transaction
func MarshalTransactions(txs []*RawTransaction) ([]byte, error) {
	transactions := TransactionsToPb(txs)
	transactionSlice := tas_middleware_pb.RawTransactionSlice{Transactions: transactions}
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

func ensureUint64(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
func ensureInt32(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
func ensureInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func byteToHash(b []byte) common.Hash {
	if len(b) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(b)
}

func pbToTransaction(t *tas_middleware_pb.RawTransaction) *RawTransaction {
	if t == nil {
		return &RawTransaction{}
	}

	var (
		target *common.Address
		source *common.Address
	)
	if t.Target != nil {
		t := common.BytesToAddress(t.Target)
		target = &t
	}
	if t.Source != nil {
		t := common.BytesToAddress(t.Source)
		source = &t
	}

	value := new(BigInt).SetBytesWithSign(t.Value)
	gasLimit := new(BigInt).SetBytesWithSign(t.GasLimit)
	gasPrice := new(BigInt).SetBytesWithSign(t.GasPrice)

	transaction := &RawTransaction{
		Source:    source,
		Data:      t.Data,
		Value:     value,
		Nonce:     ensureUint64(t.Nonce),
		Target:    target,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		ExtraData: t.ExtraData,
		Type:      int8(ensureInt32(t.Type)),
		Sign:      t.Sign,
	}
	return transaction
}

func PbToTransactions(txs []*tas_middleware_pb.RawTransaction) []*RawTransaction {
	result := make([]*RawTransaction, 0)
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

	header := BlockHeader{
		Hash:        byteToHash(h.Hash),
		Height:      ensureUint64(h.Height),
		PreHash:     byteToHash(h.PreHash),
		Elapsed:     ensureInt32(h.Elapsed),
		ProveValue:  h.ProveValue,
		CurTime:     time2.Int64MilliSecondsToTimeStamp(ensureInt64(h.CurTime)),
		Castor:      h.Castor,
		Group:       byteToHash(h.GroupId),
		Signature:   h.Signature,
		Nonce:       ensureInt32(h.Nonce),
		TxTree:      byteToHash(h.TxTree),
		ReceiptTree: byteToHash(h.ReceiptTree),
		StateTree:   byteToHash(h.StateTree),
		ExtraData:   h.ExtraData,
		TotalQN:     ensureUint64(h.TotalQN),
		Random:      h.Random,
		GasFee:      ensureUint64(h.GasFee),
	}
	return &header
}

func PbToBlock(b *tas_middleware_pb.Block) *Block {
	if b == nil {
		return nil
	}
	h := PbToBlockHeader(b.Header)
	txs := PbToTransactions(b.Transactions)
	block := &Block{Header: h, Transactions: txs}
	return block
}

func transactionToPb(t *RawTransaction) *tas_middleware_pb.RawTransaction {
	if t == nil {
		return nil
	}
	var (
		target []byte
		source []byte
	)
	if t.Target != nil {
		target = t.Target.Bytes()
	}
	if t.Source != nil {
		source = t.Source.Bytes()
	}
	tp := int32(t.Type)
	transaction := tas_middleware_pb.RawTransaction{
		Data:      t.Data,
		Value:     t.Value.GetBytesWithSign(),
		Nonce:     &t.Nonce,
		Target:    target,
		GasLimit:  t.GasLimit.GetBytesWithSign(),
		GasPrice:  t.GasPrice.GetBytesWithSign(),
		ExtraData: t.ExtraData,
		Type:      &tp,
		Sign:      t.Sign,
		Source:    source,
	}
	return &transaction
}

func TransactionsToPb(txs []*RawTransaction) []*tas_middleware_pb.RawTransaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*tas_middleware_pb.RawTransaction, 0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func BlockHeaderToPb(h *BlockHeader) *tas_middleware_pb.BlockHeader {
	ts := h.CurTime.UnixMilli()
	header := tas_middleware_pb.BlockHeader{
		Hash:        h.Hash.Bytes(),
		Height:      &h.Height,
		PreHash:     h.PreHash.Bytes(),
		Elapsed:     &h.Elapsed,
		ProveValue:  h.ProveValue,
		CurTime:     &ts,
		Castor:      h.Castor,
		GroupId:     h.Group.Bytes(),
		Signature:   h.Signature,
		Nonce:       &h.Nonce,
		TxTree:      h.TxTree.Bytes(),
		ReceiptTree: h.ReceiptTree.Bytes(),
		StateTree:   h.StateTree.Bytes(),
		ExtraData:   h.ExtraData,
		TotalQN:     &h.TotalQN,
		Random:      h.Random,
		GasFee:      &h.GasFee,
	}
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

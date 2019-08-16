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

// Package types define the key data structures for the chain
package types

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/time"
)

type AddBlockOnChainSituation string

// AddBlockResult is the result of the add-block operation
type AddBlockResult int8

// gasLimitMax expresses the max gasLimit of a transaction
var gasLimitMax = new(BigInt).SetUint64(500000)

var (
	AdminAddr         = common.StringToAddress("zv28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031b") // address of admin who can control foundation contract
	MiningPoolAddr    = common.StringToAddress("zv01cf40d3a25d0a00bb6876de356e702ae5a2a379c95e77c5fd04f4cc6bb680c0") // address of mining pool in pre-distribution
	CirculatesAddr    = common.StringToAddress("zvebb50bcade66df3fcb8df1eeeebad6c76332f2aee43c9c11b5cd30187b45f6d3") // address of circulates in pre-distribution
	UserNodeAddress   = common.StringToAddress("zve30c75b3fd8888f410ac38ec0a07d82dcc613053513855fb4dd6d75bc69e8139") // address of official reserved user node address
	DaemonNodeAddress = common.StringToAddress("zvae1889182874d8dad3c3e033cde3229a3320755692e37cbe1caab687bf6a1122") // address of official reserved daemon node address
)

var ExtractGuardNodes = []common.Address{
	common.StringToAddress("zvcf176aca3e4f1f5721d50f536e0e1e06434e188379e27d68656bef4b2ad904c6"),
	common.StringToAddress("zvf06321edb1512b17646aa8a2bea4d898758f85d7b6cd4ec9624363be00db0198"),
} // init gurad miner nodes

// defines all possible result of the add-block operation
const (
	AddBlockFailed            AddBlockResult = -1 // Means the operations is fail
	AddBlockConsensusFailed   AddBlockResult = -2 // Means the consensus is fail
	AddBlockSucc              AddBlockResult = 0  // Means success
	BlockExisted              AddBlockResult = 1  // Means the block already added before
	BlockTotalQnLessThanLocal AddBlockResult = 2  // Weight consideration
)

const (
	TVMExecutedError     = 1001
	TVMGasNotEnoughError = 1002
	TVMCheckABIError     = 1003
	TVMCallMaxDeepError  = 1004
	TVMNoCodeError       = 1005

	txFixSize = 200 // Fixed size for each transaction
)

type TransactionError struct {
	Code    int
	Message string
}

func NewTransactionError(code int, msg string) *TransactionError {
	return &TransactionError{Code: code, Message: msg}
}

// Supported transaction types
const (
	TransactionTypeTransfer       = 0
	TransactionTypeContractCreate = 1
	TransactionTypeContractCall   = 2

	// Miner operation related type
	TransactionTypeStakeAdd        = 3
	TransactionTypeMinerAbort      = 4
	TransactionTypeStakeReduce     = 5
	TransactionTypeStakeRefund     = 6
	TransactionTypeApplyGuardMiner = 7 // apply guard node
	TransactionTypeVoteMinerPool   = 8 // vote to miner pool
	TransactionTypeCancelGuard     = 9 // cancel guard node,only admin can call

	// Group operation related type
	TransactionTypeGroupPiece       = 101 //group member upload his encrypted share piece
	TransactionTypeGroupMpk         = 102 //group member upload his mpk
	TransactionTypeGroupOriginPiece = 103 //group member upload origin share piece
	TransactionTypeReward           = 104
)

// Transaction denotes one transaction infos
type Transaction struct {
	Data   []byte          `msgpack:"dt,omitempty"` // Data of the transaction, cost gas
	Value  *BigInt         `msgpack:"v"`            // The value the sender suppose to transfer
	Nonce  uint64          `msgpack:"nc"`           // The nonce indicates the transaction sequence related to sender
	Target *common.Address `msgpack:"tg,omitempty"` // The receiver address
	Type   int8            `msgpack:"tp"`           // Transaction type

	GasLimit *BigInt     `msgpack:"gl"`
	GasPrice *BigInt     `msgpack:"gp"`
	Hash     common.Hash `msgpack:"h"`

	ExtraData []byte          `msgpack:"ed"`
	Sign      []byte          `msgpack:"si"`  // The Sign of the sender
	Source    *common.Address `msgpack:"src"` // Sender address, recovered from sign
}

func (tx *Transaction) GetNonce() uint64 {
	return tx.Nonce
}

func (tx *Transaction) GetExtraData() []byte {
	return tx.ExtraData
}

func (tx *Transaction) GetSign() []byte {
	return tx.Sign
}

func (tx *Transaction) GetType() int8 {
	return tx.Type
}

// GenHash generate unique hash of the transaction. source,sign is out of the hash calculation range
func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}
	buffer := bytes.Buffer{}
	if tx.Data != nil {
		buffer.Write(tx.Data)
	}
	buffer.Write(tx.Value.GetBytesWithSign())
	buffer.Write(common.Uint64ToByte(tx.Nonce))
	if tx.Target != nil {
		buffer.Write(tx.Target.Bytes())
	}
	buffer.WriteByte(byte(tx.Type))
	buffer.Write(tx.GasLimit.GetBytesWithSign())
	buffer.Write(tx.GasPrice.GetBytesWithSign())
	if tx.ExtraData != nil {
		buffer.Write(tx.ExtraData)
	}

	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) HexSign() string {
	return common.ToHex(tx.Sign)
}

// RecoverSource recover source from the sign field.
// It returns directly if source is not nil or it is a reward transaction.
func (tx *Transaction) RecoverSource() error {
	if tx.Source != nil || tx.IsReward() {
		return nil
	}
	sign := common.BytesToSign(tx.Sign)
	if sign == nil {
		return fmt.Errorf("BytesToSign fail, sign=%x", tx.Sign)
	}
	pk, err := sign.RecoverPubkey(tx.Hash.Bytes())
	if err == nil {
		src := pk.GetAddress()
		tx.Source = &src
	}
	return err
}

func (tx *Transaction) Size() int {
	return txFixSize + len(tx.Data) + len(tx.ExtraData)
}

func (tx *Transaction) IsReward() bool {
	return tx.Type == TransactionTypeReward
}

func (tx Transaction) GetData() []byte { return tx.Data }

func (tx Transaction) GetGasLimit() uint64 {
	return tx.GasLimit.Uint64()
}
func (tx Transaction) GetValue() uint64 {
	return tx.Value.Uint64()
}

func (tx Transaction) GetSource() *common.Address { return tx.Source }
func (tx Transaction) GetTarget() *common.Address { return tx.Target }
func (tx Transaction) GetHash() common.Hash       { return tx.Hash }

// PriorityTransactions is a transaction array that determines the priority based on gasprice.
// Gasprice is placed low
type PriorityTransactions []*Transaction

func (pt PriorityTransactions) Len() int {
	return len(pt)
}
func (pt PriorityTransactions) Swap(i, j int) {
	pt[i], pt[j] = pt[j], pt[i]
}
func (pt PriorityTransactions) Less(i, j int) bool {
	return pt[i].GasPrice.Cmp(&pt[j].GasPrice.Int) < 0
}
func (pt *PriorityTransactions) Push(x interface{}) {
	item := x.(*Transaction)
	*pt = append(*pt, item)
}

func (pt *PriorityTransactions) Pop() interface{} {
	old := *pt
	n := len(old)
	item := old[n-1]

	*pt = old[0 : n-1]
	return item
}

// Reward is the reward transaction raw data
type Reward struct {
	TxHash     common.Hash
	TargetIds  []int32
	BlockHash  common.Hash
	Group      common.Hash
	Sign       []byte
	TotalValue uint64
	PackFee    uint64
}

// BlockHeader is block header structure
type BlockHeader struct {
	Hash        common.Hash    // The hash of this block
	Height      uint64         // The height of this block
	PreHash     common.Hash    // The hash of previous block
	Elapsed     int32          // The length of time from the last block
	ProveValue  []byte         // Vrf prove
	TotalQN     uint64         // QN of the entire chain
	CurTime     time.TimeStamp // Current block time
	Castor      []byte         // Proposer ID
	Group       common.Hash    // Verify group hash，hash of the seed block
	Signature   []byte         // Group signature from consensus
	Nonce       int32          // Salt
	TxTree      common.Hash    // Transaction Merkel root hash
	ReceiptTree common.Hash    // Receipte Merkel root hash
	StateTree   common.Hash    // State db Merkel root hash
	ExtraData   []byte
	Random      []byte // Random number generated during the consensus process
	GasFee      uint64 // gas fee of transaction executed in block
}

// GenHash calculates the hash of the block
func (bh *BlockHeader) GenHash() common.Hash {
	buf := bytes.NewBuffer([]byte{})

	buf.Write(common.UInt64ToByte(bh.Height))

	buf.Write(bh.PreHash.Bytes())

	buf.Write(common.Int32ToByte(bh.Elapsed))

	buf.Write(bh.ProveValue)

	buf.Write(common.UInt64ToByte(bh.TotalQN))

	buf.Write(bh.CurTime.Bytes())

	buf.Write(bh.Castor)

	buf.Write(bh.Group.Bytes())

	buf.Write(common.Int32ToByte(bh.Nonce))

	buf.Write(bh.TxTree.Bytes())
	buf.Write(bh.ReceiptTree.Bytes())
	buf.Write(bh.StateTree.Bytes())
	buf.Write(common.Uint64ToByte(bh.GasFee))
	if bh.ExtraData != nil {
		buf.Write(bh.ExtraData)
	}

	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func (bh *BlockHeader) PreTime() time.TimeStamp {
	return bh.CurTime.Add(int64(-bh.Elapsed))
}

func (bh *BlockHeader) HasTransactions() bool {
	return bh.TxTree != common.EmptyHash
}

// Block is the block data structure consists of the header and transactions as body
type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

func (b *Block) GetTransactionHashs() []common.Hash {
	if b.Transactions == nil {
		return []common.Hash{}
	}
	hashs := make([]common.Hash, 0)
	for _, tx := range b.Transactions {
		hashs = append(hashs, tx.Hash)
	}
	return hashs
}

// BlockWeight denotes the weight of one block
type BlockWeight struct {
	Hash    common.Hash
	TotalQN uint64   // Same as TotalQN field of BlockHeader
	PV      *big.Int // Converted from ProveValue field of BlockHeader
}

type CandidateBlockHeader struct {
	BW *BlockWeight
	BH *BlockHeader
}

type PvFunc func(pvBytes []byte) *big.Int

var DefaultPVFunc PvFunc

// MoreWeight checks the current block is more weight than the given one
func (bw *BlockWeight) MoreWeight(bw2 *BlockWeight) bool {
	return bw.Cmp(bw2) > 0
}

// Cmp compares the weight between current block and the given one.
// 1 returns if current block is more weight
// 0 returns if equal
// otherwise -1 is returned
func (bw *BlockWeight) Cmp(bw2 *BlockWeight) int {
	if bw.TotalQN > bw2.TotalQN {
		return 1
	} else if bw.TotalQN < bw2.TotalQN {
		return -1
	}
	return bw.PV.Cmp(bw2.PV)
}

func NewCandidateBlockHeader(bh *BlockHeader) *CandidateBlockHeader {
	bw := NewBlockWeight(bh)
	return &CandidateBlockHeader{BW: bw, BH: bh}
}

func NewBlockWeight(bh *BlockHeader) *BlockWeight {
	return &BlockWeight{
		Hash:    bh.Hash,
		TotalQN: bh.TotalQN,
		PV:      DefaultPVFunc(bh.ProveValue),
	}
}

func (bw BlockWeight) String() string {
	return fmt.Sprintf("%v-%v", bw.TotalQN, bw.Hash)
}

func IsInExtractGuardNodes(addr common.Address) bool {
	for _, addrStr := range ExtractGuardNodes {
		if addrStr == addr {
			return true
		}
	}
	return false
}

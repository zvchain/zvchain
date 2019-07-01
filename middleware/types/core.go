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

// defines all possible result of the add-block operation
const (
	AddBlockFailed            AddBlockResult = -1 // Means the operations is fail
	AddBlockSucc              AddBlockResult = 0  // Means success
	BlockExisted              AddBlockResult = 1  // Means the block already added before
	BlockTotalQnLessThanLocal AddBlockResult = 2  // Weight consideration
	Forking                   AddBlockResult = 3
)
const (
	Success                            = 0
	TxErrorBalanceNotEnough        	   = 1
	TxErrorCodeContractAddressConflict = 2
	TxFailed 						   = 3
	TxErrorCodeNoCode                  = 4

	//TODO detail error
	TVMExecutedError = 1003

	SysCheckABIError = 2002
	SysABIJSONError  = 2003

	txFixSize = 200 // Fixed size for each transaction
)

var (
	NoCodeErr                    = 4
	NoCodeErrorMsg               = "get code from address %s,but no code!"
	ABIJSONError                 = 2003
	ABIJSONErrorMsg              = "abi json format error"
	CallMaxDeepError             = 2004
	CallMaxDeepErrorMsg          = "call max deep cannot more than 8"
	InitContractError            = 2005
	InitContractErrorMsg         = "contract init error"
	TargetNilError               = 2006
	TargetNilErrorMsg            = "target nil error"
)

var (
	TxErrorBalanceNotEnoughErr          = NewTransactionError(TxErrorBalanceNotEnough, "balance not enough")
	TxErrorABIJSONErr                   = NewTransactionError(SysABIJSONError, "abi json format error")
	TxErrorFailedErr                    = NewTransactionError(TxFailed, "failed")
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
	TransactionTypeTransfer         = 0
	TransactionTypeContractCreate   = 1
	TransactionTypeContractCall     = 2
	TransactionTypeReward           = 3
	TransactionTypeMinerApply       = 4
	TransactionTypeMinerAbort       = 5
	TransactionTypeMinerRefund      = 6
	TransactionTypeMinerCancelStake = 7
	TransactionTypeMinerStake       = 8
	TransactionTypeGroupPiece       = 9
	TransactionTypeGroupMpk         = 10
	TransactionTypeGroupOriginPiece = 11

	TransactionTypeToBeRemoved = -1
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

	ExtraData     []byte          `msgpack:"ed"`
	ExtraDataType int8            `msgpack:"et,omitempty"`
	Sign          []byte          `msgpack:"si"`  // The Sign of the sender
	Source        *common.Address `msgpack:"src"` // Sender address, recovered from sign
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
	buffer.WriteByte(byte(tx.ExtraDataType))

	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) HexSign() string {
	return common.ToHex(tx.Sign)
}

// RecoverSource recover source from the sign field.
// It returns directly if source is not nil or it is a reward transaction.
func (tx *Transaction) RecoverSource() error {
	if tx.Source != nil || tx.Type == TransactionTypeReward {
		return nil
	}
	sign := common.BytesToSign(tx.Sign)
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

// BoundCheck check if the transaction param exceeds the bounds
func (tx *Transaction) BoundCheck() error {
	if tx.GasPrice == nil || !tx.GasPrice.IsUint64() {
		return fmt.Errorf("illegal tx gasPrice:%v", tx.GasPrice)
	}
	if tx.GasLimit == nil || !tx.GasLimit.IsUint64() {
		return fmt.Errorf("illegal tx gasLimit:%v", tx.GasLimit)
	}
	if tx.Value == nil || !tx.Value.IsUint64() {
		return fmt.Errorf("illegal tx value:%v", tx.Value)
	}
	if tx.Type == TransactionTypeTransfer || tx.Type == TransactionTypeContractCall {
		if tx.Target == nil {
			return fmt.Errorf("param target cannot nil")
		}
	}
	if tx.Type == TransactionTypeMinerApply || tx.Type == TransactionTypeMinerCancelStake || tx.Type ==TransactionTypeMinerStake {
		if tx.Data == nil {
			return fmt.Errorf("param data cannot nil")
		}
	}
	return nil
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
	if pt[i].Type == TransactionTypeToBeRemoved && pt[j].Type != TransactionTypeToBeRemoved {
		return true
	} else if pt[i].Type != TransactionTypeToBeRemoved && pt[j].Type == TransactionTypeToBeRemoved {
		return false
	} else {
		return pt[i].GasPrice.Cmp(&pt[j].GasPrice.Int) < 0
	}
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
	GroupID    []byte
	Sign       []byte
	TotalValue uint64
}

const (
	MinerTypeLight    = 0
	MinerTypeHeavy    = 1
	MinerStatusNormal = 0
	MinerStatusAbort  = 1
)

// Miner is the miner info including public keys and pledges
type Miner struct {
	ID           []byte
	PublicKey    []byte
	VrfPublicKey []byte
	ApplyHeight  uint64
	Stake        uint64
	AbortHeight  uint64
	Type         byte
	Status       byte
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
	GroupID     []byte         // Verify group ID，binary representation of groupsig.ID
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

	buf.Write(bh.GroupID)

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

// Member is used for one member of groups
type Member struct {
	ID     []byte
	PubKey []byte
}

// GroupHeader is the group header info
type GroupHeader struct {
	Hash          common.Hash // Group header hash
	Parent        []byte      // Parent group ID, which create the current group
	PreGroup      []byte      // Previous group ID on group chain
	Authority     uint64      // The authority given by the parent group
	Name          string      // The name given by the parent group
	BeginTime     time.TimeStamp
	MemberRoot    common.Hash // Group members list root hash
	CreateHeight  uint64      // Height of the group created
	ReadyHeight   uint64      // Latest height of ready
	WorkHeight    uint64      // Height of work
	DismissHeight uint64      // Height of dismiss
	Extends       string      // Extend data
}

func (gh *GroupHeader) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(gh.Parent)
	buf.Write(gh.PreGroup)
	buf.Write(common.Uint64ToByte(gh.Authority))
	buf.WriteString(gh.Name)

	buf.Write(gh.MemberRoot.Bytes())
	buf.Write(common.Uint64ToByte(gh.CreateHeight))
	buf.Write(common.Uint64ToByte(gh.ReadyHeight))
	buf.Write(common.Uint64ToByte(gh.WorkHeight))
	buf.Write(common.Uint64ToByte(gh.DismissHeight))
	buf.WriteString(gh.Extends)
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

// DismissedAt checks if the group dismissed at the given height
func (gh *GroupHeader) DismissedAt(h uint64) bool {
	return gh.DismissHeight <= h
}

// WorkAt checks if the group can working at the given height
func (gh *GroupHeader) WorkAt(h uint64) bool {
	return !gh.DismissedAt(h) && gh.WorkHeight <= h
}

// Group is the whole group info
type Group struct {
	Header      *GroupHeader
	ID          []byte
	PubKey      []byte
	Signature   []byte
	Members     [][]byte // Member id list
	GroupHeight uint64
}

func (g *Group) MemberExist(id []byte) bool {
	for _, mem := range g.Members {
		if bytes.Equal(mem, id) {
			return true
		}
	}
	return false
}

type StateNode struct {
	Key   []byte
	Value []byte
}

// BlockWeight denotes the weight of one block
type BlockWeight struct {
	Hash    common.Hash
	TotalQN uint64   // Same as TotalQN field of BlockHeader
	PV      *big.Int // Converted from ProveValue field of BlockHeader
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

func NewBlockWeight(bh *BlockHeader) *BlockWeight {
	return &BlockWeight{
		Hash:    bh.Hash,
		TotalQN: bh.TotalQN,
		PV:      DefaultPVFunc(bh.ProveValue),
	}
}

func (bw *BlockWeight) String() string {
	return fmt.Sprintf("%v-%v", bw.TotalQN, bw.PV.Uint64())
}

// StakeStatus indicates the stake status
type StakeStatus = int

const (
	Staked      StakeStatus = iota // Normal status
	StakeFrozen                    // Frozen status
)

// StakeDetail expresses the stake detail
type StakeDetail struct {
	Source       common.Address
	Target       common.Address
	Value        uint64
	Status       StakeStatus
	FrozenHeight uint64
}

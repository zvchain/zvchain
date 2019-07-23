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

package cli

import (
	"math/big"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
)

// Result is rpc request successfully returns the variable parameter
type Result struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
}

func (r *Result) IsSuccess() bool {
	return r.Status == 0
}

// ErrorResult is rpc request error returned variable parameter
type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RPCReqObj is complete rpc request body
type RPCReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

// RPCResObj is complete rpc response body
type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

// Transactions in the buffer pool transaction list
type Transactions struct {
	Hash      string `json:"hash"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Value     string `json:"value"`
	Height    uint64 `json:"height"`
	BlockHash string `json:"block_hash"`
}

type PubKeyInfo struct {
	PubKey string `json:"pub_key"`
	ID     string `json:"id"`
}

type ConnInfo struct {
	ID      string `json:"id"`
	IP      string `json:"ip"`
	TCPPort string `json:"tcp_port"`
}

type GroupStat struct {
	Dismissed bool  `json:"dismissed"`
	VCount    int32 `json:"v_count"`
}

type ProposerStat struct {
	Stake      uint64  `json:"stake"`
	StakeRatio float64 `json:"stake_ratio"`
	PCount     int32   `json:"p_count"`
}

type CastStat struct {
	Group    map[string]GroupStat    `json:"group"`
	Proposer map[string]ProposerStat `json:"proposer"`
}

type MortGage struct {
	Stake       uint64 `json:"stake"`
	ApplyHeight uint64 `json:"apply_height"`
	Type        string `json:"type"`
	Status      string `json:"miner_status"`
}

func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "proposal node"
	if miner.IsVerifyRole() {
		t = "verify node"
	}
	status := "prepared"
	if miner.IsActive() {
		status = "normal"
	} else if miner.IsFrozen() {
		status = "frozen"
	}
	mg := &MortGage{
		Stake:       uint64(common.RA2TAS(miner.Stake)),
		ApplyHeight: miner.ApplyHeight,
		Type:        t,
		Status:      status,
	}
	return mg
}

type StakeDetail struct {
	Value        uint64 `json:"value"`
	UpdateHeight uint64 `json:"update_height"`
	MType        string `json:"m_type"`
	Status       string `json:"stake_status"`
}

type MinerStakeDetails struct {
	Overview []*MortGage               `json:"overview,omitempty"`
	Details  map[string][]*StakeDetail `json:"details,omitempty"`
}

type NodeInfo struct {
	ID           string     `json:"id"`
	Balance      float64    `json:"balance"`
	Status       string     `json:"status"`
	WGroupNum    int        `json:"w_group_num"`
	AGroupNum    int        `json:"a_group_num"`
	NType        string     `json:"n_type"`
	TxPoolNum    int        `json:"tx_pool_num"`
	BlockHeight  uint64     `json:"block_height"`
	GroupHeight  uint64     `json:"group_height"`
	MortGages    []MortGage `json:"mort_gages"`
	VrfThreshold float64    `json:"vrf_threshold"`
}

type PageObjects struct {
	Total uint64        `json:"count"`
	Data  []interface{} `json:"data"`
}

type Block struct {
	Height      uint64      `json:"height"`
	Hash        common.Hash `json:"hash"`
	PreHash     common.Hash `json:"pre_hash"`
	CurTime     time.Time   `json:"cur_time"`
	PreTime     time.Time   `json:"pre_time"`
	Castor      groupsig.ID `json:"castor"`
	Group       common.Hash `json:"group_id"`
	Prove       string      `json:"prove"`
	TotalQN     uint64      `json:"total_qn"`
	Qn          uint64      `json:"qn"`
	TxNum       uint64      `json:"txs"`
	StateRoot   common.Hash `json:"state_root"`
	TxRoot      common.Hash `json:"tx_root"`
	ReceiptRoot common.Hash `json:"receipt_root"`
	ProveRoot   common.Hash `json:"prove_root"`
	Random      string      `json:"random"`
}

type BlockDetail struct {
	Block
	GenRewardTx   *RewardTransaction    `json:"gen_reward_tx"`
	Trans         []Transaction         `json:"trans"`
	BodyRewardTxs []RewardTransaction   `json:"body_reward_txs"`
	MinerReward   []*MinerRewardBalance `json:"miner_reward"`
	PreTotalQN    uint64                `json:"pre_total_qn"`
}

type BlockReceipt struct {
	Receipts        []*types.Receipt `json:"receipts"`
	EvictedReceipts []*types.Receipt `json:"evictedReceipts"`
}

type ExplorerBlockDetail struct {
	BlockDetail
	Receipts        []*types.Receipt `json:"receipts"`
	EvictedReceipts []*types.Receipt `json:"evictedReceipts"`
}

type Group struct {
	Seed          common.Hash `json:"id"`
	BeginHeight   uint64      `json:"begin_height"`
	DismissHeight uint64      `json:"dismiss_height"`
	Threshold     int32       `json:"threshold"`
	Members       []string    `json:"members"`
	MemSize       int         `json:"mem_size"`
	GroupHeight   uint64      `json:"group_height"`
}

type MinerRewardBalance struct {
	ID            groupsig.ID `json:"id"`
	Proposal      bool        `json:"proposal"`       // Is there a proposal
	PackRewardTx  int         `json:"pack_reward_tx"` // The counts of packed reward transaction
	VerifyBlock   int         `json:"verify_block"`   // Number of blocks verified
	PreBalance    *big.Int    `json:"pre_balance"`
	CurrBalance   *big.Int    `json:"curr_balance"`
	ExpectBalance *big.Int    `json:"expect_balance"`
	Explain       string      `json:"explain"`
}

type Transaction struct {
	Data   []byte          `json:"data"`
	Value  float64         `json:"value"`
	Nonce  uint64          `json:"nonce"`
	Source *common.Address `json:"source"`
	Target *common.Address `json:"target"`
	Type   int8            `json:"type"`

	GasLimit uint64      `json:"gas_limit"`
	GasPrice uint64      `json:"gas_price"`
	Hash     common.Hash `json:"hash"`

	ExtraData string `json:"extra_data"`
}

type Receipt struct {
	Status            int          `json:"status"`
	CumulativeGasUsed uint64       `json:"cumulativeGasUsed"`
	Logs              []*types.Log `json:"logs"`

	TxHash          common.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress common.Address `json:"contractAddress"`
	Height          uint64         `json:"height"`
	TxIndex         uint16         `json:"tx_index"`
}

type ExecutedTransaction struct {
	Receipt     *Receipt
	Transaction *Transaction
}

type RewardTransaction struct {
	Hash         common.Hash   `json:"hash"`
	BlockHash    common.Hash   `json:"block_hash"`
	GroupSeed    common.Hash   `json:"group_id"`
	TargetIDs    []groupsig.ID `json:"target_ids"`
	Value        uint64        `json:"value"`
	PackFee      uint64        `json:"pack_fee"`
	StatusReport string        `json:"status_report"`
	Success      bool          `json:"success"`
}

type Dashboard struct {
	BlockHeight uint64     `json:"block_height"`
	GroupHeight uint64     `json:"group_height"`
	WorkGNum    int        `json:"work_g_num"`
	NodeInfo    *NodeInfo  `json:"node_info"`
	Conns       []ConnInfo `json:"conns"`
}

type ExplorerAccount struct {
	Balance   *big.Int               `json:"balance"`
	Nonce     uint64                 `json:"nonce"`
	Type      uint32                 `json:"type"`
	CodeHash  string                 `json:"code_hash"`
	Code      string                 `json:"code"`
	StateData map[string]interface{} `json:"state_data"`
}

type ExploreBlockReward struct {
	ProposalID           string            `json:"proposal_id"`
	ProposalReward       uint64            `json:"proposal_reward"`
	ProposalGasFeeReward uint64            `json:"proposal_gas_fee_reward"`
	VerifierReward       RewardTransaction `json:"verifier_reward"`
	VerifierGasFeeReward uint64            `json:"verifier_gas_fee_reward"`
}

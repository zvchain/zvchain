package models

//   Copyright (C) 2018 TASChain
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

import (
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/common"
	"time"
)

const (
	VoteStatusNotBegin = iota
	VoteStatusInProcess
	VoteStatusEnded
)

type PoolStake struct {
	gorm.Model
	Address string `json:"address" gorm:"index"`
	Stake   int64  `json:"stake" gorm:"index"`
	From    string `json:"from" gorm:"index"`
}

type MinerList struct {
	gorm.Model
	Address              string `json:"address" gorm:"unique_index"`
	ProposalConfirmCount uint64 `json:"proposal_confirm_count" gorm:"index;default:0"`
	VerifyConfirmCount   uint64 `json:"verify_confirm_count" gorm:"index;default:0"`
}
type AccountList struct {
	gorm.Model
	Address          string  `json:"address" gorm:"unique_index"`
	RoleType         uint64  `json:"role_type" gorm:"index;default:10"` // user default role_type value
	ProposalStake    uint64  `json:"proposal_stake" gorm:"index"`
	VerifyStake      uint64  `json:"verify_stake" gorm:"index"`
	TotalStake       uint64  `json:"total_stake" gorm:"index"`
	StakeToOther     uint64  `json:"stake_to_other" gorm:"index"`
	OtherStake       uint64  `json:"other_stake" gorm:"index"` // meams stake from other
	Group            string  `json:"group"`
	WorkGroup        uint64  `json:"work_group" gorm:"index"`
	DismissGroup     uint64  `json:"dismiss_group" gorm:"index"`
	PrepareGroup     uint64  `json:"prepare_group" gorm:"index"`
	TotalTransaction uint64  `json:"total_transaction"`
	Rewards          float64 `json:"rewards"`
	Status           int8    `json:"status" gorm:"index;default:-1"`
	VerifyStatus     int8    `json:"verify_status" gorm:"index;default:-1"`
	StakeFrom        string  `json:"stake_from"`
	Balance          float64 `json:"balance"`
	TotalBalance     float64 `json:"total_balance"`
	ExtraData        string  `json:"extra_data" gorm:"type:TEXT;size:65000"` // roletype extra data
	ProposalCount    uint64  `json:"proposal_count" gorm:"index;default:0"`
	VerifyCount      uint64  `json:"verify_count" gorm:"index;default:0"`

	ProposalFrozenStake uint64 `json:"proposal_frozen_stake"`
	VerifyFrozenStake   uint64 `json:"verify_frozen_stake"`
}

type RecentMineBlock struct {
	gorm.Model
	Address              string `json:"address" gorm:"unique_index"`
	RecentProposalBlocks string `json:"recent_proposal_blocks" gorm:"type:LONGTEXT"`
	RecentVerifyBlocks   string `json:"recent_verify_blocks" gorm:"type:LONGTEXT"`
}

type PoolExtraData struct {
	Vote uint64 `json:"vote"`
}

type ForkNotify struct {
	PreHeight   uint64
	LocalHeight uint64
}
type Sys struct {
	gorm.Model
	Variable string `json:"variable" gorm:"unique_index"`
	Value    uint64 `json:"value"`
	SetBy    string `json:"set_by"`
}

type ContractTransaction struct {
	gorm.Model
	ContractCode string `json:"contract_code" gorm:"index"`
	Address      string `json:"address"`
	Value        uint64 `json:"value"`
	TxHash       string `json:"tx_hash" gorm:"index"`
	TxType       uint64 `json:"tx_type"`
	Status       uint64 `json:"status"`
	BlockHeight  uint64 `json:"block_height"`
}

type ContractCallTransaction struct {
	gorm.Model
	ContractCode string     `json:"contract_code" gorm:"index"`
	TxHash       string     `json:"tx_hash" gorm:"index"`
	TxType       uint64     `json:"tx_type"`
	BlockHeight  uint64     `json:"block_height"`
	CurTime      *time.Time `json:"cur_time" gorm:"index"`
	Status       uint64     `json:"status"`
}

type TokenContract struct {
	gorm.Model
	ContractAddr  string `json:"contract_addr" gorm:"index"`
	Creator       string `json:"creator" gorm:"index"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	Decimal       int64  `json:"decimal"`
	HolderNum     uint64 `json:"holder_num"`
	TransferTimes uint64 `json:"transfer_times"`
}

type DaiPriceContract struct {
	gorm.Model
	TxHash           string    `json:"tx_hash" gorm:"unique_index"`
	Price            uint64    `json:"price"`
	LiquidationPrice uint64    `json:"liquidation_price"`
	Address          string    `json:"address" gorm:"index"`
	OrderId          uint64    `json:"order_id" gorm:"index"`
	ItemName         string    `json:"item_name"`
	Num              uint64    `json:"num"`
	Coin             string    `json:"coin"`
	Liquidation      uint64    `json:"liquidation"`
	RealLiquidation  uint64    `json:"real_liquidation"`
	CurTime          time.Time `json:"cur_time" gorm:"index"`
	Status           uint64    `json:"status"`
	Phone            string    `json:"phone"`
}
type PizzaswapContract struct {
	gorm.Model
	Pair        string `json:"pair" gorm:"unique_index"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Token0name  string `json:"token0name" `
	Token1name  string `json:"token1name" `
	Decimal0    uint64 `json:"decimal0"`
	Decimal1    uint64 `json:"decimal1"`
	Decimalpair uint64 `json:"decimalpair"`

	Creator string `json:"creator" `
	Status  uint64 `json:"status" `
}

type PizzaswapPool struct {
	gorm.Model
	Pair         string `json:"pair" gorm:"unique_index"`
	Token0       string `json:"token0"`
	Token1       string `json:"token1"`
	Token0name   string `json:"token0name" `
	Token1name   string `json:"token1name" `
	Decimalpair  uint64 `json:"decimalpair"`
	DecimalPizza uint64 `json:"decimalPizza"`

	PoolId  uint64 `json:"poolid"`
	Creator string `json:"creator" `
}

type TokenContractTransaction struct {
	gorm.Model
	ContractAddr string    `json:"contract_addr" gorm:"index"`
	Source       string    `json:"source" gorm:"index"`
	Target       string    `json:"target" gorm:"index"`
	Value        string    `json:"value"`
	TxHash       string    `json:"tx_hash" gorm:"index"`
	TxType       uint64    `json:"tx_type"`
	Status       uint64    `json:"status"`
	BlockHeight  uint64    `json:"block_height"`
	CurTime      time.Time `json:"cur_time" gorm:"index"`
}

type TokenContractUser struct {
	gorm.Model
	ContractAddr string `json:"contract_addr" gorm:"index"`
	Address      string `json:"address" gorm:"index"`
	Value        string `json:"value" gorm:"index"`
}

type Group struct {
	Id            string   `json:"id" gorm:"index"`
	Height        uint64   `json:"height" gorm:"index"`
	WorkHeight    uint64   `json:"work_height"`
	DismissHeight uint64   `json:"dismiss_height"`
	Threshold     uint64   `json:"threshold"`
	Members       []string `json:"members" gorm:"-"`
	MemberCount   uint64   `json:"member_count" `
	MembersStr    string   `json:"members_str"  gorm:"type:TEXT;size:65000"`
}

type Block struct {
	gorm.Model
	Height          uint64                 `json:"height" gorm:"index"`
	CurIndex        uint64                 `json:"cur_index" gorm:"index"`
	Hash            string                 `json:"hash" gorm:"unique_index"`
	PreHash         string                 `json:"pre_hash"`
	CurTime         time.Time              `json:"cur_time" gorm:"index"`
	PreTime         time.Time              `json:"pre_time"`
	Castor          string                 `json:"castor" gorm:"index"`
	GroupID         string                 `json:"group_id" gorm:"index"`
	TotalQN         uint64                 `json:"total_qn"`
	Qn              uint64                 `json:"qn"`
	TransCount      uint64                 `json:"trans_count"`
	RewardInfo      map[string]interface{} `json:"reward_info" gorm:"-"`
	LoadVerifyCount uint32                 `json:"load_verify_count"`
	LoadVerify      bool                   `json:"load_verify"`
	Random          string                 `json:"random"`
}

type Verification struct {
	gorm.Model
	BlockHash   string `json:"block_hash" gorm:"index"`
	BlockHeight uint64 `json:"block_height" gorm:"index"`
	NodeId      string `json:"node_id" gorm:"index"`
	Value       uint64 `json:"value"`
	Type        uint64 `json:"type"`
}

type Reward struct {
	gorm.Model
	BlockHash    string    `json:"block_hash" gorm:"unique_index:idx_reward"`
	BlockHeight  uint64    `json:"block_height" gorm:"index"`
	RewardHeight uint64    `json:"reward_height" gorm:"index"`
	CurTime      time.Time `json:"cur_time" gorm:"index"`
	NodeId       string    `json:"node_id" gorm:"unique_index:idx_reward"`
	Value        uint64    `json:"value"`
	Type         uint64    `json:"type" gorm:"unique_index:idx_reward"`
	Stake        uint64    `json:"stake"`
	RoleType     uint64    `json:"role_type" gorm:"index"`
	GasFee       float64   `json:"gas_fee" gorm:"index"`
}

type MinerToBlock struct {
	gorm.Model
	Type      uint64 `json:"type" gorm:"unique_index:idx_addr_seq" `
	Address   string `json:"address" gorm:"unique_index:idx_addr_seq"`
	BlockIDs  string `json:"block_ids" gorm:"type:TEXT"`
	BlockCnts int    `json:"block_cnts"`
	Sequence  uint64 `json:"sequence" gorm:"unique_index:idx_addr_seq"`
	Max       uint64 `json:"max" gorm:"index:idx_max"`
	Min       uint64 `json:"min" gorm:"index:idx_min"`
}

type BlockToMiner struct {
	gorm.Model
	BlockHeight      uint64    `json:"block_height" gorm:"unique_index"`
	BlockHash        string    `json:"block_hash" gorm:"unique_index"`
	RewardHeight     uint64    `json:"reward_height" gorm:"index"`
	CurTime          time.Time `json:"cur_time" gorm:"index"`
	PrpsNodeID       string    `json:"prps_node_id"`
	VerfNodeIDs      string    `json:"verf_node_ids" gorm:"type:TEXT"`
	VerfNodeCnts     uint64    `json:"verf_node_cnts"`
	PrpsReward       uint64    `json:"prps_reward"`
	VerfReward       uint64    `json:"verf_reward"`
	PrpsGasFee       uint64    `json:"prps_gas_fee"`
	VerfSingleGasFee float64   `json:"verf_single_gas_fee"`
	VerfTotalGasFee  uint64    `json:"verf_total_gas_fee"`
}

type TempTransaction struct {
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
	Status    uint   `json:"status"`
}

type TempDeployToken struct {
	gorm.Model
	TxHash string `json:"tx_hash" gorm:"index"`
}

type Transaction struct {
	gorm.Model
	BlockHash   string    `json:"block_hash" gorm:"index"`
	CurIndex    uint64    `json:"cur_index" gorm:"index"`
	BlockHeight uint64    `json:"block_height" gorm:"index"`
	Data        string    `json:"data" gorm:"type:TEXT;size:65000"`
	Value       float64   `json:"value"`
	Nonce       uint64    `json:"nonce"`
	Source      string    `json:"source" gorm:"index"`
	Target      string    `json:"target" gorm:"index:idx_transactions_target_type"`
	Type        int32     `json:"type" gorm:"index:idx_transactions_target_type"`
	CurTime     time.Time `json:"cur_time" gorm:"index"`

	GasLimit          uint64   `json:"gas_limit"`
	GasPrice          uint64   `json:"gas_price"`
	CumulativeGasUsed uint64   `json:"cumulative_gas_used"`
	Hash              string   `json:"hash" gorm:"unique_index"`
	Receipt           *Receipt `json:"receipt" gorm:"-"`
	ExtraData         string   `json:"extra_data" gorm:"type:TEXT;size:65000"`
	Status            uint     `json:"status" gorm:"index"`
	ContractAddress   string   `json:"contract_address" gorm:"index"`
}

type StakeMapping struct {
	gorm.Model
	Source       string `json:"source" gorm:"unique_index:idx_stakemapping_source_target"`
	Target       string `json:"target" gorm:"unique_index:idx_stakemapping_source_target"`
	PrpsActStake uint64 `json:"prps_act_stake" gorm:"index"`
	PrpsFrzStake uint64 `json:"prps_frz_stake" gorm:"index"`
	VerfActStake uint64 `json:"verf_act_stake" gorm:"index"`
	VerfFrzStake uint64 `json:"verf_frz_stake" gorm:"index"`
	//PrpsUpdtHeight uint64	`json:"prps_updt_height"`
	//VerfUpdtHeight uint64	`json:"verf_updt_height"`
}

type Receipt struct {
	Status            uint   `json:"status"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed"`
	Logs              []*Log `json:"logs" gorm:"-"`

	TxHash          string `json:"transactionHash" gorm:"unique_index"`
	ContractAddress string `json:"contractAddress" gorm:"index"`
	BlockHash       string `json:"block_hash" gorm:"index"`
	BlockHeight     uint64 `json:"block_height" gorm:"index"`
}

type Log struct {
	gorm.Model
	Address     string `json:"address" gorm:"index"`
	Topic       string `json:"topics"`
	Data        string `json:"data"`
	BlockNumber uint64 `json:"block_number" gorm:"unique_index:idx_log"`
	TxHash      string `json:"tx_hash"  gorm:"index"`
	TxIndex     uint   `json:"tx_index" gorm:"unique_index:idx_log"`
	BlockHash   string `json:"block_hash"`
	LogIndex    uint   `json:"log_index" gorm:"unique_index:idx_log"`
	Removed     bool   `json:"removed"`
}

type BlockDetail struct {
	Block
	Trans           []*TempTransaction `json:"trans"`
	Receipts        []*Receipt         `json:"receipts"`
	EvictedReceipts []*Receipt         `json:"evictedReceipts"`
}

type Config struct {
	gorm.Model
	Variable string `json:"variable" gorm:"unique_index"`
	Value    string `json:"value" gorm:"type:TEXT"`
	SetBy    string `json:"set_by"`
}

type BlockHeights []uint64

type BlockConfirmMiner struct {
	ProHeight uint64
	VerHight  uint64
}

type Vote struct {
	gorm.Model
	VoteId         uint64    `json:"vote_id" gorm:"unique"`
	Title          string    `json:"title" gorm:"index"`       // 投票标题
	Options        string    `json:"options" gorm:"type:TEXT"` // 选项列表 json string
	Intro          string    `json:"intro" gorm:"type:TEXT"`
	Promoter       string    `json:"promoter" gorm:"index"` // 发起人
	ContractAddr   string    `json:"contract_addr" gorm:"index"`
	StartTime      time.Time `json:"start_time" gorm:"index"`
	EndTime        time.Time `json:"end_time" gorm:"index"`
	StartTimeTS    int64     `json:"start_time_ts" gorm:"-"`
	EndTimeTS      int64     `json:"end_time_ts" gorm:"-"`
	OptionsCount   uint8     `json:"options_count"`
	Status         uint8     `json:"status" gorm:"index"` // 当前投票状态:0未开始，1进行中，2已结束
	OptionsDetails string    `json:"options_details" gorm:"type:TEXT"`
	Valid          bool      `json:"valid" gorm:"index"`
	Passed         bool      `json:"passed" gorm:"index"`
	GuardCount     uint64    `json:"guard_count"`
	TotalWeight    uint64    `json:"total_weight"`
}

type VoteDetails map[uint64]*VoteStat

type Voter struct {
	Addr   string `json:"addr"`
	Weight int64  `json:"weight"`
}

type VoteStat struct {
	Count       int     `json:"count"`
	TotalWeight int     `json:"total_weight"`
	Voter       []Voter `json:"voter"`
}

func (e BlockHeights) Len() int           { return len(e) }
func (e BlockHeights) Less(i, j int) bool { return e[i] > e[j] }
func (e BlockHeights) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

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

type PoolStake struct {
	gorm.Model
	Address string `json:"address" gorm:"index"`
	Stake   int64  `json:"stake" gorm:"index"`
	From    string `json:"from" gorm:"index"`
}

type AccountList struct {
	gorm.Model
	Address          string  `json:"address" gorm:"unique_index"`
	RoleType         uint64  `json:"role_type" gorm:"index;default:10"` // user default role_type value
	ProposalStake    uint64  `json:"proposal_stake" gorm:"index"`
	VerifyStake      uint64  `json:"verify_stake" gorm:"index"`
	TotalStake       uint64  `json:"total_stake" gorm:"index"`
	OtherStake       uint64  `json:"other_stake" gorm:"index"`
	Group            string  `json:"group"`
	WorkGroup        uint64  `json:"work_group" gorm:"index"`
	DismissGroup     uint64  `json:"dismiss_group" gorm:"index"`
	PrepareGroup     uint64  `json:"prepare_group" gorm:"index"`
	TotalTransaction uint64  `json:"total_transaction"`
	Rewards          uint64  `json:"rewards"`
	Status           byte    `json:"status" gorm:"index"`
	VerifyStatus     byte    `json:"verify_status" gorm:"index"`
	StakeFrom        string  `json:"stake_from"`
	Balance          float64 `json:"balance"`
	ExtraData        string  `json:"extra_data" gorm:"type:TEXT;size:65000"` // roletype extra data

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
	Variable string `json:"variable"`
	Value    uint64 `json:"value"`
	SetBy    string `json:"set_by"`
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
	Hash            string                 `json:"hash" gorm:"index"`
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
	BlockHash    string    `json:"block_hash" gorm:"index"`
	BlockHeight  uint64    `json:"block_height" gorm:"index"`
	RewardHeight uint64    `json:"reward_height" gorm:"index"`
	CurTime      time.Time `json:"cur_time" gorm:"index"`
	NodeId       string    `json:"node_id" gorm:"index:idx_user"`
	Value        uint64    `json:"value"`
	Type         uint64    `json:"type" gorm:"index:idx_user"`
	Stake        uint64    `json:"stake"`
	RoleType     uint64    `json:"role_type" gorm:"index"`
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

type Transaction struct {
	gorm.Model
	BlockHash   string    `json:"block_hash" gorm:"index"`
	BlockHeight uint64    `json:"block_height" gorm:"index"`
	Data        string    `json:"data" gorm:"type:TEXT;size:65000"`
	Value       float64   `json:"value"`
	Nonce       uint64    `json:"nonce"`
	Source      string    `json:"source" gorm:"index"`
	Target      string    `json:"target" gorm:"index"`
	Type        int32     `json:"type"`
	CurTime     time.Time `json:"cur_time" gorm:"index"`

	GasLimit  uint64   `json:"gas_limit"`
	GasPrice  uint64   `json:"gas_price"`
	Hash      string   `json:"hash" gorm:"index"`
	Receipt   *Receipt `json:"receipt" gorm:"-"`
	ExtraData string   `json:"extra_data" gorm:"type:TEXT;size:65000"`
	Status    uint     `json:"status" gorm:"index"`
}

type Receipt struct {
	Status            uint   `json:"status"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed"`
	Logs              []*Log `json:"logs" gorm:"-"`

	TxHash          string `json:"transactionHash" gorm:"index"`
	ContractAddress string `json:"contractAddress"`
	BlockHash       string `json:"block_hash" gorm:"index"`
	BlockHeight     uint64 `json:"block_height" gorm:"index"`
}

type Log struct {
	Address string `json:"address"`

	Topics []string `json:"topics" gorm:"-"`

	Data string `json:"data"`

	BlockNumber uint64 `json:"blockNumber"  gorm:"index"`

	TxHash string `json:"transactionHash"  gorm:"index"`

	TxIndex uint `json:"transactionIndex"  gorm:"index"`

	BlockHash string `json:"blockHash"  gorm:"index"`

	Index uint `json:"logIndex"`

	Removed bool `json:"removed"`
}
type BlockDetail struct {
	Block
	Trans           []*TempTransaction `json:"trans"`
	Receipts        []*Receipt         `json:"receipts"`
	EvictedReceipts []*Receipt         `json:"evictedReceipts"`
}

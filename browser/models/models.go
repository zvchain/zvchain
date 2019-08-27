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
)

type PoolStake struct {
	gorm.Model
	Address string `json:"address" gorm:"index"`
	Stake   int64  `json:"stake" gorm:"index"`
	From    string `json:"from" gorm:"index"`
}

type Account struct {
	gorm.Model
	Address          string `json:"address" gorm:"unique_index"`
	RoleType         uint64 `json:"role_type" gorm:"index"`
	ProposalStake    uint64 `json:"proposal_stake" gorm:"index"`
	VerifyStake      uint64 `json:"verify_stake" gorm:"index"`
	TotalStake       uint64 `json:"total_stake" gorm:"index"`
	OtherStake       uint64 `json:"other_stake" gorm:"index"`
	Group            string `json:"group"`
	WorkGroup        uint64 `json:"work_group" gorm:"index"`
	DismissGroup     uint64 `json:"dismiss_group" gorm:"index"`
	PrepareGroup     uint64 `json:"prepare_group" gorm:"index"`
	TotalTransaction uint64 `json:"total_transaction"`
	Rewards          uint64 `json:"rewards" gorm:"index"`
	Status           byte   `json:"status" gorm:"index"`
	StakeFrom        string `json:"stake_from"`
	Balance          float64 `json:"balance"`
	ExtraData        string `json:"extra_data" gorm:"type:TEXT;size:65000"` // roletype extra data

}

type PoolExtraData struct {
	Vote uint64 `json:"vote"`
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

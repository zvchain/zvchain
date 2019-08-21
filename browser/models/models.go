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

type Account struct {
	gorm.Model
	Address          string `json:"address" gorm:"unique_index"`
	RoleType         uint64 `json:"role_type" gorm:"index"`
	ProposalStake    uint64 `json:"proposal_stake"`
	VerifyStake      uint64 `json:"verify_stake"`
	TotalStake       uint64 `json:"total_stake" gorm:"index"`
	OtherStake       uint64 `json:"other_stake"`
	Group            string `json:"group"`
	TotalTransaction uint64 `json:"total_transaction"`
	Rewards          uint64 `json:"rewards" gorm:"index"`
	Status           string `json:"status" gorm:"index"`
	StakeFrom        string `json:"stake_from"`
	Balance          uint64 `json:"balance"`
}

type Sys struct {
	gorm.Model
	Variable string `json:"variable"`
	Value    uint64 `json:"value"`
	SetBy    string `json:"set_by"`
}

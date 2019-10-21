//   Copyright (C) 2019 ZVChain
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

package params

import (
	"github.com/zvchain/zvchain/common"
)

// ChainConfig defines the basic params of the chain
type ChainConfig struct {
	// Chain id identifies the current chain
	ChainId uint16

	// zip001 makes the block weight comparision more fair and random
	ZIP001 uint64

	// zip002 implements the gas price calculation when multiplying
	ZIP002 uint64
}

var config = &ChainConfig{
	ZIP001: 76228, // effect at : 2019-10-30 14:00:00
	ZIP002: 78228, // effect at : 2019-10-31 14:00:00
}

func InitChainConfig(chainId uint16) {
	config.ChainId = chainId
}

func GetChainConfig() *ChainConfig {
	return config
}

func (cfg *ChainConfig) IsMainNet() bool {
	return cfg.ChainId <= (common.MaxUint16 / 2)
}

func isFork(s, head uint64) bool {
	return s <= head
}

func (cfg *ChainConfig) IsZIP001(h uint64) bool {
	return isFork(cfg.ZIP001, h)
}

func (cfg *ChainConfig) IsZIP002(h uint64) bool {
	return isFork(cfg.ZIP002, h)
}

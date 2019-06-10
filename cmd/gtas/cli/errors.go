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

import "errors"

var (
	// ErrorBlockChainUninitialized means uninitialized blockchain
	ErrorBlockChainUninitialized = errors.New("should init blockchain module first")
	// ErrorP2PUninitialized means uninitialized P2P module
	ErrorP2PUninitialized = errors.New("should init P2P module first")
	// ErrorGovUninitialized means uninitialized consensus module
	ErrorGovUninitialized = errors.New("should init Governance module first")
	// ErrorWalletsUninitialized means uninitialized wallet
	ErrorWalletsUninitialized = errors.New("should load wallets from config")
)

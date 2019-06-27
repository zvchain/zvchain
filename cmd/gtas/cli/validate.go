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

package cli

import (
	"github.com/zvchain/zvchain/middleware/types"
	"regexp"
)

var addrReg = regexp.MustCompile("^0[xX][0-9a-fA-F]{64}$")
var hashReg = regexp.MustCompile("^0[xX][0-9a-fA-F]{64}$")
var keyReg = regexp.MustCompile("^0[xX][0-9a-fA-F]{1,64}$")

func validateAddress(addr string) bool {
	return addrReg.MatchString(addr)
}

func validateHash(hash string) bool {
	return hashReg.MatchString(hash)
}

func validateKey(key string) bool {
	return keyReg.MatchString(key)
}

func validateTxType(typ int) bool {
	return typ != types.TransactionTypeReward && typ >= types.TransactionTypeTransfer && typ <= types.TransactionTypeStakeRefund
}

func validateMinerType(typ int) bool {
	mType := types.MinerType(typ)
	return mType == types.MinerTypeVerify || mType == types.MinerTypeProposal
}

func validateStakeStatus(typ int) bool {
	st := types.StakeStatus(typ)
	return st != types.Staked && st != types.StakeFrozen
}

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

package core

import (
	"github.com/zvchain/zvchain/storage/account"
)

type verifier interface {
	VerifyIntegrity(cb account.VerifyAccountIntegrityCallback) (bool, error)
}

func (chain *FullBlockChain) Verifier(h uint64) verifier {
	db, err := chain.AccountDBAt(h)
	if err != nil {
		Logger.Errorf("get account db error at %v, err %v", h, err)
		return nil
	}
	return db.(*account.AccountDB)
}

func (chain *FullBlockChain) IntegrityVerify(height uint64, cb account.VerifyAccountIntegrityCallback) (bool, error) {
	v := chain.Verifier(height)
	if v != nil {
		return v.VerifyIntegrity(cb)
	}
	return true, nil
}

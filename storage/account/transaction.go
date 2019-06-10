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

package account

import (
	"math/big"

	"github.com/zvchain/zvchain/common"
)

type transitionEntry interface {
	undo(*AccountDB)
}

type transition []transitionEntry

type (
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *accountObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool
		prevbalance *big.Int
	}

	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account  *common.Address
		key      string
		prevalue []byte
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) undo(s *AccountDB) {
	s.accountObjects.Delete(*ch.account)
	delete(s.accountObjectsDirty, *ch.account)
}

func (ch resetObjectChange) undo(s *AccountDB) {
	s.setAccountObject(ch.prev)
}

func (ch suicideChange) undo(s *AccountDB) {
	obj := s.getAccountObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *AccountDB) {
	if !ch.prev && *ch.account != ripemd {
		s.getAccountObject(*ch.account).touched = ch.prev
		if !ch.prevDirty {
			delete(s.accountObjectsDirty, *ch.account)
		}
	}
}

func (ch balanceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setBalance(ch.prev)
}

func (ch nonceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setNonce(ch.prev)
}

func (ch codeChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch storageChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setData(ch.key, ch.prevalue)
}

func (ch refundChange) undo(s *AccountDB) {
	s.refund = ch.prev
}

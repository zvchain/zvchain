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

package account

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"math/big"
	"testing"
)

func TestAccountDB_Traverse(t *testing.T) {
	// Create an empty state database
	db, _ := tasdb.NewMemDatabase()
	state, _ := NewAccountDB(common.Hash{}, NewDatabase(db, false))

	//// Update it with some accounts
	for i := byte(0); i < 2; i++ {
		addr := common.BytesToAddress(common.Sha256([]byte{i}))
		state.AddBalance(addr, big.NewInt(int64(i)))
		fmt.Println(addr.Hash().Hex(), i)
	}

	root, _ := state.Commit(true)

	err := state.db.TrieDB().Commit(root, false)
	if err != nil {
		t.Fatal(err)
	}

	state2, err := NewAccountDB(root, NewDatabase(db, false))
	if err != nil {
		t.Fatal(err)
	}

	state2.Traverse(&TraverseConfig{
		VisitAccountCb: func(stat *TraverseStat) {
			t.Logf("visit %v, balance %v", stat.Addr.Hash().Hex(), stat.Account.Balance)
		},
	})
}

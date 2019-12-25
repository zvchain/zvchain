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
	"math/rand"
	"sync"
	"testing"
)

func TestAccountDB_Traverse(t *testing.T) {
	// Create an empty state database
	db, _ := tasdb.NewMemDatabase()
	state, _ := NewAccountDB(common.Hash{}, NewDatabase(db, false))

	input := make(map[common.Address]*big.Int)
	//// Update it with some accounts
	for i := 0; i < 10200; i++ {
		addr := common.BytesToAddress(common.Sha256([]byte{byte(i)}))
		balance := big.NewInt(rand.Int63())
		state.SetBalance(addr, balance)
		input[addr] = balance
		fmt.Println(addr.Hash().Hex(), balance)
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

	output := make(map[common.Address]*big.Int)
	mu := sync.Mutex{}
	state2.Traverse(&TraverseConfig{
		VisitAccountCb: func(stat *TraverseStat) {
			t.Logf("visit %v, balance %v", stat.Addr.Hash().Hex(), stat.Account.Balance)
			mu.Lock()
			output[stat.Addr] = stat.Account.Balance
			mu.Unlock()
		},
	})

	if len(input) != len(output) {
		t.Fatalf("len error")
	}
	for addr, balance := range input {
		balance2, ok := output[addr]
		if !ok {
			t.Fatalf("addr traverse error %v", addr.Hash().Hex())
		}
		if balance.Cmp(balance2) != 0 {
			t.Fatalf("addr traverse balance error %v %v %v", addr.Hash().Hex(), balance, balance2)
		}
	}
}

func TestSlice(t *testing.T) {
	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	fmt.Printf("%p %p %p %p\n", a, a[1:], a[2:], a[:2])

	a = append(a, 10)
	b := a[5:]
	b[0] = 444

	fmt.Printf("%p %v %v\n ", b, b, a)
}

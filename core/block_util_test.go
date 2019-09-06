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

package core

import (
	"testing"

	"github.com/zvchain/zvchain/middleware/types"
)

func TestCalTree(t *testing.T) {
	tx1 := getRandomTxs()
	tree1 := tx1.calcTxTree()
	if tree1.Hex() != "0x07b89743cdd3f5efa9ab10ceadeb6e590f3583a614b558ccce144dacfad49eae" {
		t.Errorf("mismatch, expect 0x07b89743cdd3f5efa9ab10ceadeb6e590f3583a614b558ccce144dacfad49eae but got %s ", tree1.Hex())
	}
}

func getRandomTxs() txSlice {
	result := make(txSlice, 0)
	var i uint64
	for i = 0; i < 100; i++ {
		tx := types.RawTransaction{Nonce: i, Value: types.NewBigInt(100 - i)}
		result = append(result, types.NewTransaction(&tx, tx.GenHash()))
	}
	return result
}

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
package group

import (
	"github.com/zvchain/zvchain/middleware/types"
	"testing"
)

func TestRevert(t *testing.T) {
	queue := make([]types.GroupI, 0)

	for i := 0; i < 10; i++ {
		ui := uint64(i)
		gl := newGroup4Test(ui)
		queue = append(queue,gl)
	}

	re := revert(queue)
	for ii, v := range re {
		if uint64(ii) != 9 - v.Header().WorkHeight() {
			t.Error("revert failed")
		}
	}
}


func newGroup4Test(height uint64) *group {
	header := &groupHeader{WorkHeightD:height}
	return &group{HeaderD:header}
}

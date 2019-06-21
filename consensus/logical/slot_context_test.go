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

package logical

import (
	"github.com/zvchain/zvchain/common"
	"gopkg.in/fatih/set.v0"
	"math/rand"
	"sync"
	"testing"
)

func TestSlotContext_addSignedTxHash_Concurrent(t *testing.T) {
	sc := &SlotContext{signedRewardTxHashs: set.New(set.ThreadSafe)}

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			bs := common.Int32ToByte(rand.Int31n(1000000000))
			h := common.BytesToHash(bs)
			sc.addSignedTxHash(h)
			wg.Done()
		}()
	}
	wg.Wait()
	t.Logf("finished, add size %v", sc.signedRewardTxHashs.Size())
}

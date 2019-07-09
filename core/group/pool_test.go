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
	"github.com/zvchain/zvchain/common"
	"sort"
	"strconv"
	"testing"
)

func TestShift(t *testing.T) {
	queue := make([]*groupLife, 0)

	for i:=0;i<10 ; i++ {
		ui := uint64(i)
		gl := &groupLife{common.HexToHash(strconv.Itoa(i)), ui, ui,ui}
		queue = push(queue,gl)
	}
	t.Log("init queue:")
	printQueue(t,queue)
	t.Log("after remove first:")
	queue = removeFirst(queue)
	printQueue(t,queue)
	t.Log("after remove last:")
	queue = removeLast(queue)
	printQueue(t,queue)
	t.Log("peek:")
	t.Log(peek(queue).height)
	t.Log("sPeek:")
	t.Log(sPeek(queue).height)

	t.Log("after sort:")
	sort.SliceStable(queue, func(i, j int) bool {
		return queue[i].end > queue[j].end
	})
	printQueue(t,queue)

}

func printQueue(t *testing.T,queue []*groupLife)  {
	for _, v := range queue {
		t.Log(v.height)
	}
}
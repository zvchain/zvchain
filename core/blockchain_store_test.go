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
	"fmt"
	"github.com/zvchain/zvchain/common/prque"
	"strconv"
	"testing"
)

func TestGetCropHeights(t *testing.T) {
	que := prque.NewPrque()

	chain := &FullBlockChain{
		triegc:prque.NewPrque(),
	}
	for i := 10; i >= 0; i-- {
		que.Push(strconv.Itoa(i), int64(i))
	}
	cropItems := chain.getPruneHeights(5, 2)
	if len(cropItems) != 3 {
		t.Fatalf("except len is 3,but got %v", len(cropItems))
	}
	if cropItems[0].Priority != 2 || cropItems[1].Priority != 1 || cropItems[2].Priority != 0 {
		t.Fatalf("value is error")
	}
	if que.Size() != 8 {
		t.Fatalf("expect len is 8,but got %v", que.Size())
	}
	_, ht := que.Pop()
	if uint64(ht) != 10 {
		t.Fatalf("expect data is 10,but got %v", ht)
	}
	chain = &FullBlockChain{
		triegc:prque.NewPrque(),
	}
	for i := 10; i >= 0; i-- {
		que.Push(strconv.Itoa(i), int64(i))
	}
	cropItems = chain.getPruneHeights(11, 2)
	if len(cropItems) > 0 {
		t.Fatalf("expect len is 0,but got %v", len(cropItems))
	}
	if que.Size() != 11 {
		t.Fatalf("expect len is 11,but got %v", que.Size())
	}
	_, ht = que.Pop()
	if uint64(ht) != 10 {
		t.Fatalf("expect data is 10,but got %v", ht)
	}
	chain = &FullBlockChain{
		triegc:prque.NewPrque(),
	}
	for i := 10; i >= 0; i-- {
		que.Push(strconv.Itoa(i), int64(i))
	}
	cropItems = chain.getPruneHeights(10, 9)
	if len(cropItems) != 1 {
		t.Fatalf("expect len is 1,but got %v", len(cropItems))
	}
	if cropItems[0].Priority != 0 {
		t.Fatalf("expect len is 0,but got %v", cropItems[0].Priority)
	}
	if que.Size() != 10 {
		t.Fatalf("expect len is 9,but got %v", que.Size())
	}
	_, ht = que.Pop()
	if uint64(ht) != 10 {
		t.Fatalf("expect data is 10,but got %v", ht)
	}
}

func TestFullBlockChain_QueryBlockFloor(t *testing.T) {
	initContext4Test(t)
	defer clearSelf(t)
	chain := BlockChainImpl

	fmt.Println("=====")
	bh := chain.queryBlockHeaderByHeight(0)
	fmt.Println(bh, bh.Hash.Hex())
	//top := gchain.latestBlock
	//t.Log(top.Height, top.Hash.String())
	//
	//for h := uint64(4460); h <= 4480; h++ {
	//	bh := gchain.queryBlockHeaderByHeightFloor(h)
	//	t.Log(bh.Height, bh.Hash.String())
	//}

	bh = chain.queryBlockHeaderByHeightFloor(0)
	fmt.Println(bh)
}

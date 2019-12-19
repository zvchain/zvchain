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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/common/prque"
	"math/big"
	"strconv"
	"testing"
)

func TestGetRepeatCropHeights(t *testing.T) {
	que := preparedData()
	chain := &FullBlockChain{
		triegc:que,
	}
	cropItems := chain.getPruneHeights(10, 9)
	if len(cropItems) != 0 {
		t.Fatalf("expect len is 0,but got %v", len(cropItems))
	}
	if que.Size() != 20 {
		t.Fatalf("expect len is 20,but got %v", que.Size())
	}
	_, ht := que.Pop()
	if uint64(ht) != 14 {
		t.Fatalf("expect data is 14,but got %v", ht)
	}

	que = preparedData()
	chain = &FullBlockChain{
		triegc:que,
	}
	cropItems = chain.getPruneHeights(10, 8)
	if len(cropItems) != 1 {
		t.Fatalf("expect len is 0,but got %v", len(cropItems))
	}
	if que.Size() != 19 {
		t.Fatalf("expect len is 19,but got %v", que.Size())
	}
	_, ht = que.Pop()
	if uint64(ht) != 14 {
		t.Fatalf("expect data is 14,but got %v", ht)
	}

	que = preparedData()
	chain = &FullBlockChain{
		triegc:que,
	}
	cropItems = chain.getPruneHeights(10, 5)
	if len(cropItems) != 7 {
		t.Fatalf("expect len is 7,but got %v", len(cropItems))
	}
	if que.Size() != 13 {
		t.Fatalf("expect len is 13,but got %v", que.Size())
	}
	_, ht = que.Pop()
	if uint64(ht) != 14 {
		t.Fatalf("expect data is 14,but got %v", ht)
	}

	var lastHt int64 = 0
	for !que.Empty() {
		_, ht = que.Pop()
		lastHt = ht
	}
	if uint64(lastHt) != 5 {
		t.Fatalf("expect data is 5,but got %v", ht)
	}
}

func BenchmarkGetRepeatCropHeights(b *testing.B) {
	que := prque.NewPrque()
	chain := &FullBlockChain{
		triegc:que,
	}
	maxI := 10000
	for i := maxI; i >= 0; i-- {
		que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(i+1))), int64(i+1))
	}
	for i := 0; i < b.N; i++ {
		chain.getPruneHeights(9000, 960)
	}

}

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



func preparedData() *prque.Prque {
	que := prque.NewPrque()
	que.Push("1", int64(1))
	que.Push("2", int64(2))

	que.Push("3", int64(3))
	que.Push("31", int64(3))
	que.Push("32", int64(3))

	que.Push("4", int64(4))
	que.Push("41", int64(4))

	que.Push("5", int64(5))
	que.Push("51", int64(5))

	que.Push("6", int64(6))
	que.Push("7", int64(7))
	que.Push("8", int64(8))
	que.Push("9", int64(9))

	que.Push("10", int64(10))
	que.Push("11", int64(11))
	que.Push("12", int64(12))

	que.Push("13", int64(13))
	que.Push("131", int64(13))

	que.Push("14", int64(14))
	que.Push("141", int64(14))
	return que
}
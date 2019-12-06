package prque

import (
	"github.com/zvchain/zvchain/common"
	"math/big"
	"strconv"
	"testing"
)

func TestGetCropHeights(t *testing.T) {
	que := NewPrque()
	for i := 10; i >= 0; i-- {
		que.Push(strconv.Itoa(i),-int64(i))
	}
	cropItems := que.GetCropHeights(5,2)
	if len(cropItems) != 3{
		t.Fatalf("except len is 3,but got %v",len(cropItems))
	}
	if cropItems[0].priority != 0 && cropItems[1].priority != -1 && cropItems[2].priority != -2{
		t.Fatalf("value is error")
	}
	if que.Size() != 8{
		t.Fatalf("expect len is 8,but got %v",que.Size())
	}
	_,ht := que.Pop()
	if uint64(-ht) != 10{
		t.Fatalf("expect data is 10,but got %v",ht)
	}
	que = NewPrque()
	for i := 10; i >= 0; i-- {
		que.Push(strconv.Itoa(i),-int64(i))
	}
	cropItems = que.GetCropHeights(11,2)
	if len(cropItems) > 0{
		t.Fatalf("expect len is 0,but got %v",len(cropItems))
	}
	if que.Size() != 11{
		t.Fatalf("expect len is 11,but got %v",que.Size())
	}
	_,ht = que.Pop()
	if uint64(-ht) != 10{
		t.Fatalf("expect data is 10,but got %v",ht)
	}
	que = NewPrque()
	for i := 10; i >= 0; i-- {
		que.Push(strconv.Itoa(i),-int64(i))
	}
	cropItems = que.GetCropHeights(10,9)
	if len(cropItems) != 1{
		t.Fatalf("expect len is 1,but got %v",len(cropItems))
	}
	if cropItems[0].priority !=0{
		t.Fatalf("expect len is 0,but got %v",cropItems[0].priority)
	}
	if que.Size() != 10{
		t.Fatalf("expect len is 9,but got %v",que.Size())
	}
	_,ht = que.Pop()
	if uint64(-ht) != 10{
		t.Fatalf("expect data is 10,but got %v",ht)
	}
}

func preparedData()*Prque{
	que := NewPrque()
	que.Push("1",-int64(1))
	que.Push("2",-int64(2))

	que.Push("3",-int64(3))
	que.Push("31",-int64(3))
	que.Push("32",-int64(3))

	que.Push("4",-int64(4))
	que.Push("41",-int64(4))

	que.Push("5",-int64(5))
	que.Push("51",-int64(5))

	que.Push("6",-int64(6))
	que.Push("7",-int64(7))
	que.Push("8",-int64(8))
	que.Push("9",-int64(9))


	que.Push("10",-int64(10))
	que.Push("11",-int64(11))
	que.Push("12",-int64(12))

	que.Push("13",-int64(13))
	que.Push("131",-int64(13))

	que.Push("14",-int64(14))
	que.Push("141",-int64(14))
	return que
}

func TestGetRepeatCropHeights(t *testing.T) {
	que := preparedData()
	cropItems := que.GetCropHeights(10,9)
	if len(cropItems) != 0{
		t.Fatalf("expect len is 0,but got %v",len(cropItems))
	}
	if que.Size() != 20{
		t.Fatalf("expect len is 20,but got %v",que.Size())
	}
	_,ht := que.Pop()
	if uint64(-ht) != 14{
		t.Fatalf("expect data is 14,but got %v",ht)
	}

	que = preparedData()
	cropItems = que.GetCropHeights(10,8)
	if len(cropItems) != 1{
		t.Fatalf("expect len is 0,but got %v",len(cropItems))
	}
	if que.Size() != 19{
		t.Fatalf("expect len is 19,but got %v",que.Size())
	}
	_,ht = que.Pop()
	if uint64(-ht) != 14{
		t.Fatalf("expect data is 14,but got %v",ht)
	}

	que = preparedData()
	cropItems = que.GetCropHeights(10,5)
	if len(cropItems) != 7{
		t.Fatalf("expect len is 7,but got %v",len(cropItems))
	}
	if que.Size() != 13{
		t.Fatalf("expect len is 13,but got %v",que.Size())
	}
	_,ht = que.Pop()
	if uint64(-ht) != 14{
		t.Fatalf("expect data is 14,but got %v",ht)
	}

	var lastHt int64 = 0
	for !que.Empty(){
		_,ht=que.Pop()
		lastHt = ht
	}
	if uint64(-lastHt) != 5{
		t.Fatalf("expect data is 5,but got %v",ht)
	}
}

func BenchmarkGetRepeatCropHeights(b *testing.B) {
	que := NewPrque()
	maxI := 10000
	for i := maxI; i >= 0; i-- {
		que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(i+1))), -int64((i + 1)))
	}
	for i := 0; i < b.N; i++ {
		que.GetCropHeights(9000,960)
	}


}

func TestPrque(t *testing.T) {
	que := NewPrque()

	maxI := 10000
	for i := maxI; i >= 0; i-- {
		que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(i+1))), -int64((i + 1)))
	}

	var (
		firstHash          = common.Hash{}
		firstHeight uint64 = 0

		lastHash          = common.Hash{}
		lastHeight uint64 = 0
	)

	for !que.Empty() {
		root, number := que.Pop()
		if firstHeight == 0 {
			firstHeight = uint64(-number)
			firstHash = root.(common.Hash)
		}
		lastHeight = uint64(-number)
		lastHash = root.(common.Hash)
	}


	if firstHash != common.BigToHash(new(big.Int).SetUint64(uint64(maxI+1))) {
		t.Fatalf("except hash is %v,but got %v", common.BigToHash(new(big.Int).SetUint64(uint64(maxI+1))), lastHash)
	}

	if firstHeight != uint64(maxI+1) {
		t.Fatalf("except height is %v,but got %v", maxI+1, lastHeight)
	}

	if lastHash != common.BigToHash(new(big.Int).SetUint64(uint64(1))) {
		t.Fatalf("except hash is %v,but got %v", common.BigToHash(new(big.Int).SetUint64(uint64(1))), firstHash)
	}

	if lastHeight != 1 {
		t.Fatalf("except height is 1,but got %v", firstHeight)
	}
}

func TestForkPrque(t *testing.T) {
	que := NewPrque()
	que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(1))), -int64((1)))
	que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(2))), -int64((1)))
	que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(2))), -int64((2)))

	_, number := que.Pop()

	if uint64(-number) != 2 {
		t.Fatalf("expect 2 ,but got %v", -number)
	}

	_, number = que.Pop()

	if uint64(-number) != 1 {
		t.Fatalf("expect 1 ,but got %v", -number)
	}

	_, number = que.Pop()

	if uint64(-number) != 1 {
		t.Fatalf("expect 1 ,but got %v", -number)
	}

}

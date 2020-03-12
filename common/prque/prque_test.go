package prque

import (
	"github.com/zvchain/zvchain/common"
	"math/big"
	"testing"
)

func TestPrque(t *testing.T) {
	que := NewPrque()

	maxI := 10000
	for i := maxI; i >= 0; i-- {
		que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(i+1))), int64(i+1))
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
			firstHeight = uint64(number)
			firstHash = root.(common.Hash)
		}
		lastHeight = uint64(number)
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
	que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(1))), int64(1))
	que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(2))), int64(1))
	que.Push(common.BigToHash(new(big.Int).SetUint64(uint64(2))), int64(2))

	_, number := que.Pop()

	if uint64(number) != 2 {
		t.Fatalf("expect 2 ,but got %v", number)
	}

	_, number = que.Pop()

	if uint64(number) != 1 {
		t.Fatalf("expect 1 ,but got %v", number)
	}

	_, number = que.Pop()

	if uint64(number) != 1 {
		t.Fatalf("expect 1 ,but got %v", number)
	}

}

package core

import (
	"github.com/zvchain/zvchain/common"
	"testing"
)

func TestMinerManager_MaxStake(t *testing.T) {
	mm := MinerManager{}
	maxs := []uint64{2500000, 4245283, 5803571, 7203389, 8467741, 9615384, 10661764, 10915492, 11148648, 11363636, 11562500,
		11746987, 11918604, 11797752, 11684782, 11578947, 11479591, 11386138, 11298076, 11098130, 10909090, 10730088,
		10560344, 10399159, 10245901}
	for i := 0; i <= 30; i++ {
		var cur = i
		if i >= len(maxs) {
			cur = len(maxs) - 1
		}
		max := mm.MaxStake(uint64(i * 5000000))
		if max != maxs[cur]*common.TAS {
			t.Errorf("max stake wanted:%d, got %d", maxs[cur]*common.TAS, max)
		}
	}
}

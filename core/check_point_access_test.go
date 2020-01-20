package core

import (
	"github.com/zvchain/zvchain/middleware/types"
	"testing"
)

var cp *checkPointAccess

func init() {
	cp = &checkPointAccess{}
}

func TestCheckPointAccess(t *testing.T) {
	t1 := cp.Load()
	if t1 != nil {
		t.Fatalf("expect nil,but got value")
	}
	bh := &types.BlockHeader{Height: 100}
	cp.Store(bh)
	t2 := cp.Load()
	if t2 == nil {
		t.Fatalf("expect not nil,but got nil")
	}
	if t2.Height != 100 {
		t.Fatalf("expect 100,but got %v", t2.Height)
	}
	cp.Reset()
	t3 := cp.Load()
	if t3 != nil {
		t.Fatalf("expect nil,but got value")
	}
}

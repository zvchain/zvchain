package core

import (
	"github.com/zvchain/zvchain/middleware/types"
	"sync/atomic"
	"unsafe"
)

type checkPoint struct {
	blockHeader *types.BlockHeader
}

func (c *checkPoint) Load() *types.BlockHeader {
	p := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.blockHeader)))
	if p != nil {
		return (*types.BlockHeader)(p)
	}
	return nil
}

func (c *checkPoint) Store(bh *types.BlockHeader) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.blockHeader)), unsafe.Pointer(bh))
}

func (c *checkPoint) Reset() {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.blockHeader)), unsafe.Pointer(nil))
}

package core

import (
	"github.com/zvchain/zvchain/middleware/types"
	"sync/atomic"
	"unsafe"
)

type checkPointAccess struct {
	blockHeader *types.BlockHeader
}

func initCheckPointAccess() *checkPointAccess {
	return &checkPointAccess{}
}

func (c *checkPointAccess) Load() *types.BlockHeader {
	p := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.blockHeader)))
	if p != nil {
		return (*types.BlockHeader)(p)
	}
	return nil
}

func (c *checkPointAccess) Store(bh *types.BlockHeader) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.blockHeader)), unsafe.Pointer(bh))
}

func (c *checkPointAccess) Reset() {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.blockHeader)), unsafe.Pointer(nil))
}

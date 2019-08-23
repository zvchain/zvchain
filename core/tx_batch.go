package core

import (
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/zvchain/zvchain/middleware/types"
)

type txBatchAdder struct {
	pool       types.TransactionPool
	routineNum int
	mu         sync.Mutex
}

func newTxBatchAdder(pool types.TransactionPool) *txBatchAdder {
	return &txBatchAdder{
		pool:       pool,
		routineNum: runtime.NumCPU(),
	}
}

func (tv *txBatchAdder) batchAdd(txs txSlice) error {
	if len(txs) == 0 {
		return nil
	}
	tv.mu.Lock()
	defer tv.mu.Unlock()
	wg := sync.WaitGroup{}
	step := int(math.Ceil(float64(len(txs)) / float64(tv.routineNum)))

	atomicErr := atomic.Value{}

	for begin := 0; begin < len(txs); {
		end := begin + step
		if end > len(txs) {
			end = len(txs)
		}
		copySlice := txs[begin:end]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, tx := range copySlice {
				if atomicErr.Load() != nil {
					return
				}
				if err := tv.pool.AsyncAddTransaction(tx); err != nil {
					atomicErr.Store(err)
					Logger.Warnf("batch add tx error:%v, tx hash %v, source %v", err, tx.Hash, tx.Source)
					return
				}
			}
		}()
		begin = end
	}
	wg.Wait()

	e := atomicErr.Load()
	if e != nil {
		return e.(error)
	}
	return nil
}

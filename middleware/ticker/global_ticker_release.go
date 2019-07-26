// +build release

package ticker

import (
	"runtime/debug"
	"sync/atomic"

	"github.com/zvchain/zvchain/log"
)

// trigger trigger an execution
func (gt *GlobalTicker) trigger(routine *TickerRoutine) bool {
	defer func() {
		if routine.rType == rTypeOneTime {
			gt.RemoveRoutine(routine.id)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			log.DefaultLogger.Errorf(string(s))
		}
	}()

	t := gt.ticker
	lastTicker := atomic.LoadUint64(&routine.lastTicker)

	if atomic.LoadInt32(&routine.status) != running {
		return false
	}

	b := false
	if lastTicker < t && atomic.CompareAndSwapUint64(&routine.lastTicker, lastTicker, t) {
		b = routine.handler()
	}
	return b
}

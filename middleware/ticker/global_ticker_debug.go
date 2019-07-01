// +build !release

package ticker

import (
	"sync/atomic"
)

// trigger trigger an execution
func (gt *GlobalTicker) trigger(routine *TickerRoutine) bool {

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
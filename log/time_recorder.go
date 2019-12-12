package log

import (
	"sync"
	"time"
)

var Recorder TimeRecorder

type TimeRecorder struct {
	m sync.Map
}

func (recorder *TimeRecorder) Start(id string) {
	recorder.m.Range(func(key, value interface{}) bool {
		t := value.(time.Time)
		if time.Since(t) > time.Minute*5 {
			recorder.m.Delete(key)
		}
		return true
	})
	recorder.m.Store(id, time.Now())
}

func (recorder *TimeRecorder) End(id string) int64 {
	v, ok := recorder.m.Load(id)
	if !ok {
		return 0
	} else {
		start := v.(time.Time)
		return time.Since(start).Nanoseconds() / 1e6
	}
}

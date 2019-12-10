package log

import "time"

var Recorder TimeRecorder

type TimeRecorder struct {
	t map[string]time.Time
}

func (recorder *TimeRecorder) Start(id string) {
	for id, t := range recorder.t {
		if time.Since(t) > time.Minute {
			delete(recorder.t, id)
		}
	}
	recorder.t[id] = time.Now()
}

func (recorder *TimeRecorder) End(id string) int64 {
	start, ok := recorder.t[id]
	if !ok {
		return 0
	} else {
		return time.Since(start).Milliseconds()
	}
}

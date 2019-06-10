package common

import (
	"fmt"
	"sync"
	"time"
)

type TimeMetric struct {
	Total int
	Cost  time.Duration
}
type TimeStatCtx struct {
	Stats map[string]*TimeMetric
	lock  sync.Mutex
}

func NewTimeStatCtx() *TimeStatCtx {
	return &TimeStatCtx{Stats: make(map[string]*TimeMetric)}
}

func (ts *TimeStatCtx) AddStat(name string, dur time.Duration) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	if v, ok := ts.Stats[name]; ok {
		v.Cost += dur
		v.Total++
	} else {
		tm := &TimeMetric{}
		tm.Total++
		tm.Cost += dur
		ts.Stats[name] = tm
	}
}

func (ts *TimeStatCtx) Output() string {
	s := ""
	var maxM *TimeMetric
	for key, v := range ts.Stats {
		if maxM == nil || v.Cost > maxM.Cost {
			maxM = v
		}
		s += fmt.Sprintf("%v %v\n", key, v.Cost.Seconds()/float64(v.Total))
	}
	for key, v := range ts.Stats {
		s += fmt.Sprintf("%v %v\t %v\n", key, v.Cost.Seconds()/float64(v.Total), v.Cost.Seconds()/maxM.Cost.Seconds())
	}
	return s
}

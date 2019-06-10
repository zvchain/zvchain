package concurrent

import (
	"fmt"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2019-05-29 13:11
**  Description:
 */

type TaskHandler func()

type ConcurrentTasks []TaskHandler

func NewConcurentTasks() ConcurrentTasks {
	return make(ConcurrentTasks, 0)
}

func (ctask ConcurrentTasks) ExecuteSync() {
	if len(ctask) == 0 {
		return
	}
	fmt.Println(len(ctask))
	wg := sync.WaitGroup{}
	for _, h := range ctask {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h()
		}()
	}
	wg.Wait()
}

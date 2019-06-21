package logical

import (
	"sync"
	"sync/atomic"
	"testing"
)

var data uint64
var wg sync.WaitGroup
func TestAtomic(t *testing.T) {
	wg.Add(10000)
	for i := 0;i<100;i++{
		go addI()
	}
	wg.Wait()
	if data != 10000{
		t.Fatalf("except 10000,but got %v",data)
	}
}

func addI(){
	for i := 0 ;i<100;i++{
		atomic.AddUint64(&data,1)
		wg.Done()
	}
}

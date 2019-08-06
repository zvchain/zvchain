package network

import (
	"fmt"
	"testing"
)

func TestHandleGroupGenConnectNodes(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}

	nodes := make([]NodeID, 0)



	g:=netCore.groupManager.buildGroup("test",nodes)
	g.genConnectNodes()

	fmt.Printf("nodes :%v",len(g.needConnectNodes))

}

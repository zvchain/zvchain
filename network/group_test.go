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

	g := netCore.groupManager.buildGroup("test", nodes)
	g.genConnectNodes()

	fmt.Printf("nodes :%v", len(g.needConnectNodes))

}

func TestHandleGroupGenConnectNodes9(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}

	nodes := []string{
		"0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019",
		"0xec6cddf1ee4d91832ad3fc658a1e4aae1741387b3c7bc176cb15399073454f23",
		"0xc9a2afdacceb7b2d0f50a19cd5bc2683f46e70b392cb66ddf759c3927fc591bf",
		"0x032f417a71b351cd4320a0a0859271ddbd257aa8ab22a2e35aeea2a02971bd8f",
		"0xa244d146ad492a4dacd2a1edd455292464e347687b331a1e5cd7484c8bec366c",
		"0x0553ad3d999a2439b12cf66d9822bcd0531ad9b29b6fc7e0b9b925a4e8720d33",
		"0xbb2a3d5b9dcec5bb4fbe251e29492eed2b0a813b9c69ab562c7a57bffe0da94b",
		"0xcc3532d62f9e88a5d0d4648a90c4248fb42eea81074d97f9abe94da83836ed33",
		netCore.ID.GetHexString()}
	g := netServerInstance.AddGroup("test", nodes)

	if g.sliceSize != 4 {
		t.Fatalf("sliceSize is not right")
	}

	if g.sliceCount != 3 {
		t.Fatalf("sliceCount is not right")
	}
	fmt.Printf("nodes :%v", len(g.needConnectNodes))

}

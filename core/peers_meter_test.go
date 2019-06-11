package core

import (
	"testing"
)

func TestPeerManager(t *testing.T) {
	initPeerManager()
	// add and hear
	peer := peerManagerImpl.getOrAddPeer("1")
	peerManagerImpl.heardFromPeer("1")

	if peer.isEvil(){
		t.Fatalf("should not be evil")
	}

	peer.increaseTimeout()
	peer.increaseTimeout()
	peer.increaseTimeout()
	peer.increaseTimeout()


	if !peer.isEvil(){
		t.Fatalf("should be evil")
	}

	peer.decreaseTimeout()

	if peer.isEvil(){
		t.Fatalf("should not be evil")
	}
}
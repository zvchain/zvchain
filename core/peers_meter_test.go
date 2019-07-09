package core

import (
	"github.com/zvchain/zvchain/common/secp256k1"
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
	peer.increaseTimeout()


	if !peer.isEvil(){
		t.Fatalf("should be evil")
	}

	if peer.evilCount != 1{
		t.Fatalf("evil count should be 1!")
	}

	if peer.timeoutMeter != 0{
		t.Fatalf("evil timeout meter shoud be 0")
	}

	peer.resetTimeoutMeter()

}

func returnErrHash()error{
	return ErrHash
}

func returnErrSign()error{
	return ErrSign
}

func returnErrDataSizeTooLong()error{
	return ErrDataSizeTooLong
}

func returnErrInvalidMsgLen()error{
	return secp256k1.ErrInvalidMsgLen
}

func returnErrRecoverFailed()error{
	return secp256k1.ErrRecoverFailed
}

func returnErrInvalidSignatureLen()error{
	return secp256k1.ErrInvalidSignatureLen
}

func returnErrInvalidRecoveryID()error{
	return secp256k1.ErrInvalidRecoveryID
}


func TestEvilError(t *testing.T){
	err := returnErrHash()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}

	err = returnErrSign()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}

	err = returnErrDataSizeTooLong()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}

	err = returnErrInvalidMsgLen()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}

	err = returnErrRecoverFailed()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}

	err = returnErrInvalidSignatureLen()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}
	err = returnErrInvalidRecoveryID()

	if _,ok  := evilErrorMap[err];!ok{
		t.Fatalf("this error should be in map")
	}

}
package common

import (
	"fmt"
	"testing"
)

func TestStringToAddress(t *testing.T) {
	rightAddr := "zvc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"
	shortAddr := "zvc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d810"
	longAddr := "zvc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d81033"
	longAddr2 := "zvc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d810333"
	addr := StringToAddress(rightAddr)
	if addr.AddrPrefixString() != rightAddr {
		t.Errorf("wanted: %s, got: %s", rightAddr, addr.AddrPrefixString())
	}
	addr = StringToAddress(shortAddr)
	if addr.AddrPrefixString() != "zv0c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d810" {
		t.Errorf("wanted: %s, got: %s", "zv0c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d810", addr.AddrPrefixString())
	}
	addr = StringToAddress(longAddr)
	if addr.AddrPrefixString() != "zv2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d81033" {
		t.Errorf("wanted: %s, got: %s", "zv2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d81033", addr.AddrPrefixString())
	}
	addr = StringToAddress(longAddr2)
	if addr.AddrPrefixString() != "zvf067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d810333" {
		t.Errorf("wanted: %s, got: %s", "zvf067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d810333", addr.AddrPrefixString())
	}
}

func TestSha256(t *testing.T) {
	fmt.Printf("result = %v \n", ToHex(Sha256([]byte("It is a test"))))
}

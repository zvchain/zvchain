package common

import (
	"testing"
)

func TestValidateAddress(t *testing.T) {
	wrongAddr := []string{"0xed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a",
		"0xed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8az",
		"0xed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8",
		"0xed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8z",
		"zved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8aa",
		"zved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8",
		"zved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8z",
		" zved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8",
		"z ved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8",
		"zved890e78fc5d07e85e66b7926d8370 095570abb5259e346438abd3ea7a56a8a",
	}
	rightAddr := []string{
		"zved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a",
		"zVed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a",
		"Zved890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a",
		"ZVed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a",
		"ZVed890e78fc5d07e85E66b7926d8370c095570abb5259e346438Abd3ea7a56a8a",
	}
	for _, addr := range wrongAddr {
		if ValidateAddress(addr) {
			t.Error("wanted false; got true!")
		}
	}
	for _, addr := range rightAddr {
		if !ValidateAddress(addr) {
			t.Error("wanted true; got false!")
		}
	}
}

package cli

import (
	"github.com/zvchain/zvchain/common"
	"testing"
)

func TestStringToUint(t *testing.T) {

	//case 0
	numb := ""
	result, _ := parseRaFromString(numb)
	if result != 0{
		t.Fatal("should be equal")
	}


	//case 1
	numb = "123"
	result, _ = parseRaFromString(numb)
	if result != 123*common.ZVC {
		t.Fatal("should be equal")
	}

	//case 2
	numb = "1230000000.2323"
	result, _ = parseRaFromString(numb)
	if result != 1230000000*common.ZVC+232300000 {
		t.Fatal("should be equal")
	}

	//case 3
	numb = "0.232111112"
	result, _ = parseRaFromString(numb)
	if result != 232111112 {
		t.Fatal("should be equal")
	}

	//case 4
	numb = ".232111112"
	_, err := parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 5
	numb = ".232111112"
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 6
	numb = "11."
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 7
	numb = "11111111111"
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 8
	numb = "1.1111111111"
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 9
	numb = "111111a.11111"
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 10
	numb = "11111.b11111"
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}

	//case 11
	numb = "-11111.1111"
	_, err = parseRaFromString(numb)
	if err == nil {
		t.Fatal("should be error")
	}
}

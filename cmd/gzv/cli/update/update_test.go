package update

import (
	"fmt"
	"testing"
)

func TestRequest(t *testing.T) {
	RequestUrl = "http://127.0.0.1:8000/request"
	vc := NewVersionChecker()
	no, err := vc.requestVersion()
	if err != nil {
		t.Error("err :", err)
	}
	fmt.Println("no ===>", no)
}

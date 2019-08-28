package common

import "testing"

func TestCheckWeakPassword(t *testing.T){
	CheckWeakPassword("sss   ...")
}

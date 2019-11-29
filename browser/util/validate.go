package util

import "regexp"

var addrReg = regexp.MustCompile("^[Zz][Vv][0-9a-fA-F]{64}$")

func ValidateAddress(addr string) bool {
	return addrReg.MatchString(addr)
}

package util

import "regexp"

var addrReg = regexp.MustCompile("^0[xX][0-9a-fA-F]{64}$")

func ValidateAddress(addr string) bool {
	return addrReg.MatchString(addr)
}

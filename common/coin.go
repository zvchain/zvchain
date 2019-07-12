//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/*
**  Creator: pxf
**  Date: 2019/1/8 下午3:33
**  Description:
 */

const (
	RA  uint64 = 1
	KRA        = 1000
	MRA        = 1000000
	ZVC        = 1000000000
)

var (
	ErrEmptyStr   = fmt.Errorf("empty string")
	ErrIllegalStr = fmt.Errorf("illegal gasprice string")
)

var re, _ = regexp.Compile("^([0-9]+)(ra|kra|mra|tas)$")

// ParseCoin parses string to amount
func ParseCoin(s string) (uint64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, ErrEmptyStr
	}

	arr := re.FindAllStringSubmatch(s, -1)
	if arr == nil || len(arr) == 0 {
		return 0, ErrIllegalStr
	}
	ret := arr[0]
	if ret == nil || len(ret) != 3 {
		return 0, ErrIllegalStr
	}
	num, err := strconv.Atoi(ret[1])
	if err != nil {
		return 0, err
	}
	unit := RA
	if len(ret) == 3 {
		switch ret[2] {
		case "kra":
			unit = KRA
		case "mra":
			unit = MRA
		case "zvc":
			unit = ZVC
		}
	}
	//fmt.Println(re.FindAllString(s, -1))
	return uint64(num) * unit, nil
}

func TAS2RA(v uint64) uint64 {
	return v * ZVC
}

func Value2RA(v float64) uint64 {
	return uint64(v * float64(ZVC))
}

func RA2TAS(v uint64) float64 {
	return float64(v) / float64(ZVC)
}

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

import "testing"

func runParseCoin(s string, expect uint64, t *testing.T) {
	v, err := ParseCoin(s)
	if err != nil {
		t.Fatal(err)
	}
	if v != expect {
		t.Errorf("parse coin error,input %v, expect %v", s, expect)
	}
}

func TestParseCoin_Correct(t *testing.T) {
	runParseCoin("232RA", 232, t)
	runParseCoin("232ra", 232, t)
	runParseCoin("232kra", 232000, t)
	runParseCoin("232mra", 232000000, t)
	runParseCoin("232tas", 232000000000, t)
}

func runParseCoinWrong(s string, t *testing.T) {
	_, err := ParseCoin(s)
	if err == nil {
		t.Fatalf("parse error string error: %v", s)
	}
}
func TestParseCoin_Wrong(t *testing.T) {
	runParseCoinWrong("232R", t)
	runParseCoinWrong("232a", t)
	runParseCoinWrong("", t)
	runParseCoinWrong("232", t)
	runParseCoinWrong("232 ZVC", t)
}

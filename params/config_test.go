//   Copyright (C) 2019 ZVChain
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

package params

import (
	"testing"
	"time"
)

func TestCalculateZIP001Height(t *testing.T) {
	begin := time.Date(2020, 3, 5, 18, 40, 43, 0, time.Local)
	beginHeight := uint64(4632366)
	effect := time.Date(2020, 3, 13, 14, 0, 0, 0, time.Local)
	t.Log(effect.String(), effect.UTC().String())

	seconds := effect.Local().Sub(begin.Local())
	seconds2 := effect.UTC().Sub(begin.UTC())
	//4857903  2.99
	//4857151 3
	//4858660 2.98
	t.Log(seconds, seconds2)
	if seconds2 != seconds {
		t.Fatalf("sub error")
	}

	blocksDelta := uint64(seconds.Seconds() / 2.98)

	zip001 := beginHeight + blocksDelta
	t.Log(zip001)
}

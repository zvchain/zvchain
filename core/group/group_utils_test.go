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

package group

import "testing"

func TestGetSeedHeight(t *testing.T) {
	seed := getSeedHeight(1234567)
	if seed != 1234500 {
		t.Errorf("mismatch, expect 1234500 but got get %d ", seed)
	}

	seed = getSeedHeight(1)
	if seed != 0 {
		t.Errorf("mismatch, expect 0 but got get %d ", seed)
	}

	seed = getSeedHeight(0)
	if seed != 0 {
		t.Errorf("mismatch, expect 0 but got get %d ", seed)
	}

	seed = getSeedHeight(300)
	if seed != 300 {
		t.Errorf("mismatch, expect 300 but got get %d ", seed)
	}


	seed = getSeedHeight(599)
	if seed != 300 {
		t.Errorf("mismatch, expect 300 but got get %d ", seed)
	}

	seed = getSeedHeight(610)
	if seed != 600 {
		t.Errorf("mismatch, expect 600 but got get %d ", seed)
	}
}

func TestIsHeightInRound(t *testing.T) {
	var currentHeight uint64 =  1399
	rs := isInRound(currentHeight, 1)
	if rs {
		t.Errorf("1399 should not in round 1")
	}

	rs = isInRound(currentHeight, 2)
	if !rs {
		t.Errorf("1399 should in round 2")
	}

	rs = isInRound(currentHeight, 3)
	if rs {
		t.Errorf("1399 should not in round 3")
	}

}

func TestGetRoundFirstBlockHeight(t *testing.T) {
	var currentHeight uint64 =  3
	rs := getRoundFirstBlockHeight(currentHeight,1)
	t.Log(rs)

	rs = getRoundFirstBlockHeight(currentHeight,2)
	t.Log(rs)

	rs = getRoundFirstBlockHeight(currentHeight,3)
	t.Log(rs)

	rs = getRoundFirstBlockHeight(currentHeight,4)
	t.Log(rs)

	rs = getRoundFirstBlockHeight(currentHeight,5)
	t.Log(rs)

}

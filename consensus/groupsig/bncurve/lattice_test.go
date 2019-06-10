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

package bncurve

import (
	"crypto/rand"

	"testing"
)

func TestLatticeReduceCurve(t *testing.T) {
	k, _ := rand.Int(rand.Reader, Order)
	ks := curveLattice.decompose(k)

	if ks[0].BitLen() > 130 || ks[1].BitLen() > 130 {
		t.Fatal("reduction too large")
	} else if ks[0].Sign() < 0 || ks[1].Sign() < 0 {
		t.Fatal("reduction must be positive")
	}
}

func TestLatticeReduceTarget(t *testing.T) {
	k, _ := rand.Int(rand.Reader, Order)
	ks := targetLattice.decompose(k)

	if ks[0].BitLen() > 66 || ks[1].BitLen() > 66 || ks[2].BitLen() > 66 || ks[3].BitLen() > 66 {
		t.Fatal("reduction too large")
	} else if ks[0].Sign() < 0 || ks[1].Sign() < 0 || ks[2].Sign() < 0 || ks[3].Sign() < 0 {
		t.Fatal("reduction must be positive")
	}
}

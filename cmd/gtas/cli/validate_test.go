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

package cli

import "testing"

func TestValidateAddress(t *testing.T) {
	if validateAddress("0x123") {
		t.Errorf("length error")
	}
	if validateAddress("0xiop1222222223333333333333232322222222222222222222222222222222222") {
		t.Errorf("wrong letters")
	}
	if validateAddress("0x33333333333333333333333333333333333333333333333333333333333333333333333333") {
		t.Errorf("too long")
	}
	if !validateAddress("0x1231222222223333333333333232322222222222222222222222222222222222") {
		t.Errorf("correct")
	}
}

func TestValidateHash(t *testing.T) {
	if validateHash("0x123") {
		t.Errorf("length error")
	}
	if validateHash("0xiop1222222223333333333333232322222222222222222222222222222222222") {
		t.Errorf("wrong letters")
	}
	if validateHash("0x33333333333333333333333333333333333333333333333333333333333333333333333333") {
		t.Errorf("too long")
	}
	if !validateHash("0x1231222222223333333333333232322222222222222222222222222222222222") {
		t.Errorf("correct")
	}
}

func TestValidateTxType(t *testing.T) {
	if !validateTxType(1) {
		t.Errorf("validate error type 1")
	}
	if validateTxType(3) {
		t.Errorf("validate error type 3")
	}
	if validateTxType(100) {
		t.Errorf("validate error type 100")
	}
}

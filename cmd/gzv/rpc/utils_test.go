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

package rpc

import (
	"github.com/zvchain/zvchain/common"
	"regexp"
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	hexchars := "0123456789ABCDEFabcdef"
	for i := 0; i < 100; i++ {
		id := string(NewID())
		if !strings.HasPrefix(id, "0x") {
			t.Fatalf("invalid ID prefix, want '0x...', got %s", id)
		}

		id = id[2:]
		if len(id) == 0 || len(id) > 32 {
			t.Fatalf("invalid ID length, want len(id) > 0 && len(id) <= 32), got %d", len(id))
		}

		for i := 0; i < len(id); i++ {
			if strings.IndexByte(hexchars, id[i]) == -1 {
				t.Fatalf("unexpected byte, want any valid hex char, got %c", id[i])
			}
		}
	}
}

func TestAddressRegexMatch(t *testing.T) {
	reg := regexp.MustCompile("^0[xX][0-9a-fA-F]{64}$")
	b := reg.MatchString("123")
	if b {
		t.Errorf("match error of 123")
	}

	addr := common.StringToAddress("zv123")
	b = reg.MatchString(addr.AddrPrefixString())

	b = reg.MatchString("0x0123000000000000000000000000000000000000000000000000000000000000")
	if !b {
		t.Errorf("match error")
	}

	b = reg.MatchString("0x012300000000000000000000000000000000000000000000000000000000000")
	if b {
		t.Errorf("match error: length not enough")
	}
	b = reg.MatchString("0x01230000000000000000000000000000000000000000000000000000000000001")
	if b {
		t.Errorf("match error: length too long")
	}
	b = reg.MatchString("0x012300000000000000000000000000000000000000000000000000000000000I")
	if b {
		t.Errorf("match error: wrong letter")
	}
	b = reg.MatchString("0123000000000000000000000000000000000000000000000000000000000001")
	if b {
		t.Errorf("match error: no prefix")
	}
	b = reg.MatchString("a0x0123000000000000000000000000000000000000000000000000000000000000d")
	if b {
		t.Errorf("match error: begin error")
	}
	b = reg.MatchString("0x0123000000000000000000000000000000000000000000000000000006594Aef")
	if !b {
		t.Errorf("match error: right format")
	}
	b = reg.MatchString("0x01230000000000G000000000DD00000B00000000000000000000000006594Aef")
	if b {
		t.Errorf("match error: wrong letter2")
	}
}

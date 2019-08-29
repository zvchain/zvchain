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

package types

import "testing"

func TestEpochAt(t *testing.T) {
	h := uint64(0)
	ep := EpochAt(h)
	if ep.Start() != 0 {
		t.Errorf("epoch at error:%v", h)
	}
	t.Log(ep.Start(), ep.End(), ep)
	h = 100
	ep = EpochAt(h)
	t.Log(ep.Start(), ep.End(), ep)
	if ep.Start() != 0 {
		t.Errorf("epoch at error:%v", h)
	}

	h = EpochLength
	ep = EpochAt(h)
	if ep.Start() != EpochLength {
		t.Errorf("epoch at error:%v", h)
	}

	h = EpochLength + 443
	ep = EpochAt(h)
	t.Log(ep.Start(), ep.End(), ep)
	if ep.Start() != EpochLength {
		t.Errorf("epoch at error:%v", h)
	}
}

func TestEpoch_Next(t *testing.T) {
	var ep Epoch = genesisEpoch{}
	cnt := 0
	for {
		t.Log(ep.Start(), ep.End())
		if cnt > 100 {
			break
		}
		cnt++
		nxt := ep.Next()
		if nxt.Start() > 0 && nxt.Start()-ep.Start() != EpochLength {
			t.Errorf("epoch error:%v %v", ep.Start(), nxt.Start())
		}
		ep = nxt
	}
}

func TestEpoch_Prev(t *testing.T) {
	ep := EpochAt(599999)
	cnt := 0
	for {
		t.Log(ep.Start(), ep.End())
		if cnt > 100 {
			break
		}
		cnt++
		prv := ep.Prev()
		if prv.Start() > 0 && ep.Start()-prv.Start() != EpochLength {
			t.Errorf("epoch error:%v %v", ep.Start(), prv.Start())
		}
		ep = prv
	}
}

func TestEpoch_Add_Positive(t *testing.T) {
	ep := EpochAt(4544545453)
	add0 := ep.Add(0)
	t.Log("add0", add0.Start(), add0.End())
	if add0.Start() != ep.Start() {
		t.Errorf("add epoch error")
	}

	add1 := ep.Add(1)
	t.Log("add1", add1.Start(), add1.End())
	if add1.Start() != ep.Start()+EpochLength {
		t.Errorf("add epoch error")
	}

	add100 := ep.Add(100)
	t.Log("add100", add100.Start(), add100.End())
	if add100.Start() != ep.Start()+100*EpochLength {
		t.Errorf("add epoch error")
	}
}

func TestEpoch_Add_Negative(t *testing.T) {
	ep := EpochAt(1)
	sub1 := ep.Add(-1)
	t.Log("sub1", sub1.Start(), sub1.End())
	if _, ok := sub1.(*genesisEpoch); !ok {
		t.Errorf("sub error %v", -1)
	}

	sub100 := ep.Add(-100)
	t.Log("sub100", sub100.Start(), sub100.End())
	if _, ok := sub100.(*genesisEpoch); !ok {
		t.Errorf("sub error %v", -100)
	}

	ep = EpochAt(344355)
	t.Log(ep.Start(), ep.End())
	for i := uint64(0); ; i++ {
		sub := ep.Add(int(-i))
		t.Log("sub", i, sub.Start(), sub.End())
		if _, ok := sub.(*genesisEpoch); ok {
			break
		}
		if sub.Start() != ep.Start()-i*EpochLength {
			t.Errorf("sub error %v", i)
		}
	}

}

func TestCreateEpochsOfActivatedGroupsAt(t *testing.T) {
	s, e := CreateEpochsOfActivatedGroupsAt(1)
	t.Logf("start %v-%v, end %v-%v", s.Start(), s.End(), e.Start(), e.End())
	if s.Start() != 0 || e.Start() != 0 {
		t.Errorf("error")
	}

	s, e = CreateEpochsOfActivatedGroupsAt(8000)
	t.Logf("start %v-%v, end %v-%v", s.Start(), s.End(), e.Start(), e.End())
	if s.Start() != 0 || e.Start() != 0 {
		t.Errorf("error")
	}
	s, e = CreateEpochsOfActivatedGroupsAt(16000)
	t.Logf("start %v-%v, end %v-%v", s.Start(), s.End(), e.Start(), e.End())
	if s.Start() != 0 || e.Start() != 8000 {
		t.Errorf("error")
	}
	s, e = CreateEpochsOfActivatedGroupsAt(24000)
	t.Logf("start %v-%v, end %v-%v", s.Start(), s.End(), e.Start(), e.End())
	if s.Start() != 0 || e.Start() != 16000 {
		t.Errorf("error")
	}
	s, e = CreateEpochsOfActivatedGroupsAt(32000)
	t.Logf("start %v-%v, end %v-%v", s.Start(), s.End(), e.Start(), e.End())
	if s.Start() != 8000 || e.Start() != 24000 {
		t.Errorf("error")
	}
}

func TestActivateEpochOfGroupsCreatedAt(t *testing.T) {
	ep := ActivateEpochOfGroupsCreatedAt(0)
	if ep.Start() != 16000 {
		t.Errorf("error")
	}
	ep = ActivateEpochOfGroupsCreatedAt(234)
	if ep.Start() != 16000 {
		t.Errorf("error")
	}
	ep = ActivateEpochOfGroupsCreatedAt(8000)
	if ep.Start() != 24000 {
		t.Errorf("error")
	}
	ep = ActivateEpochOfGroupsCreatedAt(17000)
	if ep.Start() != 32000 {
		t.Errorf("error")
	}
}

func TestDismissEpochOfGroupsCreatedAt(t *testing.T) {
	ep := DismissEpochOfGroupsCreatedAt(0)
	if ep.Start() != 32000 {
		t.Errorf("dismiss error")
	}
	ep = DismissEpochOfGroupsCreatedAt(8000)
	if ep.Start() != 8000+EpochLength*4 {
		t.Errorf("dismiss error")
	}
	ep = DismissEpochOfGroupsCreatedAt(40000)
	if ep.Start() != 40000+EpochLength*4 {
		t.Errorf("dismiss error")
	}
}

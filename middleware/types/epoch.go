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

type Epoch interface {
	Start() uint64
	End() uint64
	Next() Epoch
	Prev() Epoch
	Add(delta int) Epoch
}

type GroupEpochAlg interface {
	// createEpochByHeight returns the creating epoch ranges of the groups at the given block height
	CreateEpochByHeight(h uint64) (start, end Epoch)
}

const EpochLength = 8000 // blocks per epoch

type epoch uint64

func (e epoch) Start() uint64 {
	return uint64(e) * EpochLength
}

func (e epoch) End() uint64 {
	return e.Start() + EpochLength
}
func (e epoch) Prev() Epoch {
	if e.Start() < 1 {
		return genesisEpoch{}
	}
	return EpochAt(e.Start() - 1)
}
func (e epoch) Next() Epoch {
	return EpochAt(e.End() + 1)
}

func (e epoch) Add(delta int) Epoch {
	if delta >= 0 {
		return EpochAt(e.Start() + uint64(delta)*EpochLength)
	}
	delta = -delta
	if e.Start() >= uint64(delta)*EpochLength {
		return EpochAt(e.Start() - uint64(delta)*EpochLength)
	}
	return &genesisEpoch{}
}

type genesisEpoch struct{}

func (ge genesisEpoch) Start() uint64 {
	return 0
}

func (ge genesisEpoch) End() uint64 {
	return 0
}

func (ge genesisEpoch) Prev() Epoch {
	return ge
}

func (ge genesisEpoch) Next() Epoch {
	return EpochAt(1)
}

func (ge genesisEpoch) Add(delta int) Epoch {
	if delta <= 0 {
		return ge
	}
	if delta == 1 {
		return EpochAt(0)
	}
	return EpochAt(ge.Start() + uint64(delta)*EpochLength)
}

func EpochAt(h uint64) Epoch {
	return epoch(h / EpochLength)
}

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

package logical

import (
	"github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
)

type proveChecker struct {
	proposalVrfHashs *lru.Cache // Recently proposed vrf prove hash
}


func newProveChecker() *proveChecker {
	return &proveChecker{
		proposalVrfHashs: common.MustNewLRUCache(50),
	}
}

func (p *proveChecker) proveExists(pi base.VRFProve) bool {
	hash := common.BytesToHash(base.VRFProof2hash(pi))
	_, ok := p.proposalVrfHashs.Get(hash)
	return ok
}

func (p *proveChecker) addProve(pi base.VRFProve) {
	hash := common.BytesToHash(base.VRFProof2hash(pi))
	p.proposalVrfHashs.Add(hash, 1)
}

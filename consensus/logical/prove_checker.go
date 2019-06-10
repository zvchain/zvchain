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
	"bytes"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
)

type proveChecker struct {
	proposalVrfHashs *lru.Cache // Recently proposed vrf prove hash
	proveRootCaches  *lru.Cache // Full account verification cache
	chain            core.BlockChain
}

type prootCheckResult struct {
	ok  bool
	err error
}

func newProveChecker(chain core.BlockChain) *proveChecker {
	return &proveChecker{
		proveRootCaches:  common.MustNewLRUCache(50),
		proposalVrfHashs: common.MustNewLRUCache(50),
		chain:            chain,
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

func (p *proveChecker) genVerifyHash(b []byte, id groupsig.ID) common.Hash {
	buf := bytes.NewBuffer([]byte{})
	if b != nil {
		buf.Write(b)
	}
	buf.Write(id.Serialize())

	h := base.Data2CommonHash(buf.Bytes())
	return h
}

// sampleBlockHeight performs block sampling on the id
func (p *proveChecker) sampleBlockHeight(heightLimit uint64, rand []byte, id groupsig.ID) uint64 {
	// Randomly extract the blocks before 10 blocks to ensure that
	// the blocks on the forks are not extracted.
	if heightLimit > 2*model.Param.Epoch {
		heightLimit -= 2 * model.Param.Epoch
	}
	return base.RandFromBytes(rand).DerivedRand(id.Serialize()).ModuloUint64(heightLimit)
}

func (p *proveChecker) genProveHash(heightLimit uint64, rand []byte, id groupsig.ID) common.Hash {
	h := p.sampleBlockHeight(heightLimit, rand, id)
	bs := p.chain.QueryBlockBytesFloor(h)
	hash := p.genVerifyHash(bs, id)

	return hash
}

func (p *proveChecker) genProveHashs(heightLimit uint64, rand []byte, ids []groupsig.ID) (proves []common.Hash) {
	hashs := make([]common.Hash, len(ids))

	for idx, id := range ids {
		hashs[idx] = p.genProveHash(heightLimit, rand, id)
	}
	proves = hashs

	return
}

func (p *proveChecker) addPRootResult(hash common.Hash, ok bool, err error) {
	p.proveRootCaches.Add(hash, &prootCheckResult{ok: ok, err: err})
}

func (p *proveChecker) getPRootResult(hash common.Hash) (exist bool, result bool, err error) {
	v, ok := p.proveRootCaches.Get(hash)
	if ok {
		r := v.(*prootCheckResult)
		return true, r.ok, r.err
	}
	return false, false, nil
}

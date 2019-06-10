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
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync/atomic"

	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
)

// defines the status of vrfWorker
const (
	prove    int32 = 0
	proposed       = 1
	success        = 2
)

var max256 *big.Rat
var rat1 *big.Rat

func init() {
	t := new(big.Int)
	t.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	max256 = new(big.Rat).SetInt(t)
	rat1 = new(big.Rat).SetInt64(1)
}

// vrfWorker do some vrf calculations during block proposal to check if the specified miner
// satisfied the propose-condition
type vrfWorker struct {
	//read only
	miner      *model.SelfMinerDO // Miner info
	baseBH     *types.BlockHeader // The block the proposal process based on
	castHeight uint64             // The height of the block to be proposed
	expire     time.TimeStamp     // The worker process deadline
	//writable
	status int32
	ts     time.TimeService
}

func newVRFWorker(miner *model.SelfMinerDO, bh *types.BlockHeader, castHeight uint64, expire time.TimeStamp, ts time.TimeService) *vrfWorker {
	return &vrfWorker{
		miner:      miner,
		baseBH:     bh,
		castHeight: castHeight,
		expire:     expire,
		status:     prove,
		ts:         ts,
	}
}

func (vrf *vrfWorker) m() []byte {
	h := vrf.castHeight - vrf.baseBH.Height
	return vrfM(vrf.baseBH.Random, h)
}

func vrfM(random []byte, h uint64) []byte {
	if h <= 0 {
		panic(fmt.Sprintf("vrf height error! deltaHeight=%v", h))
	}
	data := random
	for h > 1 {
		h--
		hash := base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	return data
}

// Prove generates VRFProve and corresponding qn for block proposal with given total stake
// which is read from chain
func (vrf *vrfWorker) Prove(totalStake uint64) (base.VRFProve, uint64, error) {
	pi, err := base.VRFGenerateProve(vrf.miner.VrfPK, vrf.miner.VrfSK, vrf.m())
	if err != nil {
		return nil, 0, err
	}
	if ok, qn := vrfSatisfy(pi, vrf.miner.Stake, totalStake); ok {

		return pi, qn, nil
	}
	return nil, 0, errors.New("proof fail")
}

func vrfThreshold(stake, totalStake uint64) *big.Rat {
	brTStake := new(big.Rat).SetFloat64(float64(totalStake))
	return new(big.Rat).Quo(new(big.Rat).SetInt64(int64(stake*uint64(model.Param.PotentialProposal))), brTStake)
}

func vrfSatisfy(pi base.VRFProve, stake uint64, totalStake uint64) (ok bool, qn uint64) {
	if totalStake == 0 {
		stdLogger.Errorf("total stake is 0!")
		return false, 0
	}
	value := base.VRFProof2hash(pi)

	br := new(big.Rat).SetInt(new(big.Int).SetBytes(value))
	pr := br.Quo(br, max256)

	vs := vrfThreshold(stake, totalStake)

	ok = pr.Cmp(vs) < 0
	// Calculate qn
	if vs.Cmp(rat1) > 0 {
		vs.Set(rat1)
	}

	step := vs.Quo(vs, new(big.Rat).SetInt64(int64(model.Param.MaxQN)))

	r, _ := pr.Quo(pr, step).Float64()
	qn = uint64(math.Floor(r) + 1)

	return
}

// vrfVerifyBlock verifies if the vrf prove of the given block is legal
func vrfVerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader, miner *model.MinerDO, totalStake uint64) (bool, error) {
	pi := base.VRFProve(bh.ProveValue)
	ok, err := base.VRFVerify(miner.VrfPK, pi, vrfM(preBH.Random, bh.Height-preBH.Height))
	if !ok {
		return ok, err
	}
	if ok, qn := vrfSatisfy(pi, miner.Stake, totalStake); ok {
		if bh.TotalQN != qn+preBH.TotalQN {
			return false, fmt.Errorf("qn error.bh hash=%v, height=%v, qn=%v,totalQN=%v, preBH totalQN=%v", bh.Hash.ShortS(), bh.Height, qn, bh.TotalQN, preBH.TotalQN)
		}
		return true, nil
	}
	return false, errors.New("proof not satisfy")
}

func (vrf *vrfWorker) markProposed() {
	atomic.CompareAndSwapInt32(&vrf.status, prove, proposed)
}

func (vrf *vrfWorker) markSuccess() {
	atomic.CompareAndSwapInt32(&vrf.status, proposed, success)
}

func (vrf *vrfWorker) getBaseBH() *types.BlockHeader {
	return vrf.baseBH
}

func (vrf *vrfWorker) isSuccess() bool {
	return vrf.getStatus() == success
}

func (vrf *vrfWorker) isProposed() bool {
	return vrf.getStatus() == proposed
}

func (vrf *vrfWorker) getStatus() int32 {
	return atomic.LoadInt32(&vrf.status)
}

func (vrf *vrfWorker) workingOn(bh *types.BlockHeader, castHeight uint64) bool {
	return bh.Hash == vrf.baseBH.Hash && castHeight == vrf.castHeight && !vrf.timeout()
}

func (vrf *vrfWorker) timeout() bool {
	return vrf.ts.NowAfter(vrf.expire)
}

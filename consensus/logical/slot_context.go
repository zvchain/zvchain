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
	"fmt"
	"sync/atomic"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"gopkg.in/fatih/set.v0"
)

const (
	slIniting int32 = iota

	// slWaiting waiting for the signature fragment to reach the threshold
	slWaiting

	// slSigned indicate whether you have signed
	slSigned

	// slRecovered recover verifyGroup signature
	slRecovered

	// slVerified indicate verifyGroup signature is verified by verifyGroup public key
	slVerified

	// slSuccess indicate already on the chain and broadcast
	slSuccess

	// slFailed failure in the process of cast block, irreversible
	slFailed

	// slRewardSignReq indicate reward transaction signature request has been sent
	slRewardSignReq

	// slRewardSent indicate reward transaction has been broadcast
	slRewardSent
)

// SlotContext stores the contextual infos of a specified block proposal
type SlotContext struct {

	// Verification related
	BH             *types.BlockHeader        // Detailed data in block header
	gSignGenerator *model.GroupSignGenerator // Block signature generator
	rSignGenerator *model.GroupSignGenerator // Random number signature generator
	slotStatus     int32                     // The current status

	castor groupsig.ID        // The proposal miner id
	pSign  groupsig.Signature // The proposer signature

	// Reward related
	rewardTrans    *types.Transaction        // The reward transaction related to the block, it should issue after the block broadcast
	rewardGSignGen *model.GroupSignGenerator // Reward transaction signature generator

	signedRewardTxHashs set.Interface // Signed reward transaction hash
}

func createSlotContext(bh *types.BlockHeader, threshold int) *SlotContext {
	return &SlotContext{
		BH:                  bh,
		castor:              groupsig.DeserializeID(bh.Castor),
		pSign:               *groupsig.DeserializeSign(bh.Signature),
		slotStatus:          slWaiting,
		gSignGenerator:      model.NewGroupSignGenerator(threshold),
		rSignGenerator:      model.NewGroupSignGenerator(threshold),
		signedRewardTxHashs: set.New(set.ThreadSafe),
	}
}

func (sc *SlotContext) setSlotStatus(st int32) {
	atomic.StoreInt32(&sc.slotStatus, st)
}

func (sc *SlotContext) IsFailed() bool {
	st := sc.GetSlotStatus()
	return st == slFailed
}
func (sc *SlotContext) IsRewardSent() bool {
	st := sc.GetSlotStatus()
	return st == slRewardSent
}
func (sc *SlotContext) GetSlotStatus() int32 {
	return atomic.LoadInt32(&sc.slotStatus)
}

func (sc SlotContext) MessageSize() int {
	return sc.gSignGenerator.WitnessSize()
}

// VerifyGroupSigns verifies both the verifyGroup signature and the random number(also a signature in fact)
func (sc *SlotContext) VerifyGroupSigns(pk groupsig.Pubkey, preRandom []byte) bool {
	if sc.IsVerified() || sc.IsSuccess() {
		return true
	}
	b := sc.gSignGenerator.VerifyGroupSign(pk, sc.BH.Hash.Bytes())
	if b {
		b = sc.rSignGenerator.VerifyGroupSign(pk, preRandom)
		if b {
			// Group signature verification
			sc.setSlotStatus(slVerified)
		}
	}
	if !b {
		sc.setSlotStatus(slFailed)
	}
	return b
}

func (sc *SlotContext) IsVerified() bool {
	return sc.GetSlotStatus() == slVerified
}

func (sc *SlotContext) IsRecovered() bool {
	return sc.GetSlotStatus() == slRecovered
}

func (sc *SlotContext) IsSuccess() bool {
	return sc.GetSlotStatus() == slSuccess
}

func (sc *SlotContext) IsWaiting() bool {
	return sc.GetSlotStatus() == slWaiting
}

// AcceptVerifyPiece received an in-verifyGroup verification signature piece
//
// Returns:
//		1, the verification piece is accepted and the threshold not reached
// 		2, the verification piece is accepted and the threshold reached
//		-1, piece denied
func (sc *SlotContext) AcceptVerifyPiece(signer groupsig.ID, sign groupsig.Signature, randomSign groupsig.Signature) (ret int8, err error) {

	add, generate := sc.gSignGenerator.AddWitness(signer, sign)
	// Has received the member's verification
	if !add {
		// ignore
		return pieceFail, fmt.Errorf("CBMR_IGNORE_REPEAT")
	}

	// Did not receive the signature of the user
	randomAdd, randomGen := sc.rSignGenerator.AddWitness(signer, randomSign)
	// Reach the verifyGroup signature condition
	if randomAdd && generate && randomGen {
		sc.setSlotStatus(slRecovered)
		return pieceThreshold, nil
	}
	return pieceNormal, nil
}

func (sc *SlotContext) isValid() bool {
	return sc.GetSlotStatus() != slIniting
}

func (sc *SlotContext) statusTransform(from int32, to int32) bool {
	return atomic.CompareAndSwapInt32(&sc.slotStatus, from, to)
}

func (sc *SlotContext) setRewardTrans(tx *types.Transaction) bool {
	if !sc.hasSignedRewardTx() && sc.statusTransform(slSuccess, slRewardSignReq) {
		sc.rewardTrans = tx
		return true
	}
	return false
}

// AcceptRewardPiece try to accept the signature piece of the reward transaction consensus
func (sc *SlotContext) AcceptRewardPiece(sd *model.SignData) (accept, recover bool) {
	if sc.rewardTrans != nil && sc.rewardTrans.Hash != sd.DataHash {
		return
	}
	if sc.rewardTrans == nil {
		return
	}
	if sc.rewardGSignGen == nil {
		sc.rewardGSignGen = model.NewGroupSignGenerator(sc.gSignGenerator.Threshold())
	}
	accept, recover = sc.rewardGSignGen.AddWitness(sd.GetID(), sd.DataSign)
	if accept && recover {
		// Cast block reward transaction using verifyGroup signature
		if sc.rewardTrans.Sign == nil {
			signBytes := sc.rewardGSignGen.GetGroupSign().Serialize()
			tmpBytes := make([]byte, common.SignLength)
			// Group signature length = 33, common signature length = 65.
			// VerifyRewardTransaction() will recover common sig to groupsig
			copy(tmpBytes[0:len(signBytes)], signBytes)
			sign := common.BytesToSign(tmpBytes)
			sc.rewardTrans.Sign = sign.Bytes()
		}
	}
	return
}

func (sc *SlotContext) addSignedTxHash(hash common.Hash) {
	sc.signedRewardTxHashs.Add(hash)
}

func (sc *SlotContext) hasSignedTxHash(hash common.Hash) bool {
	return sc.signedRewardTxHashs.Has(hash)
}

// hasSignedRewardTx means if signed a reward transaction
func (sc *SlotContext) hasSignedRewardTx() bool {
	return sc.signedRewardTxHashs.Size() > 0
}

// GetAggregatedSign returns the aggregated signature of proposer and verifier-verifyGroup
func (sc *SlotContext) GetAggregatedSign() *groupsig.Signature {
	return sc.aggregateSign(sc.gSignGenerator.GetGroupSign())
}

func (sc *SlotContext) aggregateSign(sign groupsig.Signature) *groupsig.Signature {
	if sc.pSign.IsValid() && sign.IsValid() {
		signArray := [2]groupsig.Signature{sc.pSign, sign}
		aggSign := groupsig.AggregateSigs(signArray[:])
		return &aggSign
	}
	return nil
}

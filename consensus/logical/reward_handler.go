package logical

import (
	"fmt"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

type ProcessorInterface interface {
	GetMinerID() groupsig.ID
	GetRewardManager() types.RewardManager
	GetBlockHeaderByHash(hash common.Hash) *types.BlockHeader
	GetVctxByHeight(height uint64) *VerifyContext
	GetGroupBySeed(seed common.Hash) *verifyGroup
	GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey

	AddTransaction(tx *types.Transaction) (bool, error)

	SendCastRewardSign(msg *model.CastRewardTransSignMessage)
}


type RewardHandler struct {
	processor ProcessorInterface

	futureRewardReqs *FutureMessageHolder // Store the reward sign request messages non-processable because of absence of the corresponding block
}

func NewRewardHandler(pi ProcessorInterface) *RewardHandler {
	rh := &RewardHandler{}
	rh.processor = pi
	rh.futureRewardReqs = NewFutureMessageHolder()
	return rh
}

// OnMessageCastRewardSign receives signed messages for the reward transaction from verifyGroup members
// If threshold signature received and the verifyGroup signature recovered successfully, the node will submit the reward transaction to the pool
func (rh *RewardHandler) OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage) {
	mType := "OMCRS"

	tLog := newHashTraceLog(mType, msg.BlockHash, msg.SI.GetID())

	tLog.logStart("txHash=%v", msg.ReqHash)

	var (
		send bool
		err  error
	)

	defer func() {
		tLog.logEnd("reward send:%v, ret:%v", send, err)
	}()

	// If the block related to the reward transaction is not on the chain, then drop the messages
	// This could happened after one fork process
	bh := rh.processor.GetBlockHeaderByHash(msg.BlockHash)
	if bh == nil {
		err = fmt.Errorf("block not exist, hash=%v", msg.BlockHash)
		return
	}

	gSeed := bh.Group
	group := rh.processor.GetGroupBySeed(gSeed)
	if group == nil {
		err = fmt.Errorf("verifyGroup is nil")
		return
	}
	pk := group.getMemberPubkey(msg.SI.GetID())
	if !msg.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail, pk=%v, id=%v", pk, msg.SI.GetID())
		return
	}

	vctx := rh.processor.GetVctxByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		err = fmt.Errorf("vctx is nil")
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}

	// Try to add the signature to the verifyGroup sign generator of the slot related to the block
	accept, recover := slot.AcceptRewardPiece(&msg.SI)

	// Add the reward transaction to pool if the signature is accepted and the verifyGroup signature is recovered
	if accept && recover && slot.statusTransform(slRewardSignReq, slRewardSent) {
		_, err2 := rh.processor.AddTransaction(slot.rewardTrans)
		send = true
		err = fmt.Errorf("add rewardTrans to txPool, txHash=%v, ret=%v", slot.rewardTrans.Hash, err2)

	} else {
		err = fmt.Errorf("accept %v, recover %v, %v", accept, recover, slot.rewardGSignGen.Brief())
	}
}

// OnMessageCastRewardSignReq handles reward transaction signature requests
// It signs the message if and only if the block of the transaction already added on chain,
// otherwise the message will be cached util the condition met
func (rh *RewardHandler) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	mType := "OMCRSR"
	reward := &msg.Reward
	tLog := newHashTraceLog(mType, reward.BlockHash, msg.SI.GetID())
	tLog.logStart("txHash=%v", reward.TxHash)

	var (
		send bool
		err  error
	)

	defer func() {
		tLog.logEnd("txHash=%v, %v %v", reward.TxHash, send, err)
	}()

	// At this point the block is not necessarily on the chain
	// in case that, the message will be cached
	bh := rh.processor.GetBlockHeaderByHash(reward.BlockHash)
	if bh == nil {
		err = fmt.Errorf("future reward request receive and cached, hash=%v", reward.BlockHash)
		msg.ReceiveTime = time.Now()
		rh.futureRewardReqs.addMessage(reward.BlockHash, msg)
		return
	}

	send, err = rh.signCastRewardReq(msg, bh)
	return
}

func (rh *RewardHandler) signCastRewardReq(msg *model.CastRewardTransSignReqMessage, bh *types.BlockHeader) (send bool, err error) {
	gSeed := bh.Group
	reward := &msg.Reward

	vctx := rh.processor.GetVctxByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		err = fmt.Errorf("vctx is nil,%v height=%v", vctx == nil, bh.Height)
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}

	// A dividend transaction has been sent, no longer signed for this
	if slot.IsRewardSent() {
		err = fmt.Errorf("already sent reward trans")
		return
	}

	if gSeed != reward.Group {
		err = fmt.Errorf("groupSeed not equal %v %v", bh.Group, reward.Group)
		return
	}
	if !slot.hasSignedTxHash(reward.TxHash) {
		rewardShare := rh.processor.GetRewardManager().CalculateCastRewardShare(bh.Height, bh.GasFee)
		genReward, _, err2 := rh.processor.GetRewardManager().GenerateReward(reward.TargetIds, bh.Hash, bh.Group, rewardShare.TotalForVerifier(), rewardShare.ForRewardTxPacking)
		if err2 != nil {
			err = err2
			return
		}
		if genReward.TxHash != reward.TxHash {
			err = fmt.Errorf("reward txHash diff %v %v", genReward.TxHash, reward.TxHash)
			return
		}

		if len(msg.Reward.TargetIds) != len(msg.SignedPieces) {
			err = fmt.Errorf("targetId len differ from signedpiece len %v %v", len(msg.Reward.TargetIds), len(msg.SignedPieces))
			return
		}

		group := vctx.group

		mpk := group.getMemberPubkey(msg.SI.GetID())
		if !msg.VerifySign(mpk) {
			err = fmt.Errorf("verify sign fail, gseed=%v, uid=%v", gSeed, msg.SI.GetID())
			return
		}

		// Reuse the original generator to avoid duplicate signature verification
		gSignGenerator := slot.gSignGenerator

		for idx, idIndex := range msg.Reward.TargetIds {
			mem := group.getMemberAt(int(idIndex))
			if mem == nil {
				stdLogger.Errorf("reward targets %v: %v", len(msg.Reward.TargetIds), msg.Reward.TargetIds)
				err = fmt.Errorf("member not exist, idx %v, memsize %v", idIndex, group.memberSize())
				return
			}
			sign := msg.SignedPieces[idx]

			// If there is no local id signature, you need to verify the signature.
			if sig, ok := gSignGenerator.GetWitness(mem.id); !ok {
				pk := group.getMemberPubkey(mem.id)
				if !groupsig.VerifySig(pk, bh.Hash.Bytes(), sign) {
					err = fmt.Errorf("verify member sign fail, id=%v", mem.id)
					return
				}
				// Join the generator
				gSignGenerator.AddWitnessForce(mem.id, sign)
			} else { // If the signature of the id already exists locally, just judge whether it is the same as the local signature.
				if !sign.IsEqual(sig) {
					err = fmt.Errorf("member sign different id=%v", mem.id)
					return
				}
			}
		}

		if !gSignGenerator.Recovered() {
			err = fmt.Errorf("recover verifyGroup sign fail")
			return
		}

		bhSign := groupsig.DeserializeSign(bh.Signature)
		aggSign := slot.GetAggregatedSign()
		if aggSign == nil {
			err = fmt.Errorf("obtain the Aggregated signature fail")
			return
		}
		if !aggSign.IsEqual(*bhSign) {
			err = fmt.Errorf("aggregated sign differ from bh sign, aggregated %v, bh %v", aggSign, bhSign)
			return
		}

		slot.addSignedTxHash(reward.TxHash)
	}

	send = true
	// Sign yourself
	signMsg := &model.CastRewardTransSignMessage{
		ReqHash:   reward.TxHash,
		BlockHash: reward.BlockHash,
		GSeed:     gSeed,
		Launcher:  msg.SI.GetID(),
	}
	ski := model.NewSecKeyInfo(rh.processor.GetMinerID(), rh.processor.GetGroupSignatureSeckey(gSeed))
	if signMsg.GenSign(ski, signMsg) {
		rh.processor.SendCastRewardSign(signMsg)
	} else {
		err = fmt.Errorf("signCastRewardReq genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
	}
	return
}

func (rh *RewardHandler) TriggerFutureRewardSign(bh *types.BlockHeader) {
	futures := rh.futureRewardReqs.getMessages(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	rh.futureRewardReqs.remove(bh.Hash)
	mType := "CMCRSR-Future"
	for _, msg := range futures {
		tLog := newHashTraceLog(mType, bh.Hash, groupsig.ID{})
		send, err := rh.signCastRewardReq(msg.(*model.CastRewardTransSignReqMessage), bh)
		tLog.logEnd("send %v, result %v", send, err)
	}
}
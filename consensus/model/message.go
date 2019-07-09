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

package model

import (
	"fmt"
	"github.com/zvchain/zvchain/middleware/types"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
)

// ISignedMessage defines the message functions
type ISignedMessage interface {
	// GenSign generates signature with the given secKeyInfo and hash function
	// Returns false when generating failure
	GenSign(ski SecKeyInfo, hasher Hasher) bool

	// VerifySign verifies the signature with the public key
	VerifySign(pk groupsig.Pubkey) bool
}

// Hasher defines the hash generated function for messages
type Hasher interface {
	GenHash() common.Hash
}

// BaseSignedMessage is the base class of all messages that need to be signed
type BaseSignedMessage struct {
	SI SignData
}

// GenSign generates signature with the given secKeyInfo and hash function
// Returns false when generating failure
func (sign *BaseSignedMessage) GenSign(ski SecKeyInfo, hasher Hasher) bool {
	if !ski.IsValid() {
		return false
	}
	sign.SI = GenSignData(hasher.GenHash(), ski.GetID(), ski.SK)
	return true
}

// VerifySign verifies the signature with the public key
func (sign *BaseSignedMessage) VerifySign(pk groupsig.Pubkey) (ok bool) {
	if !sign.SI.GetID().IsValid() {
		return false
	}
	ok = sign.SI.VerifySign(pk)
	if !ok {
		fmt.Printf("verifySign fail, pk=%v, id=%v, sign=%v, data=%v\n", pk.GetHexString(), sign.SI.SignMember.GetHexString(), sign.SI.DataSign.GetHexString(), sign.SI.DataHash.Hex())
	}
	return
}

// ConsensusCastMessage is the block proposal message from proposers
// and handled by the verify-group members
type ConsensusCastMessage struct {
	BH        types.BlockHeader
	ProveHash common.Hash
	BaseSignedMessage
}

func (msg *ConsensusCastMessage) GenHash() common.Hash {
	return msg.BH.GenHash()
}

func (msg *ConsensusCastMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.BH.Random)
	if sig == nil || sig.IsNil() {
		return false
	}
	return groupsig.VerifySig(pkey, preRandom, *sig)
}

// ConsensusVerifyMessage is Verification message - issued by the each members of the verify-group for a specified block
type ConsensusVerifyMessage struct {
	BlockHash  common.Hash
	RandomSign groupsig.Signature
	BaseSignedMessage
}

func (msg *ConsensusVerifyMessage) GenHash() common.Hash {
	return msg.BlockHash
}

func (msg *ConsensusVerifyMessage) GenRandomSign(skey groupsig.Seckey, preRandom []byte) {
	sig := groupsig.Sign(skey, preRandom)
	msg.RandomSign = sig
}

/*
Reward transaction
*/

// CastRewardTransSignReqMessage is the signature requesting message for reward transaction
type CastRewardTransSignReqMessage struct {
	BaseSignedMessage
	Reward       types.Reward
	SignedPieces []groupsig.Signature
	ReceiveTime  time.Time
}

func (msg *CastRewardTransSignReqMessage) GenHash() common.Hash {
	return msg.Reward.TxHash
}

// CastRewardTransSignMessage is the signature response message to requester who should be one of the group members
type CastRewardTransSignMessage struct {
	BaseSignedMessage
	ReqHash   common.Hash
	BlockHash common.Hash

	// Not serialized
	GSeed    common.Hash
	Launcher groupsig.ID
}

func (msg *CastRewardTransSignMessage) GenHash() common.Hash {
	return msg.ReqHash
}

// ReqProposalBlock requests the block body when the verification consensus is finished by the group members
type ReqProposalBlock struct {
	Hash common.Hash
}

// ResponseProposalBlock responses the corresponding block body to the requester
type ResponseProposalBlock struct {
	Hash         common.Hash
	Transactions []*types.Transaction
}

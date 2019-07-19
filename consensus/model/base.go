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

// Package model defines core data structures  used in the consensus process
package model

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
)

// SignData is data signature structure
type SignData struct {
	Version    int32              // Protocol version
	DataHash   common.Hash        // Hash value which is the signed message
	DataSign   groupsig.Signature // The signature
	SignMember groupsig.ID        // User ID who does the signing work
}

func (sd SignData) IsEqual(rhs SignData) bool {
	return sd.DataHash == rhs.DataHash && sd.SignMember.IsEqual(rhs.SignMember) && sd.DataSign.IsEqual(rhs.DataSign)
}

// GenSignData generate SignData
func GenSignData(h common.Hash, id groupsig.ID, sk groupsig.Seckey) SignData {
	return SignData{
		DataHash:   h,
		DataSign:   groupsig.Sign(sk, h.Bytes()),
		SignMember: id,
		Version:    common.ConsensusVersion,
	}
}

func (sd SignData) GetID() groupsig.ID {
	return sd.SignMember
}

// GenSign generate signature with sk
func (sd *SignData) GenSign(sk groupsig.Seckey) bool {
	b := sk.IsValid()
	if b {
		sd.DataSign = groupsig.Sign(sk, sd.DataHash.Bytes())
	}
	return b
}

// VerifySign verify the signature with pk, verify that it returns true, otherwise false.
func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
	return groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
}

// HasSign means is there already signature data?
func (sd SignData) HasSign() bool {
	return sd.DataSign.IsValid() && sd.SignMember.IsValid()
}

// PubKeyInfo means Id->public key pair
type PubKeyInfo struct {
	ID groupsig.ID
	PK groupsig.Pubkey
}

func NewPubKeyInfo(id groupsig.ID, pk groupsig.Pubkey) PubKeyInfo {
	return PubKeyInfo{
		ID: id,
		PK: pk,
	}
}

func (p PubKeyInfo) IsValid() bool {
	return p.ID.IsValid() && p.PK.IsValid()
}

func (p PubKeyInfo) GetID() groupsig.ID {
	return p.ID
}

// SecKeyInfo means Id->private key pair
type SecKeyInfo struct {
	ID groupsig.ID
	SK groupsig.Seckey
}

func NewSecKeyInfo(id groupsig.ID, sk groupsig.Seckey) SecKeyInfo {
	return SecKeyInfo{
		ID: id,
		SK: sk,
	}
}

func (s SecKeyInfo) IsValid() bool {
	return s.ID.IsValid() && s.SK.IsValid()
}

func (s SecKeyInfo) GetID() groupsig.ID {
	return s.ID
}

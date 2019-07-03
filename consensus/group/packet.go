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

package group

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
)

type originSenderSharePiece struct {
	receiver groupsig.ID
	piece    groupsig.Seckey
}

func (sp *originSenderSharePiece) Receiver() []byte {
	return sp.receiver.Serialize()
}

func (sp *originSenderSharePiece) PieceData() []byte {
	return sp.piece.Serialize()
}

type encryptedSenderSharePiece struct {
	*originSenderSharePiece
	receiverPubkey groupsig.Pubkey
	sourceSecKey   groupsig.Seckey
}

func (sp *encryptedSenderSharePiece) PieceData() []byte {
	return sp.piece.Serialize()
}

type originSenderSharePiecePacket struct {
	seed   common.Hash
	sender groupsig.ID
	pieces []types.SenderPiece
}

func (sp *originSenderSharePiecePacket) Seed() common.Hash {
	return sp.seed
}

func (sp *originSenderSharePiecePacket) Sender() []byte {
	return sp.sender.Serialize()
}

func (sp *originSenderSharePiecePacket) Pieces() []types.SenderPiece {
	return sp.pieces
}

type encryptedSenderSharePiecePacket struct {
	*originSenderSharePiecePacket
	pubkey groupsig.Pubkey
}

func (sp *encryptedSenderSharePiecePacket) Pubkey() []byte {
	return sp.pubkey.Serialize()
}

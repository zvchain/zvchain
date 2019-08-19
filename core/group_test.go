////   Copyright (C) 2018 ZVChain
////
////   This program is free software: you can redistribute it and/or modify
////   it under the terms of the GNU General Public License as published by
////   the Free Software Foundation, either version 3 of the License, or
////   (at your option) any later version.
////
////   This program is distributed in the hope that it will be useful,
////   but WITHOUT ANY WARRANTY; without even the implied warranty of
////   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
////   GNU General Public License for more details.
////
////   You should have received a copy of the GNU General Public License
////   along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
package core

import (
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/middleware/types"
)

// TestGroupCreateTxs tests interface types.GroupPacketSender and types.GroupStoreReader
func TestGroupCreateTxs(t *testing.T) {
	err := initContext4Test()
	if err != nil {
		t.Fatalf("fail to initContext4Test")
	}

	defer clear()
	castor := new([]byte)

	var block *types.Block
	account := getAccount()

	seed := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")

	sender := common.StringToAddress(account.Address).Bytes()
	groupSender := group.NewPacketSender(BlockChainImpl)

	//Round 1
	data := &group.EncryptedSharePiecePacketImpl{}
	data.SeedD = seed
	data.SenderD = sender
	data.Pubkey0D = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555555")
	data.PiecesD = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f00000000000000")

	err = groupSender.SendEncryptedPiecePacket(data)
	if err != nil {
		t.Fatalf("fail to SendEncryptedPiecePacket %v", err)
	}

	block = BlockChainImpl.CastBlock(1, common.Hex2Bytes("11"), 0, *castor, seed)
	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block: %v", err)
	}

	store := group.NewStore(BlockChainImpl)
	pieces, err := store.GetEncryptedPiecePackets(data)
	if err != nil {
		t.Fatalf("fail to GetEncryptedPiecePackets %v", err)
	}
	if len(pieces) == 0 {
		t.Fatalf("the length of pieces sould be 1 but got 0")
	}

	// Round 2
	mpkData := &group.MpkPacketImpl{}
	mpkData.SeedD = seed
	mpkData.SenderD = sender
	mpkData.MpkD = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555522")
	mpkData.SignD = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555533")

	err = groupSender.SendMpkPacket(mpkData)
	if err != nil {
		t.Fatalf("fail to SendMpkPacket: %v", err)
	}
	block = BlockChainImpl.CastBlock(2, common.Hex2Bytes("12"), 1, *castor, seed)
	if 0 != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block: %v", err)
	}

	store = group.NewStore(BlockChainImpl)
	mpks, err := store.GetMpkPackets(mpkData)
	if err != nil {
		t.Fatalf("fail to GetMpkPackets %v", err)
	}
	if len(mpks) == 0 {
		t.Fatalf("the length of mpks sould be 1 but got 0")
	}

	//Round 3
	//SendOriginPiecePacket
	dataOp := &group.OriginSharePiecePacketImpl{}
	dataOp.SeedD = seed
	dataOp.SenderD = sender
	dataOp.EncSeckeyD = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f755555555544")
	dataOp.PiecesD = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555566")

	err = groupSender.SendOriginPiecePacket(dataOp)
	if err != nil {
		t.Fatalf("fail to SendOriginPiecePacket: %v", err)
	}

	block = BlockChainImpl.CastBlock(3, common.Hex2Bytes("13"), 2, *castor, seed)
	// 上链
	if 0 != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block: %v", err)
	}

	store = group.NewStore(BlockChainImpl)
	ops, err := store.GetOriginPiecePackets(dataOp)
	if err != nil {
		t.Fatalf("fail to GetOriginPiecePackets %v", err)
	}
	if len(ops) == 0 {
		t.Fatalf("the length of ops sould be 1 but got 0")
	}

	hasPieceSent := store.HasSentEncryptedPiecePacket(sender, dataOp)
	if !hasPieceSent {
		t.Fatalf("fail to test HasSentEncryptedPiecePacket, should returns ture but got false")
	}

	hasMpkSent := store.HasSentMpkPacket(sender, dataOp)
	if !hasMpkSent {
		t.Fatalf("fail to test HasSentMpkPacket, should returns ture but got false")
	}

	isOrgPieceRequired := store.IsOriginPieceRequired(dataOp)
	if isOrgPieceRequired {
		t.Fatalf("fail to test IsOriginPieceRequired, should returns false but got true")
	}

	hasOrgPieceSent := store.HasSentOriginPiecePacket(sender, dataOp)
	if !hasOrgPieceSent {
		t.Fatalf("fail to test HasSentOriginPiecePacket, should returns ture but got false")
	}
	//
	//TODO: test GetGroupInfoBySeed()
	//groupInfo := store.GetGroupInfoBySeed(dataOp)
	//if groupInfo == nil {
	//	t.Fatalf("fail to test GetGroupBySeed, should returns object but got nil" )
	//}
	//
	//TODO: test GetAvailableGroupSeeds()
	//hasOrgPieceSent := store.GetAvailableGroupSeeds(3)
	//if !hasOrgPieceSent {
	//	t.Fatalf("fail to test GetAvailableGroupSeeds, should returns ture but got false" )
	//}

}

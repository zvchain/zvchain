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
	"fmt"
	"github.com/zvchain/zvchain/middleware/notify"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type PacketSender struct {
	chain        chainReader
	baseGasPrice *types.BigInt
	baseGasLimit *types.BigInt
}

func NewPacketSender(chain chainReader) types.GroupPacketSender {
	return &PacketSender{
		chain:        chain,
		baseGasPrice: types.NewBigInt(uint64(common.GlobalConf.GetInt("chain", "group_tx_gas_price", 2000))),
		baseGasLimit: types.NewBigInt(uint64(common.GlobalConf.GetInt("chain", "group_tx_gas_limit", 13000))), // actually used 7186
	}
}

// SendEncryptedPiecePacket send transaction the miner's encrypted piece  in round one
func (p *PacketSender) SendEncryptedPiecePacket(packet types.EncryptedSharePiecePacket) (err error) {
	source := common.BytesToAddress(packet.Sender())
	data := &EncryptedSharePiecePacketImpl{}
	data.SeedD = packet.Seed()
	data.SenderD = packet.Sender()
	data.Pubkey0D = packet.Pubkey0()
	data.PiecesD = packet.Pieces()
	byteData, err := msgpack.Marshal(data)
	if err != nil {
		return
	}
	tx, err := p.toTx(source, byteData, types.TransactionTypeGroupPiece)
	if err != nil {
		return
	}
	err = p.sendTransaction(tx)
	if err != nil {
		return
	}
	// print the message to console
	printToConsole(fmt.Sprintf("auto send group piece, hash = %v \n", tx.Hash.Hex()))
	return nil
}

// SendEncryptedPiecePacket send transaction the miner mpk and sign  in round two
func (p *PacketSender) SendMpkPacket(packet types.MpkPacket) (err error) {
	source := common.BytesToAddress(packet.Sender())
	data := &MpkPacketImpl{}
	data.SeedD = packet.Seed()
	data.MpkD = packet.Mpk()
	data.SignD = packet.Sign()
	data.SenderD = packet.Sender()

	byteData, err := msgpack.Marshal(data)
	if err != nil {
		return
	}
	tx, err := p.toTx(source, byteData, types.TransactionTypeGroupMpk)
	if err != nil {
		return
	}
	err = p.sendTransaction(tx)
	if err != nil {
		return
	}
	// print the message to console
	printToConsole(fmt.Sprintf("auto send group mpk, hash = %v \n", tx.Hash.Hex()))
	return nil
}

// SendEncryptedPiecePacket send transaction the miner origin in round three
func (p PacketSender) SendOriginPiecePacket(packet types.OriginSharePiecePacket) (err error) {
	source := common.BytesToAddress(packet.Sender())
	data := &OriginSharePiecePacketImpl{}
	data.SeedD = packet.Seed()
	data.SenderD = packet.Sender()
	data.EncSeckeyD = packet.EncSeckey()
	data.PiecesD = packet.Pieces()

	byteData, err := msgpack.Marshal(data)
	if err != nil {
		return
	}
	tx, err := p.toTx(source, byteData, types.TransactionTypeGroupOriginPiece)
	if err != nil {
		return
	}
	err = p.sendTransaction(tx)
	if err != nil {
		return
	}
	// print the message to console
	printToConsole(fmt.Sprintf("auto send group origin piece, hash = %v \n", tx.Hash.Hex()))
	return nil
}

func (p *PacketSender) toTx(source common.Address, data []byte, txType int8) (*types.Transaction, error) {
	db, err := p.chain.LatestStateDB()
	if err != nil {
		logger.Error("failed to get last db")
		return nil, err
	}

	tx := &types.Transaction{}
	tx.Data = data
	tx.Type = txType
	tx.GasPrice = p.baseGasPrice
	tx.GasLimit = p.baseGasLimit
	tx.Nonce = db.GetNonce(source) + 1
	tx.Hash = tx.GenHash()

	sk := common.HexToSecKey(p.chain.MinerSk())
	if sk == nil {
		return nil, fmt.Errorf("fail to get miner's sk")
	}
	sign, err := sk.Sign(tx.Hash.Bytes())
	if err != nil {
		return nil, err
	}
	tx.Sign = sign.Bytes()
	return tx, nil
}

func (p *PacketSender) sendTransaction(tx *types.Transaction) error {
	if tx.Sign == nil {
		return fmt.Errorf("transaction sign is empty")
	}
	if ok, err := p.chain.AddTransactionToPool(tx); err != nil || !ok {
		return common.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err)
	}

	logger.Debugf("[group] sendTransaction success. type = %d, hash = %v ", tx.Type, tx.Hash)
	return nil
}

func printToConsole(msg string) {
	notify.BUS.Publish(notify.MessageToConsole, &consoleMsg{msg})
}

type consoleMsg struct {
	msg string
}

func (m *consoleMsg) GetRaw() []byte {
	return []byte{}
}
func (m *consoleMsg) GetData() interface{} {
	return m.msg
}

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

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
)

type CheckerContext struct {
	height uint64
}

func (c *CheckerContext) Height() uint64 {
	return c.height
}

// Operation define some functions on create create transaction
type Operation interface {
	ParseTransaction() error // Parse the input transaction
	Operation() error        // Do the operation
}

func newBaseOperation(db vm.AccountDB, tx types.Transaction, height uint64, checker types.GroupCreateChecker) *baseOperation {
	return &baseOperation{
		accountDB: db,
		tx:        tx,
		height:    height,
		checker:   checker,
	}
}

type baseOperation struct {
	accountDB vm.AccountDB
	tx        types.Transaction
	height    uint64
	checker   types.GroupCreateChecker
}

// NewOperation creates the mOperation instance base on msg type
func (m *Manager) NewOperation(db vm.AccountDB, tx types.Transaction, height uint64) Operation {
	baseOp := newBaseOperation(db, tx, height, m.checker)
	var operation Operation
	switch tx.Type {
	case types.TransactionTypeGroupPiece:
		operation = &sendPieceOp{baseOperation: baseOp}
	case types.TransactionTypeGroupMpk:
		operation = &sendMpkOp{baseOperation: baseOp}
	case types.TransactionTypeGroupOriginPiece:
		operation = &sendOriginPieceOp{baseOperation: baseOp}
	}
	return operation
}

// sendPieceOp is for the group piece upload operation in round one
type sendPieceOp struct {
	*baseOperation
	data types.EncryptedSharePiecePacket
}

func (op *sendPieceOp) ParseTransaction() (err error) {
	if op.tx.Data == nil {
		err = fmt.Errorf("payload length error")
		return
	}
	var data EncryptedSharePiecePacketImpl
	err = msgpack.Unmarshal(op.tx.Data, &data)
	if err != nil {
		return
	}
	op.data = &data
	ctx := &CheckerContext{op.height}
	err = op.checker.CheckEncryptedPiecePacket(&data, ctx)
	return
}

func (op *sendPieceOp) Operation() error {
	seedAddr := common.HashToAddress(op.data.Seed())
	source := op.tx.Source
	key := &txDataKey{dataVersion, dataTypePiece, *source}
	byteKey := keyToByte(key)
	op.accountDB.SetData(seedAddr, byteKey, op.tx.Data)
	return nil
}

// sendMpkOp is for the group piece upload operation in round two
type sendMpkOp struct {
	*baseOperation
	data types.MpkPacket
}

func (op *sendMpkOp) ParseTransaction() (err error) {
	if op.tx.Data == nil {
		err = fmt.Errorf("payload length error")
		return
	}
	var data MpkPacketImpl
	err = msgpack.Unmarshal(op.tx.Data, &data)
	if err != nil {
		return
	}
	op.data = &data
	ctx := &CheckerContext{op.height}
	err = op.checker.CheckMpkPacket(&data, ctx)
	return
}

func (op *sendMpkOp) Operation() error {
	seedAddr := common.HashToAddress(op.data.Seed())
	source := op.tx.Source

	key := &txDataKey{dataVersion, dataTypeMpk, *source}
	byteKey := keyToByte(key)
	op.accountDB.SetData(seedAddr, byteKey, op.tx.Data)

	return nil
}

// sendOriginPieceOp is for the group piece upload operation in round three
type sendOriginPieceOp struct {
	*baseOperation
	data types.OriginSharePiecePacket
}

func (op *sendOriginPieceOp) ParseTransaction() (err error) {
	if op.tx.Data == nil {
		err = fmt.Errorf("payload length error")
		return
	}
	var data OriginSharePiecePacketImpl
	err = msgpack.Unmarshal(op.tx.Data, &data)
	if err != nil {
		return
	}
	op.data = &data

	ctx := &CheckerContext{op.height}
	err = op.checker.CheckOriginPiecePacket(&data, ctx)
	return
}

func (op *sendOriginPieceOp) Operation() error {
	seedAddr := common.HashToAddress(op.data.Seed())
	source := op.tx.Source

	key := &txDataKey{dataVersion, dataTypeOriginPiece, *source}
	byteKey := keyToByte(key)
	op.accountDB.SetData(seedAddr, byteKey, op.tx.Data)

	return nil
}

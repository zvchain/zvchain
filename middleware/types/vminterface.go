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

/*
	Package vm is used as the vm call chain
*/
package types

import (
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/trie"
)

type AccountDB interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	GetRefund() uint64

	GetData(common.Address, []byte) []byte
	SetData(common.Address, []byte, []byte)
	RemoveData(common.Address, []byte)
	DataIterator(common.Address, []byte) *trie.Iterator
	//DataNext(iterator uintptr) []byte

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	Exist(common.Address) bool
	Empty(common.Address) bool

	RevertToSnapshot(int)
	Snapshot() int

	Transfer(common.Address, common.Address, *big.Int)
	CanTransfer(common.Address, *big.Int) bool
}

type ChainReader interface {
	Height() uint64
	QueryTopBlock() *BlockHeader
	QueryBlockHeaderByHash(hash common.Hash) *BlockHeader
	QueryBlockHeaderByHeight(height uint64) *BlockHeader
	HasBlock(hash common.Hash) bool
	HasHeight(height uint64) bool
}

// MinerOperationMessage generated when operate miner stake info
type TxMessage interface {
	OpType() int8
	Operator() *common.Address
	OpTarget() *common.Address
	Amount() *big.Int // Operated value
	Payload() []byte  // Data transfer by the message
	GetExtraData() []byte
	GetGasLimit() uint64
	GetHash() common.Hash
	GetValue() uint64
	GetNonce() uint64
	GetGasLimitOriginal()*big.Int
}

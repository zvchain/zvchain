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

package notify

// defines all of current used event ids
const (
	BlockAddSucc = "block_add_succ"

	GroupAddSucc = "group_add_succ"

	NewBlock = "new_block"

	NewBlockHeader = "new_block_header"

	BlockBodyReq = "block_body_req"

	BlockBody = "block_body"

	StateInfoReq = "state_info_req"

	StateInfo = "state_info"

	BlockReq = "block_req"

	BlockResponse = "block_response"

	BlockInfoNotify = "block_info_notify"

	ChainPieceBlockReq = "chain_piece_block_req"

	ChainPieceBlock = "chain_piece_block"

	GroupHeight = "group_height"

	GroupReq = "group_req"

	Group = "group"

	TransactionReq = "transaction_req"

	TransactionGot = "transaction_got"

	TransactionGotAddSucc = "transaction_got_add_succ"

	BlockSync = "block_sync"
	GroupSync = "group_sync"

	TxSyncNotify   = "tx_sync_notify"
	TxSyncReq      = "tx_sync_req"
	TxSyncResponse = "tx_sync_response"

	TxPoolAddTxs = "tx_pool_add_txs"
)

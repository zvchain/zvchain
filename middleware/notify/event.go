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
	BlockAddSucc     = "block_add_succ"
	GroupAddSucc     = "group_add_succ"
	BlockSync        = "block_sync"
	MessageToConsole = "message_to_console"

	BlockInfoNotify = "block_info_notify"
	BlockReq        = "block_req"
	BlockResponse   = "block_response"
	NewBlock        = "new_block"

	ForkFindAncestorResponse = "fork_find_ancestor_response"
	ForkFindAncestorReq      = "fork_find_ancestor_req"
	ForkChainSliceReq        = "fork_block_req"
	ForkChainSliceResponse   = "fork_block_response"

	TxSyncNotify   = "tx_sync_notify"
	TxSyncReq      = "tx_sync_req"
	TxSyncResponse = "tx_sync_response"
)

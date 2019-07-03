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

package core

import "github.com/zvchain/zvchain/middleware/types"

// InitCore initialize the peerManagerImpl, BlockChainImpl and GroupChainImpl
func InitCore(helper types.ConsensusHelper, account Account) error {
	initPeerManager()
	if nil == BlockChainImpl {
		err := initBlockChain(helper, account)
		if err != nil {
			return err
		}
	}

	if nil == GroupChainImpl && helper != nil {
		err := initGroupChain(helper.GenerateGenesisInfo(), helper)
		if err != nil {
			return err
		}
	}
	return nil
}

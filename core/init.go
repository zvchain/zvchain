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

var InitializationImpl *InitializationProcesser
type InitialFunc func()


func initInitialization(){
	InitializationImpl = &InitializationProcesser{}
}

type InitializationProcesser struct {
	processes[]InitialFunc
}

func(f*InitializationProcesser)Register(fn InitialFunc){
	f.processes = append(f.processes,fn)
}

func(f*InitializationProcesser)Process(){
	if f.processes != nil && len(f.processes) > 0{
		for _,p := range f.processes{
			p()
		}
	}
}


// InitCore initialize the peerManagerImpl, BlockChainImpl and GroupChainImpl
func InitCore(helper types.ConsensusHelper, account types.Account) error {
	initPeerManager()
	initInitialization()
	if nil == BlockChainImpl {
		err := initBlockChain(helper, account)
		if err != nil {
			return err
		}
	}
	return nil
}

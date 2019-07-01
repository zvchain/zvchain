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

package group

import (
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)



func (m *Manager) getGroupCreatingDataFromDb(seed common.Hash) (cData groupCreatingData, err error) {
	data := m.chain.LatestStateDB().GetData(common.GroupCreatingDBAddress,seed.Hex())
	err = msgpack.Unmarshal(data, &cData)
	if err != nil {
		err = log().Errorf("getGroupCreatingData: Unmarshal %v error, ", seed)
	}
	return
}

func (m *Manager) setGroupCreatingDataToDb(seed common.Hash, cData groupCreatingData) {
	//TODO: should we ignore this error? or just panic?
	data, _ := msgpack.Marshal(cData)
	m.chain.LatestStateDB().SetData(common.GroupCreatingDBAddress,seed.Hex(),data)
}

func (m *Manager) saveGroupDataToDb(seed common.Hash, cData *types.Group) {
	//TODO: should we ignore this error? or just panic?
	data, _ := msgpack.Marshal(cData)
	m.chain.LatestStateDB().SetData(common.GroupDBAddress,seed.Hex(),data)
}

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

package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

func ExportChainData(output string, helper types.ConsensusHelper) (err error) {
	var configSec = "chain"
	conf := common.GlobalConf.GetSectionManager(configSec)
	dbFile := common.GlobalConf.GetString(configSec, "db_blocks", "d_b")
	smallDbFile := common.GlobalConf.GetString(configSec, "small_db", "d_small")
	cacheDir := conf.GetString("state_cache_dir", "state_cache")
	stateCacheSize := common.GlobalConf.GetInt(configSec, "db_state_cache", 256)

	genesisGroup := helper.GenerateGenesisInfo()

	sdbExist, err := pathExists(smallDbFile)
	if err != nil {
		return err
	}
	if !sdbExist {
		smallDbFile = ""
	}

	tailor, err := NewOfflineTailor(genesisGroup, dbFile, smallDbFile, stateCacheSize, cacheDir, "", false)
	if err != nil {
		return err
	}
	err = tailor.Export(output)
	if err != nil {
		return err
	}
	return
}

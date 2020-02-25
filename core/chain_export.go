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
	"fmt"
	"os"
	"path/filepath"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
)

func ExportChainData(output string, helper types.ConsensusHelper) (err error) {
	chain, err := getMvpChain(helper, false)
	if err != nil {
		return err
	}
	pruneMode := chain.config.pruneMode
	if !pruneMode {
		printToConsole("You were run the chain in a none prune mode. It is highly recommended that prune the database before doing the exporting.")
	}

	// Should merge small db to chain db before exporting
	var configSec = "chain"
	sdbDir := common.GlobalConf.GetString(configSec, "small_db", "d_small")

	if sdbDir != "" {
		smallStateDs, err := tasdb.NewDataSource(sdbDir, nil)
		if err != nil {
			Logger.Errorf("new small state datasource error:%v", err)
			return err
		}
		smallStateDb, err := smallStateDs.NewPrefixDatabase("")
		if err != nil {
			Logger.Errorf("new small state db error:%v", err)
			return err
		}
		chain.smallStateDb = initSmallStore(smallStateDb)
		err = chain.repairStateDatabase(chain.latestBlock)
		if err != nil {
			return err
		}
	}

	err = doExport(chain, output)

	if err != nil {
		return err
	}
	return
}

func doExport(chain *FullBlockChain, dist string) error {
	last := chain.getLatestBlock()
	tpFile := filepath.Join(chain.config.dbfile, trustHashFile)
	// close the db to avoid the leveldb's compaction
	chain.stateDb.Close()
	err := saveTrustHash(last.Hash, tpFile)
	if err != nil {
		return err
	}
	err = zipit(chain.config.dbfile, dist)
	if err != nil {
		return err
	}
	printToConsole(fmt.Sprintf("Export success. The output file is: %v. The top hash is: %v", dist, last.Hash.Hex()))
	return nil
}

func saveTrustHash(trustHash common.Hash, filename string) error {
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		return err
	} else {
		_, err = f.Write(trustHash.Bytes())
		if err != nil {
			return err
		}
	}
	return nil

}

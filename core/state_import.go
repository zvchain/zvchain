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
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/storage/tasdb"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

var stateValidateBlockNum = 1      // how many blocks need to validate the state tree
var trustHashFile = "tp"


func ImportFromArchive(importFile string, helper types.ConsensusHelper) (err error) {
	dbFile := getBlockChainConfig().dbfile
	dbExist, err := pathExists(dbFile)
	if err != nil {
		return err
	}
	importFile = "db_export"
	archiveExist, err := pathExists(importFile)
	if err != nil {
		return err
	}

	if !archiveExist {
		return errors.New("importing file not exist")
	}
	if dbExist {
		return errors.New(fmt.Sprintf("You have set the '--import' parameter in the start command. please delete the folder %v or remove the '--import' from the start command and try again.", dbFile))
	}


	targetDb := getBlockChainConfig().dbfile
	defer func() {
		if err != nil {
			os.RemoveAll(targetDb)
		}
	}()
	err = unzip(importFile, targetDb)
	if err != nil {
		return err
	}
	tpFile := filepath.Join(targetDb,  trustHashFile)
	trustHash, err := getTrustHash(tpFile)
	if err != nil {
		return err
	}

	chain, err := getMvpChain(helper)
	if err != nil {
		return err
	}
	defer func() {
		chain.stateDb.Close()
	}()

	err = checkTrustDb(chain, trustHash)
	return
}

func getTrustHash(dbPath string) (common.Hash, error) {
	f, err := os.OpenFile(dbPath, os.O_RDONLY, 0600)
	defer f.Close()
	if err != nil {
		return common.EmptyHash, err
	}
	contentByte, _ := ioutil.ReadAll(f)
	return common.BytesToHash(contentByte), nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getMvpChain(helper types.ConsensusHelper) (*FullBlockChain, error) {
	chain := &FullBlockChain{
		config:          getBlockChainConfig(),
		init:            true,
		isAdjusting:     false,
		topRawBlocks:    common.MustNewLRUCache(20),
		consensusHelper: helper,
	}

	Logger = log.CoreLogger

	//options := &opt.Options{
	//	ReadOnly: true,
	//	Filter:   filter.NewBloomFilter(10),
	//}

	ds, err := tasdb.NewDataSource(chain.config.dbfile, nil)
	if err != nil {
		Logger.Errorf("new datasource error:%v", err)
		return nil, err
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.txDb, err = ds.NewPrefixDatabase(chain.config.tx)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.stateDb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}


	stateCacheSize := common.GlobalConf.GetInt(configSec, "db_state_cache", 256)

	chain.stateCache = account.NewDatabaseWithCache(chain.stateDb, false, stateCacheSize, "")

	GroupManagerImpl = group.NewManager(chain, helper)

	chain.cpChecker = newCpChecker(GroupManagerImpl, chain)
	sp := newStateProcessor(chain)
	chain.stateProc = sp

	return chain, nil
}

//
//func NewDbImporter(importFile string) (importer *DbImporter, err error) {
//
//	chain := &FullBlockChain{
//		config:       getBlockChainConfig(),
//		init:         true,
//		isAdjusting:  false,
//		topRawBlocks: common.MustNewLRUCache(20),
//	}
//
//	Logger = log.CoreLogger
//	//chain := &FullBlockChain{
//	//	config:           getBlockChainConfig(),
//	//	latestBlock:      nil,
//	//	init:             true,
//	//	isAdjusting:      false,
//	//	consensusHelper:  helper,
//	//	ticker:           ticker.NewGlobalTicker("chain"),
//	//	triegc:           prque.NewPrque(),
//	//	ts:               time2.TSInstance,
//	//	futureRawBlocks:  common.MustNewLRUCache(100),
//	//	verifiedBlocks:   common.MustNewLRUCache(10),
//	//	topRawBlocks:     common.MustNewLRUCache(20),
//	//	newBlockMessages: common.MustNewLRUCache(100),
//	//	Account:          minerAccount,
//	//}
//
//	options := &opt.Options{
//		ReadOnly:true,
//		Filter:                 filter.NewBloomFilter(10),
//	}
//
//	ds, err := tasdb.NewDataSource(chain.config.dbfile, options)
//	if err != nil {
//		Logger.Errorf("new datasource error:%v", err)
//		return
//	}
//
//	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
//	if err != nil {
//		Logger.Errorf("Init block chain error! Error:%s", err.Error())
//		return
//	}
//
//	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
//	if err != nil {
//		Logger.Errorf("Init block chain error! Error:%s", err.Error())
//		return
//	}
//	chain.txDb, err = ds.NewPrefixDatabase(chain.config.tx)
//	if err != nil {
//		Logger.Errorf("Init block chain error! Error:%s", err.Error())
//		return
//	}
//	chain.stateDb, err = ds.NewPrefixDatabase(chain.config.state)
//	if err != nil {
//		Logger.Errorf("Init block chain error! Error:%s", err.Error())
//		return
//	}
//
//	receiptdb, err := ds.NewPrefixDatabase(chain.config.receipt)
//	if err != nil {
//		Logger.Errorf("Init block chain error! Error:%s", err.Error())
//		return
//	}
//	smallStateDs, err := tasdb.NewDataSource(common.GlobalConf.GetString(configSec, "small_db", "d_small"), nil)
//	if err != nil {
//		Logger.Errorf("new small state datasource error:%v", err)
//		return
//	}
//	smallStateDb, err := smallStateDs.NewPrefixDatabase("")
//	if err != nil {
//		Logger.Errorf("new small state db error:%v", err)
//		return
//	}
//
//
//
//	GroupManagerImpl = group.NewManager(chain, helper)
//
//	chain.cpChecker = newCpChecker(GroupManagerImpl, chain)
//	sp := newStateProcessor(chain)
//	chain.stateProc = sp
//
//	chain.insertGenesisBlock()
//}


func checkTrustDb(chain *FullBlockChain, trustHash common.Hash) (err error) {
	trustBl := chain.queryBlockHeaderByHash(trustHash)
	if trustBl == nil {
		err = errors.New(printToConsole("Can't find the trust block hash in database. Please set the right hash and restart the program!"))
		return
	}
	printToConsole(fmt.Sprintf("Your trust point hash is %v and height is %v", trustBl.Hash, trustBl.Height))

	err = validateHeaders(chain, trustHash)
	if err != nil {
		printToConsole(err.Error())
		err = errors.New(printToConsole("Illegal database! The import file is untrusted."))
		return
	}
	printToConsole("Validating block headers finish")

	err = validateStateDb(chain, trustBl)
	if err != nil {
		Logger.Errorf("VerifyIntegrity failed: %v", err)
		printToConsole(err.Error())
		err = errors.New(printToConsole("Illegal database! The import file is untrusted."))
		return
	}
	printToConsole(fmt.Sprintf("Validating state tree finish, reset top to the trust point: %v and start syncing", trustBl.Height))
	err = chain.blocks.Put([]byte(blockStatusKey), trustBl.Hash.Bytes())
	if err != nil {
		Logger.Errorf("ResetTop failed: %v", err)
		printToConsole(err.Error())
		err = errors.New(printToConsole("Failed to reset top to trust block!"))
		return
	}

	return
}

func validateHeaders(chain *FullBlockChain, trustHash common.Hash) (err error) {
	printToConsole("Start validating block headers ...")
	genesisBl := chain.insertGenesisBlock(false)
	currentHash := trustHash
	var last *types.BlockHeader

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			printToConsole("validating block headers ...")
		}
	}()

	for {
		current := chain.queryBlockHeaderByHash(currentHash)
		if current == nil {
			return fmt.Errorf("validate header fail, miss block: %v", currentHash)
		}

		if current.Hash != current.GenHash() {
			return fmt.Errorf("validate header fail, block hash error: %v", currentHash)
		}

		if last != nil && last.Height <= current.Height {
			return fmt.Errorf("validate header fail, block height error: %v", currentHash)
		}

		if current.Height < 0 {
			return fmt.Errorf("validate header fail, negative block height error: %v, %v", currentHash, current.Height)
		}

		if current.Height == 0 {
			if current.Hash != genesisBl.Header.Hash {
				return fmt.Errorf("validate header fail, genesis block hash error: %v", currentHash)
			}
			return
		}

		last = current
		currentHash = current.PreHash
	}
}

func validateStateDb(chain *FullBlockChain, trustHash *types.BlockHeader) error {
	printToConsole("Start validating state tree ...")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			printToConsole("Validating state tree ...")
		}
	}()
	start := time.Now()
	Logger.Debugf("validateStateDb cost: %v ", time.Since(start))

	currentHash := trustHash.Hash
	for i := 0; i < stateValidateBlockNum; i++ {
		current := chain.queryBlockHeaderByHash(currentHash)
		db, err := account.NewAccountDB(current.StateTree, chain.stateCache)
		if err != nil {
			return err
		}
		printToConsole(fmt.Sprintf("Validating state tree for block height = %d, remaining %d blocks", current.Height, stateValidateBlockNum-i))

		ok, err := db.VerifyIntegrity(nil, nil, true)
		if !ok {
			return fmt.Errorf("validate state fail, block height: %v", current.Height)
		}
		if err != nil {
			return err
		}
		if current.Height == 0 {
			return nil
		}
		currentHash = current.PreHash
	}
	return nil
}

func doDoubleConfirm(topHeight uint64, trustHeight uint64) bool {
	scanLine := func() string {
		var c byte
		var err error
		var b []byte
		for err == nil {
			_, err = fmt.Scanf("%c", &c)
			if c != '\n' {
				b = append(b, c)
			} else {
				break
			}
		}
		return string(b)
	}
	printToConsole(fmt.Sprintf("Your current local top %d is higher than trust block over %d blocks", topHeight, topHeight-trustHeight))
	for {
		printToConsole(fmt.Sprintf("Are you sure you want to reset to the trust block and validate the database? [Y/n]"))
		cmd := scanLine()
		if cmd == "" || cmd == "Y" || cmd == "y" {
			Logger.Debugln("user choose Y to continue validation")
			return true
		} else if cmd == "N" || cmd == "n" {
			printToConsole("You choose to skip the trust block validation. You can remove the -t or --trusthash option from the starting command parameters next time.")
			return false
		}
	}
}

func zipit(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	//info, err := os.Stat(source)
	//if err != nil {
	//	return nil
	//}

	var baseDir string
	//if info.IsDir() {
	//	//baseDir = filepath.Base(source)
	//}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		printToConsole(fmt.Sprintf("compressing %v", path))
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		printToConsole(fmt.Sprintf("uncompressing %v", path))
		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
		targetFile.Close()
		_ = fileReader.Close()
	}

	return nil
}

func printToConsole(msg string) string {
	//Logger.Debugln(msg)
	fmt.Println(msg)
	return msg
}

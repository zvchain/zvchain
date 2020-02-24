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

	"github.com/cheggaaa/pb/v3"

	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/storage/tasdb"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

var trustHashFile = "tp"

var peekForImporting = false
var peekStartHeight uint64 = 0

func EnableChainPeek() {
	peekForImporting = true
}

func addBlockSuccessForImporting(db types.AccountDB, bh *types.BlockHeader) {
	if peekStartHeight == 0 {
		peekStartHeight = bh.Height
	}
	if bh.Height-peekStartHeight > TriesInMemory {
		printToConsole(fmt.Sprintf("%d blocks added.", TriesInMemory))
		printToConsole("The importing process end success")
		os.Exit(0)
	}
}

func ImportChainData(importFile string, helper types.ConsensusHelper) (err error) {
	begin := time.Now()
	defer func() {
		if err == nil {
			printToConsole(fmt.Sprintf("Import database finish, costs %v.", time.Since(begin).String()))
			printToConsole(fmt.Sprintf("Will try to sync %v blocks from the network.", TriesInMemory))
		}
	}()
	dbFile := getBlockChainConfig().dbfile
	// check file existing
	dbExist, err := pathExists(dbFile)
	if err != nil {
		return err
	}
	archiveExist, err := pathExists(importFile)
	if err != nil {
		return err
	}

	if !archiveExist {
		return fmt.Errorf("importing file: %v not exist", importFile)
	}
	if dbExist {
		return fmt.Errorf("You already have a database folder. please delete the folder %v try again.", dbFile)
	}

	//unzip the archive
	targetDb := getBlockChainConfig().dbfile
	defer func() {
		if err != nil {
			_ = os.RemoveAll(targetDb)
		}
	}()
	err = unzip(importFile, targetDb)
	if err != nil {
		return err
	}
	tpFile := filepath.Join(targetDb, trustHashFile)
	trustHash, err := getTrustHash(tpFile)
	if err != nil {
		return err
	}
	err = confirmTrustHash(trustHash)
	if err != nil {
		return err
	}

	//set top block
	chain, err := getMvpChain(helper, false)
	if err != nil {
		return err
	}
	updateTopBlock(chain, trustHash)
	chain.stateDb.Close()

	// check block headers and state db
	err = checkTrustDb(helper, trustHash)
	if err != nil {
		return err
	}

	return
}

// getTrustHash returns the trust block hash from archive file
func getTrustHash(dbPath string) (common.Hash, error) {
	f, err := os.OpenFile(dbPath, os.O_RDONLY, 0600)
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		return common.EmptyHash, err
	}
	contentByte, _ := ioutil.ReadAll(f)
	return common.BytesToHash(contentByte), nil
}

func confirmTrustHash(trustHash common.Hash) error {
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
	printToConsole(fmt.Sprintf("You are importing the chain data from an archive file which may not be trustable. You should manual check below hash is existing in the main ZVChain. (eg. you can check the hash on the https://explorer.zvchain.io)"))
	printToConsole(trustHash.Hex())
	for {
		printToConsole(fmt.Sprintf("Are you sure the above hash is on the main ZVchain? [N/y]"))
		cmd := scanLine()
		if cmd == "" || cmd == "N" || cmd == "n" {
			printToConsole("You choose 'N'")
			return errors.New("Illegal database! The import file is untrusted.")
		} else if cmd == "Y" || cmd == "y" {
			return nil
		}
	}
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

func getMvpChain(helper types.ConsensusHelper, readOnly bool) (*FullBlockChain, error) {
	chain := &FullBlockChain{
		config:          getBlockChainConfig(),
		init:            true,
		isAdjusting:     false,
		topRawBlocks:    common.MustNewLRUCache(20),
		consensusHelper: helper,
	}

	Logger = log.CoreLogger

	options := &opt.Options{
		ReadOnly: readOnly,
		Filter:   filter.NewBloomFilter(10),
	}

	ds, err := tasdb.NewDataSource(chain.config.dbfile, options)
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
	chain.latestBlock = chain.loadCurrentBlock()

	stateCacheSize := common.GlobalConf.GetInt(configSec, "db_state_cache", 256)

	chain.stateCache = account.NewDatabaseWithCache(chain.stateDb, false, stateCacheSize, "")

	GroupManagerImpl = group.NewManager(chain, helper)

	chain.cpChecker = newCpChecker(GroupManagerImpl, chain)
	sp := newStateProcessor(chain)
	chain.stateProc = sp

	return chain, nil
}

func checkTrustDb(helper types.ConsensusHelper, trustHash common.Hash) (err error) {
	chain, err := getMvpChain(helper, true)
	if err != nil {
		return err
	}
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
		err = errors.New("Illegal database! The import file is untrusted.")
		return
	}
	chain.stateDb.Close()
	printToConsole("Validating state tree finish")
	return
}

func validateHeaders(chain *FullBlockChain, trustHash common.Hash) (err error) {
	printToConsole("Start validating block headers ...")
	genesisBl, _ := chain.createGenesisBlock()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			printToConsole("validating block headers ...")
		}
	}()
	trustBh := chain.queryBlockHeaderByHash(trustHash)
	topHeight := trustBh.Height

	// check genesis block
	last := chain.queryBlockHeaderByHeight(0)
	if last.Hash != genesisBl.Header.Hash {
		return fmt.Errorf("validate header fail, genesis block hash error: %v", last.Hash)
	}

	var indexHeight uint64 = 1
	for ; indexHeight <= topHeight; indexHeight++ {
		current := chain.queryBlockHeaderByHeight(indexHeight)
		if current == nil {
			// no block in this height
			continue
		}

		if current.Hash != current.GenHash() {
			return fmt.Errorf("validate header fail, block hash error: %v", current.Hash)
		}

		if current.PreHash != last.Hash {
			return fmt.Errorf("validate header fail, pre hash error: %v", current.Hash)
		}
		last = current
	}

	if last.Hash != trustHash {
		return fmt.Errorf("validate header fail, last hash error: %v", last.Hash)
	}

	return nil
}

func validateStateDb(chain *FullBlockChain, trustBl *types.BlockHeader) error {
	traverseConfig := &account.TraverseConfig{
		CheckHash:           true,
		VisitedRoots:        make(map[common.Hash]struct{}),
	}

	ok, err := chain.Traverse(trustBl.Height, traverseConfig)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("validate failed")
	}
	return nil
}

func updateTopBlock(chain *FullBlockChain, trustBl common.Hash) {
	err := chain.blocks.Put([]byte(blockStatusKey), trustBl.Bytes())
	if err != nil {
		Logger.Errorf("ResetTop failed: %v", err)
		printToConsole(err.Error())
		err = errors.New(printToConsole("Failed to reset top to trust block!"))
		return
	}
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func zipit(source, target string) error {
	printToConsole("compressing data:" + source)
	total, err := dirSize(source)
	if err != nil {
		return err
	}
	// start new bar
	bar := pb.Full.Start64(total)

	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	var baseDir string

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
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
		writer = bar.NewProxyWriter(writer)
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
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
	bar.Finish()
	return err
}

func unzip(archive, target string) error {
	start := time.Now()
	defer func() {
		printToConsole(fmt.Sprintf("uncompressing takes %f seconds" , time.Since(start).Seconds()))
	}()
	printToConsole("uncompressing data:" + archive)
	total, err := os.Stat(archive)
	if err != nil {
		return err
	}
	// start new bar
	bar := pb.Full.Start64(total.Size())

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
		fileReader = bar.NewProxyReader(fileReader)
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
		_ = targetFile.Close()
		_ = fileReader.Close()
	}
	bar.Finish()

	return nil
}

func printToConsole(msg string) string {
	fmt.Println(msg)
	return msg
}

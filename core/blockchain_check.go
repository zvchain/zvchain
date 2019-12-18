package core

import (
	"fmt"
	"os"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/params"
	"github.com/zvchain/zvchain/storage/account"
)

const skipCheckBlockNum = 1000 * 10000 // how many blocks exceed the trust block to skip the checking
const stateValidateBlockNum = 1        // how many blocks need to validate the state tree

func (chain *FullBlockChain) checkTrustDb() *types.BlockHeader {
	hs := params.GetChainConfig().TrustHash
	if hs != "" {
		printToConsole("You have set the trust point block hash in the running command, starting validating the chain between the genesis block and trust point block ...")

		trustHash := common.HexToHash(hs)
		trustBl := chain.queryBlockHeaderByHash(trustHash)
		if trustBl == nil {
			printToConsole("Can't find the trust block hash in database, skip the validation and start the syncing")
			return nil
		}
		printToConsole(fmt.Sprintf("Your trust point hash is %v and height is %v", trustBl.Hash, trustBl.Height))
		// check executed
		top := chain.loadCurrentBlock()
		if top == nil {
			// top is nil, will start with genesis block
			Logger.Infoln("Top is nil, will start with genesis block")
			return nil
		}
		if top.Height-trustBl.Height > skipCheckBlockNum {
			confirmed := doDoubleConfirm(top.Height, trustBl.Height)
			if !confirmed {
				return nil
			}
		}

		err := validateHeaders(chain, trustHash)
		if err != nil {
			printToConsole(err.Error())
			printToConsole("Illegal database! Please delete the directory d_b and restart the program!")
			os.Exit(0)
		}
		printToConsole("Validating block headers finish")

		//validate state  tree
		err = validateStateDb(chain, trustBl)
		if err != nil {
			Logger.Errorf("VerifyIntegrity failed: %v", err)
			printToConsole(err.Error())
			printToConsole("Illegal database! Please delete the directory d_b and restart the program!")
			os.Exit(0)
		}
		printToConsole(fmt.Sprintf("Validating state tree finish, reset top to the trust point: %v and start syncing", trustBl.Height))
		chain.ResetTop(trustBl)
		return trustBl
	}
	return nil
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
	printToConsole(fmt.Sprintf("Your current top height is %d and the turst block height %d", topHeight, trustHeight))
	for {
		printToConsole(fmt.Sprintf("Do you want to reset the top to the trust block and validate the database? (It is highly recommend to choose [Y] if you copied this database from internet and run it first time)  [Y/n]"))
		cmd := scanLine()
		if cmd == "" || cmd == "Y" || cmd == "y" {
			Logger.Debugln("user choose Y to continue validation")
			return true
		} else if cmd == "N" || cmd == "n" {
			printToConsole("You choose to skip the trust block validation. You can remove the -t or --trustHash option from the starting command parameters.")
			return false
		}
	}
}

func printToConsole(msg string) {
	Logger.Debugln(msg)
	fmt.Println(msg)
	//notify.BUS.Publish(notify.MessageToConsole, &consoleMsg{msg})
}

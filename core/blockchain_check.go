package core

import (
	"fmt"
	"os"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/params"
)

func (chain *FullBlockChain) checkTrustDb() *types.BlockHeader {
	hs := params.GetChainConfig().TrustHash
	if hs != "" {
		printToConsole("You have set the trust point block hash in the running command, starting validating the chain between the genesis block and trust point block...")

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
		if top.Height-trustBl.Height > 28000*30 {
			printToConsole("Determining your current top block  height is bigger that the trust block over 28000, skip the validation and start the syncing")
			return nil
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
	printToConsole("Start validating block headers...")
	genesisBl := chain.insertGenesisBlock(false)
	hash := trustHash
	var last *types.BlockHeader
	for {
		current := chain.queryBlockHeaderByHash(hash)
		if current == nil {
			return fmt.Errorf("validate header fail, miss block: %v", hash)
		}

		if current.Hash != current.GenHash() {
			return fmt.Errorf("validate header fail, block hash error: %v", hash)
		}

		if last != nil && last.Height <= current.Height {
			return fmt.Errorf("validate header fail, block height error: %v", hash)
		}

		if current.Height < 0 {
			return fmt.Errorf("validate header fail, negative block height error: %v, %v", hash, current.Height)
		}

		if current.Height == 0 {
			if current.Hash != genesisBl.Header.Hash {
				return fmt.Errorf("validate header fail, genesis block hash error: %v", hash)
			}
			return
		}

		last = current
		hash = current.PreHash
	}
}

func validateStateDb(chain *FullBlockChain, trustHash *types.BlockHeader) (err error) {
	printToConsole("Start validating state tree...")
	return
}

func printToConsole(msg string) {
	fmt.Println(msg)
	//notify.BUS.Publish(notify.MessageToConsole, &consoleMsg{msg})
}

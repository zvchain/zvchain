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

package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/peterh/liner"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/tvm"
)

type baseCmd struct {
	name string
	help string
	fs   *flag.FlagSet
}

func genbaseCmd(n string, h string) *baseCmd {
	return &baseCmd{
		name: n,
		help: h,
		fs:   flag.NewFlagSet(n, flag.ContinueOnError),
	}
}

type newAccountCmd struct {
	baseCmd
	password string
	miner    bool
}

func genNewAccountCmd() *newAccountCmd {
	c := &newAccountCmd{
		baseCmd: *genbaseCmd("newaccount", "create account"),
	}
	c.fs.StringVar(&c.password, "password", "", "password for the account")
	c.fs.BoolVar(&c.miner, "miner", false, "create the account for miner if set")
	return c
}

func (c *newAccountCmd) parse(args []string) bool {
	err := c.fs.Parse(args)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.password) == "" {
		fmt.Println("please input the password")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type unlockCmd struct {
	baseCmd
	addr string
}

func genUnlockCmd() *unlockCmd {
	c := &unlockCmd{
		baseCmd: *genbaseCmd("unlock", "unlock the account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *unlockCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type balanceCmd struct {
	baseCmd
	addr string
}

func genBalanceCmd() *balanceCmd {
	c := &balanceCmd{
		baseCmd: *genbaseCmd("balance", "get the balance of the current unlocked account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *balanceCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type nonceCmd struct {
	baseCmd
	addr string
}

func genNonceCmd() *nonceCmd {
	c := &nonceCmd{
		baseCmd: *genbaseCmd("nonce", "get the nonce of the current unlocked account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *nonceCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type minerInfoCmd struct {
	baseCmd
	addr string
}

func genMinerInfoCmd() *minerInfoCmd {
	c := &minerInfoCmd{
		baseCmd: *genbaseCmd("minerinfo", "get the info of the miner"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the miner address")
	return c
}

func (c *minerInfoCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type connectCmd struct {
	baseCmd
	host string
	port int
}

func genConnectCmd() *connectCmd {
	c := &connectCmd{
		baseCmd: *genbaseCmd("connect", "connect to one tas node"),
	}
	c.fs.StringVar(&c.host, "host", "", "the node ip")
	c.fs.IntVar(&c.port, "port", 8101, "the node port, default is 8101")
	return c
}

func (c *connectCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.host) == "" {
		fmt.Println("please input the host,available testnet hosts are node1.taschain.cn,node2.taschain.cn,node3.taschain.cn,node4.taschain.cn,node5.taschain.cn")
		c.fs.PrintDefaults()
		return false
	}
	if c.port == 0 {
		fmt.Println("please input the port")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type txCmd struct {
	baseCmd
	hash     string
	executed bool
}

func genTxCmd() *txCmd {
	c := &txCmd{
		baseCmd: *genbaseCmd("tx", "get transaction detail"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex transaction hash")
	c.fs.BoolVar(&c.executed, "executed", false, "get executed transaction detail")
	return c
}

func (c *txCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.hash) == "" {
		fmt.Println("please input the transaction hash")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type blockCmd struct {
	baseCmd
	hash   string
	height uint64
}

func genBlockCmd() *blockCmd {
	c := &blockCmd{
		baseCmd: *genbaseCmd("block", "get block detail"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex block hash")
	c.fs.Uint64Var(&c.height, "height", 0, "the block height")
	return c
}

func (c *blockCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

type gasBaseCmd struct {
	baseCmd
	gaslimit    uint64
	gasPriceStr string
	gasPrice    uint64
}

func genGasBaseCmd(n string, h string) *gasBaseCmd {
	c := &gasBaseCmd{
		baseCmd: *genbaseCmd(n, h),
	}
	return c
}

func (c *gasBaseCmd) parseGasPrice() bool {
	gp, err := common.ParseCoin(c.gasPriceStr)
	if err != nil {
		fmt.Println(fmt.Sprintf("%v:%v, correct example: 100RA,100kRA,1mRA,1TAS", err, c.gasPriceStr))
		return false
	}
	c.gasPrice = gp
	return true
}

func (c *gasBaseCmd) initBase() {
	c.fs.Uint64Var(&c.gaslimit, "gaslimit", 1000, "gas limit, default 1000")
	c.fs.StringVar(&c.gasPriceStr, "gasprice", "100RA", "gas price, default 100RA")
}

type sendTxCmd struct {
	gasBaseCmd
	to           string
	value        float64
	data         string
	nonce 	     uint64
	contractName string
	contractPath string
	txType       int
}

func genSendTxCmd() *sendTxCmd {
	c := &sendTxCmd{
		gasBaseCmd: *genGasBaseCmd("sendtx", "send a transaction to the tas system"),
	}
	c.initBase()
	c.fs.StringVar(&c.to, "to", "", "the transaction receiver address")
	c.fs.Float64Var(&c.value, "value", 0.0, "transfer value in tas unit")
	c.fs.StringVar(&c.data, "data", "", "transaction data")
	c.fs.Uint64Var(&c.nonce, "nonce", 0, "nonce, optional. will use default nonce on chain if not specified")
	c.fs.StringVar(&c.contractName, "contractname", "", "the name of the contract.")
	c.fs.StringVar(&c.contractPath, "contractpath", "", "the path to the contract file.")
	c.fs.IntVar(&c.txType, "type", 0, "transaction type: 0=general tx, 1=contract create, 2=contract call, 3=bonus, 4=miner apply,5=miner abort, 6=miner refund")
	return c
}

func (c *sendTxCmd) toTxRaw() *txRawData {
	return &txRawData{
		Target:   c.to,
		Value:    common.Value2RA(c.value),
		TxType:   c.txType,
		Data:     c.data,
		Gas:      c.gaslimit,
		Gasprice: c.gasPrice,
		Nonce:    c.nonce,
	}
}

func (c *sendTxCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.to) == "" {
		fmt.Println("please input the target address")
		c.fs.PrintDefaults()
		return false
	}
	if !c.parseGasPrice() {
		return false
	}

	if c.txType == 1 { // Release contract preprocessing
		if strings.TrimSpace(c.contractName) == "" { // Contract name is not empty
			fmt.Println("please input the contractName")
			c.fs.PrintDefaults()
			return false
		}

		if strings.TrimSpace(c.contractPath) == "" { // Contract file path is not empty
			fmt.Println("please input the contractPath")
			c.fs.PrintDefaults()
			return false
		}

		f, err := ioutil.ReadFile(c.contractPath) // Read file
		if err != nil {
			fmt.Println("read the "+c.contractPath+"file failed ", err)
			c.fs.PrintDefaults()
			return false
		}
		contract := tvm.Contract{Code: string(f), ContractName: c.contractName, ContractAddress: nil}

		jsonBytes, errMarsh := json.Marshal(contract)
		if errMarsh != nil {
			fmt.Println("Marshal contract failed: ", errMarsh)
			c.fs.PrintDefaults()
			return false
		}
		c.data = string(jsonBytes)

	} else if c.txType == 2 { // Release contract preprocessing
		if strings.TrimSpace(c.contractPath) == "" { // Contract file path is not empty
			fmt.Println("please input the contractPath")
			c.fs.PrintDefaults()
			return false
		}

		f, err := ioutil.ReadFile(c.contractPath) // Read file
		if err != nil {
			fmt.Println("read the "+c.contractPath+"file failed ", err)
			c.fs.PrintDefaults()
			return false
		}
		c.data = string(f)
	}

	return true
}

type minerApplyCmd struct {
	gasBaseCmd
	stake uint64
	mtype int
}

func genMinerApplyCmd() *minerApplyCmd {
	c := &minerApplyCmd{
		gasBaseCmd: *genGasBaseCmd("minerapply", "apply to be a miner"),
	}
	c.initBase()
	c.fs.Uint64Var(&c.stake, "stake", 100, "freeze stake of tas, default 100TAS")
	c.fs.IntVar(&c.mtype, "type", 0, "apply miner type: 0=verify node, 1=proposal node, default 0")
	return c
}

func (c *minerApplyCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return c.parseGasPrice()
}

type minerAbortCmd struct {
	gasBaseCmd
	mtype int
}

func genMinerAbortCmd() *minerAbortCmd {
	c := &minerAbortCmd{
		gasBaseCmd: *genGasBaseCmd("minerabort", "abort a miner identifier"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "abort miner type: 0=verify node, 1=proposal node, default 0")
	return c
}

func (c *minerAbortCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return c.parseGasPrice()
}

type minerRefundCmd struct {
	gasBaseCmd
	mtype int
	addr  string
}

func genMinerRefundCmd() *minerRefundCmd {
	c := &minerRefundCmd{
		gasBaseCmd: *genGasBaseCmd("minerrefund", "apply to refund the miner freeze stake"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "refund miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.StringVar(&c.addr, "addr", "", "refund miner addr, default self")
	return c
}

func (c *minerRefundCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return c.parseGasPrice()
}

type minerStakeCmd struct {
	gasBaseCmd
	mtype int
	addr  string
	value uint64
}

func genMinerStakeCmd() *minerStakeCmd {
	c := &minerStakeCmd{
		gasBaseCmd: *genGasBaseCmd("minerstake", "stake TAS to an address"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "receiver's type staked to: 0=verify node, 1=proposal node, default 0")
	c.fs.StringVar(&c.addr, "addr", "", "receiver's addr staked to, default self")
	c.fs.Uint64Var(&c.value, "value", 0, "stake value, default 0TAS")
	return c
}

func (c *minerStakeCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return c.parseGasPrice()
}

type minerCancelStakeCmd struct {
	gasBaseCmd
	mtype int
	addr  string
	value uint64
}

func genMinerCancelStakeCmd() *minerCancelStakeCmd {
	c := &minerCancelStakeCmd{
		gasBaseCmd: *genGasBaseCmd("minercancelstake", "cancel stake TAS of an address"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "receiver's type: 0=verify node, 1=proposal node, default 0")
	c.fs.StringVar(&c.addr, "addr", "", "receiver's addr, default self")
	c.fs.Uint64Var(&c.value, "value", 0, "refund value, default 0TAS")
	return c
}

func (c *minerCancelStakeCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return c.parseGasPrice()
}

type viewContractCmd struct {
	baseCmd
	addr string
}

func genViewContractCmd() *viewContractCmd {
	c := &viewContractCmd{
		baseCmd: *genbaseCmd("viewcontract", "view contract data"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "address of the contract")
	return c
}

func (c *viewContractCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if c.addr == "" {
		fmt.Println("please input the contract address")
		return false
	}
	return true
}

var cmdNewAccount = genNewAccountCmd()
var cmdExit = genbaseCmd("exit", "quit  gtas")
var cmdHelp = genbaseCmd("help", "show help info")
var cmdAccountList = genbaseCmd("accountlist", "list the account of the keystore")
var cmdUnlock = genUnlockCmd()
var cmdBalance = genBalanceCmd()
var cmdNonce = genNonceCmd()
var cmdAccountInfo = genbaseCmd("accountinfo", "get the info of the current unlocked account")
var cmdDelAccount = genbaseCmd("delaccount", "delete the info of the current unlocked account")
var cmdMinerInfo = genMinerInfoCmd()
var cmdConnect = genConnectCmd()
var cmdBlockHeight = genbaseCmd("blockheight", "the current block height")
var cmdGroupHeight = genbaseCmd("groupheight", "the current group height")
var cmdTx = genTxCmd()
var cmdBlock = genBlockCmd()
var cmdSendTx = genSendTxCmd()

var cmdMinerApply = genMinerApplyCmd()
var cmdMinerAbort = genMinerAbortCmd()
var cmdMinerRefund = genMinerRefundCmd()
var cmdMinerStake = genMinerStakeCmd()
var cmdMinerCancelStake = genMinerCancelStakeCmd()
var cmdViewContract = genViewContractCmd()

var list = make([]*baseCmd, 0)

func init() {
	list = append(list, cmdHelp)
	list = append(list, &cmdNewAccount.baseCmd)
	list = append(list, cmdAccountList)
	list = append(list, &cmdUnlock.baseCmd)
	list = append(list, &cmdBalance.baseCmd)
	list = append(list, &cmdNonce.baseCmd)
	list = append(list, cmdAccountInfo)
	list = append(list, cmdDelAccount)
	list = append(list, &cmdMinerInfo.baseCmd)
	list = append(list, &cmdConnect.baseCmd)
	list = append(list, cmdBlockHeight)
	list = append(list, cmdGroupHeight)
	list = append(list, &cmdTx.baseCmd)
	list = append(list, &cmdBlock.baseCmd)
	list = append(list, &cmdSendTx.baseCmd)
	list = append(list, &cmdMinerApply.baseCmd)
	list = append(list, &cmdMinerAbort.baseCmd)
	list = append(list, &cmdMinerRefund.baseCmd)
	list = append(list, &cmdViewContract.baseCmd)
	list = append(list, &cmdMinerCancelStake.baseCmd)
	list = append(list, &cmdMinerStake.baseCmd)
	list = append(list, cmdExit)
}

func Usage() {
	fmt.Println("Usage:")
	for _, cmd := range list {
		fmt.Println(" " + cmd.name + ":\t" + cmd.help)
		cmd.fs.PrintDefaults()
		fmt.Print("\n")
	}
}

func ConsoleInit(keystore, host string, port int, show bool, rpcport int) error {
	aop, err := initAccountManager(keystore, false)
	if err != nil {
		return err
	}
	chainop := InitRemoteChainOp(host, port, show, aop)
	if chainop.base != "" {

	}

	if rpcport > 0 {
		ws := NewWalletServer(rpcport, aop)
		if err := ws.Start(); err != nil {
			return err
		}
	}

	loop(aop, chainop)

	return nil
}

func handleCmd(handle func() *Result) {
	ret := handle()
	if !ret.IsSuccess() {
		fmt.Println(ret.Message)
	} else {
		bs, err := json.MarshalIndent(ret, "", "\t")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(bs))
		}
	}
}

func unlockLoop(cmd *unlockCmd, acm accountOp) {
	c := 0

	for c < 3 {
		c++

		bs, err := gopass.GetPasswdPrompt("please input password: ", true, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		ret := acm.UnLock(cmd.addr, string(bs))
		if ret.IsSuccess() {
			fmt.Printf("unlock will last %v secs:%v\n", accountUnLockTime.Seconds(), cmd.addr)
			break
		} else {
			fmt.Fprintln(os.Stderr, ret.Message)
		}
	}
}

func loop(acm accountOp, chainOp chainOp) {

	re, _ := regexp.Compile("\\s{2，}")

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	items := make([]string, len(list))
	for idx, cmd := range list {
		items[idx] = cmd.name
	}

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range items {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	for {
		ep := chainOp.Endpoint()
		if ep == ":0" {
			ep = "not connected"
		}
		input, err := line.Prompt(fmt.Sprintf("gtas:%v > ", ep))
		if err != nil {
			if err == liner.ErrPromptAborted {
				line.Close()
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, err)
		}
		input = strings.TrimSpace(input)
		input = re.ReplaceAllString(input, " ")
		inputArr := strings.Split(input, " ")

		line.AppendHistory(input)

		if len(inputArr) == 0 {
			continue
		}
		cmdStr := inputArr[0]
		args := inputArr[1:]

		switch cmdStr {
		case "":
			break
		case cmdNewAccount.name:
			cmd := genNewAccountCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return acm.NewAccount(cmd.password, cmd.miner)
				})
			}
		case cmdExit.name, "quit":
			fmt.Printf("thank you, bye\n")
			line.Close()
			os.Exit(0)
		case cmdHelp.name:
			Usage()
		case cmdAccountList.name:
			handleCmd(func() *Result {
				return acm.AccountList()
			})
		case cmdUnlock.name:
			cmd := genUnlockCmd()
			if cmd.parse(args) {
				unlockLoop(cmd, acm)
			}
		case cmdAccountInfo.name:
			handleCmd(func() *Result {
				return acm.AccountInfo()
			})
		case cmdDelAccount.name:
			handleCmd(func() *Result {
				return acm.DeleteAccount()
			})
		case cmdConnect.name:
			cmd := genConnectCmd()
			if cmd.parse(args) {
				chainOp.Connect(cmd.host, cmd.port)
			}

		case cmdBalance.name:
			cmd := genBalanceCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.Balance(cmd.addr)
				})
			}
		case cmdNonce.name:
			cmd := genNonceCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.Nonce(cmd.addr)
				})
			}

		case cmdMinerInfo.name:
			cmd := genMinerInfoCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.MinerInfo(cmd.addr)
				})
			}
		case cmdBlockHeight.name:
			handleCmd(func() *Result {
				return chainOp.BlockHeight()
			})
		case cmdGroupHeight.name:
			handleCmd(func() *Result {
				return chainOp.GroupHeight()
			})
		case cmdTx.name:
			cmd := genTxCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					if cmd.executed {
						return chainOp.TxReceipt(cmd.hash)
					}
					return chainOp.TxInfo(cmd.hash)
				})
			}
		case cmdBlock.name:
			cmd := genBlockCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					if cmd.hash != "" {
						return chainOp.BlockByHash(cmd.hash)
					}
					return chainOp.BlockByHeight(cmd.height)
				})
			}
		case cmdSendTx.name:
			cmd := genSendTxCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.SendRaw(cmd.toTxRaw())
				})
			}
		case cmdMinerApply.name:
			cmd := genMinerApplyCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.ApplyMiner(cmd.mtype, cmd.stake, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdMinerAbort.name:
			cmd := genMinerAbortCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.AbortMiner(cmd.mtype, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdMinerRefund.name:
			cmd := genMinerRefundCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.RefundMiner(cmd.mtype, cmd.addr, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdMinerStake.name:
			cmd := genMinerStakeCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.MinerStake(cmd.mtype, cmd.addr, cmd.value, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdMinerCancelStake.name:
			cmd := genMinerCancelStakeCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.MinerCancelStake(cmd.mtype, cmd.addr, cmd.value, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdViewContract.name:
			cmd := genViewContractCmd()
			if cmd.parse(args) {
				handleCmd(func() *Result {
					return chainOp.ViewContract(cmd.addr)
				})
			}
		default:
			fmt.Printf("not supported command %v\n", cmdStr)
			Usage()
		}
	}
}

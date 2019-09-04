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
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
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

func genBaseCmd(n string, h string) *baseCmd {
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

func output(msg ...interface{}) {
	fmt.Println(msg...)
}

func outputJSONErr(result *ErrorResult) {
	bs, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		output(err.Error())
	} else {
		output(string(bs))
	}
}

func genNewAccountCmd() *newAccountCmd {
	c := &newAccountCmd{
		baseCmd: *genBaseCmd("newaccount", "create account"),
	}
	c.fs.StringVar(&c.password, "password", "", "password for the account")
	c.fs.BoolVar(&c.miner, "miner", false, "create the account for miner if set")
	return c
}

func (c *newAccountCmd) parse(args []string) bool {
	err := c.fs.Parse(args)
	if err != nil {
		output(err.Error())
		return false
	}
	pass := strings.TrimSpace(c.password)
	if len(pass) == 0 {
		output("Please input password")
		return false
	}
	if len(pass) > common.MaxPasswordLength || len(pass) < common.MinPasswordLength {
		fmt.Printf("password length should between %d-%d \n", common.MinPasswordLength, common.MaxPasswordLength)
		return false
	}
	return true
}

type unlockCmd struct {
	baseCmd
	addr     string
	duration uint
}

func genUnlockCmd() *unlockCmd {
	c := &unlockCmd{
		baseCmd: *genBaseCmd("unlock", "unlock the account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	c.fs.UintVar(&c.duration, "duration", 120, "unlock duration, default 120 secs")
	return c
}

func (c *unlockCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		output("please input the address")
		c.fs.PrintDefaults()
		return false
	}

	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
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
		baseCmd: *genBaseCmd("balance", "get the balance of the current unlocked account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *balanceCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		output("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
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
		baseCmd: *genBaseCmd("nonce", "get the nonce of the current unlocked account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *nonceCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		output("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	return true
}

type minerPoolInfoCmd struct {
	baseCmd
	addr string
}

func genMinerPoolInfoCmd() *minerPoolInfoCmd {
	c := &minerPoolInfoCmd{
		baseCmd: *genBaseCmd("minerpoolinfo", "view miner pool info by address"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "miner pool info address")
	return c
}

func (c *minerPoolInfoCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	c.addr = strings.TrimSpace(c.addr)
	if c.addr == "" {
		output("please input the address")
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	return true
}

type voteMinerPoolCmd struct {
	gasBaseCmd
	addr string
}

func genVoteMinerPoolCmd() *voteMinerPoolCmd {
	c := &voteMinerPoolCmd{
		gasBaseCmd: *genGasBaseCmd("voteminerpool", "only guard miner node can for vote miner pool, each guard miner node can only vote once"),
	}
	c.initBase()
	c.fs.StringVar(&c.addr, "addr", "", "your vote address")
	return c
}

func (c *voteMinerPoolCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	c.addr = strings.TrimSpace(c.addr)
	if c.addr == "" {
		output("please input the address")
		return false
	}
	if !common.ValidateAddress(c.addr) {
		output("Wrong address format")
		return false
	}
	return c.parseGasPrice()
}

type applyGuardMinerCmd struct {
	gasBaseCmd
}

func genApplyGuardMinerCmd() *applyGuardMinerCmd {
	c := &applyGuardMinerCmd{
		gasBaseCmd: *genGasBaseCmd("applyguard", "apply guard miner node"),
	}
	c.initBase()
	return c
}

func (c *applyGuardMinerCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	return c.parseGasPrice()
}

type minerInfoCmd struct {
	baseCmd
	addr   string
	detail string
}

func genMinerInfoCmd() *minerInfoCmd {
	c := &minerInfoCmd{
		baseCmd: *genBaseCmd("minerinfo", "get the info of the miner"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the miner address")
	c.fs.StringVar(&c.detail, "detail", "", "show the details of the stake from the given address, no details shows if empty")
	return c
}

func (c *minerInfoCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		output("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	if c.detail != "" && !common.ValidateAddress(c.detail) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
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
		baseCmd: *genBaseCmd("connect", "connect to one ZV node"),
	}
	c.fs.StringVar(&c.host, "host", "", "the node ip")
	c.fs.IntVar(&c.port, "port", 8101, "the node port, default is 8101")
	return c
}

func (c *connectCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.host) == "" {
		output("please input the host,available testnet hosts are node1.taschain.cn,node2.taschain.cn,node3.taschain.cn,node4.taschain.cn,node5.taschain.cn")
		c.fs.PrintDefaults()
		return false
	}
	if c.port == 0 {
		output("please input the port")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type txCmd struct {
	baseCmd
	hash string
}

func genTxCmd() *txCmd {
	c := &txCmd{
		baseCmd: *genBaseCmd("tx", "get transaction detail"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex transaction hash")
	return c
}

func (c *txCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.hash) == "" {
		output("please input the transaction hash")
		c.fs.PrintDefaults()
		return false
	}
	if !validateHash(c.hash) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong hash format")))
		return false
	}
	return true
}

type receiptCmd struct {
	baseCmd
	hash string
}

func genReceiptCmd() *receiptCmd {
	c := &receiptCmd{
		baseCmd: *genBaseCmd("receipt", "get transaction receipt"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex transaction hash")
	return c
}

func (c *receiptCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if strings.TrimSpace(c.hash) == "" {
		output("please input the transaction hash")
		c.fs.PrintDefaults()
		return false
	}
	if !validateHash(c.hash) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong hash format")))
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
		baseCmd: *genBaseCmd("block", "get block detail"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex block hash")
	c.fs.Uint64Var(&c.height, "height", 0, "the block height")
	return c
}

func (c *blockCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if len(c.hash) > 0 {
		if !validateHash(c.hash) {
			outputJSONErr(opErrorRes(fmt.Errorf("wrong hash format")))
			return false
		}
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
		baseCmd: *genBaseCmd(n, h),
	}
	return c
}

func (c *gasBaseCmd) parseGasPrice() bool {
	gp, err := common.ParseCoin(c.gasPriceStr)
	if err != nil {
		outputJSONErr(opErrorRes(fmt.Errorf("%v:%v, correct example: 100RA,100kRA,1mRA,1ZVC", err, c.gasPriceStr)))
		return false
	}
	c.gasPrice = gp
	return true
}

func (c *gasBaseCmd) initBase() {
	c.fs.Uint64Var(&c.gaslimit, "gaslimit", 3000, "gas limit, default 3000")
	c.fs.StringVar(&c.gasPriceStr, "gasprice", "500RA", "gas price, default 500RA")
}

type sendTxCmd struct {
	gasBaseCmd
	to           string
	value        string
	data         string
	nonce        uint64
	contractName string
	contractPath string
	txType       int
	extraData    string
}

func genSendTxCmd() *sendTxCmd {
	c := &sendTxCmd{
		gasBaseCmd: *genGasBaseCmd("sendtx", "send a transaction to the ZV system"),
	}
	c.initBase()
	c.fs.StringVar(&c.to, "to", "", "the transaction receiver address")
	c.fs.StringVar(&c.value, "value", "", "transfer value in ZVC unit")
	c.fs.StringVar(&c.data, "data", "", "transaction data")
	c.fs.StringVar(&c.extraData, "extra", "", "transaction extra data, user defined")
	c.fs.Uint64Var(&c.nonce, "nonce", 0, "nonce, optional. will use default nonce on chain if not specified")
	c.fs.StringVar(&c.contractName, "contractname", "", "the name of the contract.")
	c.fs.StringVar(&c.contractPath, "contractpath", "", "the path to the contract file.")
	c.fs.IntVar(&c.txType, "type", 0, "transaction type: 0=general tx, 1=contract create, 2=contract call, 4=stake add ,5=miner abort, 6=stake reduce, 7=stake refund")
	return c
}

func (c *sendTxCmd) toTxRaw() *TxRawData {
	value, _ := parseRaFromString(c.value)
	return &TxRawData{
		Target:    c.to,
		Value:     value,
		TxType:    c.txType,
		Data:      []byte(c.data),
		GasLimit:  c.gaslimit,
		GasPrice:  c.gasPrice,
		Nonce:     c.nonce,
		ExtraData: []byte(c.extraData),
	}
}

func (c *sendTxCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if !validateTxType(c.txType) {
		outputJSONErr(opErrorRes(fmt.Errorf("not supported transaction type")))
		return false
	}
	if c.txType == types.TransactionTypeTransfer || c.txType == types.TransactionTypeContractCall {
		if strings.TrimSpace(c.to) == "" {
			output("please input the target address")
			c.fs.PrintDefaults()
			return false
		} else {
			if !common.ValidateAddress(strings.TrimSpace(c.to)) {
				outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
				return false
			}
		}
	}

	if !c.parseGasPrice() {
		return false
	}

	if _, err := parseRaFromString(c.value); err != nil {
		outputJSONErr(opErrorRes(err))
		return false
	}

	if c.txType == types.TransactionTypeContractCreate { // Release contract preprocessing
		if strings.TrimSpace(c.contractName) == "" { // Contract name is not empty
			output("please input the contractName")
			c.fs.PrintDefaults()
			return false
		}

		if strings.TrimSpace(c.contractPath) == "" { // Contract file path is not empty
			output("please input the contractPath")
			c.fs.PrintDefaults()
			return false
		}

		f, err := ioutil.ReadFile(c.contractPath) // Read file
		if err != nil {
			outputJSONErr(opErrorRes(fmt.Errorf("read the "+c.contractPath+"file failed ", err)))
			c.fs.PrintDefaults()
			return false
		}
		contract := tvm.Contract{Code: string(f), ContractName: c.contractName, ContractAddress: nil}

		jsonBytes, errMarsh := json.Marshal(contract)
		if errMarsh != nil {
			outputJSONErr(opErrorRes(fmt.Errorf("marshal contract failed: %s", errMarsh.Error())))
			c.fs.PrintDefaults()
			return false
		}
		c.data = string(jsonBytes)

	} else if c.txType == types.TransactionTypeContractCall { // Release contract preprocessing
		if strings.TrimSpace(c.contractPath) == "" { // Contract file path is not empty
			output("please input the contractPath")
			c.fs.PrintDefaults()
			return false
		}

		f, err := ioutil.ReadFile(c.contractPath) // Read file
		if err != nil {
			outputJSONErr(opErrorRes(fmt.Errorf("read the "+c.contractPath+"file failed ", err)))
			c.fs.PrintDefaults()
			return false
		}
		c.data = string(f)
	}

	return true
}

func parseRaFromString(number string) (uint64, error) {
	if len(number) == 0 {
		return 0, nil
	}

	numberSplit := strings.Split(number, ".")
	lengthOfNumber := len(numberSplit)
	if lengthOfNumber > 2 || lengthOfNumber < 1 {
		return 0, fmt.Errorf("illegal number")
	}

	var numReg = regexp.MustCompile("^[0-9]{1,10}$") //check the format
	if !numReg.MatchString(numberSplit[0]) {
		return 0, fmt.Errorf("illegal number")
	}

	bigNumber, err := strconv.ParseUint(numberSplit[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var decimal uint64
	if lengthOfNumber == 2 {
		var digital = regexp.MustCompile("^[0-9]{1,9}$") //check the format
		if !digital.MatchString(numberSplit[1]) {
			return 0, fmt.Errorf("illegal number")
		}
		realNumber := numberSplit[1]
		for i := len(numberSplit[1]); i < 9; i++ {
			realNumber += "0"
		}
		decimal, err = strconv.ParseUint(realNumber, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return bigNumber*common.ZVC + decimal, nil
}

type stakeAddCmd struct {
	gasBaseCmd
	value  uint64
	mtype  int
	target string
}

func genStakeAddCmd() *stakeAddCmd {
	c := &stakeAddCmd{
		gasBaseCmd: *genGasBaseCmd("stakeadd", "add value for the target miner"),
	}
	c.initBase()
	c.fs.Uint64Var(&c.value, "value", 500, "freeze value of ZVC, default 500ZVC")
	c.fs.IntVar(&c.mtype, "type", 0, "apply miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.StringVar(&c.target, "target", "", "value add target address, default the operator if not specified")
	return c
}

func (c *stakeAddCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if !validateMinerType(c.mtype) {
		outputJSONErr(opErrorRes(fmt.Errorf("unsupported miner type")))
		return false
	}
	if len(strings.TrimSpace(c.target)) > 0 {
		if !common.ValidateAddress(c.target) {
			outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
			return false
		}
	}
	return c.parseGasPrice()
}

type minerAbortCmd struct {
	gasBaseCmd
	mtype      int
	forceAbort bool
}

func genMinerAbortCmd() *minerAbortCmd {
	c := &minerAbortCmd{
		gasBaseCmd: *genGasBaseCmd("minerabort", "abort a miner identifier"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "abort miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.BoolVar(&c.forceAbort, "f", false, "operation won't success if the miner was currently selected to join a group if not specified")
	return c
}

func (c *minerAbortCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if !validateMinerType(c.mtype) {
		outputJSONErr(opErrorRes(fmt.Errorf("unsupported miner type")))
		return false
	}
	return c.parseGasPrice()
}

type changeGuardNodeCmd struct {
	gasBaseCmd
	mode int
}

func genChangeGuardNodeCmd() *changeGuardNodeCmd {
	c := &changeGuardNodeCmd{
		gasBaseCmd: *genGasBaseCmd("changemode", "only can changed by fund guard node"),
	}
	c.initBase()
	c.fs.IntVar(&c.mode, "mode", 0, "mode type :0=6+5, 1= 6+6")
	return c
}

func (c *changeGuardNodeCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if !validateFundGuardMode(c.mode) {
		output(fmt.Sprintf("Unsupported mode type %d", c.mode))
		return false
	}
	return c.parseGasPrice()
}

type stakeRefundCmd struct {
	gasBaseCmd
	mtype  int
	target string
}

func genStakeRefundCmd() *stakeRefundCmd {
	c := &stakeRefundCmd{
		gasBaseCmd: *genGasBaseCmd("stakerefund", "apply to refund the miner freeze value"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "refund miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.StringVar(&c.target, "target", "", "refund target address, default the operator if not specified")
	return c
}

func (c *stakeRefundCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if c.target != "" && !common.ValidateAddress(c.target) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	if !validateMinerType(c.mtype) {
		outputJSONErr(opErrorRes(fmt.Errorf("unsupported miner type")))
		return false
	}
	return c.parseGasPrice()
}

type stakeReduceCmd struct {
	gasBaseCmd
	mtype  int
	target string
	value  uint64
}

func genStakeReduceCmd() *stakeReduceCmd {
	c := &stakeReduceCmd{
		gasBaseCmd: *genGasBaseCmd("stakereduce", "reduce value of the given address"),
	}
	c.initBase()
	c.fs.IntVar(&c.mtype, "type", 0, "receiver's type: 0=verify node, 1=proposal node, default 0")
	c.fs.StringVar(&c.target, "target", "", "reduce target address, default the operator if not specified")
	c.fs.Uint64Var(&c.value, "value", 0, "reduce value, default 0ZVC")
	return c
}

func (c *stakeReduceCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if c.target != "" && !common.ValidateAddress(c.target) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	if !validateMinerType(c.mtype) {
		outputJSONErr(opErrorRes(fmt.Errorf("unsupported miner type")))
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
		baseCmd: *genBaseCmd("viewcontract", "view contract data"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "address of the contract")
	return c
}

func (c *viewContractCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if c.addr == "" {
		output("please input the contract address")
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	return true
}

type importKeyCmd struct {
	baseCmd
	key      string
	password string
	miner    bool
}

func genImportKeyCmd() *importKeyCmd {
	c := &importKeyCmd{
		baseCmd: *genBaseCmd("importkey", "import private key"),
	}
	c.fs.StringVar(&c.key, "privatekey", "", "private key imported for the account")
	c.fs.StringVar(&c.password, "password", "", "password for the account")
	c.fs.BoolVar(&c.miner, "miner", false, "create the account for miner if set")
	return c
}

func (c *importKeyCmd) parse(args []string) bool {
	err := c.fs.Parse(args)
	if err != nil {
		output(err.Error())
		return false
	}
	key := strings.TrimSpace(c.key)
	if len(key) == 0 {
		output("Please input private key")
		return false
	}
	if !validateKey(key) {
		outputJSONErr(opErrorRes(fmt.Errorf("private key is invalid")))
		return false
	}
	pass := strings.TrimSpace(c.password)
	if len(pass) == 0 {
		output("Please input password")
		return false
	}
	if len(pass) > common.MaxPasswordLength || len(pass) < common.MinPasswordLength {
		fmt.Printf("password length should between %d-%d \n", common.MinPasswordLength, common.MaxPasswordLength)
		return false
	}
	return true
}

type exportKeyCmd struct {
	baseCmd
	addr string
}

func genExportKeyCmd() *exportKeyCmd {
	c := &exportKeyCmd{
		baseCmd: *genBaseCmd("exportkey", "export private key"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "address of the account")
	return c
}

func (c *exportKeyCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if c.addr == "" {
		output("please input the account address")
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	return true
}

type groupCheckCmd struct {
	baseCmd
	addr string
}

func genGroupCheckCmd() *groupCheckCmd {
	c := &groupCheckCmd{
		baseCmd: *genBaseCmd("groupcheck", "check joining group info of the given miner address"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the address of miner")
	return c
}

func (c *groupCheckCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		output(err.Error())
		return false
	}
	if c.addr == "" {
		output("please input the address")
		return false
	}
	if !common.ValidateAddress(c.addr) {
		outputJSONErr(opErrorRes(fmt.Errorf("wrong address format")))
		return false
	}
	return true
}

var cmdNewAccount = genNewAccountCmd()
var cmdExit = genBaseCmd("exit", "quit  gzv")
var cmdHelp = genBaseCmd("help", "show help info")
var cmdAccountList = genBaseCmd("accountlist", "list the account of the keystore")
var cmdUnlock = genUnlockCmd()
var cmdBalance = genBalanceCmd()
var cmdNonce = genNonceCmd()
var cmdMinerPoolInfo = genMinerPoolInfoCmd()
var cmdAccountInfo = genBaseCmd("accountinfo", "get the info of the current unlocked account")
var cmdDelAccount = genBaseCmd("delaccount", "delete the info of the current unlocked account")
var cmdMinerInfo = genMinerInfoCmd()
var cmdConnect = genConnectCmd()
var cmdBlockHeight = genBaseCmd("blockheight", "the current block height")
var cmdGroupHeight = genBaseCmd("groupheight", "the current group height")
var cmdTx = genTxCmd()
var cmdReceipt = genReceiptCmd()
var cmdBlock = genBlockCmd()
var cmdSendTx = genSendTxCmd()
var cmdApplyGuardMiner = genApplyGuardMinerCmd()
var cmdVoteMinerPool = genVoteMinerPoolCmd()

var cmdStakeAdd = genStakeAddCmd()
var cmdMinerAbort = genMinerAbortCmd()
var cmdChangeGuardNode = genChangeGuardNodeCmd()
var cmdStakeRefund = genStakeRefundCmd()
var cmdStakeReduce = genStakeReduceCmd()
var cmdViewContract = genViewContractCmd()

var cmdImportKey = genImportKeyCmd()
var cmdExportKey = genExportKeyCmd()
var cmdGroupCheck = genGroupCheckCmd()

var list = make([]*baseCmd, 0)

func init() {
	list = append(list, cmdHelp)
	list = append(list, &cmdNewAccount.baseCmd)
	list = append(list, cmdAccountList)
	list = append(list, &cmdUnlock.baseCmd)
	list = append(list, &cmdBalance.baseCmd)
	list = append(list, &cmdNonce.baseCmd)
	list = append(list, &cmdMinerPoolInfo.baseCmd)
	list = append(list, cmdAccountInfo)
	list = append(list, cmdDelAccount)
	list = append(list, &cmdMinerInfo.baseCmd)
	list = append(list, &cmdApplyGuardMiner.baseCmd)
	list = append(list, &cmdVoteMinerPool.baseCmd)
	list = append(list, &cmdConnect.baseCmd)
	list = append(list, cmdBlockHeight)
	list = append(list, cmdGroupHeight)
	list = append(list, &cmdTx.baseCmd)
	list = append(list, &cmdReceipt.baseCmd)
	list = append(list, &cmdBlock.baseCmd)
	list = append(list, &cmdSendTx.baseCmd)
	list = append(list, &cmdStakeAdd.baseCmd)
	list = append(list, &cmdMinerAbort.baseCmd)
	list = append(list, &cmdChangeGuardNode.baseCmd)
	list = append(list, &cmdStakeRefund.baseCmd)
	list = append(list, &cmdViewContract.baseCmd)
	list = append(list, &cmdStakeReduce.baseCmd)
	list = append(list, &cmdImportKey.baseCmd)
	list = append(list, &cmdExportKey.baseCmd)
	list = append(list, &cmdGroupCheck.baseCmd)
	list = append(list, cmdExit)
}

func Usage() {
	output("Usage:")
	for _, cmd := range list {
		output(" " + cmd.name + ":\t" + cmd.help)
		cmd.fs.PrintDefaults()
		fmt.Print("\n")
	}
}

func ConsoleInit(keystore, host string, port int, show bool, rpcport int) error {
	aop, err := initAccountManager(keystore, false, "")
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

func handleCmdForChain(handle func() *RPCResObjCmd) {
	res := handle()
	if res != nil {
		if res.Error != nil {
			bs, err := json.MarshalIndent(res.Error, "", "\t")
			if err != nil {
				output(err.Error())
			} else {
				output(string(bs))
			}
		} else {
			bs, err := json.MarshalIndent(res.Result, "", "\t")
			if err != nil {
				output(err.Error())
			} else {
				output(string(bs))
			}
		}
	}
}

func handleCmdForAccount(handle func() (interface{}, error)) {
	res, err := handle()
	if err != nil {
		output(err)
	} else {
		bs, err := json.MarshalIndent(res, "", "\t")
		if err != nil {
			output(err.Error())
		} else {
			output(string(bs))
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

		resErr := acm.UnLock(cmd.addr, string(bs), cmd.duration)
		if resErr == nil {
			fmt.Printf("unlock will last %v secs:%v\n", cmd.duration, cmd.addr)
			break
		} else {
			fmt.Fprintln(os.Stderr, resErr.Error())
		}
	}
}

func loop(acm accountOp, chainOp chainOp) {

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
		input, err := line.Prompt(fmt.Sprintf("gzv:%v > ", ep))
		if err != nil {
			if err == liner.ErrPromptAborted {
				line.Close()
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, err)
		}

		inputArr, err := parseCommandLine(input)
		if err != nil {
			fmt.Printf("%s", err.Error())
		}

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
				handleCmdForAccount(func() (interface{}, error) {
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
			handleCmdForAccount(func() (interface{}, error) {
				return acm.AccountList()
			})
		case cmdUnlock.name:
			cmd := genUnlockCmd()
			if cmd.parse(args) {
				unlockLoop(cmd, acm)
			}
		case cmdAccountInfo.name:
			handleCmdForAccount(func() (interface{}, error) {
				return acm.AccountInfo()
			})
		case cmdDelAccount.name:
			handleCmdForAccount(func() (interface{}, error) {
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
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.Balance(cmd.addr)
				})
			}
		case cmdNonce.name:
			cmd := genNonceCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.Nonce(cmd.addr)
				})
			}

		case cmdMinerInfo.name:
			cmd := genMinerInfoCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.MinerInfo(cmd.addr, cmd.detail)
				})
			}
		case cmdMinerPoolInfo.name:
			cmd := genMinerPoolInfoCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.MinerPoolInfo(cmd.addr)
				})
			}
		case cmdApplyGuardMiner.name:
			cmd := genApplyGuardMinerCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.ApplyGuardMiner(cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdVoteMinerPool.name:
			cmd := genVoteMinerPoolCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.VoteMinerPool(cmd.addr, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdBlockHeight.name:
			handleCmdForChain(func() *RPCResObjCmd {
				return chainOp.BlockHeight()
			})
		case cmdGroupHeight.name:
			handleCmdForChain(func() *RPCResObjCmd {
				return chainOp.GroupHeight()
			})
		case cmdTx.name:
			cmd := genTxCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.TxInfo(cmd.hash)
				})
			}
		case cmdReceipt.name:
			cmd := genReceiptCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.TxReceipt(cmd.hash)
				})
			}
		case cmdBlock.name:
			cmd := genBlockCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					if cmd.hash != "" {
						return chainOp.BlockByHash(cmd.hash)
					}
					return chainOp.BlockByHeight(cmd.height)
				})
			}
		case cmdSendTx.name:
			cmd := genSendTxCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.SendRaw(cmd.toTxRaw())
				})
			}
		case cmdStakeAdd.name:
			cmd := genStakeAddCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.StakeAdd(cmd.target, cmd.mtype, cmd.value, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdMinerAbort.name:
			cmd := genMinerAbortCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.MinerAbort(cmd.mtype, cmd.gaslimit, cmd.gasPrice, cmd.forceAbort)
				})
			}
		case cmdChangeGuardNode.name:
			cmd := genChangeGuardNodeCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.ChangeFundGuardMode(cmd.mode, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdStakeRefund.name:
			cmd := genStakeRefundCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.StakeRefund(cmd.target, cmd.mtype, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdStakeReduce.name:
			cmd := genStakeReduceCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.StakeReduce(cmd.target, cmd.mtype, cmd.value, cmd.gaslimit, cmd.gasPrice)
				})
			}
		case cmdViewContract.name:
			cmd := genViewContractCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.ViewContract(cmd.addr)
				})
			}
		case cmdImportKey.name:
			cmd := genImportKeyCmd()
			if cmd.parse(args) {
				handleCmdForAccount(func() (interface{}, error) {
					return acm.NewAccountByImportKey(cmd.key, cmd.password, cmd.miner)
				})
			}
		case cmdExportKey.name:
			cmd := genExportKeyCmd()
			if cmd.parse(args) {
				handleCmdForAccount(func() (interface{}, error) {
					return acm.ExportKey(cmd.addr)
				})
			}
		case cmdGroupCheck.name:
			cmd := genGroupCheckCmd()
			if cmd.parse(args) {
				handleCmdForChain(func() *RPCResObjCmd {
					return chainOp.GroupCheck(cmd.addr)
				})
			}
		default:
			fmt.Printf("not supported command %v\n", cmdStr)
			Usage()
		}
	}
}

func parseCommandLine(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, fmt.Errorf("Unclosed quote in command line: %s", command)
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}

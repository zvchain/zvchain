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
	"errors"
	"fmt"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware"
	"os"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/network"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/zvchain/zvchain/consensus/mediator"
	chandler "github.com/zvchain/zvchain/consensus/net"

	"encoding/json"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/monitor"
)

const (
	// Section is default section configuration
	Section = "gtas"
	// ini configuration file instance section
	instanceSection = "instance"
	// The key below the instance section
	indexKey = "index"
	// ini configuration file chain section
	chainSection = "chain"
	// The key below the chain section
	databaseKey = "database"
	// ini configuration file statistics section
	statisticsSection = "statistics"
)

type Gtas struct {
	inited       bool
	account      Account
	config       *minerConfig
	rpcInstances []rpcApi
}

// miner start miner node
func (gtas *Gtas) miner(cfg *minerConfig) error {
	gtas.config = cfg
	gtas.runtimeInit()
	err := gtas.fullInit()
	if err != nil {
		return err
	}
	if cfg.rpcEnable() {
		err = gtas.startRPC()
		if err != nil {
			return err
		}
	}
	ok := mediator.StartMiner()

	fmt.Println("Syncing block and group info from ZV net.Waiting...")
	core.InitBlockSyncer(core.BlockChainImpl)

	// Auto apply miner role when balance enough
	var appFun applyFunc
	if len(cfg.applyRole) > 0 {
		fmt.Printf("apply role: %v\n", cfg.applyRole)
		mtype := types.MinerTypeProposal
		if cfg.applyRole == "light" {
			mtype = types.MinerTypeVerify
		}
		appFun = func() {
			gtas.autoApplyMiner(mtype)
		}
	}
	initMsgShower(mediator.Proc.GetMinerID().Serialize(), appFun)

	gtas.inited = true
	if !ok {
		return fmt.Errorf("start miner fail")
	}
	return nil
}

func (gtas *Gtas) runtimeInit() {
	debug.SetGCPercent(100)
	debug.SetMaxStack(2 * 1000000000)
	log.DefaultLogger.Info("setting gc 100%, max memory 2g")

}

func (gtas *Gtas) exit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	if core.BlockChainImpl == nil {
		return
	}
	fmt.Println("exiting...")
	core.BlockChainImpl.Close()
	//taslog.Close()
	mediator.StopMiner()
	if gtas.inited {
		quit <- true
	} else {
		os.Exit(0)
	}
}

func (gtas *Gtas) Run() {
	var err error

	// Control+c interrupt signal
	ctrlC := signals()
	quitChan := make(chan bool)
	go gtas.exit(ctrlC, quitChan)
	app := kingpin.New("gzv", "A blockchain application.")
	app.HelpFlag.Short('h')
	configFile := app.Flag("config", "Config file").Default("zv.ini").String()
	_ = app.Flag("metrics", "enable metrics").Bool()
	_ = app.Flag("dashboard", "enable metrics dashboard").Bool()
	pprofPort := app.Flag("pprof", "enable pprof").Default("23333").Uint()
	statisticsEnable := app.Flag("statistics", "enable statistics").Bool()
	keystore := app.Flag("keystore", "the keystore path, default is current path").Default("keystore").Short('k').String()
	*statisticsEnable = false

	// Console
	consoleCmd := app.Command("console", "start gzv console")
	showRequest := consoleCmd.Flag("show", "show the request json").Short('v').Bool()
	remoteHost := consoleCmd.Flag("host", "the node host address to connect").Short('i').String()
	remotePort := consoleCmd.Flag("port", "the node host port to connect").Short('p').Default("8101").Int()
	rpcPort := consoleCmd.Flag("rpcport", "gzv console will listen at the port for wallet service").Short('r').Default("0").Int()

	// Version
	versionCmd := app.Command("version", "show gzv version")

	// Mine
	mineCmd := app.Command("miner", "miner start")

	// Rpc analysis
	rpc := mineCmd.Flag("rpc", "start rpc server and specify the rpc service level").Default(strconv.FormatInt(int64(rpcLevelNone), 10)).Int()
	enableMonitor := mineCmd.Flag("monitor", "enable monitor").Default("false").Bool()
	addrRPC := mineCmd.Flag("rpcaddr", "rpc service host").Short('r').Default("0.0.0.0").IP()
	rpcServicePort := mineCmd.Flag("rpcport", "rpc service port").Short('p').Default("8101").Uint16()
	super := mineCmd.Flag("super", "start super node").Bool()
	instanceIndex := mineCmd.Flag("instance", "instance index").Short('i').Default("0").Int()
	passWd := mineCmd.Flag("password", "login password").Default(common.DefaultPassword).String()
	apply := mineCmd.Flag("apply", "apply heavy or light miner").String()
	if *apply == "heavy" {
		fmt.Println("Welcome to be a ZV propose miner!")
	} else if *apply == "light" {
		fmt.Println("Welcome to be a ZV verify miner!")
	}
	autoCreateAccount := mineCmd.Flag("createaccount", "if account not exists,create it by password").Bool()
	reset := mineCmd.Flag("reset", "reset the local top to block of the given hash").Default("").String()

	// In test mode, P2P NAT is closed
	testMode := mineCmd.Flag("test", "test mode").Bool()
	seedAddr := mineCmd.Flag("seed", "seed address").String()
	natAddr := mineCmd.Flag("nat", "nat server address").String()
	natPort := mineCmd.Flag("natport", "nat server port").Default("0").Uint16()
	chainID := mineCmd.Flag("chainid", "chain id").Default("0").Uint16()

	clearCmd := app.Command("clear", "Clear the data of blockchain")

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}

	gtas.simpleInit(*configFile)

	switch command {
	case versionCmd.FullCommand():
		fmt.Println("gzv Version:", common.GtasVersion)
		os.Exit(0)
	case consoleCmd.FullCommand():
		err := ConsoleInit(*keystore, *remoteHost, *remotePort, *showRequest, *rpcPort)
		if err != nil {
			fmt.Println(err.Error())
		}
	case mineCmd.FullCommand():
		log.Init()
		common.InstanceIndex = *instanceIndex
		go func() {
			http.ListenAndServe(fmt.Sprintf(":%d", *pprofPort), nil)
			runtime.SetBlockProfileRate(1)
			runtime.SetMutexProfileFraction(1)
		}()

		common.GlobalConf.SetInt(instanceSection, indexKey, *instanceIndex)
		databaseValue := "d" + strconv.Itoa(*instanceIndex)
		common.GlobalConf.SetString(chainSection, databaseKey, databaseValue)
		common.GlobalConf.SetBool(statisticsSection, "enable", *statisticsEnable)
		types.InitMiddleware()

		if *natAddr != "" {
			log.DefaultLogger.Infof("NAT server ip:%s", *natAddr)
		}

		cfg := &minerConfig{
			rpcLevel:          rpcLevel(*rpc),
			rpcAddr:           addrRPC.String(),
			rpcPort:           *rpcServicePort,
			super:             *super,
			testMode:          *testMode,
			natIP:             *natAddr,
			natPort:           *natPort,
			seedIP:            *seedAddr,
			applyRole:         *apply,
			keystore:          *keystore,
			enableMonitor:     *enableMonitor,
			chainID:           *chainID,
			password:          *passWd,
			autoCreateAccount: *autoCreateAccount,
			resetHash:         *reset,
		}

		// Start miner
		err := gtas.miner(cfg)
		if err != nil {
			output("initialize fai l:", err)
			log.DefaultLogger.Errorf("initialize fail:%v", err)
			os.Exit(-1)
		}
	case clearCmd.FullCommand():
		err := ClearBlock()
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("clear blockchain successfully")
		}
	}
	<-quitChan
}

// ClearBlock delete local blockchain data
func ClearBlock() error {
	err := core.InitCore(mediator.NewConsensusHelper(groupsig.ID{}), nil)
	if err != nil {
		return err
	}
	return core.BlockChainImpl.Clear()
}

func (gtas *Gtas) simpleInit(configPath string) {
	common.InitConf(configPath)
}

func (gtas *Gtas) checkAddress(keystore, address, password string, autoCreateAccount bool) error {
	aop, err := initAccountManager(keystore, autoCreateAccount, password)
	if err != nil {
		return err
	}
	defer aop.Close()

	acm := aop.(*AccountManager)
	if address != "" {
		aci, err := acm.checkMinerAccount(address, password)
		if err != nil {
			return fmt.Errorf("init miner error, err is %v", err.Error())
		}
		if aci.Miner == nil {
			return fmt.Errorf("the address is not a miner account: %v", address)
		}
		gtas.account = aci.Account
		return nil
	}
	acc := acm.getFirstMinerAccount(password)
	if acc != nil {
		gtas.account = *acc
		return nil
	}
	return fmt.Errorf("please provide a miner account and correct password! ")
}

func (gtas *Gtas) fullInit() error {
	var err error

	// Initialization middlewarex
	middleware.InitMiddleware()

	cfg := gtas.config

	addressConfig := common.GlobalConf.GetString(Section, "miner", "")

	err = gtas.checkAddress(cfg.keystore, addressConfig, cfg.password, cfg.autoCreateAccount)
	if err != nil {
		return err
	}

	common.GlobalConf.SetString(Section, "miner", gtas.account.Address)
	fmt.Println("Your Miner Address:", gtas.account.Address)

	//set the time for proposer package
	timeForPackage := common.GlobalConf.GetInt(Section, "time_for_package", 2000)
	if timeForPackage > 100 && timeForPackage < 2000 {
		core.ProposerPackageTime = time.Duration(timeForPackage) * time.Millisecond
		log.DefaultLogger.Infof("proposer uses the package config: timeForPackage %d ", timeForPackage)
	}

	//set the block gas limit for proposer package
	gasLimitForPackage := common.GlobalConf.GetInt(Section, "gas_limit_for_package", core.GasLimitPerBlock)
	if gasLimitForPackage > 10000 && gasLimitForPackage < core.GasLimitPerBlock {
		core.GasLimitForPackage = uint64(gasLimitForPackage)
		log.DefaultLogger.Infof("proposer uses the package config: gasLimitForPackage %d ", gasLimitForPackage)
	}

	//set the ignoreVmCall option for proposer package. the option shouldn't be set true only if you know what you are doing.
	core.IgnoreVmCall = common.GlobalConf.GetBool(Section, "ignore_vm_call", false)

	sk := common.HexToSecKey(gtas.account.Sk)
	minerInfo, err := model.NewSelfMinerDO(sk)
	if err != nil {
		return err
	}

	id := minerInfo.ID.GetAddrString()
	genesisMembers := make([]string, 0)
	helper := mediator.NewConsensusHelper(minerInfo.ID)
	for _, mem := range helper.GenerateGenesisInfo().Group.Members() {
		genesisMembers = append(genesisMembers, common.ToAddrHex(mem.ID()))
	}

	netCfg := network.NetworkConfig{
		IsSuper:         cfg.super,
		TestMode:        cfg.testMode,
		NatAddr:         cfg.natIP,
		NatPort:         cfg.natPort,
		SeedAddr:        cfg.seedIP,
		NodeIDHex:       id,
		ChainID:         cfg.chainID,
		ProtocolVersion: common.ProtocolVersion,
		SeedIDs:         genesisMembers,
		PK:              gtas.account.Pk,
		SK:              gtas.account.Sk,
	}

	err = network.Init(&common.GlobalConf, chandler.MessageHandler, netCfg)

	if err != nil {
		return err
	}

	err = core.InitCore(helper, &gtas.account)
	if err != nil {
		return err
	}
	if cfg.resetHash != "" {
		bh := core.BlockChainImpl.QueryBlockHeaderByHash(common.HexToHash(cfg.resetHash))
		if bh == nil {
			return fmt.Errorf("block not exists of the hash %v", cfg.resetHash)
		}
		core.BlockChainImpl.ResetTop(bh)
		output("reset local top to block:%v-%v", bh.Height, bh.Hash.Hex())
	}

	enableTraceLog := common.GlobalConf.GetBool("gtas", "enable_trace_log", false)
	if enableTraceLog {
		monitor.InitPerformTraceLogger()
	}

	// Print related content
	ShowPubKeyInfo(minerInfo, id)
	ok := mediator.ConsensusInit(minerInfo, common.GlobalConf)
	if !ok {
		return errors.New("consensus module error")
	}
	if cfg.enableMonitor || common.GlobalConf.GetBool("gtas", "enable_monitor", false) {
		monitor.InitLogService(id)
	}
	return nil
}

func ShowPubKeyInfo(info model.SelfMinerDO, id string) {
	pubKey := info.GetDefaultPubKey().GetHexString()
	log.DefaultLogger.Infof("Miner PubKey: %s;\n", pubKey)
	js, err := json.Marshal(PubKeyInfo{pubKey, id})
	if err != nil {
		log.DefaultLogger.Errorf(err.Error())
	} else {
		log.DefaultLogger.Infof("pubkey_info json: %s\n", js)
	}
}

func NewGtas() *Gtas {
	return &Gtas{}
}

func (gtas *Gtas) autoApplyMiner(mType types.MinerType) {
	miner := mediator.Proc.GetMinerInfo()
	if miner.ID.GetAddrString() != gtas.account.Address {
		// exit if miner's id not match the the account
		panic(fmt.Errorf("id error %v %v", miner.ID.GetAddrString(), gtas.account.Address))
	}

	pks := &types.MinerPks{
		MType: types.MinerType(mType),
		Pk:    miner.PK.Serialize(),
		VrfPk: miner.VrfPK,
	}

	data, err := types.EncodePayload(pks)
	if err != nil {
		log.DefaultLogger.Debugf("auto apply fail:%v", err)
		return
	}

	nonce := core.BlockChainImpl.GetNonce(miner.ID.ToAddress()) + 1
	api := &RpcDevImpl{}
	ret, err := api.TxUnSafe(gtas.account.Sk, gtas.account.Address, uint64(common.RA2TAS(core.MinMinerStake)), 20000, 500, nonce, types.TransactionTypeStakeAdd, string(data))
	log.DefaultLogger.Debug("apply result", ret, err)
}

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
	"github.com/zvchain/zvchain/consensus/base"
	"os"

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

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/monitor"
	"github.com/zvchain/zvchain/taslog"
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
func (gtas *Gtas) miner(cfg *minerConfig) {
	gtas.config = cfg
	gtas.runtimeInit()
	err := gtas.fullInit()
	if err != nil {
		fmt.Println(err.Error())
		common.DefaultLogger.Error(err.Error())
		return
	}
	if cfg.rpcEnable() {
		err = gtas.startRPC()
		if err != nil {
			common.DefaultLogger.Errorf(err.Error())
			return
		}
	}
	ok := mediator.StartMiner()

	fmt.Println("Syncing block and group info from tas net.Waiting...")
	core.InitGroupSyncer(core.GroupChainImpl, core.BlockChainImpl.(*core.FullBlockChain))
	core.InitBlockSyncer(core.BlockChainImpl.(*core.FullBlockChain))

	// Auto apply miner role when balance enough
	var appFun applyFunc
	if len(cfg.applyRole) > 0 {
		fmt.Printf("apply role: %v\n", cfg.applyRole)
		mtype := types.MinerTypeHeavy
		if cfg.applyRole == "light" {
			mtype = types.MinerTypeLight
		}
		appFun = func() {
			gtas.autoApplyMiner(mtype)
		}
	}
	initMsgShower(mediator.Proc.GetMinerID().Serialize(), appFun)

	gtas.inited = true
	if !ok {
		return
	}
}

func (gtas *Gtas) runtimeInit() {
	debug.SetGCPercent(100)
	debug.SetMaxStack(2 * 1000000000)
	common.DefaultLogger.Infof("setting gc 100%, max memory 2g")

}

func (gtas *Gtas) exit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	if core.BlockChainImpl == nil {
		return
	}
	fmt.Println("exiting...")
	core.BlockChainImpl.Close()
	taslog.Close()
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
	app := kingpin.New("GTAS", "A blockchain application.")
	app.HelpFlag.Short('h')
	configFile := app.Flag("config", "Config file").Default("tas.ini").String()
	_ = app.Flag("metrics", "enable metrics").Bool()
	_ = app.Flag("dashboard", "enable metrics dashboard").Bool()
	pprofPort := app.Flag("pprof", "enable pprof").Default("23333").Uint()
	statisticsEnable := app.Flag("statistics", "enable statistics").Bool()
	keystore := app.Flag("keystore", "the keystore path, default is current path").Default("keystore").Short('k').String()
	*statisticsEnable = false

	// Console
	consoleCmd := app.Command("console", "start gtas console")
	showRequest := consoleCmd.Flag("show", "show the request json").Short('v').Bool()
	remoteHost := consoleCmd.Flag("host", "the node host address to connect").Short('i').String()
	remotePort := consoleCmd.Flag("port", "the node host port to connect").Short('p').Default("8101").Int()
	rpcPort := consoleCmd.Flag("rpcport", "gtas console will listen at the port for wallet service").Short('r').Default("0").Int()

	// Version
	versionCmd := app.Command("version", "show gtas version")

	// Mine
	mineCmd := app.Command("miner", "miner start")

	// Rpc analysis
	rpc := mineCmd.Flag("rpc", "start rpc server and specify the rpc service level").Default(strconv.FormatInt(int64(rpcLevelNone), 10)).Int()
	enableMonitor := mineCmd.Flag("monitor", "enable monitor").Default("false").Bool()
	addrRPC := mineCmd.Flag("rpcaddr", "rpc service host").Short('r').Default("0.0.0.0").IP()
	rpcServicePort := mineCmd.Flag("rpcport", "rpc service port").Short('p').Default("8101").Uint16()
	super := mineCmd.Flag("super", "start super node").Bool()
	instanceIndex := mineCmd.Flag("instance", "instance index").Short('i').Default("0").Int()
	apply := mineCmd.Flag("apply", "apply heavy or light miner").String()
	if *apply == "heavy" {
		fmt.Println("Welcome to be a tas propose miner!")
	} else if *apply == "heavy" {
		fmt.Println("Welcome to be a tas verify miner!")
	}

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
		fmt.Println("Gtas Version:", common.GtasVersion)
		os.Exit(0)
	case consoleCmd.FullCommand():
		err := ConsoleInit(*keystore, *remoteHost, *remotePort, *showRequest, *rpcPort)
		if err != nil {
			fmt.Println(err.Error())
		}
	case mineCmd.FullCommand():
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
		common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
		types.InitMiddleware()

		if *natAddr != "" {
			common.DefaultLogger.Infof("NAT server ip:%s", *natAddr)
		}

		cfg := &minerConfig{
			rpcLevel:      rpcLevel(*rpc),
			rpcAddr:       addrRPC.String(),
			rpcPort:       *rpcServicePort,
			super:         *super,
			testMode:      *testMode,
			natIP:         *natAddr,
			natPort:       *natPort,
			seedIP:        *seedAddr,
			applyRole:     *apply,
			keystore:      *keystore,
			enableMonitor: *enableMonitor,
			chainID:       *chainID,
		}

		// Start miner
		gtas.miner(cfg)
	case clearCmd.FullCommand():
		err := ClearBlock()
		if err != nil {
			common.DefaultLogger.Error(err.Error())
		} else {
			common.DefaultLogger.Infof("clear blockchain successfully")
		}
	}
	<-quitChan
}

// ClearBlock delete local blockchain data
func ClearBlock() error {
	err := core.InitCore(mediator.NewConsensusHelper(groupsig.ID{}))
	if err != nil {
		return err
	}
	return core.BlockChainImpl.Clear()
}

func (gtas *Gtas) simpleInit(configPath string) {
	common.InitConf(configPath)
}

func (gtas *Gtas) checkAddress(keystore, address string) error {
	aop, err := initAccountManager(keystore, true)
	if err != nil {
		return err
	}
	defer aop.Close()

	acm := aop.(*AccountManager)
	if address != "" {
		aci, err := acm.getAccountInfo(address)
		if err != nil {
			return fmt.Errorf("cannot get miner, err:%v", err.Error())
		}
		if aci.Miner == nil {
			return fmt.Errorf("the address is not a miner account: %v", address)
		}
		gtas.account = aci.Account
		return nil

	}
	aci := acm.getFirstMinerAccount()
	if aci != nil {
		gtas.account = *aci
		return nil
	}
	return fmt.Errorf("please create a miner account first")
}

func (gtas *Gtas) fullInit() error {
	var err error

	// Initialization middleware
	middleware.InitMiddleware()

	cfg := gtas.config

	addressConfig := common.GlobalConf.GetString(Section, "miner", "")
	err = gtas.checkAddress(cfg.keystore, addressConfig)
	if err != nil {
		return err
	}

	common.GlobalConf.SetString(Section, "miner", gtas.account.Address)
	fmt.Println("Your Miner Address:", gtas.account.Address)

	//minerInfo := model.NewSelfMinerDO(common.HexToSecKey(gtas.account.Sk))
	var minerInfo model.SelfMinerDO
	if gtas.account.Miner != nil {
		prk := common.HexToSecKey(gtas.account.Sk)
		dBytes := prk.PrivKey.D.Bytes()
		tempBuf := make([]byte, 32)
		if len(dBytes) < 32 {
			copy(tempBuf[32-len(dBytes):32], dBytes[:])
		} else {
			copy(tempBuf[:], dBytes[len(dBytes)-32:])
		}
		minerInfo.SecretSeed = base.RandFromBytes(tempBuf[:])
		minerInfo.SK = *groupsig.NewSeckeyFromHexString(gtas.account.Miner.BSk)
		minerInfo.PK = *groupsig.NewPubkeyFromHexString(gtas.account.Miner.BPk)
		minerInfo.ID = *groupsig.NewIDFromString(gtas.account.Address)
		minerInfo.VrfSK = base.Hex2VRFPrivateKey(gtas.account.Miner.VrfSk)
		minerInfo.VrfPK = base.Hex2VRFPublicKey(gtas.account.Miner.VrfPk)
	}
	//import end.   gtas.account --> minerInfo

	err = core.InitCore(mediator.NewConsensusHelper(minerInfo.ID))
	if err != nil {
		return err
	}
	id := minerInfo.ID.GetHexString()

	netCfg := network.NetworkConfig{IsSuper: cfg.super,
		TestMode:        cfg.testMode,
		NatAddr:         cfg.natIP,
		NatPort:         cfg.natPort,
		SeedAddr:        cfg.seedIP,
		NodeIDHex:       id,
		ChainID:         cfg.chainID,
		ProtocolVersion: common.ProtocalVersion,
		SeedIDs:         core.GroupChainImpl.GenesisMembers(),
	}

	err = network.Init(common.GlobalConf, chandler.MessageHandler, netCfg)

	if err != nil {
		return err
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

	mediator.Proc.BeginGenesisGroupMember()

	return nil
}

func ShowPubKeyInfo(info model.SelfMinerDO, id string) {
	pubKey := info.GetDefaultPubKey().GetHexString()
	common.DefaultLogger.Infof("Miner PubKey: %s;\n", pubKey)
	js, err := json.Marshal(PubKeyInfo{pubKey, id})
	if err != nil {
		common.DefaultLogger.Errorf(err.Error())
	} else {
		common.DefaultLogger.Infof("pubkey_info json: %s\n", js)
	}
}

func NewGtas() *Gtas {
	return &Gtas{}
}

func (gtas *Gtas) autoApplyMiner(mtype int) {
	miner := mediator.Proc.GetMinerInfo()
	if miner.ID.GetHexString() != gtas.account.Address {
		panic(fmt.Errorf("id error %v %v", miner.ID.GetHexString(), gtas.account.Address))
	}

	tm := &types.Miner{
		ID:           miner.ID.Serialize(),
		PublicKey:    miner.PK.Serialize(),
		VrfPublicKey: miner.VrfPK,
		Stake:        common.VerifyStake,
		Type:         byte(mtype),
	}
	data, err := msgpack.Marshal(tm)
	if err != nil {
		common.DefaultLogger.Errorf("err marhsal types.Miner", err)
		return
	}

	nonce := core.BlockChainImpl.GetNonce(miner.ID.ToAddress()) + 1
	api := &RpcDevImpl{}
	ret, err := api.TxUnSafe(gtas.account.Sk, "", 0, 20000, 100, nonce, types.TransactionTypeMinerApply, common.ToHex(data))
	common.DefaultLogger.Debugf("apply result", ret, err)

}

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
	databaseKey = "db_blocks"
)

type Gzv struct {
	Inited       bool
	Account      Account
	Config       *minerConfig
	RpcInstances []rpcApi
}

// miner start miner node
func (gzv *Gzv) miner(cfg *minerConfig) {
	gzv.Config = cfg
	gzv.runtimeInit()
	err := gzv.fullInit()
	if err != nil {
		fmt.Println(err.Error())
		log.DefaultLogger.Error(err.Error())
		return
	}
	if cfg.rpcEnable() {
		err = gzv.startRPC()
		if err != nil {
			log.DefaultLogger.Errorf(err.Error())
			return
		}
	}
	ok := mediator.StartMiner()

	fmt.Println("Syncing block and group info from ZV net.Waiting...")
	core.InitBlockSyncer(core.BlockChainImpl.(*core.FullBlockChain))

	// Auto apply miner role when balance enough
	var appFun applyFunc
	if len(cfg.applyRole) > 0 {
		fmt.Printf("apply role: %v\n", cfg.applyRole)
		mtype := types.MinerTypeProposal
		if cfg.applyRole == "light" {
			mtype = types.MinerTypeVerify
		}
		appFun = func() {
			gzv.autoApplyMiner(mtype)
		}
	}
	initMsgShower(mediator.Proc.GetMinerID().Serialize(), appFun)

	gzv.Inited = true
	if !ok {
		return
	}
}

func (gzv *Gzv) runtimeInit() {
	debug.SetGCPercent(100)
	debug.SetMaxStack(2 * 1000000000)
	log.DefaultLogger.Info("setting gc 100%, max memory 2g")

}

func (gzv *Gzv) exit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	if core.BlockChainImpl == nil {
		return
	}
	fmt.Println("exiting...")
	core.BlockChainImpl.Close()
	//taslog.Close()
	mediator.StopMiner()
	if gzv.Inited {
		quit <- true
	} else {
		os.Exit(0)
	}
}

func (gzv *Gzv) Run() {
	//var err error

	// Control+c interrupt signal
	ctrlC := Signals()
	quitChan := make(chan bool)
	go gzv.exit(ctrlC, quitChan)
	kingpin.HelpFlag.Short('h')
	configFile := kingpin.Flag("config", "Config file").Default("zv.ini").String()
	pprofPort := kingpin.Flag("pprof", "enable pprof").Default("23333").Uint()
	keystore := kingpin.Flag("keystore", "the keystore path, default is current path").Default("keystore").Short('k').String()

	// Rpc analysis
	rpc := kingpin.Flag("rpc", "start rpc server and specify the rpc service level").Default(strconv.FormatInt(int64(rpcLevelNone), 10)).Int()
	enableMonitor := kingpin.Flag("monitor", "enable monitor").Default("false").Bool()
	addrRPC := kingpin.Flag("rpcaddr", "rpc service host").Short('r').Default("0.0.0.0").IP()
	rpcServicePort := kingpin.Flag("rpcport", "rpc service port").Short('p').Default("8101").Uint16()
	super := kingpin.Flag("super", "start super node").Bool()
	instanceIndex := kingpin.Flag("instance", "instance index").Short('i').Default("0").Int()
	passWd := kingpin.Flag("password", "login password").Default("123").String()
	apply := kingpin.Flag("apply", "apply heavy or light miner").String()
	if *apply == "heavy" {
		fmt.Println("Welcome to be a ZV propose miner!")
	} else if *apply == "light" {
		fmt.Println("Welcome to be a ZV verify miner!")
	}

	// In test mode, P2P NAT is closed
	testMode := kingpin.Flag("test", "test mode").Bool()
	seedAddr := kingpin.Flag("seed", "seed address").String()
	natAddr := kingpin.Flag("nat", "nat server address").String()
	natPort := kingpin.Flag("natport", "nat server port").Default("0").Uint16()
	chainID := kingpin.Flag("chainid", "chain id").Default("0").Uint16()

	kingpin.Parse()
	gzv.simpleInit(*configFile)

	log.Init()
	common.InstanceIndex = *instanceIndex
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", *pprofPort), nil)
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
	}()

	common.GlobalConf.SetInt(instanceSection, indexKey, *instanceIndex)
	databaseValue := "d_b" + strconv.Itoa(*instanceIndex)
	common.GlobalConf.SetString(chainSection, databaseKey, databaseValue)
	types.InitMiddleware()

	if *natAddr != "" {
		log.DefaultLogger.Infof("NAT server ip:%s", *natAddr)
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
		password:      *passWd,
	}

	// Start gzv
	gzv.miner(cfg)
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

func (gzv *Gzv) simpleInit(configPath string) {
	common.InitConf(configPath)
}

func (gzv *Gzv) checkAddress(keystore, address, password string) error {
	if !dirExists(keystore) {
		return fmt.Errorf("the given path \"%s\" is not exist", keystore)
	}

	keystoreMgr, err := NewkeyStoreMgr(keystore)
	if err != nil {
		return err
	}
	defer keystoreMgr.Close()

	if address != "" {
		aci, err := keystoreMgr.CheckMinerAccount(address, password)
		if err != nil {
			return fmt.Errorf("cannot get miner, err:%v", err.Error())
		}
		if aci.Miner == nil {
			return fmt.Errorf("the address is not a miner account: %v", address)
		}
		gzv.Account = *aci
		return nil
	}
	acc := keystoreMgr.GetFirstMinerAccount(password)
	if acc != nil {
		gzv.Account = *acc
		return nil
	}
	return fmt.Errorf("please provide a miner account and correct password! ")
}

func (gzv *Gzv) fullInit() error {
	var err error

	// Initialization middlewarex
	middleware.InitMiddleware()

	cfg := gzv.Config

	addressConfig := common.GlobalConf.GetString(Section, "miner", "")
	err = gzv.checkAddress(cfg.keystore, addressConfig, cfg.password)
	if err != nil {
		return err
	}

	common.GlobalConf.SetString(Section, "miner", gzv.Account.Address)
	fmt.Println("Your Miner Address:", gzv.Account.Address)

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

	sk := common.HexToSecKey(gzv.Account.Sk)
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
		PK:              gzv.Account.Pk,
		SK:              gzv.Account.Sk,
	}

	err = network.Init(&common.GlobalConf, chandler.MessageHandler, netCfg)

	if err != nil {
		return err
	}

	err = core.InitCore(helper, &gzv.Account)
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

func NewGzv() *Gzv {
	return &Gzv{}
}

func (gzv *Gzv) autoApplyMiner(mType types.MinerType) {
	miner := mediator.Proc.GetMinerInfo()
	if miner.ID.GetAddrString() != gzv.Account.Address {
		// exit if miner's id not match the the account
		panic(fmt.Errorf("id error %v %v", miner.ID.GetAddrString(), gzv.Account.Address))
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
	ret, err := api.TxUnSafe(gzv.Account.Sk, gzv.Account.Address, uint64(common.RA2TAS(core.MinMinerStake)), 20000, 500, nonce, types.TransactionTypeStakeAdd, string(data))
	log.DefaultLogger.Debug("apply result", ret, err)
}

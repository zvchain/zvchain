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
	"fmt"
	"github.com/zvchain/zvchain/log"
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
		log.DefaultLogger.Error(err.Error())
		return
	}

	gtas.inited = true
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
	passWd := mineCmd.Flag("password", "login password").Default("123").String()
	apply := mineCmd.Flag("apply", "apply heavy or light miner").String()
	if *apply == "heavy" {
		fmt.Println("Welcome to be a ZV propose miner!")
	} else if *apply == "light" {
		fmt.Println("Welcome to be a ZV verify miner!")
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

		// Start miner
		gtas.miner(cfg)
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

func (gtas *Gtas) checkAddress(keystore, address, password string) error {
	aop, err := initAccountManager(keystore, true)
	if err != nil {
		return err
	}
	defer aop.Close()

	acm := aop.(*AccountManager)
	if address != "" {
		aci, err := acm.checkMinerAccount(address, password)
		if err != nil {
			return fmt.Errorf("cannot get miner, err:%v", err.Error())
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

	cfg := gtas.config

	addressConfig := common.GlobalConf.GetString(Section, "miner", "")
	err = gtas.checkAddress(cfg.keystore, addressConfig, cfg.password)
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

	id := minerInfo.ID.GetHexString()
	genesisMembers := make([]string, 0)
	//helper := mediator.NewConsensusHelper(minerInfo.ID)
	//for _, mem := range helper.GenerateGenesisInfo().Group.Members() {
	//	genesisMembers = append(genesisMembers, common.ToHex(mem.ID()))
	//}

	genesisMembers = append(genesisMembers, "0x03ea3d0d2baff52e785525227ab81ec37825fa49b41a64c5136ec58e9be279f4")
	genesisMembers = append(genesisMembers, "0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4")
	genesisMembers = append(genesisMembers, "0x08847f5c81aee6a2d195723ca3961219dc7a1bee341776cbe2fa64e6ca1426b1")

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

	fmt.Println(time.Now().UnixNano() / 1000000)
	gtas.TestSpreadAmongGroup()

	// Print related content
	ShowPubKeyInfo(minerInfo, id)
	return nil
}

func (gtas *Gtas) TestSpreadAmongGroup() {
	groupID := "group"
	//members := []string{"0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02","0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76","0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"}
	members := []string{
		"0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02",

		"0x3b2cdf5aaa6de805087677ed7f8560aef370e771b17558e75fa9efc9a19c9ddc",
		"0x83361a38e4ac2cc0147f0c253d69ca0d18b69bc14d2b35a0bd4767c88439465e",
		"0x0ccb56badf284764e20629e4bcf4fa3aadb3fd6cdc4275d3c41485db4168d3d1",
		"0x26dea2122d15d0ed803a9d6d90f7363bff2e056721e9eb1ea2c53bb2ff099d7f",
		"0xe9264622cff077a2b3a8b0242ae88ae1fbdf8c5416f28bf45d5b308b12c81ffa",
		"0x0d7a8f69f94a92f088fb42e642406e977285d1f5cabc8cf57b72f0e328ec4a6c",
		"0x08847f5c81aee6a2d195723ca3961219dc7a1bee341776cbe2fa64e6ca1426b1",
		"0xbbd5a63b2f262b16b33ead5cd02e84655420b1c68118bc414612998c8d0c6826",
		"0xab68cb652e7bd40cc9dcfa1154b69b2006c08cff25917878bd012b9ab26d8ba4",
		"0xcb25178d694b214710f25acf2abadfae15dc9dac045112670fe6a1b557f472e4",
		"0xab79c2bf02ab023a0541b9ca215277c3f7e34b68ce15f70ac93a7bbdbcb810b8",
		"0x933f366a84d1e06ac0fdcfba815e6592d2c547b343ca391e70f0f20102bd91fc",
		"0x9a5261dde18a30b2b0dc317c561cba4b3d2f86f4a15fcd3722c533573e0b5a8d",
		"0x47bdf5b424ff948695655ed91c12bdc9f53199be9ebfbfe1159a67a6af74b66b",
		"0x16330ec817a52a568415f583d63f5a887b7ff98460567505ffa7903c3d2d0aac",
		"0xf55d6b1f19a3cd92ee2e9c6e495ccefc7c2f33b2e2efd72b42916744bad0c4fa",
		"0x0a03a4b9b05377a1c1da25e5570ceed7dac0d7b3bd0e11aefddee62518e2e9d8",
		"0x3e6a3e7f29ca62effa66aa1d3f9b5f12ca002ba24274cc992df0c69a70255e41",
		"0x0bc915e09dec4b3ae2427464513eabec188f0c3b8142a0b0155f166971c5c368",
		"0xcfe0a90f7044fbc8d3519b06a46ade1ba05f9d5420c2caece19ad1ce50fe321e",
		"0x64baa8f97e4a25c6dd4744fcad3e0eb938c49c6c9a795c21f643eda7f88633ec",
		"0x6e1512f604f0d07de1991b3db51cd5b9fba7d4663e40523b1d312a0c5bcf9679",
		"0xde60b502e83bfa00c81ab8ca2d6379b2d40bee156dd124fd193a3e2e61c67801",
		"0xcae5f1217a1fd02e8d722b62ea65e2bec79da5929818b1008bf3b2b322fbca3e",
		"0x0bf173f737196c9f8e2f274d359b4cec9a3f049a6a542017276e67d85c8a2992",
		"0xf32195b2938bd1477466eae87b74186fd8731694e30ef752918277a11e7fe3a5",
		"0x08e7107620a38eae98abae0a85ec80240728909af1c4b8bcfd4de0eaac3880a6",
		"0xe9c4c5c372ce94699343174339fa92f892c71196c965236d25229ebce2e7bf44",
		"0x6aaba378e56eb0d3d90737c2cbb3b55407ffe690c2ce2150c9e130ea04f0ff18",
		"0xc690291fc4a4132a4ce08f71aa3db6bef9c6c7a2da9952c35538fcae94ae4ce1",
		"0x03ea3d0d2baff52e785525227ab81ec37825fa49b41a64c5136ec58e9be279f4",
		"0x6c7c4724a0abfff3fc109ed6ef71ce82aba7ada1c4e498674458bc2045af21cd",
		"0xc7d83d1e57ac5e2df25df8c569e87f62fc1173039faf3cb30f65d0efab9ecc50",
		"0x0cc66654d0099fdc10709df3bdb4eab4a68c97ec5dc66d087361d0cabb37db80",
		"0x69d959d090df8c77adc85b5294871a904b0294eb85fb8251ba903710805d64c2",
		"0x6c5cc60dbcdb1d63e739ebde3a076c32e37e6bf46f9e15809ccf73348f24acd9",
		"0x75df2d0e7b7aaf2b6a5e25856afd0da84da08dbe935cd0e67582bc7b8dad4f81",
		"0xb4c45be38afcc52fcbc6aa6706897b0c0f943bed691be2ac2beb18e5f2a07890",
		"0x806ec4eb2d7a2ba0ebee40e2a39e3e8b1f3a09d91ae78bd7fdf30c77e543f545",
		"0x27af3b203839ba7d47ceaac70b81dbe57074ade843c87b0e5562fd2a3dd3c990",
		"0x214f1f91ee41d33494575d43bcf41244145d90b569fb16fc93e212c3338caa78",
		"0xb0fa580541b684390b6cbb292a0e6ef2dd8998de98e487cc5952d3d75db0ec1c",
		"0xb2e882c6d59b37636d65cd6e023d4f2bd49f25947c37221ac52b3c9b60278813",
		"0x586e580e7d3352d617f35189ed4995679729a1a8b53ed8d91e46d7f8970d4737",
		"0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4",
	}
	network.GetNetInstance().BuildGroupNet(groupID, members)

	go func() {
		//var count uint32 = 32
		for {

			fmt.Println(len(network.GetNetInstance().ConnInfo()), network.GetNetInstance().ConnInfo())
			//if len(network.GetNetInstance().ConnInfo()) >= 37 {
			//	count++
			//	msg := network.Message{
			//		Code: count,
			//		Body: []byte("helloworld"),
			//	}
			//	network.GetNetInstance().SpreadAmongGroup(groupID, msg)
			//}
			time.Sleep(2 * time.Second)
		}
	}()
}

func (gtas *Gtas) TestSendEveryone() {
	members := []string{
		"0x3b2cdf5aaa6de805087677ed7f8560aef370e771b17558e75fa9efc9a19c9ddc",
		"0x83361a38e4ac2cc0147f0c253d69ca0d18b69bc14d2b35a0bd4767c88439465e",
		"0x0ccb56badf284764e20629e4bcf4fa3aadb3fd6cdc4275d3c41485db4168d3d1",
		"0x26dea2122d15d0ed803a9d6d90f7363bff2e056721e9eb1ea2c53bb2ff099d7f",
		"0xe9264622cff077a2b3a8b0242ae88ae1fbdf8c5416f28bf45d5b308b12c81ffa",
		"0x0d7a8f69f94a92f088fb42e642406e977285d1f5cabc8cf57b72f0e328ec4a6c",
		"0x08847f5c81aee6a2d195723ca3961219dc7a1bee341776cbe2fa64e6ca1426b1",
		"0xbbd5a63b2f262b16b33ead5cd02e84655420b1c68118bc414612998c8d0c6826",
		"0xab68cb652e7bd40cc9dcfa1154b69b2006c08cff25917878bd012b9ab26d8ba4",
		"0xcb25178d694b214710f25acf2abadfae15dc9dac045112670fe6a1b557f472e4",
		"0xab79c2bf02ab023a0541b9ca215277c3f7e34b68ce15f70ac93a7bbdbcb810b8",
		"0x933f366a84d1e06ac0fdcfba815e6592d2c547b343ca391e70f0f20102bd91fc",
		"0x9a5261dde18a30b2b0dc317c561cba4b3d2f86f4a15fcd3722c533573e0b5a8d",
		"0x47bdf5b424ff948695655ed91c12bdc9f53199be9ebfbfe1159a67a6af74b66b",
		"0x16330ec817a52a568415f583d63f5a887b7ff98460567505ffa7903c3d2d0aac",
		"0xf55d6b1f19a3cd92ee2e9c6e495ccefc7c2f33b2e2efd72b42916744bad0c4fa",
		"0x0a03a4b9b05377a1c1da25e5570ceed7dac0d7b3bd0e11aefddee62518e2e9d8",
		"0x3e6a3e7f29ca62effa66aa1d3f9b5f12ca002ba24274cc992df0c69a70255e41",
		"0x0bc915e09dec4b3ae2427464513eabec188f0c3b8142a0b0155f166971c5c368",
		"0xcfe0a90f7044fbc8d3519b06a46ade1ba05f9d5420c2caece19ad1ce50fe321e",
		"0x64baa8f97e4a25c6dd4744fcad3e0eb938c49c6c9a795c21f643eda7f88633ec",
		"0x6e1512f604f0d07de1991b3db51cd5b9fba7d4663e40523b1d312a0c5bcf9679",
		"0xde60b502e83bfa00c81ab8ca2d6379b2d40bee156dd124fd193a3e2e61c67801",
		"0xcae5f1217a1fd02e8d722b62ea65e2bec79da5929818b1008bf3b2b322fbca3e",
		"0x0bf173f737196c9f8e2f274d359b4cec9a3f049a6a542017276e67d85c8a2992",
		"0xf32195b2938bd1477466eae87b74186fd8731694e30ef752918277a11e7fe3a5",
		"0x08e7107620a38eae98abae0a85ec80240728909af1c4b8bcfd4de0eaac3880a6",
		"0xe9c4c5c372ce94699343174339fa92f892c71196c965236d25229ebce2e7bf44",
		"0x6aaba378e56eb0d3d90737c2cbb3b55407ffe690c2ce2150c9e130ea04f0ff18",
		"0xc690291fc4a4132a4ce08f71aa3db6bef9c6c7a2da9952c35538fcae94ae4ce1",
		"0x03ea3d0d2baff52e785525227ab81ec37825fa49b41a64c5136ec58e9be279f4",
		"0x6c7c4724a0abfff3fc109ed6ef71ce82aba7ada1c4e498674458bc2045af21cd",
		"0xc7d83d1e57ac5e2df25df8c569e87f62fc1173039faf3cb30f65d0efab9ecc50",
		"0x0cc66654d0099fdc10709df3bdb4eab4a68c97ec5dc66d087361d0cabb37db80",
		"0x69d959d090df8c77adc85b5294871a904b0294eb85fb8251ba903710805d64c2",
		"0x6c5cc60dbcdb1d63e739ebde3a076c32e37e6bf46f9e15809ccf73348f24acd9",
		"0x75df2d0e7b7aaf2b6a5e25856afd0da84da08dbe935cd0e67582bc7b8dad4f81",
		"0xb4c45be38afcc52fcbc6aa6706897b0c0f943bed691be2ac2beb18e5f2a07890",
		"0x806ec4eb2d7a2ba0ebee40e2a39e3e8b1f3a09d91ae78bd7fdf30c77e543f545",
		"0x27af3b203839ba7d47ceaac70b81dbe57074ade843c87b0e5562fd2a3dd3c990",
		"0x214f1f91ee41d33494575d43bcf41244145d90b569fb16fc93e212c3338caa78",
		"0xb0fa580541b684390b6cbb292a0e6ef2dd8998de98e487cc5952d3d75db0ec1c",
		"0xb2e882c6d59b37636d65cd6e023d4f2bd49f25947c37221ac52b3c9b60278813",
		"0x586e580e7d3352d617f35189ed4995679729a1a8b53ed8d91e46d7f8970d4737",
		"0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4",
	}
	go func() {
		var count uint32 = 21100
		body := []byte("helloworld")//make([]byte, 1024 * (1024 / 2))
		for {
			fmt.Println(len(network.GetNetInstance().ConnInfo()), network.GetNetInstance().ConnInfo())
			if len(network.GetNetInstance().ConnInfo()) >= 10 {
				count++
				msg := network.Message{
					Code: count,
					Body: body,
				}
				for _, m := range members {
					network.GetNetInstance().Send(m, msg)
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
}

func (gtas *Gtas) TestBroadcast() {
	go func() {
		var count uint32 = 52000
		body := make([]byte, 1024 * (1024 / 2))//[]byte("helloworld")
		for {
			fmt.Println(len(network.GetNetInstance().ConnInfo()), network.GetNetInstance().ConnInfo())
			if len(network.GetNetInstance().ConnInfo()) >= 10 {
				count++
				msg := network.Message{
					Code: count,
					Body: body,
				}
				network.GetNetInstance().Broadcast(msg)
			}
			time.Sleep(2 * time.Second)
		}
	}()
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

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

	genesisMembers = append(genesisMembers, "0x0077464c5d2b1152d945fee3c3e3f2bf09fdfad68e22d954e34e8b1dfa550ff5")
	genesisMembers = append(genesisMembers, "0x0959fc1dff5127ff2b7dede48be6a60c8ee8533543775462fdd4f5b5d6a5f5e4")
	genesisMembers = append(genesisMembers, "0x0ba0e0074b7f84250184f294fe0be8f4056174c826bad09b6982a0d93312a30c")

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
	//log.ELKLogger.WithFields(logrus.Fields{
	//	"addr": gtas.account.Address,
	//	"counter": 2,
	//}).Debug("Init")
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

		"0x0077464c5d2b1152d945fee3c3e3f2bf09fdfad68e22d954e34e8b1dfa550ff5",
		"0x032245e4a28e9505c2b7d088e055641e7e38f5192219de966436e53f626dbd31",
		"0x038d881c8b2d7976e9e806b6a5a590601dee0cc8be48106aab1c83adbe4aeac6",
		"0x0959fc1dff5127ff2b7dede48be6a60c8ee8533543775462fdd4f5b5d6a5f5e4",
		"0x0a1f13a1bf593540862e31fc959eff50ff8c1fe67c2bfa2a6ee05e3c745e803d",
		"0x0a26c83ccd511329db33fed03751f1b3e7b8bf1272fc450a8d343d51bfb98c3b",
		"0x0ba0e0074b7f84250184f294fe0be8f4056174c826bad09b6982a0d93312a30c",
		"0x0e6a9d512d6a8bc82766ba4227c21f0dfe7718e3496ec0908cd8540139c0c619",
		"0x0fcd9cbd225544c2f92fee071c2cfc7c5d59854c87e5ee26c40a25f438aa3924",
		"0x105a0823aa54b47fb0eee1ebd6a8e3e45101562c011635087a75cc6b8701ef49",
		"0x13e8930b896623c1363f2d7aa1be79ac09c25e3a51f3d8ba07ad2c3096659524",
		"0x1551f3b4a14de262cb5844704fbf6269d40219fee86c4f19a903ea16c7b696e6",
		"0x158cf8df0316d9f6a20615d62e6ae7c45303181001e3158db579e94946dbd49f",
		"0x1c7ff1c98f03a18f2c84676862e653ad733d71291f1db2ab530370cf84e14dc6",
		"0x1d574d31bdbfdcbede3e05756adbd6326e029d46054e97b4c3443a40bc8ede7a",
		"0x1eebf400815d74de39a1179c38b959fd16fc287ac06f6940580ea13d96757624",
		"0x215b8758ff2af792679952a7c336bcacb62eedd72879cb20e658fdcec9771359",
		"0x232cfa4827ad61227376111504fd42b2387a0997d9d26a6133242177025433f2",
		"0x25a7f2decbb8b75d4d1d2b7c058dcc61c5dff7b3991b546d231960c958b01505",
		"0x2714712ded76e1a68a1d301514f275535e8ea3a64bb1e28e094d801109a05a41",
		"0x27940ae3b775788fc4bccecbf18be2d39158c7a03c9f2f26da9354c0c75e62e9",
		"0x2819c2b0c7a100bd3ba214b3758c78dc70c5591df78d9082a6b9164c61506052",
		"0x2959c45de9fe3b051d999a940f99450999110fe6a4d04cb94a2f6547ebc4dcef",
		"0x2c570d07a5c1295f323346b0f61a14efaa415cdf2ed0d37006726221b638cb86",
		"0x2ecca96cf50799e005c727eaa974bf4142d28a4f352d10cc7c4535924f4354b1",
		"0x2edbfe440bfdad74f651243ea27d4a529c2f54cb3572ca599abfa6e86c663b08",
		"0x3045d9f52fc7125bd48fdd1e2126ce8ea8ee3066c4c9c309de1e985c7c54a1c3",
		"0x30cd28b2ad3e620465302d24061996c2a81ef1cea0b0f455ab0e6d86cf41d1e2",
		"0x340568cf75cc4741c43c5afd7c20a6a60c610fc1ddc828ad786f23b7413034f7",
		"0x347ba167ea7bb5fb7fb031e703159a204fca3b6b5c0377a9149e712212a47890",
		"0x34859de1e30f5ac64dcce9a78304b8a09c1e3d731555cdcd400b5ab810310344",
		"0x35b3f591ef58f597c2755fd96e9369c023db8b7e9ff084af0f7ca7924f8072f5",
		"0x3abdfc6ff3d7a13c36edbddb9451190965bec2f623633b67eaed6658b7ac9587",
		"0x3fdaddb996208af206e123560199decfd0efb5d1e65ea4cbe3bfc36d8af74b26",
		"0x3ff6607f0ca170dd749d0012f8ba90e2cdd8d59e3727a2d2457a6259357c97ea",
		"0x400d124c8c2e07a5a541b916679be6dcba1df25e318cf07447d5fbdaec900590",
		"0x41e14ff4b43dbf38ce7927f2fe9f8b82e234a510aaf94e22f96660b4a0f38dee",
		"0x49db49f9de6dc6eb8ef7b62d717502965efe6e416a34af7fa3df621053080370",
		"0x4db7f8541368edd508e164a9423fdc9a2917174e1f49fe993f9ae2ef3effa11d",
		"0x52e6ba4d2585c8f5ec5c94402b3d67c2cbc415486c40d30045160aafc8cbf1d4",
		"0x5344632d941279b8bb669b734ea42cc08ecb89a56059f24e3738b6cad8fa4b23",
		"0x53556a84b524a6d9484b2560d50d3fea55cd42b49b7bce98794df92e55356442",
		"0x5470277e95c66f8e6c69b62ca9c7d0508fe12d8a270d49f3201f362c6f94d93d",
		"0x597ebcdffaf98d6f82856c667517c34199248ba7f280b4a5bd95a0d69e22c9c1",
		"0x6495cc494946f03ae3e89f2363ac88e52f11ed7964c75aaa01567a16d3deeee0",
		"0x751f76aff30e2b8d58245a7e60e3f6a2173a170f48da9d22e5fa43afcbf45c65",
		"0x77cef301c02f209ea93abbce4e4a209e78432bfa9f5652643c3af115f8501fe5",
		"0x7cd8a381168d3de03391930fb1392ddb3dc8659bbb0f0923fc7e7273419a7e20",
		"0x83584da1d2966c31b65b9a404b62ecbb439c2c67eb78e8ed0add8f4249b16f6d",
		"0x8a058a2b22d22a62a9de40d7f4002f24550b38668992f728b86461de4c334792",
		"0x8fef5b6a83f0f29af5630f5fde16b1a21cf63369683ea3d5d1ccf66ec3620aa7",
		"0x90a3fc82433306be85f7e84993812be48a50a980627cfd0441adb0bc7fe2d207",
		"0x93a8b30fc8303907deaca066a2cd83add32855870047a46e0ea7a2473c423d01",
		"0x98dfc297280754201f659dae82873bcc45f899739d6997096b839678863d62bf",
		"0x99489535d3cb3e3ea032b21e49d56e9c6e6597232404f15f1b06cf93016d0302",
		"0x9cd65a1289752f801b3307eddab141fc372459709a55b9385c3e9c17f35bf83e",
		"0x9d25efaf407d8c766f74cf08555c7649347f8bdb662858077f7664ce83543f91",
		"0xa36a0ed8549ab42789e294b667c528072060a8b9c447fe37d51ebcbe325d4a39",
		"0xa97a53609050f7fca6307e491b3515a8a1a15b8c978391a5a7b5c0ef016fddb4",
		"0xa98a31428e7888b3bfca8ac3afbd3d94dc76713fe21f6b60cf9e1b9a90893808",
		"0xae089cdd1a6809cb4ef05ca46ebc14d9debd7f25779ef877ca91f0367ee9d47f",
		"0xae346fd20103cdf71d79e48665e3966f4fb4ac1018ad0af8ca7dba779f2ca11c",
		"0xae89a12f527df0cbf5c43bd6a9e31f1bac03f7bd479b0b5d3049a6fca2a65bd1",
		"0xb31da91b385f7279c45aeab5604efd48fe72e2bfb35d278341dfce80aebf5f50",
		"0xb327c461305f85e12679a7b98d640addc646e305ebe243fb83299416c82ea9dd",
		"0xb642220718e794c48b91dbc41b7aac7a18bc9459476c4f84aa8e36f09ac5c0f1",
		"0xb8d7d57f03da0fc4bd0892a3f4fb741c0aa6a3b94489c6e06f8ff8f6642e01d5",
		"0xb8f480d0f83ff0b7d61b378f23d0f3ea71c9f75398f9e51eb1c6cde91f29ae6a",
		"0xbb75c2d7646dc98d8aa8db31a0a292b47de3b3d5b0eb540e23572dac6387eb75",
		"0xbc491e69008781e3ddf13fcf8df681e3672f2efb525c451c7797c4c83340e8ef",
		"0xbea8dd36fffb62f3b1b9c0354a524d0500151a0b4b189cd183611743f268f4f5",
		"0xbec8ef83be0ce5adf1b150130c95586d82435d22dc1b3765f1c7dcf85e14b9ff",
		"0xc1c609fbcf3430bd9bed7c5bfd518dd412c43fb34a26b5a455e296a46f65794e",
		"0xcdf9258aa8478d88f5ef4cc301454882336be16f822d7a06a8b168e7bef0f570",
		"0xd88f1b55f803bd101b46be55037fe4134c136d55fcc32e65e31e4b35cf2bb19f",
		"0xdc395846ba4a6693d37df9e27b15dfb3ab0ca65e480c71f1b3d03e0bece458bd",
		"0xdddc95a0d0b2a29f556991a1888869de6c59113aad37cf7cc2bd17419cca3085",
		"0xe056fd63f7563834ed3b7a507dd0657a4a6fba224cace18fb7e6102ae6065277",
		"0xe755cca13c9a946af6ec249fd2169fc43a8d6123fa0f000340ee755447ff235f",
		"0xea77d068e5acd93b1804f0f29b1641113b4f2e54c94303b4fbc41d8d82de53cb",
		"0xeacaf3b8fad2b4522807aa402417a29391b05d44c5015e0fd467646eaeb15843",
		"0xed5f0f1e9107f82f21aeff5b4858787362edb83756fe243c2e7e1d45ad533b99",
		"0xef3b15976491d037367f25e8d77ddcbc5867996cc08cea4215d6999729441583",
		"0xef774e74e128e2dd51f7ca20f1d22c5d0f43fa9374ef11f8f9a11f26e27f5332",
		"0xef8bdb5105791b0e757a814b3ed4248652ba16e9289bcceab6da69f636bf711e",
		"0xf1510ed56f854cc98e93b5337de218cd5e589c9d237eb60eaf2c45f81966be1a",
		"0xfb4db4f9e3634496bd721d607d60f3ddf804b0be33fe8a36c2b3b61fc94ddd24",
		"0xfcb213902b0b4fad6c073ed1e771b181e5757825aad86f076e271b839dbdbaa4",
		"0xfd7d9063e3bf57c69f8756c7bece466aff064af8163b2fb25b42e5861347cfba",
		"0xfdeecd56d3191d9f3c1cccc04ac2bc1f469b036810ca08cda3bce916c7bf31af",
		"0xff1522f11a666e69d7fb96fcf47582e289966a9b6618417d1830ebf9d8265db5",
	}
	network.GetNetInstance().BuildGroupNet(groupID, members)

	//go func() {
	//	var count uint32 = 60000
	//	body := make([]byte, 1024 * (1024 / 2))//[]byte("helloworld")
	//	for {
	//
	//		fmt.Println(len(network.GetNetInstance().ConnInfo()), network.GetNetInstance().ConnInfo())
	//		if len(network.GetNetInstance().ConnInfo()) >= 10 {
	//			count++
	//			msg := network.Message{
	//				Code: count,
	//				Body: body,
	//			}
	//			network.GetNetInstance().SpreadAmongGroup(groupID, msg)
	//		}
	//		time.Sleep(2 * time.Second)
	//	}
	//}()
}

func (gtas *Gtas) TestSendEveryone() {
	members := []string{
		"0x0077464c5d2b1152d945fee3c3e3f2bf09fdfad68e22d954e34e8b1dfa550ff5",
		"0x032245e4a28e9505c2b7d088e055641e7e38f5192219de966436e53f626dbd31",
		"0x038d881c8b2d7976e9e806b6a5a590601dee0cc8be48106aab1c83adbe4aeac6",
		"0x0959fc1dff5127ff2b7dede48be6a60c8ee8533543775462fdd4f5b5d6a5f5e4",
		"0x0a1f13a1bf593540862e31fc959eff50ff8c1fe67c2bfa2a6ee05e3c745e803d",
		"0x0a26c83ccd511329db33fed03751f1b3e7b8bf1272fc450a8d343d51bfb98c3b",
		"0x0ba0e0074b7f84250184f294fe0be8f4056174c826bad09b6982a0d93312a30c",
		"0x0e6a9d512d6a8bc82766ba4227c21f0dfe7718e3496ec0908cd8540139c0c619",
		"0x0fcd9cbd225544c2f92fee071c2cfc7c5d59854c87e5ee26c40a25f438aa3924",
		"0x105a0823aa54b47fb0eee1ebd6a8e3e45101562c011635087a75cc6b8701ef49",
		"0x13e8930b896623c1363f2d7aa1be79ac09c25e3a51f3d8ba07ad2c3096659524",
		"0x1551f3b4a14de262cb5844704fbf6269d40219fee86c4f19a903ea16c7b696e6",
		"0x158cf8df0316d9f6a20615d62e6ae7c45303181001e3158db579e94946dbd49f",
		"0x1c7ff1c98f03a18f2c84676862e653ad733d71291f1db2ab530370cf84e14dc6",
		"0x1d574d31bdbfdcbede3e05756adbd6326e029d46054e97b4c3443a40bc8ede7a",
		"0x1eebf400815d74de39a1179c38b959fd16fc287ac06f6940580ea13d96757624",
		"0x215b8758ff2af792679952a7c336bcacb62eedd72879cb20e658fdcec9771359",
		"0x232cfa4827ad61227376111504fd42b2387a0997d9d26a6133242177025433f2",
		"0x25a7f2decbb8b75d4d1d2b7c058dcc61c5dff7b3991b546d231960c958b01505",
		"0x2714712ded76e1a68a1d301514f275535e8ea3a64bb1e28e094d801109a05a41",
		"0x27940ae3b775788fc4bccecbf18be2d39158c7a03c9f2f26da9354c0c75e62e9",
		"0x2819c2b0c7a100bd3ba214b3758c78dc70c5591df78d9082a6b9164c61506052",
		"0x2959c45de9fe3b051d999a940f99450999110fe6a4d04cb94a2f6547ebc4dcef",
		"0x2c570d07a5c1295f323346b0f61a14efaa415cdf2ed0d37006726221b638cb86",
		"0x2ecca96cf50799e005c727eaa974bf4142d28a4f352d10cc7c4535924f4354b1",
		"0x2edbfe440bfdad74f651243ea27d4a529c2f54cb3572ca599abfa6e86c663b08",
		"0x3045d9f52fc7125bd48fdd1e2126ce8ea8ee3066c4c9c309de1e985c7c54a1c3",
		"0x30cd28b2ad3e620465302d24061996c2a81ef1cea0b0f455ab0e6d86cf41d1e2",
		"0x340568cf75cc4741c43c5afd7c20a6a60c610fc1ddc828ad786f23b7413034f7",
		"0x347ba167ea7bb5fb7fb031e703159a204fca3b6b5c0377a9149e712212a47890",
		"0x34859de1e30f5ac64dcce9a78304b8a09c1e3d731555cdcd400b5ab810310344",
		"0x35b3f591ef58f597c2755fd96e9369c023db8b7e9ff084af0f7ca7924f8072f5",
		"0x3abdfc6ff3d7a13c36edbddb9451190965bec2f623633b67eaed6658b7ac9587",
		"0x3fdaddb996208af206e123560199decfd0efb5d1e65ea4cbe3bfc36d8af74b26",
		"0x3ff6607f0ca170dd749d0012f8ba90e2cdd8d59e3727a2d2457a6259357c97ea",
		"0x400d124c8c2e07a5a541b916679be6dcba1df25e318cf07447d5fbdaec900590",
		"0x41e14ff4b43dbf38ce7927f2fe9f8b82e234a510aaf94e22f96660b4a0f38dee",
		"0x49db49f9de6dc6eb8ef7b62d717502965efe6e416a34af7fa3df621053080370",
		"0x4db7f8541368edd508e164a9423fdc9a2917174e1f49fe993f9ae2ef3effa11d",
		"0x52e6ba4d2585c8f5ec5c94402b3d67c2cbc415486c40d30045160aafc8cbf1d4",
		"0x5344632d941279b8bb669b734ea42cc08ecb89a56059f24e3738b6cad8fa4b23",
		"0x53556a84b524a6d9484b2560d50d3fea55cd42b49b7bce98794df92e55356442",
		"0x5470277e95c66f8e6c69b62ca9c7d0508fe12d8a270d49f3201f362c6f94d93d",
		"0x597ebcdffaf98d6f82856c667517c34199248ba7f280b4a5bd95a0d69e22c9c1",
		"0x6495cc494946f03ae3e89f2363ac88e52f11ed7964c75aaa01567a16d3deeee0",
		"0x751f76aff30e2b8d58245a7e60e3f6a2173a170f48da9d22e5fa43afcbf45c65",
		"0x77cef301c02f209ea93abbce4e4a209e78432bfa9f5652643c3af115f8501fe5",
		"0x7cd8a381168d3de03391930fb1392ddb3dc8659bbb0f0923fc7e7273419a7e20",
		"0x83584da1d2966c31b65b9a404b62ecbb439c2c67eb78e8ed0add8f4249b16f6d",
		"0x8a058a2b22d22a62a9de40d7f4002f24550b38668992f728b86461de4c334792",
		"0x8fef5b6a83f0f29af5630f5fde16b1a21cf63369683ea3d5d1ccf66ec3620aa7",
		"0x90a3fc82433306be85f7e84993812be48a50a980627cfd0441adb0bc7fe2d207",
		"0x93a8b30fc8303907deaca066a2cd83add32855870047a46e0ea7a2473c423d01",
		"0x98dfc297280754201f659dae82873bcc45f899739d6997096b839678863d62bf",
		"0x99489535d3cb3e3ea032b21e49d56e9c6e6597232404f15f1b06cf93016d0302",
		"0x9cd65a1289752f801b3307eddab141fc372459709a55b9385c3e9c17f35bf83e",
		"0x9d25efaf407d8c766f74cf08555c7649347f8bdb662858077f7664ce83543f91",
		"0xa36a0ed8549ab42789e294b667c528072060a8b9c447fe37d51ebcbe325d4a39",
		"0xa97a53609050f7fca6307e491b3515a8a1a15b8c978391a5a7b5c0ef016fddb4",
		"0xa98a31428e7888b3bfca8ac3afbd3d94dc76713fe21f6b60cf9e1b9a90893808",
		"0xae089cdd1a6809cb4ef05ca46ebc14d9debd7f25779ef877ca91f0367ee9d47f",
		"0xae346fd20103cdf71d79e48665e3966f4fb4ac1018ad0af8ca7dba779f2ca11c",
		"0xae89a12f527df0cbf5c43bd6a9e31f1bac03f7bd479b0b5d3049a6fca2a65bd1",
		"0xb31da91b385f7279c45aeab5604efd48fe72e2bfb35d278341dfce80aebf5f50",
		"0xb327c461305f85e12679a7b98d640addc646e305ebe243fb83299416c82ea9dd",
		"0xb642220718e794c48b91dbc41b7aac7a18bc9459476c4f84aa8e36f09ac5c0f1",
		"0xb8d7d57f03da0fc4bd0892a3f4fb741c0aa6a3b94489c6e06f8ff8f6642e01d5",
		"0xb8f480d0f83ff0b7d61b378f23d0f3ea71c9f75398f9e51eb1c6cde91f29ae6a",
		"0xbb75c2d7646dc98d8aa8db31a0a292b47de3b3d5b0eb540e23572dac6387eb75",
		"0xbc491e69008781e3ddf13fcf8df681e3672f2efb525c451c7797c4c83340e8ef",
		"0xbea8dd36fffb62f3b1b9c0354a524d0500151a0b4b189cd183611743f268f4f5",
		"0xbec8ef83be0ce5adf1b150130c95586d82435d22dc1b3765f1c7dcf85e14b9ff",
		"0xc1c609fbcf3430bd9bed7c5bfd518dd412c43fb34a26b5a455e296a46f65794e",
		"0xcdf9258aa8478d88f5ef4cc301454882336be16f822d7a06a8b168e7bef0f570",
		"0xd88f1b55f803bd101b46be55037fe4134c136d55fcc32e65e31e4b35cf2bb19f",
		"0xdc395846ba4a6693d37df9e27b15dfb3ab0ca65e480c71f1b3d03e0bece458bd",
		"0xdddc95a0d0b2a29f556991a1888869de6c59113aad37cf7cc2bd17419cca3085",
		"0xe056fd63f7563834ed3b7a507dd0657a4a6fba224cace18fb7e6102ae6065277",
		"0xe755cca13c9a946af6ec249fd2169fc43a8d6123fa0f000340ee755447ff235f",
		"0xea77d068e5acd93b1804f0f29b1641113b4f2e54c94303b4fbc41d8d82de53cb",
		"0xeacaf3b8fad2b4522807aa402417a29391b05d44c5015e0fd467646eaeb15843",
		"0xed5f0f1e9107f82f21aeff5b4858787362edb83756fe243c2e7e1d45ad533b99",
		"0xef3b15976491d037367f25e8d77ddcbc5867996cc08cea4215d6999729441583",
		"0xef774e74e128e2dd51f7ca20f1d22c5d0f43fa9374ef11f8f9a11f26e27f5332",
		"0xef8bdb5105791b0e757a814b3ed4248652ba16e9289bcceab6da69f636bf711e",
		"0xf1510ed56f854cc98e93b5337de218cd5e589c9d237eb60eaf2c45f81966be1a",
		"0xfb4db4f9e3634496bd721d607d60f3ddf804b0be33fe8a36c2b3b61fc94ddd24",
		"0xfcb213902b0b4fad6c073ed1e771b181e5757825aad86f076e271b839dbdbaa4",
		"0xfd7d9063e3bf57c69f8756c7bece466aff064af8163b2fb25b42e5861347cfba",
		"0xfdeecd56d3191d9f3c1cccc04ac2bc1f469b036810ca08cda3bce916c7bf31af",
		"0xff1522f11a666e69d7fb96fcf47582e289966a9b6618417d1830ebf9d8265db5",
	}
	go func() {
		var count uint32 = 72102
		body := make([]byte, 1024 * (1024 / 2))//[]byte("helloworld")
		for {
			fmt.Println(len(network.GetNetInstance().ConnInfo()), network.GetNetInstance().ConnInfo())
			if len(network.GetNetInstance().ConnInfo()) >= 2 {
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
		var count uint32 = 92000
		body := make([]byte, 1024 * (1024 / 2))//[]byte("helloworld")
		for {
			fmt.Println(len(network.GetNetInstance().ConnInfo()), network.GetNetInstance().ConnInfo())
			if len(network.GetNetInstance().ConnInfo()) >= 2 {
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

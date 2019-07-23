package logical

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
)

var data uint64
var wg sync.WaitGroup

func TestAtomic(t *testing.T) {
	wg.Add(10000)
	for i := 0; i < 100; i++ {
		go addI()
	}
	wg.Wait()
	if data != 10000 {
		t.Fatalf("except 10000,but got %v", data)
	}
}

func addI() {
	for i := 0; i < 100; i++ {
		atomic.AddUint64(&data, 1)
		wg.Done()
	}
}

var processorTest *Processor

func TestProcessor_OnMessageResponseProposalBlock(t *testing.T) {
	_ = initContext4Test()
	defer clear()

	pt := NewProcessorTest()
	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	txs := make([]*types.Transaction, 0)

	type args struct {
		msg *model.ResponseProposalBlock
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "ok",
			args: args{
				msg: &model.ResponseProposalBlock{
					pt.blockHeader.Hash,
					txs,
				},
			},
			expected: "success",
		},
		{
			name: "bad hash",
			args: args{
				msg: &model.ResponseProposalBlock{
					common.EmptyHash,
					txs,
				},
			},
			expected: "vctx is nil",
		},
		{
			name: "block exist",
			args: args{
				msg: &model.ResponseProposalBlock{
					core.BlockChainImpl.QueryTopBlock().Hash,
					txs,
				},
			},
			expected: "block onchain",
		},
	}

	p := processorTest
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := p.OnMessageResponseProposalBlock(tt.args.msg)
			if msg != tt.expected {
				t.Errorf("OnMessageResponseProposalBlock failed, expected %s but got %s", tt.expected, msg)
			}
		})
	}
}

func TestProcessor_OnMessageReqProposalBlock(t *testing.T) {
	type fields struct {
		mi               *model.SelfMinerDO
		genesisMember    bool
		minerReader      *MinerPoolReader
		blockContexts    *castBlockContexts
		futureVerifyMsgs *FutureMessageHolder
		proveChecker     *proveChecker
		Ticker           *ticker.GlobalTicker
		ready            bool
		MainChain        types.BlockChain
		vrf              atomic.Value
		NetServer        net.NetworkServer
		conf             common.ConfManager
		isCasting        int32
		castVerifyCh     chan *types.BlockHeader
		blockAddCh       chan *types.BlockHeader
		groupReader      *groupReader
		ts               time.TimeService
		rewardHandler    *RewardHandler
	}
	type args struct {
		msg      *model.ReqProposalBlock
		sourceID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Processor{
				mi:               tt.fields.mi,
				genesisMember:    tt.fields.genesisMember,
				minerReader:      tt.fields.minerReader,
				blockContexts:    tt.fields.blockContexts,
				futureVerifyMsgs: tt.fields.futureVerifyMsgs,
				proveChecker:     tt.fields.proveChecker,
				Ticker:           tt.fields.Ticker,
				ready:            tt.fields.ready,
				MainChain:        tt.fields.MainChain,
				vrf:              tt.fields.vrf,
				NetServer:        tt.fields.NetServer,
				conf:             tt.fields.conf,
				isCasting:        tt.fields.isCasting,
				castVerifyCh:     tt.fields.castVerifyCh,
				blockAddCh:       tt.fields.blockAddCh,
				groupReader:      tt.fields.groupReader,
				ts:               tt.fields.ts,
				rewardHandler:    tt.fields.rewardHandler,
			}
			p.OnMessageReqProposalBlock(tt.args.msg, tt.args.sourceID)
		})
	}
}

func initContext4Test() error {
	common.InitConf("./tas_config_test.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	err := middleware.InitMiddleware()
	if err != nil {
		return err
	}
	err = core.InitCore(NewConsensusHelper4Test(groupsig.ID{}), nil)
	core.GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})

	processorTest = &Processor{}
	sk := common.HexToSecKey(getAccount().Sk)
	minerInfo, _ := model.NewSelfMinerDO(sk)
	InitConsensus()
	processorTest.Init(minerInfo, common.GlobalConf)
	//hijack some external interface to avoid error
	processorTest.MainChain = &chain4Test{core.BlockChainImpl}
	processorTest.NetServer = &networkServer4Test{processorTest.NetServer}
	return err
}

func getAccount() *Account4Test {
	var ksr = new(KeyStoreRaw4Test)
	ksr.Key = common.Hex2Bytes("A8cmFJACR7VqbbuKwDYu/zj/hn6hcox97ujw2TNvCYk=")
	secKey := new(common.PrivateKey)
	if !secKey.ImportKey(ksr.Key) {
		_ = fmt.Errorf("failed to import key")
		return nil
	}

	account := &Account4Test{
		Sk:       secKey.Hex(),
		Pk:       secKey.GetPubKey().Hex(),
		Address:  secKey.GetPubKey().GetAddress().Hex(),
		Password: "Password",
	}
	return account
}

func NewConsensusHelper4Test(id groupsig.ID) types.ConsensusHelper {
	return &ConsensusHelperImpl4Test{ID: id}
}

type ConsensusHelperImpl4Test struct {
	ID groupsig.ID
}

func (helper *ConsensusHelperImpl4Test) GenerateGenesisInfo() *types.GenesisInfo {
	info := &types.GenesisInfo{}
	info.Group = &group4Test{&GroupHeader4Test{}}
	info.VrfPKs = make([][]byte, 0)
	info.Pks = make([][]byte, 0)
	info.VrfPKs = append(info.VrfPKs, common.FromHex("vrfPks"))
	info.Pks = append(info.Pks, common.FromHex("Pks"))
	return info
}

func (helper *ConsensusHelperImpl4Test) VRFProve2Value(prove []byte) *big.Int {
	if len(prove) == 0 {
		return big.NewInt(0)
	}
	return big.NewInt(1)
}

func (helper *ConsensusHelperImpl4Test) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	//return Proc.checkProveRoot(bh)
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) VerifyNewBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) VerifyBlockHeader(bh *types.BlockHeader) (bool, error) {
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) CheckGroup(g *types.GroupI) (ok bool, err error) {
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) VerifyRewardTransaction(tx *types.Transaction) (ok bool, err error) {
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) EstimatePreHeight(bh *types.BlockHeader) uint64 {
	height := bh.Height
	if height == 1 {
		return 0
	}
	return height - uint64(math.Ceil(float64(bh.Elapsed)/float64(model.Param.MaxGroupCastTime)))
}

func (helper *ConsensusHelperImpl4Test) CalculateQN(bh *types.BlockHeader) uint64 {
	return uint64(11)
}

type Account4Test struct {
	Address  string
	Pk       string
	Sk       string
	Password string
}

func (a *Account4Test) MinerSk() string {
	return a.Sk
}

type KeyStoreRaw4Test struct {
	Key     []byte
	IsMiner bool
}

type group4Test struct {
	header types.GroupHeaderI
}

func (g *group4Test) Header() types.GroupHeaderI {
	return g.header
}

func (g *group4Test) Members() []types.MemberI {
	members := make([]types.MemberI, 0)
	mem := &member4Test{
		common.FromHex("0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676"),
		common.FromHex("0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676")}
	members = append(members, mem)
	return members
}

type GroupHeader4Test struct {
}

func (g *GroupHeader4Test) Seed() common.Hash {
	return common.EmptyHash
}
func (g *GroupHeader4Test) WorkHeight() uint64 {
	return uint64(1)
}
func (g *GroupHeader4Test) DismissHeight() uint64 {
	return uint64(1)
}
func (g *GroupHeader4Test) PublicKey() []byte {
	return make([]byte, 0)
}
func (g *GroupHeader4Test) Threshold() uint32 {
	return uint32(1)
}
func (g *GroupHeader4Test) GroupHeight() uint64 {
	return uint64(1)
}

type member4Test struct {
	Id []byte
	Pk []byte
}

func (m *member4Test) ID() []byte {
	return m.Id
}

func (m *member4Test) PK() []byte {
	return m.Pk
}

type GroupCreateChecker4Test struct {
}

func (g *GroupCreateChecker4Test) CheckEncryptedPiecePacket(packet types.EncryptedSharePiecePacket, ctx types.CheckerContext) error {
	return nil
}
func (g *GroupCreateChecker4Test) CheckMpkPacket(packet types.MpkPacket, ctx types.CheckerContext) error {
	return nil
}
func (g *GroupCreateChecker4Test) CheckGroupCreateResult(ctx types.CheckerContext) types.CreateResult {
	return nil
}
func (g *GroupCreateChecker4Test) CheckOriginPiecePacket(packet types.OriginSharePiecePacket, ctx types.CheckerContext) error {
	return nil
}
func (g *GroupCreateChecker4Test) CheckGroupCreatePunishment(ctx types.CheckerContext) (types.PunishmentMsg, error) {
	return nil, fmt.Errorf("do not need punishment")
}

func clear() {
	fmt.Println("---clear---")
	if core.BlockChainImpl != nil {
		core.BlockChainImpl.Close()
		taslog.Close()
		core.BlockChainImpl = nil
	}

	dir, err := ioutil.ReadDir(".")
	if err != nil {
		return
	}
	for _, d := range dir {
		if d.IsDir() && (d.Name() == "logs" || strings.HasPrefix(d.Name(), "d_") || strings.HasPrefix(d.Name(), "groupstore") ||
			strings.HasPrefix(d.Name(), "database")) {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Println("error while removing /s", d.Name())
			}
		}
	}
	err = os.Remove("tas_config_test.ini")
	if err != nil {
		fmt.Println("error while removing tas_config_test.ini")
	}
}

type chain4Test struct {
	types.BlockChain
}

func (c *chain4Test) AddBlockOnChain(source string, b *types.Block) types.AddBlockResult {
	fmt.Printf("AddBlockOnChain called, source = %v, b = %v\n", source, b)
	return types.AddBlockSucc
}

type networkServer4Test struct {
	net.NetworkServer
}

func (n *networkServer4Test) BroadcastNewBlock(block *types.Block, group *net.GroupBrief) {
	fmt.Printf("BroadcastNewBlock called, block = %v, group = %v \n", block, group)
}

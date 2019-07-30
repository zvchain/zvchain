package logical

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	time2 "time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/time"
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
var existBlockHash = "0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501"

func TestProcessor_OnMessageResponseProposalBlock(t *testing.T) {
	defer clear()
	err := initContext4Test()
	if err != nil {
		t.Errorf("failed to init context: %v\n", err)
	}

	pt := NewProcessorTest()
	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	//processorTest.blockContexts.attachVctx(core.BlockChainImpl.QueryTopBlock(), pt.verifyContext)
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
					common.HexToHash(existBlockHash),
					txs,
				},
			},
			expected: "block onchain",
		},

		{
			name: "tx nil",
			args: args{
				msg: &model.ResponseProposalBlock{
					pt.blockHeader.Hash,
					nil,
				},
			},
			expected: "success",
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
	defer clear()
	err := initContext4Test()
	if err != nil {
		t.Errorf("failed to init context: %v\n", err)
	}

	pt := NewProcessorTest()
	//processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	latestBlock := core.BlockChainImpl.QueryTopBlock()
	block := new(types.Block)
	block.Header = &types.BlockHeader{
		Height:     1,
		ProveValue: common.EmptyHash.Bytes(),
		Castor:     processorTest.mi.ID.Serialize(),
		Group:      pt.verifyGroup.header.Seed(),
		TotalQN:    latestBlock.TotalQN + 5,
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		PreHash:    latestBlock.Hash,
		Nonce:      common.ChainDataVersion,
	}
	block.Header.Hash = block.Header.GenHash()
	processorTest.blockContexts.addProposed(block)

	type args struct {
		msg      *model.ReqProposalBlock
		sourceID string
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "ok",
			args: args{
				msg:      &model.ReqProposalBlock{block.Header.Hash},
				sourceID: "111",
			},
			expected: "success",
		},
		{
			name: "bad hash",
			args: args{
				msg:      &model.ReqProposalBlock{common.EmptyHash},
				sourceID: "111",
			},
			expected: "block is nil",
		},
		{
			name: "ok, adding response count",
			args: args{
				msg:      &model.ReqProposalBlock{block.Header.Hash},
				sourceID: "111",
			},
			expected: "success",
		},
		{
			name: "response count exceed",
			args: args{
				msg:      &model.ReqProposalBlock{block.Header.Hash},
				sourceID: "111",
			},
			expected: "response count exceed:3 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := processorTest
			msg := p.OnMessageReqProposalBlock(tt.args.msg, tt.args.sourceID)
			if msg != tt.expected {
				t.Errorf("OnMessageReqProposalBlock failed, expected %s but got %s", tt.expected, msg)
			}
		})
	}
}

//---------------below are functions for mock data------

func initContext4Test() error {
	index := rand.Int()
	path := fmt.Sprintf("./tas_config_test%d.ini", index)
	common.InitConf(path)
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
	clearTicker()
	return err
}

func clearTicker() {
	if core.MinerManagerImpl != nil {
		core.MinerManagerImpl.ClearTicker()
	}
	if core.TxSyncer != nil {
		core.TxSyncer.ClearTicker()
	}
}

type MinerPoolTest struct {
	pks         []groupsig.Pubkey
	ids         []groupsig.ID
	verifyGroup *verifyGroup
}

func NewMinerPoolTest(pks []groupsig.Pubkey, ids []groupsig.ID, verifyGroup *verifyGroup) *MinerPoolTest {
	mpt := MinerPoolTest{}
	mpt.pks = pks
	mpt.ids = ids
	mpt.verifyGroup = verifyGroup
	return &mpt
}

func (m MinerPoolTest) GetLatestMiner(address common.Address, mType types.MinerType) *types.Miner {
	return &types.Miner{
		Status:       types.MinerStatusActive,
		Type:         types.MinerTypeProposal,
		ID:           m.ids[1].Serialize(),
		PublicKey:    m.pks[1].Serialize(),
		VrfPublicKey: base.Hex2VRFPublicKey("0x666a589f1bbc74ad4bc24c67c0845bd4e74d83f0e3efa3a4b465bf6e5600871c"),
	}
}

//copy from ver_test.go
//const pks = "885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"
//const sks = "1fcce948db9fc312902d49745249cfd287de1a764fd48afb3cd0bdd0a8d74674885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"
func (m MinerPoolTest) GetMiner(address common.Address, mType types.MinerType, height uint64) *types.Miner {
	mi := &types.Miner{
		Status:    types.MinerStatusActive,
		Type:      types.MinerTypeProposal,
		ID:        m.ids[1].Serialize(),
		PublicKey: m.pks[1].Serialize(),
		//VrfPublicKey: base.Hex2VRFPublicKey("0x666a589f1bbc74ad4bc24c67c0845bd4e74d83f0e3efa3a4b465bf6e5600871c"),
		VrfPublicKey: base.Hex2VRFPublicKey("885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"),
		Stake:        100000,
	}
	if address == common.HexToAddress(inActiveCastor) {
		mi.Status = types.MinerStatusPrepare
	}
	return mi
}

func (MinerPoolTest) GetProposalTotalStake(height uint64) uint64 {
	return 1000000
}

func (MinerPoolTest) GetAllMiners(mType types.MinerType, height uint64) []*types.Miner {
	panic("implement me")
}

func clear() {
	fmt.Println("---clear---")
	if core.BlockChainImpl != nil {
		core.BlockChainImpl.Close()
		taslog.Close()
		core.BlockChainImpl = nil
	}
	common.GlobalConf = nil

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
		if strings.HasPrefix(d.Name(), "tas_config_test") {
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Println("error while removing /s", d.Name())
			}
		}
		if d.Name() == "groupsk.store" {
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Println("error while removing /s", d.Name())
			}
		}

	}

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
	n := 5 //member number
	info := &types.GenesisInfo{}
	info.Group = &group4Test{n, &GroupHeader4Test{}}
	info.VrfPKs = make([][]byte, 0)
	info.Pks = make([][]byte, 0)
	for i := 0; i < n; i++ {
		info.VrfPKs = append(info.VrfPKs, common.UInt32ToByte(uint32(i)))
		info.Pks = append(info.Pks, common.UInt32ToByte(uint32(i)))
	}
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
	n      int
	header types.GroupHeaderI
}

func (g *group4Test) Header() types.GroupHeaderI {
	return g.header
}

func (g *group4Test) Members() []types.MemberI {
	members := make([]types.MemberI, 0)
	for i := 0; i < g.n; i++ {
		mem := &member4Test{
			common.UInt32ToByte(uint32(i)),
			common.UInt32ToByte(uint32(i))}
		members = append(members, mem)
	}
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

type chain4Test struct {
	types.BlockChain
}

func (c *chain4Test) AddBlockOnChain(source string, b *types.Block) types.AddBlockResult {
	fmt.Printf("AddBlockOnChain called, source = %v, b = %v\n", source, b)
	return types.AddBlockSucc
}

func (c *chain4Test) HasBlock(hash common.Hash) bool {

	if hash == common.HexToHash(existBlockHash) {
		return true
	} else {
		return false
	}
}

func (c *chain4Test) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	if hash == common.HexToHash("0x01") {
		return nil
	}
	if hash == common.HexToHash("0x02") {
		return &types.BlockHeader{CurTime: time.TimeToTimeStamp(time2.Now()) - 5, Height: 1, Random: common.FromHex("0x03")}
	}
	if hash == common.HexToHash("0x03") {
		return &types.BlockHeader{CurTime: time.TimeToTimeStamp(time2.Now()) - 8, Height: 2, Random: common.FromHex("0x03")}
	}
	if hash == common.HexToHash(goodPreHash) {
		return &types.BlockHeader{CurTime: time.TimeToTimeStamp(time2.Now()) - 8, Height: 2, Random: common.FromHex("0x03")}
	}
	return &types.BlockHeader{CurTime: time.TimeToTimeStamp(time2.Now()) - 2, Random: common.FromHex("0x03")}
}

type networkServer4Test struct {
	net.NetworkServer
}

func (n *networkServer4Test) BroadcastNewBlock(block *types.Block, group *net.GroupBrief) {
	fmt.Printf("BroadcastNewBlock called, block = %v, group = %v \n", block, group)
}

func (n *networkServer4Test) ResponseProposalBlock(msg *model.ResponseProposalBlock, target string) {
	fmt.Printf("BroadcastNewBlock called, msg = %v, target = %v \n", msg, target)
}

func (n *networkServer4Test) SendVerifiedCast(cvm *model.ConsensusVerifyMessage, gSeed common.Hash) {
	fmt.Printf("SendVerifiedCast called, cvm = %v, gSeed = %v \n", cvm, gSeed)
}

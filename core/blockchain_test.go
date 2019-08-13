////   Copyright (C) 2018 ZVChain
////
////   This program is free software: you can redistribute it and/or modify
////   it under the terms of the GNU General Public License as published by
////   the Free Software Foundation, either version 3 of the License, or
////   (at your option) any later version.
////
////   This program is distributed in the hope that it will be useful,
////   but WITHOUT ANY WARRANTY; without even the implied warranty of
////   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
////   GNU General Public License for more details.
////
////   You should have received a copy of the GNU General Public License
////   along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
package core

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/storage/account"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware"
	zvtime "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
)

var source = "100"

func TestPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	fmt.Println("Current test filename: " + filename)
}

func TestBlockChain_AddBlock(t *testing.T) {
	err := initContext4Test()
	defer clear()
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}

	//BlockChainImpl.Clear()
	initBalance()

	queryAddr := "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"
	b := BlockChainImpl.GetBalance(common.HexToAddress(queryAddr))
	addr := genHash("1")
	fmt.Printf("balance = %d \n", b)
	fmt.Printf("addr = %s \n", common.BytesToAddress(addr))

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 1000000 {
		//t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}
	//	code := `
	//import account
	//def Test(a, b, c, d):
	//	print("hehe")
	//`

	nonce := uint64(1)
	// 交易1
	tx := genTestTx(500, "100", nonce, 1)
	var sign = common.BytesToSign(tx.Sign)
	pk, err := sign.RecoverPubkey(tx.Hash.Bytes())
	src := pk.GetAddress()
	balance := uint64(100000000)

	stateDB, err := BlockChainImpl.LatestAccountDB()
	if err != nil {
		t.Fatalf("get status error!")
	}
	oldBalance := stateDB.GetBalance(src)
	stateDB.AddBalance(src, new(big.Int).SetUint64(balance))
	if err != nil {
		t.Fatalf("error")
	}
	balance2 := stateDB.GetBalance(src).Uint64()
	if balance2 != balance+oldBalance.Uint64() {
		t.Fatalf("set balance fail")
	}

	_, err = txpool.AddTransaction(tx)

	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}
	//txpool.AddTransaction(genContractTx(1, 20000000, "1", "", 1, 0, []byte(code), nil, 0))
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine([]byte("1"), common.Uint64ToByte(0))))
	//交易2
	nonce++
	_, err = txpool.AddTransaction(genTestTx(500, "2", nonce, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}

	//交易3 执行失败的交易
	nonce++
	_, err = txpool.AddTransaction(genTestTx(500, "2", nonce, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}
	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")

	// 铸块1
	block := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 0, *castor, groupid)

	fmt.Printf("block.Header.CurTime = %v \n", block.Header.CurTime)
	time.Sleep(time.Second * 2)
	delta := zvtime.TSInstance.Since(block.Header.CurTime)

	if delta > 3 || delta < 1 {
		t.Fatalf("zvtime.TSInstance.Since test failed, delta should be 2 but got %v", delta)
	}

	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链

	if 0 != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block")
	}

	//最新块是块1
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 1 != blockHeader.Height {
		t.Fatalf("add block1 failed")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999999 {
		//t.Fatalf("fail to transfer 1 from 1  to 2")
	}

	// 池子中交易的数量为0

	if 0 != len(txpool.GetReceived()) {
		t.Fatalf("fail to remove transactions after addBlock")
	}

	//交易3
	transaction := genTestTx(1111, "1", 4, 10)
	sign = common.BytesToSign(transaction.Sign)
	pk, err = sign.RecoverPubkey(transaction.Hash.Bytes())
	src = pk.GetAddress()

	stateDB, error := BlockChainImpl.LatestAccountDB()
	if error != nil {
		t.Fatalf("status failed")
	}
	stateDB.AddBalance(src, new(big.Int).SetUint64(111111111222))
	_, err = txpool.AddTransaction(transaction)
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}

	//txpool.AddTransaction(genContractTx(1, 20000000, "1", contractAddr.GetHexString(), 3, 0, []byte(`{"FuncName": "Test", "Args": [10.123, "ten", [1, 2], {"key":"value", "key2":"value2"}]}`), nil, 0))
	fmt.Println(contractAddr.Hex())
	// 铸块2
	block2 := BlockChainImpl.CastBlock(2, common.Hex2Bytes("123"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add empty block")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999989 {
		//t.Fatalf("fail to transfer 10 from 1 to 2")
	}

	//最新块是块2
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block2.Header.Hash || block.Header.Hash != block2.Header.PreHash {
		t.Fatalf("add block2 failed")
	}
	blockHeader = BlockChainImpl.QueryBlockByHash(block2.Header.Hash).Header
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHash, hash: %x ", block2.Header.Hash)
	}

	blockHeader = BlockChainImpl.QueryBlockByHeight(2).Header
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHeight, height: %d ", 2)
	}

	// 铸块3 空块
	block3 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("125"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block3) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 3 != blockHeader.Height || blockHeader.Hash != block3.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block4 := BlockChainImpl.CastBlock(4, common.Hex2Bytes("126"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block4) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 4 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block5 := BlockChainImpl.CastBlock(5, common.Hex2Bytes("126"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block5) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块5
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 5 != blockHeader.Height || blockHeader.Hash != block5.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	BlockChainImpl.Close()
}

func TestBlockChain_CastingBlock(t *testing.T) {
	err := initContext4Test()
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clear()
	castor := []byte{1, 2}
	group := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("1"), 1, castor, group)
	if nil == block1 {
		t.Fatalf("fail to cast block1")
	}

	BlockChainImpl.Close()
}

func TestBlockChain_GetBlockMessage(t *testing.T) {
	err := initContext4Test()
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clear()
	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("125"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block1) {
		t.Fatalf("fail to add empty block")
	}

	block2 := BlockChainImpl.CastBlock(2, common.Hex2Bytes("1256"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add empty block")
	}

	block3 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("1257"), 0, *castor, groupid)

	if 0 != BlockChainImpl.AddBlockOnChain(source, block3) {
		t.Fatalf("fail to add empty block")
	}

	if 3 != BlockChainImpl.Height() {
		t.Fatalf("fail to add 3 blocks")
	}
	chain := BlockChainImpl.(*FullBlockChain)

	header1 := chain.queryBlockHeaderByHeight(uint64(1))
	header2 := chain.queryBlockHeaderByHeight(uint64(2))
	header3 := chain.queryBlockHeaderByHeight(uint64(3))

	b1 := chain.queryBlockByHash(header1.Hash)
	b2 := chain.queryBlockByHash(header2.Hash)
	b3 := chain.queryBlockByHash(header3.Hash)

	fmt.Printf("1: %d\n", b1.Header.Nonce)
	fmt.Printf("2: %d\n", b2.Header.Nonce)
	fmt.Printf("3: %d\n", b3.Header.Nonce)

}

func TestBlockChain_GetTopBlocks(t *testing.T) {
	err := initContext4Test()
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clear()

	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")

	var i uint64
	for i = 1; i < 2000; i++ {
		block := BlockChainImpl.CastBlock(i, common.Hex2Bytes(strconv.FormatInt(int64(i), 10)), 0, *castor, groupid)

		if 0 != BlockChainImpl.AddBlockOnChain(source, block) {
			t.Fatalf("fail to add empty block")
		}
	}
	chain := BlockChainImpl.(*FullBlockChain)
	lent := chain.topBlocks.Len()
	fmt.Printf("len = %d \n", lent)
	if 20 != chain.topBlocks.Len() {
		t.Fatalf("error for size:20")
	}

	for i = BlockChainImpl.Height() - 19; i < 2000; i++ {
		lowestLDB := chain.queryBlockHeaderByHeight(i)
		if nil == lowestLDB {
			t.Fatalf("fail to get lowest block from ldb,%d", i)
		}

		lowest, ok := chain.topBlocks.Get(lowestLDB.Hash)
		if !ok || nil == lowest {
			t.Fatalf("fail to get lowest block from cache,%d", i)
		}
	}
}

func TestBlockChain_StateTree(t *testing.T) {
	err := initContext4Test()
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clear()

	chain := BlockChainImpl.(*FullBlockChain)

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 100 {
		//t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")

	block0 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 0, *castor, groupid)
	// 上链

	if 0 != BlockChainImpl.AddBlockOnChain(source, block0) {
		t.Fatalf("fail to add block0")
	}

	// 交易1
	_, _ = txpool.AddTransaction(genTestTx(12345, "1", 1, 1))

	//交易2
	_, _ = txpool.AddTransaction(genTestTx(123456, "2", 2, 2))

	// 交易3 失败的交易
	_, _ = txpool.AddTransaction(genTestTx(123457, "1", 2, 3))

	// 铸块1
	block := BlockChainImpl.CastBlock(2, common.Hex2Bytes("12"), 0, *castor, groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链

	if 0 != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block")
	}

	// 铸块2
	block2 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("22"), 0, *castor, groupid)
	if nil == block2 {
		t.Fatalf("fail to cast new block")
	}

	// 上链

	if 0 != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add block")
	}

	fmt.Printf("state: %d\n", chain.latestBlock.StateTree)

	// 铸块3
	block3 := BlockChainImpl.CastBlock(4, common.Hex2Bytes("12"), 0, *castor, groupid)
	if nil == block3 {
		t.Fatalf("fail to cast new block")
	}

	// 上链

	if 0 != BlockChainImpl.AddBlockOnChain(source, block3) {
		t.Fatalf("fail to add block")
	}

	fmt.Printf("state: %d\n", chain.getLatestBlock().StateTree)
}

var privateKey = "0x045c8153e5a849eef465244c0f6f40a43feaaa6855495b62a400cc78f9a6d61c76c09c3aaef393aa54bd2adc5633426e9645dfc36723a75af485c5f5c9f2c94658562fcdfb24e943cf257e25b9575216c6647c4e75e264507d2d57b3c8bc00b361"

func genTestTx(price uint64, target string, nonce uint64, value uint64) *types.Transaction {

	targetbyte := common.BytesToAddress(genHash(target))

	tx := &types.Transaction{
		GasPrice: types.NewBigInt(price),
		GasLimit: types.NewBigInt(5000),
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    types.NewBigInt(value),
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(privateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())

	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source

	return tx
}

func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return common.Sha256(bytes3)
}

func initBalance() {
	blocks := GenCorrectBlocks()
	stateDB1, _ := account.NewAccountDB(common.Hash{}, BlockChainImpl.(*FullBlockChain).stateCache)
	stateDB1.AddBalance(common.HexToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"), new(big.Int).SetUint64(100000000000000000))
	exc := &executePostState{state: stateDB1}
	root := stateDB1.IntermediateRoot(true)
	blocks[0].Header.StateTree = common.BytesToHash(root.Bytes())
	_, _ = BlockChainImpl.(*FullBlockChain).commitBlock(blocks[0], exc)
}

func clear() {
	fmt.Println("---clear---")
	if BlockChainImpl != nil {
		BlockChainImpl.Close()
		//taslog.Close()
		BlockChainImpl = nil
	}

	dir, err := ioutil.ReadDir(".")
	if err != nil {
		return
	}
	for _, d := range dir {
		if d.IsDir() && (strings.HasPrefix(d.Name(), "d_") || strings.HasPrefix(d.Name(), "groupstore") ||
			strings.HasPrefix(d.Name(), "database")) {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Println("error while removing /s", d.Name())
			}
		}
	}

}

func clearTicker() {
	if MinerManagerImpl != nil && MinerManagerImpl.ticker != nil {
		MinerManagerImpl.ticker.RemoveRoutine("build_virtual_net")
	}
	if TxSyncer != nil && TxSyncer.ticker != nil {
		TxSyncer.ticker.RemoveRoutine(txNotifyRoutine)
		TxSyncer.ticker.RemoveRoutine(txReqRoutine)
	}
}

func initContext4Test() error {
	common.InitConf("../tas_config_all.ini")
	network.Logger = log.P2PLogger
	err := middleware.InitMiddleware()
	if err != nil {
		return err
	}
	BlockChainImpl = nil

	err = InitCore(NewConsensusHelper4Test(groupsig.ID{}), getAccount())
	GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})
	clearTicker()
	return err
}

func getAccount() *Account4Test {
	var ksr = new(KeyStoreRaw4Test)

	ksr.Key = common.Hex2Bytes("A8cmFJACR7VqbbuKwDYu/zj/hn6hcox97ujw2TNvCYk=")
	secKey := new(common.PrivateKey)
	if !secKey.ImportKey(ksr.Key) {
		fmt.Errorf("failed to import key")
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
	return true, nil //上链时不再校验，只在共识时校验（update：2019-04-23）
}

func (helper *ConsensusHelperImpl4Test) VerifyNewBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) VerifyBlockSign(bh *types.BlockHeader) (bool, error) {
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

func (helper *ConsensusHelperImpl4Test) VerifyBlockHeaders(pre, bh *types.BlockHeader) (ok bool, err error) {
	return true, nil
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

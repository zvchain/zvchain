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
	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/params"
	"github.com/zvchain/zvchain/storage/tasdb"
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

func init() {
	log.ELKLogger.SetLevel(logrus.ErrorLevel)
}

func TestPath(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	fmt.Println("Current test filename: " + filename)
}

func TestBlockChain_AddBlock(t *testing.T) {
	err := initContext4Test(t)
	defer clearSelf(t)
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}

	//BlockChainImpl.Clear()
	initBalance()

	queryAddr := "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"
	b := BlockChainImpl.GetBalance(common.StringToAddress(queryAddr))
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
	nonce++
	_, err = txpool.AddTransaction(genTestTx(500, "2", nonce, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}

	nonce++
	_, err = txpool.AddTransaction(genTestTx(500, "2", nonce, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}
	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")

	block := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 0, *castor, groupid)

	fmt.Printf("block.Header.CurTime = %v \n", block.Header.CurTime)
	time.Sleep(time.Second * 2)
	delta := zvtime.TSInstance.SinceSeconds(block.Header.CurTime)

	if delta > 3 || delta < 1 {
		t.Fatalf("zvtime.TSInstance.SinceSeconds test failed, delta should be 2 but got %v", delta)
	}

	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block")
	}

	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 1 != blockHeader.Height {
		t.Fatalf("add block1 failed")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999999 {
		//t.Fatalf("fail to transfer 1 from 1  to 2")
	}

	if 0 != len(txpool.GetReceived()) {
		t.Fatalf("fail to remove transactions after addBlock")
	}

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
	fmt.Println(contractAddr.AddrPrefixString())

	block2 := BlockChainImpl.CastBlock(2, common.Hex2Bytes("123"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add empty block")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999989 {
		//t.Fatalf("fail to transfer 10 from 1 to 2")
	}

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

	block3 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("125"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block3) {
		t.Fatalf("fail to add empty block")
	}
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 3 != blockHeader.Height || blockHeader.Hash != block3.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block4 := BlockChainImpl.CastBlock(4, common.Hex2Bytes("126"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block4) {
		t.Fatalf("fail to add empty block")
	}
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 4 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block5 := BlockChainImpl.CastBlock(5, common.Hex2Bytes("126"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block5) {
		t.Fatalf("fail to add empty block")
	}
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 5 != blockHeader.Height || blockHeader.Hash != block5.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	time.Sleep(1 * time.Second)

	BlockChainImpl.Close()
}

func TestBalanceLackFork(t *testing.T) {
	err := initContext4Test(t)
	defer clearSelf(t)
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}

	initBalance()
	block0 := BlockChainImpl.QueryTopBlock()

	nonce := uint64(1)
	tx := genTestTx(500, "100", nonce, (100000000-1)*common.ZVC)

	txpool := BlockChainImpl.GetTransactionPool()
	_, err = txpool.AddTransaction(tx)

	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}

	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 0, *castor, groupid)
	time.Sleep(time.Second * 2)
	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block1) {
		t.Fatalf("fail to add block")
	}

	// resetTop to make sure block2 can pack the tx and tx2
	_ = BlockChainImpl.resetTop(block0)
	tx2 := genTestTx(500, "100", nonce+1, 1)
	_, err = txpool.AddTransaction(tx2)
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}
	block2 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 1, *castor, groupid)
	time.Sleep(time.Second * 2)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block1) {
		t.Fatalf("fail to add block1")
	}

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add block2")
	}
	lastTop := BlockChainImpl.QueryTopBlock()
	if lastTop.Hash != block2.Header.Hash {
		t.Fatalf("should fork to block2, but not")
	}
}

func TestBlockChain_CastingBlock(t *testing.T) {
	err := initContext4Test(t)
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clearSelf(t)
	castor := []byte{1, 2}
	group := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("1"), 1, castor, group)
	if nil == block1 {
		t.Fatalf("fail to cast block1")
	}

	BlockChainImpl.Close()
}

func TestBlockChain_GetBlockMessage(t *testing.T) {
	err := initContext4Test(t)
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clearSelf(t)
	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("125"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block1) {
		t.Fatalf("fail to add empty block")
	}

	block2 := BlockChainImpl.CastBlock(2, common.Hex2Bytes("1256"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add empty block")
	}

	block3 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("1257"), 0, *castor, groupid)

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block3) {
		t.Fatalf("fail to add empty block")
	}

	if 3 != BlockChainImpl.Height() {
		t.Fatalf("fail to add 3 blocks")
	}
	chain := BlockChainImpl

	header1 := chain.queryBlockHeaderByHeight(uint64(1))
	header2 := chain.queryBlockHeaderByHeight(uint64(2))
	header3 := chain.queryBlockHeaderByHeight(uint64(3))

	b1 := chain.queryBlockByHash(header1.Hash)
	b2 := chain.queryBlockByHash(header2.Hash)
	b3 := chain.queryBlockByHash(header3.Hash)

	fmt.Printf("1: %d\n", b1.Header.Nonce)
	fmt.Printf("2: %d\n", b2.Header.Nonce)
	fmt.Printf("3: %d\n", b3.Header.Nonce)
	time.Sleep(time.Second)
}

func TestBlockChain_GetTopBlocks(t *testing.T) {
	err := initContext4Test(t)
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clearSelf(t)

	castor := new([]byte)
	groupid := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")

	var i uint64
	for i = 1; i < 2000; i++ {
		block := BlockChainImpl.CastBlock(i, common.Hex2Bytes(strconv.FormatInt(int64(i), 10)), 0, *castor, groupid)

		if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block) {
			t.Fatalf("fail to add empty block")
		}
	}
	chain := BlockChainImpl
	lent := chain.topRawBlocks.Len()
	fmt.Printf("len = %d \n", lent)
	if 20 != chain.topRawBlocks.Len() {
		t.Fatalf("error for size:20")
	}

	for i = BlockChainImpl.Height() - 19; i < 2000; i++ {
		lowestLDB := chain.queryBlockHeaderByHeight(i)
		if nil == lowestLDB {
			t.Fatalf("fail to get lowest block from ldb,%d", i)
		}

		lowest, ok := chain.topRawBlocks.Get(lowestLDB.Hash)
		if !ok || nil == lowest {
			t.Fatalf("fail to get lowest block from cache,%d", i)
		}
	}
	time.Sleep(time.Second)
}

func TestBlockChain_StateTree(t *testing.T) {
	err := initContext4Test(t)
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}
	defer clearSelf(t)

	chain := BlockChainImpl

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

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block0) {
		t.Fatalf("fail to add block0")
	}

	_, _ = txpool.AddTransaction(genTestTx(12345, "1", 1, 1))

	_, _ = txpool.AddTransaction(genTestTx(123456, "2", 2, 2))

	_, _ = txpool.AddTransaction(genTestTx(123457, "1", 2, 3))

	block := BlockChainImpl.CastBlock(2, common.Hex2Bytes("12"), 0, *castor, groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block) {
		t.Fatalf("fail to add block")
	}

	block2 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("22"), 0, *castor, groupid)
	if nil == block2 {
		t.Fatalf("fail to cast new block")
	}

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block2) {
		t.Fatalf("fail to add block")
	}

	fmt.Printf("state: %d\n", chain.latestBlock.StateTree)

	block3 := BlockChainImpl.CastBlock(4, common.Hex2Bytes("12"), 0, *castor, groupid)
	if nil == block3 {
		t.Fatalf("fail to cast new block")
	}

	if types.AddBlockSucc != BlockChainImpl.AddBlockOnChain(source, block3) {
		t.Fatalf("fail to add block")
	}

	fmt.Printf("state: %d\n", chain.getLatestBlock().StateTree)
	time.Sleep(time.Second)
}

var privateKey = "0x045c8153e5a849eef465244c0f6f40a43feaaa6855495b62a400cc78f9a6d61c76c09c3aaef393aa54bd2adc5633426e9645dfc36723a75af485c5f5c9f2c94658562fcdfb24e943cf257e25b9575216c6647c4e75e264507d2d57b3c8bc00b361"

func genTestTx(price uint64, target string, nonce uint64, value uint64) *types.Transaction {
	targetbyte := common.BytesToAddress(genHash(target))
	raw := &types.RawTransaction{
		GasPrice: types.NewBigInt(price),
		GasLimit: types.NewBigInt(5000),
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    types.NewBigInt(value),
	}

	sk := common.HexToSecKey(privateKey)
	source := sk.GetPubKey().GetAddress()
	raw.Source = &source
	tx := types.NewTransaction(raw, raw.GenHash())
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()
	return tx
}

func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return common.Sha256(bytes3)
}

func initBalance() {
	blocks := GenCorrectBlocks()
	stateDB, _ := BlockChainImpl.LatestAccountDB()
	stateDB1 := stateDB.(*account.AccountDB)
	//stateDB1, _ := account.NewAccountDB(common.Hash{}, BlockChainImpl.stateCache)
	stateDB1.AddBalance(common.StringToAddress("zvc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"), new(big.Int).SetUint64(100000000000000000))
	exc := &executePostState{state: stateDB1}
	root := stateDB1.IntermediateRoot(true)
	blocks[0].Header.StateTree = common.BytesToHash(root.Bytes())
	_, _ = BlockChainImpl.commitBlock(blocks[0], exc)
}

func clearAllFolder() {
	fmt.Println("---clearAllFolder---")
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
		if d.IsDir() && (strings.HasPrefix(d.Name(), "d_") || (strings.HasPrefix(d.Name(), "Test")) ||
			strings.HasPrefix(d.Name(), "database")) {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Println("error while removing /s", d.Name())
			}
		}
	}

}

func clearSelf(t *testing.T) {
	if BlockChainImpl != nil {
		BlockChainImpl.Close()
		BlockChainImpl = nil
	}

	dir, err := ioutil.ReadDir(".")
	if err != nil {
		return
	}
	for _, d := range dir {
		if d.IsDir() && d.Name() == t.Name() {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Println("error while removing /s", d.Name())
			}
		}
	}
}

func clearTicker() {
	if TxSyncer != nil && TxSyncer.ticker != nil {
		TxSyncer.ticker.RemoveRoutine(txNotifyRoutine)
		TxSyncer.ticker.RemoveRoutine(txReqRoutine)
	}
}

func initContext4Test(t *testing.T) error {
	common.InitConf("../tas_config_all.ini")
	common.GlobalConf.SetString(configSec, "db_blocks", t.Name())
	common.GlobalConf.SetInt(configSec, "db_node_cache", 0)
	network.Logger = log.P2PLogger
	err := middleware.InitMiddleware()
	if err != nil {
		return err
	}
	BlockChainImpl = nil

	err = InitCore(NewConsensusHelper4Test(groupsig.ID{}), getAccount())
	clearTicker()
	sp := newStateProcessor(BlockChainImpl)
	BlockChainImpl.stateProc = sp
	GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})
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
		Address:  secKey.GetPubKey().GetAddress().AddrPrefixString(),
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

func (helper *ConsensusHelperImpl4Test) GroupSkipCountsBetween(preBH *types.BlockHeader, h uint64) map[common.Hash]uint16 {
	return nil
}

func (helper *ConsensusHelperImpl4Test) GetBlockMinElapse(height uint64) int32 {
	return 1
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
	return height - uint64(math.Ceil(float64(bh.Elapsed)/float64(model.Param.MaxGroupCastTime*1e3)))
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
	return uint64(0)
}
func (g *GroupHeader4Test) DismissHeight() uint64 {
	return common.MaxUint64
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

func newBlockChainByDB(db string) (*FullBlockChain, error) {
	chain := &FullBlockChain{
		config: &BlockChainConfig{
			dbfile:      db,
			block:       "bh",
			blockHeight: "hi",
			state:       "st",
			reward:      "nu",
			tx:          "tx",
			receipt:     "rc",
		},
		latestBlock:      nil,
		init:             true,
		isAdjusting:      false,
		ticker:           ticker.NewGlobalTicker("chain"),
		futureRawBlocks:  common.MustNewLRUCache(100),
		verifiedBlocks:   common.MustNewLRUCache(10),
		topRawBlocks:     common.MustNewLRUCache(20),
		newBlockMessages: common.MustNewLRUCache(100),
	}

	options := &opt.Options{
		OpenFilesCacheCapacity:        100,
		BlockCacheCapacity:            16 * opt.MiB,
		WriteBuffer:                   16 * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           4 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
		CompactionTotalSize:           16 * opt.MiB,
		BlockSize:                     64 * opt.KiB,
		ReadOnly:                      true,
	}

	ds, err := tasdb.NewDataSource(chain.config.dbfile, options)
	if err != nil {
		Logger.Errorf("new datasource error:%v", err)
		return nil, err
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.txDb, err = ds.NewPrefixDatabase(chain.config.tx)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.stateDb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}

	chain.stateCache = account.NewDatabase(chain.stateDb)

	latestBH := chain.loadCurrentBlock()
	chain.latestBlock = latestBH
	return chain, nil
}

func TestStatProposalRate(t *testing.T) {
	params.InitChainConfig(1)
	common.InitConf("test1.ini")
	chain, err := newBlockChainByDB("d_b")
	if err != nil {
		t.Log(err)
		return
	}
	mm := &MinerManager{}

	statFunc := func(stat map[string]int, b uint64) {
		block := chain.QueryBlockHeaderByHeight(b)
		if block == nil {
			return
		}
		p := common.ToHex(block.Castor)
		if v, ok := stat[p]; ok {
			stat[p] = v + 1
		} else {
			stat[p] = 1
		}
	}

	beforeZIP := make(map[string]int)
	for b := uint64(1); b < params.GetChainConfig().ZIP001; b++ {
		statFunc(beforeZIP, b)
	}
	afterZIP := make(map[string]int)
	for b := params.GetChainConfig().ZIP001; b <= chain.Height(); b++ {
		statFunc(afterZIP, b)
	}
	db, _ := chain.AccountDBAt(chain.Height())
	stakeMap := make(map[string]uint64)
	for p, _ := range beforeZIP {
		m := mm.getMiner(db, common.BytesToAddress(common.FromHex(p)), types.MinerTypeProposal)
		if m == nil {
			panic(p)
		}
		stakeMap[p] = m.Stake
	}
	for p, _ := range afterZIP {
		m := mm.getMiner(db, common.BytesToAddress(common.FromHex(p)), types.MinerTypeProposal)

		stakeMap[p] = m.Stake
	}
	for p, v := range beforeZIP {
		t.Log(stakeMap[p]/common.ZVC, v, float64(v)/float64(stakeMap[p]/common.ZVC))
	}
	t.Log("====================================")
	for p, v := range afterZIP {
		t.Log(stakeMap[p]/common.ZVC, v, float64(v)/float64(stakeMap[p]/common.ZVC))
	}
}

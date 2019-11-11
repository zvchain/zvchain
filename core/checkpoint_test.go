//   Copyright (C) 2019 ZVChain
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

package core

import (
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/trie"
	"math/big"
	"math/rand"
	"os"
	"testing"
)

type groupHeader4CPTest struct {
	seed common.Hash
	wh   uint64
	dh   uint64
}

func newGroupHeader4CPTest(wh, dh uint64) *groupHeader4CPTest {
	seed := common.EmptyHash
	if dh < common.MaxUint64 {
		seed = common.BytesToHash(common.Int64ToByte(rand.Int63()))
	}
	return &groupHeader4CPTest{
		seed: seed,
		wh:   wh,
		dh:   dh,
	}
}

func (gh *groupHeader4CPTest) Seed() common.Hash {
	return gh.seed
}

func (gh *groupHeader4CPTest) WorkHeight() uint64 {
	return gh.wh
}

func (gh *groupHeader4CPTest) DismissHeight() uint64 {
	return gh.dh
}

func (gh *groupHeader4CPTest) PublicKey() []byte {
	panic("implement me")
}

func (gh *groupHeader4CPTest) Threshold() uint32 {
	panic("implement me")
}

func (gh *groupHeader4CPTest) GroupHeight() uint64 {
	panic("implement me")
}

type group4CPTest struct {
	h *groupHeader4CPTest
}

func (g *group4CPTest) Header() types.GroupHeaderI {
	return g.h
}

func (g *group4CPTest) Members() []types.MemberI {
	return nil
}

func newGroup4CPTest(wh, dh uint64) *group4CPTest {
	return &group4CPTest{h: newGroupHeader4CPTest(wh, dh)}
}

type groupReader4CPTest struct {
	groups map[uint64][]types.GroupI
}

func (gr *groupReader4CPTest) GetActivatedGroupsAt(h uint64) []types.GroupI {
	return gr.groups[types.EpochAt(h).Start()]
}

func initGroupReader4CPTest(epNum int) activatedGroupReader {
	ep := types.EpochAt(0)
	cnt := 0
	gr := &groupReader4CPTest{
		groups: make(map[uint64][]types.GroupI),
	}
	for cnt < epNum {
		n := rand.Int31n(30) + 5
		if ep.Start() < types.EpochLength {
			n = 1
		}
		gs := make([]types.GroupI, n)
		gs[0] = newGroup4CPTest(ep.Start(), common.MaxUint64)
		for i := 1; i < len(gs); i++ {
			gs[i] = newGroup4CPTest(ep.Start(), ep.End())
		}
		gr.groups[ep.Start()] = gs
		cnt++
		Logger.Debugf("active groups at %v-%v size %v", ep.Start(), ep.End(), len(gs))
		ep = ep.Next()
	}
	return gr
}

type accountDB4CPTest struct {
	datas map[string][]byte
}

func (db *accountDB4CPTest) Database() account.AccountDatabase {
	panic("implement me")
}

func (db *accountDB4CPTest) CreateAccount(common.Address) {
	panic("implement me")
}

func (db *accountDB4CPTest) GetStateObject(common.Address) account.AccAccesser {
	panic("implement me")
}

func (db *accountDB4CPTest) SubBalance(common.Address, *big.Int) {
	panic("implement me")
}

func (db *accountDB4CPTest) AddBalance(common.Address, *big.Int) {
	panic("implement me")
}

func (db *accountDB4CPTest) GetBalance(common.Address) *big.Int {
	panic("implement me")
}

func (db *accountDB4CPTest) GetNonce(common.Address) uint64 {
	panic("implement me")
}

func (db *accountDB4CPTest) SetNonce(common.Address, uint64) {
	panic("implement me")
}

func (db *accountDB4CPTest) GetCodeHash(common.Address) common.Hash {
	panic("implement me")
}

func (db *accountDB4CPTest) GetCode(common.Address) []byte {
	panic("implement me")
}

func (db *accountDB4CPTest) SetCode(common.Address, []byte) {
	panic("implement me")
}

func (db *accountDB4CPTest) GetCodeSize(common.Address) int {
	panic("implement me")
}

func (db *accountDB4CPTest) AddRefund(uint64) {
	panic("implement me")
}

func (db *accountDB4CPTest) GetRefund() uint64 {
	panic("implement me")
}

func (db *accountDB4CPTest) GetData(add common.Address, bs []byte) []byte {
	return db.datas[string(bs)]
}

func (db *accountDB4CPTest) SetData(a common.Address, k []byte, v []byte) {
	db.datas[string(k)] = v
}

func (db *accountDB4CPTest) RemoveData(common.Address, []byte) {
	panic("implement me")
}

func (db *accountDB4CPTest) DataIterator(common.Address, []byte) *trie.Iterator {
	panic("implement me")
}

func (db *accountDB4CPTest) Suicide(common.Address) bool {
	panic("implement me")
}

func (db *accountDB4CPTest) HasSuicided(common.Address) bool {
	panic("implement me")
}

func (db *accountDB4CPTest) Exist(common.Address) bool {
	panic("implement me")
}

func (db *accountDB4CPTest) Empty(common.Address) bool {
	panic("implement me")
}

func (db *accountDB4CPTest) RevertToSnapshot(int) {
	panic("implement me")
}

func (db *accountDB4CPTest) Snapshot() int {
	panic("implement me")
}

func (db *accountDB4CPTest) Transfer(common.Address, common.Address, *big.Int) {
	panic("implement me")
}

func (db *accountDB4CPTest) CanTransfer(common.Address, *big.Int) bool {
	panic("implement me")
}

func newAccountDB4Test() *accountDB4CPTest {
	return &accountDB4CPTest{
		datas: make(map[string][]byte),
	}
}

type blockReader4CPTest struct {
	blocks     []*types.BlockHeader
	blockIndex map[common.Hash]*types.BlockHeader
	dbs        map[uint64]*accountDB4CPTest
}

func (br *blockReader4CPTest) Height() uint64 {
	top := br.QueryTopBlock()
	if top == nil {
		return 0
	}
	return top.Height
}
func (br *blockReader4CPTest) QueryTopBlock() *types.BlockHeader {
	if len(br.blocks) == 0 {
		return nil
	}
	return br.blocks[len(br.blocks)-1]
}

func (br *blockReader4CPTest) AccountDBAt(height uint64) (types.AccountDB, error) {
	if height >= br.QueryTopBlock().Height {
		height = br.QueryTopBlock().Height
	}
	var db *accountDB4CPTest
	for db == nil {
		db = br.dbs[height]
		if height == 0 {
			break
		}
		height--
	}

	newDB := newAccountDB4Test()
	for key, value := range db.datas {
		newDB.datas[key] = value
	}
	return newDB, nil
}

func (br *blockReader4CPTest) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	return br.blockIndex[hash]
}

func (br *blockReader4CPTest) addBlock(bh *types.BlockHeader, db types.AccountDB) {
	br.blocks = append(br.blocks, bh)
	br.blockIndex[bh.Hash] = bh
	br.dbs[bh.Height] = db.(*accountDB4CPTest)
}

func (br *blockReader4CPTest) getBlocksAfter(h uint64) []*types.Block {
	bs := make([]*types.Block, 0)
	for _, bh := range br.blocks {
		if bh.Height >= h {
			bs = append(bs, &types.Block{Header: bh})
		}
	}
	return bs
}

func initBlockReader4CPTest() *blockReader4CPTest {
	bq := &blockReader4CPTest{
		blocks:     make([]*types.BlockHeader, 0),
		blockIndex: make(map[common.Hash]*types.BlockHeader),
		dbs:        make(map[uint64]*accountDB4CPTest),
	}
	bh := &types.BlockHeader{
		Height: 0,
		Group:  common.Hash{},
	}
	bh.Hash = bh.GenHash()
	db := newAccountDB4Test()
	cp := &cpChecker{}
	cp.setGroupVotes(db, []uint16{1})
	bq.addBlock(bh, db)
	return bq
}

func TestActiveGroupReader(t *testing.T) {
	gr := initGroupReader4CPTest(10)
	for i := 0; i < 10; i++ {
		gs := gr.GetActivatedGroupsAt(uint64(i * types.EpochLength))
		for _, g := range gs {
			t.Log(i, g.Header().Seed(), g.Header().WorkHeight())
		}
	}
}

func init() {
	Logger = logrus.StandardLogger()
}

func TestCheckpoint_init(t *testing.T) {
	gr := initGroupReader4CPTest(5)
	br := initChainReader4CPTest(gr)
	for h := uint64(1); h < 1000; h++ {
		addRandomBlock(br, h)
	}

	cp := newCpChecker(gr, br)
	cp.init()
}

func initChainReader4CPTest(gr activatedGroupReader) *FullBlockChain {
	common.InitConf("test1.ini")
	common.GlobalConf.SetString(configSec, "db_blocks", "d_b")
	err := initBlockChain(NewConsensusHelper4Test(groupsig.ID{}), nil)
	clearTicker()
	Logger = logrus.StandardLogger()
	if err != nil {
		//Logger.Panicf("init chain error:%v", err)
		return nil
	}
	chain := BlockChainImpl

	Logger = logrus.StandardLogger()
	// mock the tvm stateProc
	tvm := newStateProcessor(chain)
	// mock the cp checker
	chain.cpChecker = newCpChecker(gr, chain)

	tvm.addPostProcessor(chain.cpChecker.updateVotes)
	chain.stateProc = tvm

	chain.cpChecker.init()
	return chain
}

func TestCheckpoint_checkAndUpdate(t *testing.T) {
	os.RemoveAll("d_b")
	defer func() {
		os.RemoveAll("d_b")
	}()
	epochNum := 20
	gr := initGroupReader4CPTest(epochNum)
	br := initChainReader4CPTest(gr)
	if br == nil {
		return
	}
	Logger = logrus.StandardLogger()
	top := br.Height()
	for h := uint64(1); h < uint64(epochNum*types.EpochLength); h += uint64(rand.Int31n(2)) + 1 {
		groups := gr.GetActivatedGroupsAt(h)
		Logger.Debugf("groupsize %v at %v", len(groups), h)

		if h > types.EpochLength+100 {
			t.Log("")
		}
		if h > top {
			addRandomBlock(br, h)
		}

		cpBlock := br.CheckPointAt(h)
		Logger.Debugf("cp at %v is %v %v", h, cpBlock.Height, cpBlock.Hash)

	}

}

func TestCheckpoint_CheckPointOf(t *testing.T) {
	defer func() {
		os.RemoveAll("d_b")
	}()
	epochNum := 20
	gr := initGroupReader4CPTest(epochNum)
	br := initChainReader4CPTest(gr)
	Logger = logrus.StandardLogger()
	if br == nil {
		return
	}
	top := br.Height()
	for h := uint64(1); h < uint64(epochNum*types.EpochLength); h += uint64(rand.Int31n(2)) + 1 {
		if h > top {
			addRandomBlock(br, h)
		}
	}

	testNum := 2000
	for i := 0; i < testNum; i++ {
		h := uint64(rand.Int63n(int64(br.Height())))
		if !br.HasHeight(h) {
			continue
		}
		cp := br.CheckPointAt(h)
		if cp == nil || cp.Height < 2 {
			continue
		}
		t.Log(i, h, cp.Height)
		blocks := br.BatchGetBlocksBetween(cp.Height-2, h+1)
		cp2 := br.cpChecker.checkPointOf(blocks)
		if cp2 == nil {
			t.Errorf("checkpoint error at %v, cp1 %v", h, cp.Height)
			br.CheckPointAt(h)
			blocks := br.BatchGetBlocksBetween(cp.Height-2, h+1)
			br.cpChecker.checkPointOf(blocks)
			continue
		}
		if cp2.Hash != cp.Hash {
			t.Errorf("checkpoint error at %v, cp1 %v, cp2 %v", h, cp.Height, cp2.Height)
			br.cpChecker.checkPointOf(blocks)
		}
	}
}

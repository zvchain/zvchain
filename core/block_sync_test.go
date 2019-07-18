package core

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	tas_middleware_test "github.com/zvchain/zvchain/core/test"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/taslog"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
)

var blockSyncForTest *blockSyncer

func init(){
	resetDb()
	initContext4Test()
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	blockSyncForTest = newBlockSyncer(BlockChainImpl.(*FullBlockChain))
	blockSyncForTest.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, "1")
	initPeerManager()
	types.DefaultPVFunc = PvFuncTest
	insertBlocks()
}
func TestGetBestCandidate(t *testing.T) {
	for i := 0; i < 100; i++ {
		blockSync.addCandidatePool(strconv.Itoa(i), &types.BlockHeader{Hash: common.BigToAddress(big.NewInt(int64(i))).Hash(), TotalQN: uint64(i), ProveValue: genHash(strconv.Itoa(i))})
		peerManagerImpl.getOrAddPeer(strconv.Itoa(i))
	}
	for i := 0; i < 50; i++ {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	blockSync.getCandidateById("")

	if len(blockSync.candidatePool) != 50 {
		t.Fatalf("expect len is %d,but got %d", 50, peerManagerImpl.peerMeters.Len())
	}

	sc, _ := blockSync.getCandidateById("51")
	if sc == "" {
		t.Fatalf("expect get source,but got '' ")
	}
	sc, _ = blockSync.getCandidateById("49")
	if sc != "" {
		t.Fatalf("expect get '',but got %s ", sc)
	}

	for i := 50; i < 100; i++ {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	sc, _ = blockSync.getCandidateById("")
	if sc != "" {
		t.Fatalf("expect get '',but got %s ", sc)
	}

}

func PvFuncTest(pvBytes []byte) *big.Int {
	return new(big.Int)
}


func TestTopBlockInfoNotifyHandler(t *testing.T){
	//add a nil blockheader
	source := "0x111"
	blockSyncForTest.topBlockInfoNotifyHandler(tas_middleware_test.NewNilHeaderMessage(source))
	isPeerExists := peerManagerImpl.isPeerExists(source)
	if isPeerExists{
		t.Fatalf("expect nil,but got data")
	}
	candieData := blockSyncForTest.getPeerTopBlock (source)
	if candieData != nil{
		t.Fatalf("expect nil,but got data")
	}

	//add
	for i:=0;i<100;i++{
		msg := tas_middleware_test.GenErrorDefaultMessage(i)
		blockSyncForTest.topBlockInfoNotifyHandler(msg)
	}

	//check
	for i:=0;i<100;i++{
		isPeerExists := peerManagerImpl.isPeerExists(strconv.Itoa(i))
		if !isPeerExists{
			t.Fatalf("expect got data,but got nil")
		}
	}
}

func TestBlockReqHandler(t *testing.T){
	var ReqHeight uint64 = 10
	ReqSize := 10
	blocks := BlockChainImpl.(*FullBlockChain).BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) !=10{
		t.Fatalf("expect 10,bug got %d",len(blocks))
	}

	ReqHeight = 0
	ReqSize = 16
	blocks = BlockChainImpl.(*FullBlockChain).BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) !=16{
		t.Fatalf("expect 16,bug got %d",len(blocks))
	}

	ReqHeight = 83
	ReqSize = 16
	blocks = BlockChainImpl.(*FullBlockChain).BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) !=16{
		t.Fatalf("expect 16,bug got %d",len(blocks))
	}

	ReqHeight = 99
	ReqSize = 16
	blocks = BlockChainImpl.(*FullBlockChain).BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) !=16{
		t.Fatalf("expect 16,bug got %d",len(blocks))
	}
}


func resetDb() error {
	dir, err := ioutil.ReadDir(".")
	if err != nil {
		return err
	}
	for _, d := range dir {
		if d.IsDir() && strings.HasPrefix(d.Name(), "d_") {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func insertBlocks(){
	blocks := GenBlocks()
	stateDB,_ := account.NewAccountDB(common.Hash{}, BlockChainImpl.(*FullBlockChain).stateCache)
	exc := &executePostState{state: stateDB}
	for i:=0;i<len(blocks);i++{
		BlockChainImpl.(*FullBlockChain).commitBlock(blocks[i],exc)
	}
}

func GenBlocks()[]*types.Block{
	blocks := []*types.Block{}
	for i:= 0;i<100;i++{
		bh := tas_middleware_test.NewRandomFullBlockHeader(uint64(i))
		blocks = append(blocks,&types.Block{Header:bh})
	}
	return blocks
}

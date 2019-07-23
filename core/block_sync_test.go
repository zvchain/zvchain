package core

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	tas_middleware_test "github.com/zvchain/zvchain/core/test"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
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
	types.InitMiddleware()
	types.DefaultPVFunc = PvFuncTest
}
func TestGetBestCandidate(t *testing.T) {
	for i := 0; i < 100; i++ {
		blockSyncForTest.addCandidatePool(strconv.Itoa(i), &types.BlockHeader{Hash: common.BigToAddress(big.NewInt(int64(i))).Hash(), TotalQN: uint64(i), ProveValue: genHash(strconv.Itoa(i))})
		peerManagerImpl.getOrAddPeer(strconv.Itoa(i))
	}
	for i := 0; i < 50; i++ {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	blockSyncForTest.getCandidateById("")

	if len(blockSyncForTest.candidatePool) != 50 {
		t.Fatalf("expect len is %d,but got %d", 50, peerManagerImpl.peerMeters.Len())
	}

	sc, _ := blockSyncForTest.getCandidateById("51")
	if sc == "" {
		t.Fatalf("expect get source,but got '' ")
	}
	sc, _ = blockSyncForTest.getCandidateById("49")
	if sc != "" {
		t.Fatalf("expect get '',but got %s ", sc)
	}

	for i := 50; i < 100; i++ {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	sc, _ = blockSyncForTest.getCandidateById("")
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
	insertBlocks()
	bts,_ := tas_middleware_test.MarshalNilSyncRequest()
	msg := tas_middleware_test.GenDefaultMessageWithBytes(111,bts)

	err := blockSyncForTest.blockReqHandler(msg)
	if err == nil{
		t.Fatalf("except got error,but got nil")
	}

	bts,_ = tas_middleware_test.MarshalErorSyncRequest()
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111,bts)

	err = blockSyncForTest.blockReqHandler(msg)
	if err == nil{
		t.Fatalf("except got error,but got nil")
	}

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

	//max height is 99
	ReqHeight = 99
	ReqSize = 16
	blocks = BlockChainImpl.(*FullBlockChain).BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) !=1{
		t.Fatalf("expect 1,bug got %d",len(blocks))
	}
}

func TestBlockResponseMsgHandler(t *testing.T){
	blocks := tas_middleware_test.GenBlocks()
	pbblocks := blocksToPb(blocks)
	message := tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts,_:=proto.Marshal(&message)
	msg := tas_middleware_test.GenDefaultMessageWithBytes(111,bts)
	err := blockSyncForTest.blockResponseMsgHandler(msg)
	if  err != nil{
		t.Fatalf("except err nil,but got error")
	}


	//error protobuf format
	errorPbblocks := blocksToErrorPb(blocks)
	errorMsg := tas_middleware_test.BlockResponseMsg{Blocks: errorPbblocks}
	bts,_ = proto.Marshal(&errorMsg)
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111,bts)
	err = blockSyncForTest.blockResponseMsgHandler(msg)
	if  err != nil{
		t.Fatalf("except err nil,but got error")
	}
}


func TestNewBlockHandler(t *testing.T){
	blocks := tas_middleware_test.GenBlocks()
	pbblocks := blocksToPb(blocks)
	message := tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts,_:=proto.Marshal(&message)
	msg := tas_middleware_test.GenDefaultMessageWithBytes(111,bts)


	err := BlockChainImpl.(*FullBlockChain).newBlockHandler(msg)
	if err != nil{
		t.Fatalf("except got no error,but got error")
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


func blocksToPb(blocks[]*types.Block)[]*tas_middleware_pb.Block{
	pbblocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range blocks {
		pb := types.BlockToPb(b)
		pbblocks = append(pbblocks, pb)
	}
	return pbblocks
}

func blocksToErrorPb(blocks[]*types.Block)[]*tas_middleware_test.Block{
	pbblocks := make([]*tas_middleware_test.Block, 0)
	for _, b := range blocks {
		pb := BlockToErrorPb(b)
		pbblocks = append(pbblocks, pb)
	}
	return pbblocks
}


func BlockToErrorPb(b *types.Block) *tas_middleware_test.Block {
	if b == nil {
		return nil
	}
	header := BlockHeaderToErrorPb(b.Header)
	block := tas_middleware_test.Block{Header: header}
	return &block
}



func BlockHeaderToErrorPb(h *types.BlockHeader) *tas_middleware_test.BlockHeader {
	ts := h.CurTime.Unix()
	str := "............."
	header := tas_middleware_test.BlockHeader{
		Hash:        h.Hash.Bytes(),
		Height:      &str,
		PreHash:     h.PreHash.Bytes(),
		Elapsed:     &h.Elapsed,
		ProveValue:  h.ProveValue,
		CurTime:     &ts,
		Castor:      h.Castor,
		GroupId:     h.Group.Bytes(),
		Signature:   h.Signature,
		Nonce:       &h.Nonce,
		TxTree:      h.TxTree.Bytes(),
		ReceiptTree: h.ReceiptTree.Bytes(),
		StateTree:   h.StateTree.Bytes(),
		ExtraData:   h.ExtraData,
		TotalQN:     &h.TotalQN,
		Random:      h.Random,
		GasFee:      &h.GasFee,
	}
	return &header
}
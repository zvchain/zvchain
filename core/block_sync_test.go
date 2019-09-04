package core

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	tas_middleware_test "github.com/zvchain/zvchain/core/test"
	"github.com/zvchain/zvchain/log"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

var blockSyncForTest *blockSyncer
var lastBlockHash common.Hash
var middleBlock *types.Block
var middleBlockHash common.Hash

func init() {
	clearAllFolder()
	log.Init()
}
func initContext(t *testing.T) {
	initContext4Test(t)
	blockSyncForTest = newBlockSyncer(BlockChainImpl)
	blockSyncForTest.logger = log.BlockSyncLogger

	initPeerManager()
	types.DefaultPVFunc = PvFuncTest
}
func TestGetBestCandidate(t *testing.T) {
	initContext(t)
	defer clearSelf(t)
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
	if len(pvBytes) != 81 {
		return big.NewInt(0)
	}
	return base.VRFProof2hash(base.VRFProve(pvBytes)).Big()
}

func TestTopBlockInfoNotifyHandler(t *testing.T) {
	initContext(t)
	defer clearSelf(t)

	//add a nil blockheader
	source := "0x111"
	blockSyncForTest.topBlockInfoNotifyHandler(tas_middleware_test.NewNilHeaderMessage(source))
	isPeerExists := peerManagerImpl.isPeerExists(source)
	if isPeerExists {
		t.Fatalf("expect nil,but got data")
	}
	candieData := blockSyncForTest.getPeerTopBlock(source)
	if candieData != nil {
		t.Fatalf("expect nil,but got data")
	}

	//add
	for i := 0; i < 100; i++ {
		msg := tas_middleware_test.GenErrorDefaultMessage(i)
		blockSyncForTest.topBlockInfoNotifyHandler(msg)
	}

	//check
	for i := 0; i < 100; i++ {
		isPeerExists := peerManagerImpl.isPeerExists(strconv.Itoa(i))
		if !isPeerExists {
			t.Fatalf("expect got data,but got nil")
		}
	}
}

func TestBlockReqHandler(t *testing.T) {
	initContext(t)
	defer clearSelf(t)
	insertBlocks()
	bts, _ := tas_middleware_test.MarshalNilSyncRequest()
	msg := tas_middleware_test.GenDefaultMessageWithBytes(111, bts)

	err := blockSyncForTest.blockReqHandler(msg)
	if err == nil {
		t.Fatalf("expect got error,but got nil")
	}

	bts, _ = tas_middleware_test.MarshalErorSyncRequest()
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111, bts)

	err = blockSyncForTest.blockReqHandler(msg)
	if err == nil {
		t.Fatalf("expect got error,but got nil")
	}

	var ReqHeight uint64 = 10
	ReqSize := 10
	blocks := BlockChainImpl.BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) != 10 {
		t.Fatalf("expect 10,bug got %d", len(blocks))
	}

	ReqHeight = 0
	ReqSize = 16
	blocks = BlockChainImpl.BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) != 16 {
		t.Fatalf("expect 16,bug got %d", len(blocks))
	}

	ReqHeight = 83
	ReqSize = 16
	blocks = BlockChainImpl.BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) != 16 {
		t.Fatalf("expect 16,bug got %d", len(blocks))
	}

	//max height is 99
	ReqHeight = 99
	ReqSize = 16
	blocks = BlockChainImpl.BatchGetBlocksAfterHeight(ReqHeight, ReqSize)
	if len(blocks) != 1 {
		t.Fatalf("expect 1,bug got %d", len(blocks))
	}
}

func TestBlockResponseMsgHandler_bug(t *testing.T) {
	//error blocks
	clearSelf(t)
	initContext(t)
	defer clearSelf(t)
	insertCorrectHashBlocks()
	blocks := tas_middleware_test.GenBlocks()
	pbblocks := blocksToPb(blocks)
	message := tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts, _ := proto.Marshal(&message)
	msg := tas_middleware_test.GenDefaultMessageWithBytes(111, bts)
	err := blockSyncForTest.blockResponseMsgHandler(msg)
	if err == nil {
		t.Fatalf("expect got err,but got nil")
	}

	//error protobuf format
	errorPbblocks := blocksToErrorPb(blocks)
	errorMsg := tas_middleware_test.BlockResponseMsg{Blocks: errorPbblocks}
	bts, _ = proto.Marshal(&errorMsg)
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111, bts)
	err = blockSyncForTest.blockResponseMsgHandler(msg)
	if err == nil {
		t.Fatalf("expect err nil,but got error")
	}

	//only txs
	blocks = tas_middleware_test.GenOnlyHasTransBlocks()
	pbblocks = BlockToPbOnlyTxs(blocks)
	message = tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts, _ = proto.Marshal(&message)
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111, bts)
	err = blockSyncForTest.blockResponseMsgHandler(msg)
	if err == nil {
		t.Fatalf("expect got error,but got nil")
	}

	//tx sign error
	blocks = tas_middleware_test.GenHashCorrectBlocks()
	pbblocks = blocksToPb(blocks)
	message = tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts, _ = proto.Marshal(&message)
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111, bts)
	err = blockSyncForTest.blockResponseMsgHandler(msg)
	if err == nil {
		t.Fatalf("expect got error,but got nil")
	}

	//correct block hash
	blocks = tas_middleware_test.GenHashCorrectBlocksByFirstHash(lastBlockHash, 200)
	pbblocks = blocksToPb(blocks)
	message = tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts, _ = proto.Marshal(&message)
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111, bts)
	err = blockSyncForTest.blockResponseMsgHandler(msg)
	if err == nil {
		t.Fatalf("expect got error,but got nil")
	}

	//rock back attack
	blocks = tas_middleware_test.GenBlocksByBlock(middleBlock)
	pbblocks = blocksToPb(blocks)
	message = tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts, _ = proto.Marshal(&message)
	msg = tas_middleware_test.GenDefaultMessageWithBytes(111, bts)
	err = blockSyncForTest.blockResponseMsgHandler(msg)

	//this is roll back error!
	if BlockChainImpl.latestBlock.Hash == middleBlockHash {
		t.Fatalf("hash error")
	}
	if err == nil {
		t.Fatalf("expect got error,but got nil")
	}
}

func TestNewBlockHandler(t *testing.T) {
	initContext(t)
	defer clearSelf(t)
	blocks := tas_middleware_test.GenBlocks()
	pbblocks := blocksToPb(blocks)
	message := tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	bts, _ := proto.Marshal(&message)
	msg := tas_middleware_test.GenDefaultMessageWithBytes(111, bts)

	err := BlockChainImpl.newBlockHandler(msg)
	if err != nil {
		t.Fatalf("expect got no error,but got error")
	}
}

func clearDB() {
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
		if d.IsDir() && (strings.HasPrefix(d.Name(), "d_") || strings.HasPrefix(d.Name(), "test_db") || strings.HasPrefix(d.Name(), "groupstore") ||
			strings.HasPrefix(d.Name(), "database")) || (strings.HasSuffix(d.Name(), ".log")) {
			fmt.Printf("deleting folder: %s \n", d.Name())
			err = os.RemoveAll(d.Name())
			if err != nil {
				fmt.Printf("error while removing %s,error=%v", d.Name(), err)
			}
		}
	}

}

func insertBlocks() {
	blocks := GenBlocks()
	stateDB, _ := account.NewAccountDB(common.Hash{}, BlockChainImpl.stateCache)
	exc := &executePostState{state: stateDB}
	for i := 0; i < len(blocks); i++ {
		BlockChainImpl.commitBlock(blocks[i], exc)
		lastBlockHash = blocks[i].Header.Hash
	}
}

func insertCorrectHashBlocks() {
	blocks := GenCorrectBlocks()
	stateDB, _ := account.NewAccountDB(common.Hash{}, BlockChainImpl.stateCache)
	exc := &executePostState{state: stateDB}
	for i := 0; i < len(blocks); i++ {
		root := stateDB.IntermediateRoot(true)
		blocks[i].Header.StateTree = common.BytesToHash(root.Bytes())
		BlockChainImpl.commitBlock(blocks[i], exc)
		lastBlockHash = blocks[i].Header.Hash
		if i == 4 {
			middleBlockHash = blocks[i].Header.Hash
		}
		if i == 5 {
			middleBlock = blocks[i]
		}
	}
}

func GenCorrectBlocks() []*types.Block {
	blocks := []*types.Block{}
	var hash = common.Hash{}
	for i := 0; i < 100; i++ {
		bh := tas_middleware_test.NewRandomFullBlockHeader(uint64(i))
		bh.Hash = bh.GenHash()
		if i == 0 {
			bh.PreHash = bh.Hash
		} else {
			bh.PreHash = hash
		}
		hash = bh.Hash
		blocks = append(blocks, &types.Block{Header: bh})
	}
	return blocks
}

func GenBlocks() []*types.Block {
	blocks := []*types.Block{}
	for i := 0; i < 100; i++ {
		bh := tas_middleware_test.NewRandomFullBlockHeader(uint64(i))
		blocks = append(blocks, &types.Block{Header: bh})
	}
	return blocks
}

func blocksToPb(blocks []*types.Block) []*tas_middleware_pb.Block {
	pbblocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range blocks {
		pb := types.BlockToPb(b)
		pbblocks = append(pbblocks, pb)
	}
	return pbblocks
}

func BlockToPbOnlyTxs(blocks []*types.Block) []*tas_middleware_pb.Block {
	pbblocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range blocks {
		transactions := TransactionsToPb(b.Transactions)
		block := tas_middleware_pb.Block{Transactions: transactions}
		pbblocks = append(pbblocks, &block)
	}
	return pbblocks
}

func TransactionsToPb(txs []*types.RawTransaction) []*tas_middleware_pb.RawTransaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*tas_middleware_pb.RawTransaction, 0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func transactionToPb(t *types.RawTransaction) *tas_middleware_pb.RawTransaction {
	if t == nil {
		return nil
	}
	var (
		target []byte
	)
	if t.Target != nil {
		target = t.Target.Bytes()
	}
	tp := int32(t.Type)
	transaction := tas_middleware_pb.RawTransaction{
		Data:      t.Data,
		Value:     t.Value.GetBytesWithSign(),
		Nonce:     &t.Nonce,
		Target:    target,
		GasLimit:  t.GasLimit.GetBytesWithSign(),
		GasPrice:  t.GasPrice.GetBytesWithSign(),
		ExtraData: t.ExtraData,
		Type:      &tp,
		Sign:      t.Sign,
	}
	return &transaction
}

func blocksToErrorPb(blocks []*types.Block) []*tas_middleware_test.Block {
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
	ts := h.CurTime.UnixMilli()
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

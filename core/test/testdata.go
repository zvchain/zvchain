package tas_middleware_test

import (
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"math"
	"math/big"
	"math/rand"
	"strconv"
)

var (
	errorHashs = []common.Hash{
		common.Hash{},
		common.BytesToHash([]byte("i am evil.")),
		common.BigToHash(big.NewInt(11111)),
		common.BytesToHash([]byte("&^&^%%$%#))()SDDD")),
		common.BigToHash(big.NewInt(-11111111)),
		common.BigToHash(big.NewInt(00000000000000)),
		common.BytesToHash([]byte("0x9999991323232222222222222222222222222222222222222222222222222222222222222222222227           %%%%%%%%%%      hhhhhh 55555%%%%%%%%%%%%%%%%%%%%%%%%%%%5222")),
	}

	errorInt64 = []int64{
		math.MaxInt64,
		-11111111,
		0,
	}

	errorInt32 = []int32{
		math.MaxInt32,
		-11111111,
		0,
	}

	errorUint64 = []uint64{
		math.MaxInt64,
		math.MaxInt64 + 1,
		0,
		math.MaxUint64,
	}

	errorBytes = [][]byte{
		big.NewInt(11111).Bytes(),
		[]byte("i am evil."),
		[]byte("0x9999991323232222222222222222222222222222222222222222222222222222222222222222222227           %%%%%%%%%%      hhhhhh 55555%%%%%%%%%%%%%%%%%%%%%%%%%%%5222"),
		big.NewInt(00000000000000).Bytes(),
		big.NewInt(-11111111).Bytes(),
	}
)

func NewNilHeaderMessage(source string) *notify.DefaultMessage {
	return notify.NewDefaultMessage(nil, source, 0, 0)
}

func GenErrorDefaultMessage(source int) *notify.DefaultMessage {
	bh := NewRandomFullBlockHeader(uint64(rand.Intn(10000)))
	return GenErrorDefaultMessageWithBlockHeader(source, bh)
}

func GenDefaultMessageWithBytes(source int, data []byte) *notify.DefaultMessage {
	return notify.NewDefaultMessage(data, strconv.Itoa(source), uint16(rand.Intn(100)), uint16(rand.Intn(100)))
}

func GenErrorDefaultMessageWithBlockHeader(source int, bh *types.BlockHeader) *notify.DefaultMessage {
	blockHeader := types.BlockHeaderToPb(bh)

	blockInfo := tas_middleware_pb.TopBlockInfo{TopHeader: blockHeader}

	bt, _ := proto.Marshal(&blockInfo)

	return notify.NewDefaultMessage(bt, strconv.Itoa(source), uint16(rand.Intn(100)), uint16(rand.Intn(100)))
}

func NewRandomFullBlockHeader(height uint64) *types.BlockHeader {
	return &types.BlockHeader{
		Hash:        common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Height:      height,
		PreHash:     common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Elapsed:     int32(rand.Intn(10000)),
		ProveValue:  big.NewInt(int64(rand.Intn(10000))).Bytes(),
		TotalQN:     uint64(rand.Intn(10000)),
		CurTime:     -111111,
		Castor:      big.NewInt(int64(rand.Intn(10000))).Bytes(),
		Group:       common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Signature:   big.NewInt(int64(rand.Intn(10000))).Bytes(),
		Nonce:       int32(rand.Intn(10000)),
		TxTree:      common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		ReceiptTree: common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		StateTree:   common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		ExtraData:   big.NewInt(int64(rand.Intn(10000))).Bytes(),
		Random:      big.NewInt(int64(rand.Intn(10000))).Bytes(),
		GasFee:      uint64(rand.Intn(10000)),
	}
}

func NewBlock(bh *types.BlockHeader, txs []*types.RawTransaction) *types.Block {
	return &types.Block{
		Header:       bh,
		Transactions: txs,
	}
}

func NewBlockWithTxs(txs []*types.RawTransaction) *types.Block {
	return &types.Block{
		Transactions: txs,
	}
}

func GenBlocks() []*types.Block {
	blocks := []*types.Block{}
	bhs := GenErrorBlockHeaders()
	for _, data := range bhs {
		blocks = append(blocks, NewBlock(data, nil))
	}
	return blocks
}

func GenOnlyHasTransBlocks() []*types.Block {
	blocks := []*types.Block{}

	txs := []*types.RawTransaction{}
	tx := &types.RawTransaction{Nonce: 100}
	txs = append(txs, tx)
	for i := 0; i < 10; i++ {
		blocks = append(blocks, NewBlockWithTxs(txs))
	}
	return blocks
}

func GenHashCorrectBlocks() []*types.Block {
	blocks := []*types.Block{}
	txs := []*types.RawTransaction{}
	bhs := GeneCorrectBlockHeaders()
	nonce := 0
	for _, data := range bhs {
		target := common.StringToAddress("zv999")
		sign := []byte{}
		for i := 0; i < 20; i++ {
			sign = append(sign, 0)
		}
		raw := &types.RawTransaction{Type: 0, Value: types.NewBigInt(10), Nonce: uint64(nonce), Target: &target, GasLimit: types.NewBigInt(uint64(10000) + uint64(nonce)), GasPrice: types.NewBigInt(uint64(500) + uint64(nonce)), Sign: sign}
		nonce++
		txs = append(txs, raw)
		blocks = append(blocks, NewBlock(data, txs))
		txs = []*types.RawTransaction{}
	}
	return blocks
}

func GenBlocksByBlock(bc *types.Block) []*types.Block {
	blocks := []*types.Block{}
	bc.Header.Hash = common.BytesToHash([]byte{1, 2, 3})
	blocks = append(blocks, bc)
	return blocks
}

func GenHashCorrectBlocksByFirstHash(firstBlockHash common.Hash, len int) []*types.Block {
	blocks := []*types.Block{}
	txs := []*types.RawTransaction{}
	bhs := GeneCorrectBlockHeadersByFirstHash(firstBlockHash, len)
	nonce := 0
	for _, data := range bhs {
		target := common.StringToAddress("zv999")
		sign := []byte{}
		for i := 0; i < 20; i++ {
			sign = append(sign, 0)
		}
		tx := &types.RawTransaction{Type: 0, Value: types.NewBigInt(10), Nonce: uint64(nonce), Target: &target, GasLimit: types.NewBigInt(uint64(10000) + uint64(nonce)), GasPrice: types.NewBigInt(uint64(500) + uint64(nonce)), Sign: sign}
		nonce++
		txs = append(txs, tx)
		blocks = append(blocks, NewBlock(data, txs))
		txs = []*types.RawTransaction{}
	}
	return blocks
}

func GenErrorBlockHeaders() []*types.BlockHeader {
	bhs := []*types.BlockHeader{}
	bhs = append(bhs, NewNilBlockHeader())
	bhs = append(bhs, GeneErrorBlockHeaders()...)
	return bhs
}

func NewNilBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{}
}

func NewCorrectBlockHeader(hash common.Hash, preHash common.Hash) *types.BlockHeader {
	return &types.BlockHeader{
		Hash:        hash,
		Height:      errorUint64[rand.Intn(len(errorUint64))],
		PreHash:     preHash,
		Elapsed:     errorInt32[rand.Intn(len(errorInt32))],
		ProveValue:  errorBytes[rand.Intn(len(errorBytes))],
		TotalQN:     errorUint64[rand.Intn(len(errorUint64))],
		Castor:      errorBytes[rand.Intn(len(errorBytes))],
		Group:       errorHashs[rand.Intn(len(errorHashs))],
		Signature:   errorBytes[rand.Intn(len(errorBytes))],
		Nonce:       errorInt32[rand.Intn(len(errorInt32))],
		TxTree:      errorHashs[rand.Intn(len(errorHashs))],
		ReceiptTree: errorHashs[rand.Intn(len(errorHashs))],
		StateTree:   errorHashs[rand.Intn(len(errorHashs))],
		ExtraData:   errorBytes[rand.Intn(len(errorBytes))],
		Random:      errorBytes[rand.Intn(len(errorBytes))],
		GasFee:      errorUint64[rand.Intn(len(errorUint64))],
	}
}

func NewRandomErrorBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		Hash:        errorHashs[rand.Intn(len(errorHashs))],
		Height:      errorUint64[rand.Intn(len(errorUint64))],
		PreHash:     errorHashs[rand.Intn(len(errorHashs))],
		Elapsed:     errorInt32[rand.Intn(len(errorInt32))],
		ProveValue:  errorBytes[rand.Intn(len(errorBytes))],
		TotalQN:     errorUint64[rand.Intn(len(errorUint64))],
		Castor:      errorBytes[rand.Intn(len(errorBytes))],
		Group:       errorHashs[rand.Intn(len(errorHashs))],
		Signature:   errorBytes[rand.Intn(len(errorBytes))],
		Nonce:       errorInt32[rand.Intn(len(errorInt32))],
		TxTree:      errorHashs[rand.Intn(len(errorHashs))],
		ReceiptTree: errorHashs[rand.Intn(len(errorHashs))],
		StateTree:   errorHashs[rand.Intn(len(errorHashs))],
		ExtraData:   errorBytes[rand.Intn(len(errorBytes))],
		Random:      errorBytes[rand.Intn(len(errorBytes))],
		GasFee:      errorUint64[rand.Intn(len(errorUint64))],
	}
}

func GeneErrorBlockHeaders() []*types.BlockHeader {
	bhs := []*types.BlockHeader{}
	for i := 0; i < 200; i++ {
		bhs = append(bhs, NewRandomErrorBlockHeader())
	}
	return bhs
}

func GeneCorrectBlockHeaders() []*types.BlockHeader {
	bhs := []*types.BlockHeader{}
	hash := errorHashs[rand.Intn(len(errorHashs))]
	preHash := hash
	for i := 0; i < 200; i++ {
		bh := NewCorrectBlockHeader(hash, preHash)
		bh.Hash = bh.GenHash()
		hash = bh.Hash
		bhs = append(bhs, bh)
		preHash = hash
	}
	return bhs
}

func GeneCorrectBlockHeadersByFirstHash(firstHash common.Hash, len int) []*types.BlockHeader {
	bhs := []*types.BlockHeader{}
	hash := common.Hash{}
	preHash := firstHash
	for i := 0; i < len; i++ {
		bh := NewCorrectBlockHeader(hash, preHash)
		bh.Hash = bh.GenHash()
		hash = bh.Hash
		bhs = append(bhs, bh)
		preHash = hash
	}
	return bhs
}

func MarshalNilSyncRequest() ([]byte, error) {
	pbr := &SyncRequest{}
	return proto.Marshal(pbr)
}

func MarshalErorSyncRequest() ([]byte, error) {
	var height int64 = 100
	var size int64 = -10
	pbr := &SyncRequest{
		ReqHeight: &height,
		ReqSize:   &size,
	}
	return proto.Marshal(pbr)
}

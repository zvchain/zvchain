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
	time2 "github.com/zvchain/zvchain/middleware/time"
	"strconv"
	"time"
)
var(
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
func NewNilHeaderMessage(source string)*notify.DefaultMessage{
	return notify.NewDefaultMessage(nil, source, 0, 0)
}


func GenErrorDefaultMessage(source int)*notify.DefaultMessage{
	bh := NewRandomFullBlockHeader(uint64(rand.Intn(10000)))
	return GenErrorDefaultMessageWithBlockHeader(source,bh)
}


func GenDefaultMessageWithBytes(source int,data []byte)*notify.DefaultMessage{
	return notify.NewDefaultMessage(data,strconv.Itoa(source),uint16(rand.Intn(100)),uint16(rand.Intn(100)))
}

func GenErrorDefaultMessageWithBlockHeader(source int,bh *types.BlockHeader)*notify.DefaultMessage{
	blockHeader := types.BlockHeaderToPb(bh)

	blockInfo := tas_middleware_pb.TopBlockInfo{TopHeader: blockHeader}

	bt,_ := proto.Marshal(&blockInfo)

	return notify.NewDefaultMessage(bt,strconv.Itoa(source),uint16(rand.Intn(100)),uint16(rand.Intn(100)))
}


func NewRandomFullBlockHeader(height uint64)*types.BlockHeader{
	return &types.BlockHeader{
		Hash:common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Height: height,
		PreHash:common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Elapsed:     int32(rand.Intn(10000)),
		ProveValue: big.NewInt(int64(rand.Intn(10000))).Bytes(),
		TotalQN : uint64(rand.Intn(10000)),
		CurTime : time2.TimeToTimeStamp(time.Now()),
		Castor:      big.NewInt(int64(rand.Intn(10000))).Bytes(),
		Group  : common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Signature :big.NewInt(int64(rand.Intn(10000))).Bytes(),
		Nonce :int32(rand.Intn(10000)),
		TxTree :common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		ReceiptTree: common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		StateTree:common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		ExtraData:big.NewInt(int64(rand.Intn(10000))).Bytes(),
		Random :big.NewInt(int64(rand.Intn(10000))).Bytes(),
		GasFee :uint64(rand.Intn(10000)),
	}
}


func NewBlock(bh *types.BlockHeader)*types.Block{
	return &types.Block{
		Header : bh,
	}
}

func GenBlocks()[]*types.Block{
	blocks := []*types.Block{}
	bhs := GenErrorBlockHeaders()
	for _,data := range bhs{
		blocks = append(blocks,NewBlock(data))
	}
	return blocks
}

func GenErrorBlockHeaders()[]*types.BlockHeader{
	bhs := []*types.BlockHeader{}
	bhs = append(bhs,NewNilBlockHeader())
	bhs = append(bhs,GeneErrorBlockHeaders()...)
	return bhs
}

func NewNilBlockHeader()*types.BlockHeader{
	return &types.BlockHeader{}
}

func NewRandomErrorBlockHeader()*types.BlockHeader{
	return &types.BlockHeader{
		Hash:errorHashs[rand.Intn(len(errorHashs))],
		Height:errorUint64[rand.Intn(len(errorUint64))],
		PreHash:errorHashs[rand.Intn(len(errorHashs))],
		Elapsed:errorInt32[rand.Intn(len(errorInt32))],
		ProveValue:errorBytes[rand.Intn(len(errorBytes))],
		TotalQN:errorUint64[rand.Intn(len(errorUint64))],
		Castor:errorBytes[rand.Intn(len(errorBytes))],
		Group:errorHashs[rand.Intn(len(errorHashs))],
		Signature:errorBytes[rand.Intn(len(errorBytes))],
		Nonce :errorInt32[rand.Intn(len(errorInt32))],
		TxTree:errorHashs[rand.Intn(len(errorHashs))],
		ReceiptTree:errorHashs[rand.Intn(len(errorHashs))],
		StateTree :errorHashs[rand.Intn(len(errorHashs))],
		ExtraData:errorBytes[rand.Intn(len(errorBytes))],
		Random:errorBytes[rand.Intn(len(errorBytes))],
		GasFee:errorUint64[rand.Intn(len(errorUint64))],
	}
}


func GeneErrorBlockHeaders()[]*types.BlockHeader{
	bhs := []*types.BlockHeader{}
	for i := 0;i<200;i++{
		bhs = append(bhs,NewRandomErrorBlockHeader())
	}
	return bhs
}

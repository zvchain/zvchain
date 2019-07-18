package tas_middleware_test

import (
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"math/rand"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"strconv"
	"time"
)
var(
	errorHashs = []common.Hash{
		nil,
		common.Hash{},
		common.BytesToHash([]byte("i am evil.")),
		common.BigToHash(big.NewInt(11111)),
		common.BytesToHash([]byte("&^&^%%$%#))()SDDD")),
		common.BigToHash(big.NewInt(-11111111)),
		common.BigToHash(big.NewInt(00000000000000)),
		common.BytesToHash([]byte("0x999999132323222222222222222222222222222222222222222222222222222222222222222222222222")),
	}
)
func NewNilHeaderMessage(source string)*notify.DefaultMessage{
	return notify.NewDefaultMessage(nil, source, 0, 0)
}


func GenErrorDefaultMessage(source int)*notify.DefaultMessage{
	bh := NewRandomFullBlockHeader(uint64(rand.Intn(10000)))
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

func NewNilBlockHeader()*types.BlockHeader{
	return &types.BlockHeader{}
}


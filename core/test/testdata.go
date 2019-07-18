package tas_middleware_test

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"math/rand"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"time"
)


func NewNilHeaderMessage(source string)*notify.DefaultMessage{
	return notify.NewDefaultMessage(nil, source, 0, 0)
}


func GenErrorDefaultMessage(){
	bh := NewRandomFullBlockHeader()
	types.BlockHeaderToPb(bh)
	types.marshalTopBlockInfo
	return notify.DefaultMessage()
}


func NewRandomFullBlockHeader()*types.BlockHeader{
	return &types.BlockHeader{
		Hash:common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Height: uint64(rand.Intn(10000)),
		PreHash:common.BigToHash(big.NewInt(int64(rand.Intn(10000)))),
		Elapsed:     int32(rand.Intn(10000)),
		ProveValue: big.NewInt(int64(rand.Intn(10000))).Bytes(),
		TotalQN : uint64(rand.Intn(10000)),         // QN of the entire chain
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
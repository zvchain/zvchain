package tvm

import (
	"bytes"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/browser/ldb"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/common"
	"io"
	"math/big"
	"sync"
)

var ContractTransferData = make(chan *ContractTransfer, 500)
var MapTokenChan = make(map[string]chan *TokenContractTransfer)
var Lock = new(sync.RWMutex)
var MapTokenContractData = new(sync.Map)

type TokenContractTransfer struct {
	ContractAddr string
	Addr         string
	Value        string
	TxHash       string
	BlockHash    string
}

type ContractTransfer struct {
	Value        uint64
	Address      string
	TxHash       string
	BlockHeight  uint64
	ContractCode string
}

func ProduceTokenContractTransfer(txhash string, blockHash string, contracttoken string, addr []byte, value []byte) {
	contract := &TokenContractTransfer{
		ContractAddr: contracttoken,
		Addr:         string(addr),
		Value:        Valuetransfer(VmDataConvert(value)),
		BlockHash:    blockHash,
		TxHash:       txhash,
	}
	contracts := make([]*TokenContractTransfer, 1)
	contracts[0] = contract
	Lock.Lock()
	defer Lock.Unlock()
	if obj, ok := MapTokenContractData.Load(blockHash); ok {
		objToken := obj.([]*TokenContractTransfer)
		objToken = append(objToken, contract)
		fmt.Println("MapTokenContractData,exist:", "height:", util.ObjectTojson(objToken))
		MapTokenContractData.Store(blockHash, objToken)
	} else {
		MapTokenContractData.Store(blockHash, contracts)
	}
	//SetTokenContractMapToLdb(blockHash)
	fmt.Println("ProduceTokenContract,addr:", string(addr), "hash:", blockHash, ",contractcode:", contracttoken, "value", contract.Value)
}

func SetTokenContractMapToLdb(blockHash string, height uint64) {
	if obj, ok := (MapTokenContractData).Load(blockHash); ok {
		fmt.Println("SetTokenContractMapToLdb,exist:", "height:", height, util.ObjectTojson(obj))
		objToken := obj.([]*TokenContractTransfer)
		fmt.Println("addLdbData,exist:", util.ObjectTojson(objToken))

		addLdbData(blockHash, objToken)
	}
}

func GetTokenContractldbdata(blockkey string) ([]*TokenContractTransfer, error) {
	if blockkey == "" {
		return nil, fmt.Errorf("token data is empty")
	}
	data, _ := ldb.TokenSetdata.Get([]byte(blockkey))

	if len(data) == 0 {
		return nil, fmt.Errorf("token data is empty")
	}
	txs := make([]*TokenContractTransfer, 0)
	dataReader := bytes.NewReader(data)

	twoBytes := make([]byte, 2)
	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	txNum := common.ByteToUInt16(twoBytes)
	if txNum == 0 {
		return txs, nil
	}
	lenBytes := make([]byte, txNum*2)
	if _, err := io.ReadFull(dataReader, lenBytes); err != nil {
		return nil, err
	}

	for i := 0; i < int(txNum); i++ {
		txLen := common.ByteToUInt16(lenBytes[2*i : 2*(i+1)])
		txBytes := make([]byte, txLen)
		_, err := io.ReadFull(dataReader, txBytes)
		if err != nil {
			return nil, err
		}
		tx, err := unmarshalTo(txBytes)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func unmarshalTo(data []byte) (*TokenContractTransfer, error) {
	var tx TokenContractTransfer
	err := msgpack.Unmarshal(data, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func addLdbData(blockkey string, data []*TokenContractTransfer) {
	dataBuf := bytes.NewBuffer([]byte{})
	dataBuf.Write(common.UInt16ToByte(uint16(len(data))))
	txBuf := bytes.NewBuffer([]byte{})
	for _, tt := range data {
		txBytes, err := msgpack.Marshal(tt)
		if err != nil {
			continue
		}
		dataBuf.Write(common.UInt16ToByte(uint16(len(txBytes))))
		// Write each transaction length
		txBuf.Write(txBytes)
	}
	dataBuf.Write(txBuf.Bytes())
	ldb.TokenSetdata.Put([]byte(blockkey), dataBuf.Bytes())
}

func ProduceContractTransfer(txHash string,
	addr string,
	value uint64,
	contractCode string,
	blockHeight uint64) {
	contract := &ContractTransfer{
		Value:        value,
		Address:      addr,
		TxHash:       txHash,
		ContractCode: contractCode,
		BlockHeight:  blockHeight,
	}
	ContractTransferData <- contract
	fmt.Println("ProduceContractTransfer,addr:", addr, ",contractcode:", contractCode)
}

func Valuetransfer(valuedata interface{}) string {
	var valuestring string
	if value, ok := valuedata.(int64); ok {
		valuestring = big.NewInt(value).String()
	} else if value, ok := valuedata.(*big.Int); ok {
		valuestring = value.String()
	}
	return valuestring
}

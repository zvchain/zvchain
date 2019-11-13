package tvm

import (
	"fmt"
	"sync"
)

var ContractTransferData = make(chan *ContractTransfer, 500)
var MapTokenChan = make(map[string]chan *TokenContractTransfer)
var lock = new(sync.RWMutex)

type TokenContractTransfer struct {
	ContractAddr string
	Addr         []byte
	Value        interface{}
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
		Addr:         addr,
		Value:        VmDataConvert(value),
		BlockHash:    blockHash,
		TxHash:       txhash,
	}
	lock.Lock()
	if MapTokenChan[blockHash] == nil {
		MapTokenChan[blockHash] = make(chan *TokenContractTransfer, 500)
	}
	lock.Unlock()
	setmap(blockHash, contract)
	//TokenTransferData <- contract
	fmt.Println("ProduceTokenContractTransfer,addr:", string(addr), "hash:", blockHash, ",contractcode:", contracttoken, "value", contract.Value)
}
func setmap(blockHash string, contract *TokenContractTransfer) {
	lock.Lock()
	MapTokenChan[blockHash] <- contract
	lock.Unlock()

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

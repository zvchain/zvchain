package tvm

import (
	"fmt"
)

var ContractTransferData = make(chan *ContractTransfer, 500)
var TokenTransferData = make(chan *TokenContractTransfer, 500)

type TokenContractTransfer struct {
	ContractAddr string
	Addr         []byte
	Value        interface{}
	TxHash       string
	BlockHeight  uint64
}

type ContractTransfer struct {
	Value        uint64
	Address      string
	TxHash       string
	BlockHeight  uint64
	ContractCode string
}

func ProduceTokenContractTransfer(txhash string, blockHeight uint64, contracttoken string, addr []byte, value []byte) {
	contract := &TokenContractTransfer{
		ContractAddr: contracttoken,
		Addr:         addr,
		Value:        VmDataConvert(value),
		BlockHeight:  blockHeight,
		TxHash:       txhash,
	}
	TokenTransferData <- contract
	fmt.Println("ProduceTokenContractTransfer,addr:", string(addr), "height:", blockHeight, ",contractcode:", contracttoken, "value", contract.Value)
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

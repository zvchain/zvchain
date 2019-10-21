package tvm

import (
	"fmt"
)

var ContractTransferData chan *ContractTransfer

type ContractTransfer struct {
	Value        uint64
	Address      string
	TxHash       string
	BlockHeight  uint64
	ContractCode string
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

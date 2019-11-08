package tvm

import (
	"fmt"
)

var ContractTransferData chan *ContractTransfer
var TokenTransferData chan *TokenContractTransfer

type TokenContractTransfer struct {
	ContractAddr string
	Addr         []byte
	Value        interface{}
}

type ContractTransfer struct {
	Value        uint64
	Address      string
	TxHash       string
	BlockHeight  uint64
	ContractCode string
}

func ProduceTokenContractTransfer(contracttoken string, addr []byte, value []byte) {
	contract := &TokenContractTransfer{
		ContractAddr: contracttoken,
		Addr:         addr,
		Value:        VmDataConvert(value),
	}
	TokenTransferData <- contract
	fmt.Println("ProduceTokenContractTransfer,addr:", string(addr), ",contractcode:", contracttoken, "value", contract.Value)
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

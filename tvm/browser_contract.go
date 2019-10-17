package tvm

import (
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/browser/util"
)

func ProduceContractTransfer(txHash string, addr string, value uint64, contractCode string) {
	contractTransaction := &models.ContractTransaction{
		ContractCode: contractCode,
		Address:      addr,
		Value:        value,
		TxHash:       txHash,
	}
	mysql.DBStorage.AddContractTransaction(contractTransaction)
	fmt.Println("ProduceContractTransfer", util.ObjectTojson(contractTransaction))
}

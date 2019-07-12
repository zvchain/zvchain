package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"testing"
)

func TestCheckReceivedHashesInHitRate(t *testing.T) {
	peer := newPeerTxsKeys()
	txs := makeOrderTransactions()
	for i := 0; i < len(txs); i++ {
		peer.addSendHash(txs[i].Hash)
	}

	isSuccess := peer.checkReceivedHashesInHitRate(txs)
	if !isSuccess {
		t.Fatalf("except success,but got failed!")
	}

	changeHalfTxsToInvaildHashs(txs)

	isSuccess = peer.checkReceivedHashesInHitRate(txs)
	if !isSuccess {
		t.Fatalf("except success,but got failed!")
	}
	txlen := len(txs)

	txs[txlen-1].Hash = common.BigToAddress(big.NewInt(int64(300000))).Hash()
	isSuccess = peer.checkReceivedHashesInHitRate(txs)
	if isSuccess {
		t.Fatalf("except success,but got failed!")
	}
	changeHashsToSame(txs)

	isSuccess = peer.checkReceivedHashesInHitRate(txs)
	if isSuccess {
		t.Fatalf("except success,but got failed!")
	}
}

func changeHalfTxsToInvaildHashs(txs []*types.Transaction) {
	for i := 0; i < len(txs)/2; i++ {
		txs[i].Hash = common.BigToAddress(big.NewInt(int64(i + 30000))).Hash()
	}
}

func changeHashsToSame(txs []*types.Transaction) {
	for i := 0; i < len(txs); i++ {
		txs[i].Hash = common.BigToAddress(big.NewInt(int64(1))).Hash()
	}
}

func makeOrderTransactions() []*types.Transaction {
	txs := []*types.Transaction{}
	for i := 1; i <= 200; i++ {
		tx := &types.Transaction{Hash: common.BigToAddress(big.NewInt(int64(i))).Hash()}
		txs = append(txs, tx)
	}
	return txs
}

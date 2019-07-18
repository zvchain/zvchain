package core

import (
	"github.com/zvchain/zvchain/common"
	tas_middleware_test "github.com/zvchain/zvchain/core/test"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/taslog"
	"math/big"
	"strconv"
	"testing"
)

var blockSyncForTest *blockSyncer

func init(){
	blockSyncForTest = newBlockSyncer(nil)
	blockSyncForTest.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, "1")
	initPeerManager()
}
func TestGetBestCandidate(t *testing.T) {
	types.DefaultPVFunc = PvFuncTest
	for i := 0; i < 100; i++ {
		blockSync.addCandidatePool(strconv.Itoa(i), &types.BlockHeader{Hash: common.BigToAddress(big.NewInt(int64(i))).Hash(), TotalQN: uint64(i), ProveValue: genHash(strconv.Itoa(i))})
		peerManagerImpl.getOrAddPeer(strconv.Itoa(i))
	}
	for i := 0; i < 50; i++ {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	blockSync.getCandidateById("")

	if len(blockSync.candidatePool) != 50 {
		t.Fatalf("expect len is %d,but got %d", 50, peerManagerImpl.peerMeters.Len())
	}

	sc, _ := blockSync.getCandidateById("51")
	if sc == "" {
		t.Fatalf("expect get source,but got '' ")
	}
	sc, _ = blockSync.getCandidateById("49")
	if sc != "" {
		t.Fatalf("expect get '',but got %s ", sc)
	}

	for i := 50; i < 100; i++ {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	sc, _ = blockSync.getCandidateById("")
	if sc != "" {
		t.Fatalf("expect get '',but got %s ", sc)
	}

}

func PvFuncTest(pvBytes []byte) *big.Int {
	return new(big.Int)
}


func TestTopBlockInfoNotifyHandler(t *testing.T){
	//add a nil blockheader
	source := "0x111"
	blockSyncForTest.topBlockInfoNotifyHandler(tas_middleware_test.NewNilHeaderMessage(source))
	isPeerExists := peerManagerImpl.isPeerExists(source)
	if isPeerExists{
		t.Fatalf("expect nil,but got data")
	}
	candieData := blockSyncForTest.getPeerTopBlock (source)
	if candieData != nil{
		t.Fatalf("expect nil,but got data")
	}

	for i:=0;i<100;i++{

	}
}




package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/taslog"
	"math/big"
	"strconv"
	"testing"
)



func TestGetBestCandidate(t *testing.T){
	blockSync:=newBlockSyncer(nil)

	types.DefaultPVFunc = PvFuncTest
	blockSync.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, "1")
	initPeerManager()
	for i:=0; i<100;i++  {
		blockSync.addCandidatePool(strconv.Itoa(i),&types.BlockHeader{Hash:common.BigToAddress(big.NewInt(int64(i))).Hash(),TotalQN:uint64(i),ProveValue:genHash(strconv.Itoa(i))})
		peerManagerImpl.getOrAddPeer(strconv.Itoa(i))
	}
	for i:=0; i<50;i++  {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	blockSync.getCandidateById("")

	if len(blockSync.candidatePool) != 50{
		t.Fatalf("expect len is %d,but got %d",50,peerManagerImpl.peerMeters.Len())
	}

	sc,_ := blockSync.getCandidateById("51")
	if sc == ""{
		t.Fatalf("expect get source,but got '' ")
	}
	sc,_ = blockSync.getCandidateById("49")
	if sc != ""{
		t.Fatalf("expect get '',but got %s ",sc)
	}

	for i:=50; i<100;i++  {
		peerManagerImpl.addEvilCount(strconv.Itoa(i))
	}
	sc,_=blockSync.getCandidateById("")
	if sc != ""{
		t.Fatalf("expect get '',but got %s ",sc)
	}

}

func PvFuncTest(pvBytes []byte)*big.Int{
	return new(big.Int)
}





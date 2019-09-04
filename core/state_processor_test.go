package core

import (
	"fmt"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/log"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"

	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	executor  *stateProcessor
	adb       *account.AccountDB
	accountdb account.AccountDatabase
)

type cp4Test struct {
}

func (cp *cp4Test) updateVotes(db types.AccountDB, bh *types.BlockHeader) {
	return
}

func initExecutor() {
	executor = &stateProcessor{
		bc: &FullBlockChain{
			consensusHelper: NewConsensusHelper4Test(groupsig.ID{}),
			rewardManager:   NewRewardManager(),
		},
	}
	options := &opt.Options{
		OpenFilesCacheCapacity:        100,
		BlockCacheCapacity:            16 * opt.MiB,
		WriteBuffer:                   32 * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           4 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
		CompactionTotalSize:           16 * opt.MiB,
		BlockSize:                     2 * opt.MiB,
	}
	ds, err := tasdb.NewDataSource("test_db", options)
	if err != nil {
		panic(err)
	}

	statedb, err := ds.NewPrefixDatabase("state")
	if err != nil {
		panic(fmt.Sprintf("Init block chain error! Error:%s", err.Error()))
	}
	accountdb = account.NewDatabase(statedb)
	Logger = log.DefaultLogger

	executor = &stateProcessor{
		bc: &FullBlockChain{
			consensusHelper: NewConsensusHelper4Test(groupsig.ID{}),
			rewardManager:   NewRewardManager(),
		},
	}
	if BlockChainImpl == nil {
		BlockChainImpl = executor.bc.(*FullBlockChain)
	}

	GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})

}

func randomAddress() common.Address {
	r := rand.Uint64()
	return common.BytesToAddress(common.Uint64ToByte(r))
}

func genRandomTx() *types.Transaction {
	target := randomAddress()
	source := randomAddress()
	tx := &types.RawTransaction{
		Value:    types.NewBigInt(1),
		Nonce:    1,
		Target:   &target,
		Source:   &source,
		Type:     types.TransactionTypeTransfer,
		GasLimit: types.NewBigInt(10000),
		GasPrice: types.NewBigInt(1000),
	}
	return types.NewTransaction(tx, tx.GenHash())
}

func TestStateProcessor_process(t *testing.T) {
	initExecutor()
	defer clearDB()
	txNum := 10
	txs := make([]*types.Transaction, txNum)
	for i := 0; i < txNum; i++ {
		txs[i] = genRandomTx()
	}
	adb, err := account.NewAccountDB(common.Hash{}, accountdb)
	if err != nil {
		t.Fatal(err)
	}
	stateHash, evts, executed, receptes, _, err := executor.process(adb, &types.BlockHeader{}, txs, false, nil)
	if err != nil {
		t.Fatalf("execute error :%v", err)
	}
	t.Log(stateHash, len(executed), len(receptes))
	if len(txs) != len(executed)+len(evts) {
		t.Error("executed tx num error")
	}
}

func BenchmarkStateProcessor_process(b *testing.B) {
	txNum := 5400
	var state common.Hash
	var ts = common.NewTimeStatCtx()
	for i := 0; i < b.N; i++ {
		adb, err := account.NewAccountDB(state, accountdb)
		if err != nil {
			panic(err)
		}
		txs := make([]*types.Transaction, txNum)
		for i := 0; i < txNum; i++ {
			txs[i] = genRandomTx()
		}
		b := time.Now()
		executor.process(adb, &types.BlockHeader{}, txs, false, ts)
		ts.AddStat("process", time.Since(b))
	}
	b.Log(ts.Output())

}

func writeFile(f *os.File, bs *[]byte) {
	f.Write(*bs)
}
func TestReadWriteFile(t *testing.T) {
	file := "test_file"
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	begin := time.Now()
	cost := time.Duration(0)
	bs := make([]byte, 1024*1024*2)
	for i := 0; i < 100; i++ {
		b := time.Now()
		writeFile(f, &bs)
		cost += time.Since(b)
		//sha3.Sum256(randomAddress().Bytes())

	}
	t.Log(time.Since(begin).String(), cost.String())
}

func Test_validGasPrice(t *testing.T) {
	if validGasPrice(big.NewInt(1), 1) {
		t.Errorf("validGasPrice error -2, wanned false, got true!")
	}
	if validGasPrice(big.NewInt(1), 100000000) {
		t.Errorf("validGasPrice error -1, wanned false, got true!")
	}
	if !validGasPrice(big.NewInt(500), 1) {
		t.Errorf("validGasPrice error 0, wanned true, got false!")
	}
	if validGasPrice(big.NewInt(999), 30000000) {
		t.Errorf("validGasPrice error 1, wanned false, got true!")
	}
	if !validGasPrice(big.NewInt(1000), 30000000) {
		t.Errorf("validGasPrice error 2, wanned true, got false!")
	}
	if validGasPrice(big.NewInt(1999), 60000000) {
		t.Errorf("validGasPrice error 3, wanned true, got false!")
	}
	if !validGasPrice(big.NewInt(2000), 60000000) {
		t.Errorf("validGasPrice error 3, wanned true, got false!")
	}
	if validGasPrice(big.NewInt(3999), 90000000) {
		t.Errorf("validGasPrice error 3, wanned true, got false!")
	}

	if validGasPrice(big.NewInt(3999), 120000000) {
		t.Errorf("validGasPrice error 3, wanned true, got false!")
	}

	if !validGasPrice(big.NewInt(4000), 120000000) {
		t.Errorf("validGasPrice error 3, wanned true, got false!")
	}

}

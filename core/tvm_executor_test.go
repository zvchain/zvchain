package core

import (
	"common"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"math/rand"
	"middleware/types"
	"os"
	"storage/account"
	"storage/tasdb"
	"taslog"
	"testing"
	"time"
	"utility"
)

var (
	executor  *TVMExecutor
	adb       *account.AccountDB
	accountdb account.AccountDatabase
)

func init() {
	executor = &TVMExecutor{}
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

	Logger = taslog.GetLogger("")
}

func randomAddress() common.Address {
	r := rand.Uint64()
	return common.BytesToAddress(common.Uint64ToByte(r))
}

func genRandomTx() *types.Transaction {
	target := randomAddress()
	source := randomAddress()
	tx := &types.Transaction{
		Value:    1,
		Nonce:    1,
		Target:   &target,
		Source:   &source,
		Type:     types.TransactionTypeTransfer,
		GasLimit: 10000,
		GasPrice: 1000,
	}
	tx.Hash = tx.GenHash()
	return tx
}

func TestTVMExecutor_Execute(t *testing.T) {
	executor := &TVMExecutor{}
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
		t.Fatalf("new datasource error:%v", err)
	}

	statedb, err := ds.NewPrefixDatabase("state")
	if err != nil {
		t.Fatalf("Init block chain error! Error:%s", err.Error())
	}
	db := account.NewDatabase(statedb)

	adb, err := account.NewAccountDB(common.Hash{}, db)
	if err != nil {
		t.Fatal(err)
	}

	txNum := 10
	txs := make([]*types.Transaction, txNum)
	for i := 0; i < txNum; i++ {
		txs[i] = genRandomTx()
	}
	stateHash, evts, executed, receptes, err := executor.Execute(adb, &types.BlockHeader{}, txs, false, nil)
	if err != nil {
		t.Fatalf("execute error :%v", err)
	}
	t.Log(stateHash, evts, len(executed), len(receptes))
	if len(txs) != len(executed) {
		t.Error("executed tx num error")
	}
	for i, tx := range txs {
		if executed[i].Hash != tx.Hash {
			t.Error("execute tx error")
		}
	}
}

func BenchmarkTVMExecutor_Execute(b *testing.B) {
	txNum := 5400
	var state common.Hash
	var ts = utility.NewTimeStatCtx()
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
		executor.Execute(adb, &types.BlockHeader{}, txs, false, ts)
		ts.AddStat("Execute", time.Since(b))
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

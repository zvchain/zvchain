//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/zvchain/zvchain/common"
)

func TestCreatePool(t *testing.T) {
	err := initContext4Test(t)
	if err != nil {
		t.Fatalf("init fail:%v", err)
	}
	initBalance()
	defer clearSelf(t)
	pool := BlockChainImpl.GetTransactionPool()

	fmt.Printf("received: %d transactions\n", len(pool.GetReceived()))

	transaction := genTestTx(123457, "1", 1, 3)

	sign := common.BytesToSign(transaction.Sign)
	pk, err := sign.RecoverPubkey(transaction.Hash.Bytes())
	src := pk.GetAddress()

	accountDB, error := BlockChainImpl.LatestAccountDB()
	if error != nil {
		t.Fatalf("fail get account db")
	}
	accountDB.AddBalance(src, new(big.Int).SetUint64(111111111222))

	_, err = pool.AddTransaction(transaction)
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}
	fmt.Printf("received: %d transactions\n", len(pool.GetReceived()))

	h := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	transaction = genTestTx(12347, "1", 2, 3)

	_, err = pool.AddTransaction(transaction)
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}

	fmt.Printf("received: %d transactions\n", len(pool.GetReceived()))

	tGet := pool.GetTransaction(false, h)
	println(tGet)

	casting := pool.PackForCast()
	if len(casting) != 2 {
		t.Fatalf("casting length is wrong")
	}
}

func TestContainer(t *testing.T) {
	err := initContext4Test(t)
	if err != nil {
		t.Fatalf("init fail:%v", err)
	}
	defer clearSelf(t)
	initBalance()
	pool := BlockChainImpl.GetTransactionPool()

	var gasePrice1 uint64 = 12347
	var gasePrice2 uint64 = 12345

	transaction1 := genTestTx(gasePrice1, "1", 1, 3)

	var sign = common.BytesToSign(transaction1.Sign)
	pk, err := sign.RecoverPubkey(transaction1.Hash.Bytes())
	if err != nil {
		t.Fatalf("error")
	}
	src := pk.GetAddress()

	accountDB, err := BlockChainImpl.LatestAccountDB()
	if err != nil {
		t.Fatalf("get status failed")
	}
	accountDB.AddBalance(src, new(big.Int).SetUint64(111111111111111111))

	_, err = pool.AddTransaction(transaction1)
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}

	transaction2 := genTestTx(gasePrice2, "1", 2, 3)
	_, err = pool.AddTransaction(transaction2)
	if err != nil {
		t.Fatalf("fail to AddTransaction %v", err)
	}

	tGet := pool.GetTransaction(false, transaction1.Hash)
	if tGet.GasPrice.Uint64() != gasePrice1 {
		t.Fatalf("gas price is wrong")
	}

	tGet = pool.GetTransaction(false, transaction2.Hash)
	if tGet.GasPrice.Uint64() != gasePrice2 {
		t.Fatalf("gas price is wrong")
	}

}

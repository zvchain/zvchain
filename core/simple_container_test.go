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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"testing"
)

//var container = newSimpleContainer(6, 2)

const testTxCountPerBlock = 3

var (
	source1 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71111111111111"
	source2 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71222222222222"
	source3 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f71333333333333"
	source4 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f74444444444444"
	source5 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555555"
	source0 = "65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f00000000000000"

	addr1 = common.BytesToAddress(common.Hex2Bytes(source1))
	addr2 = common.BytesToAddress(common.Hex2Bytes(source2))
	addr3 = common.BytesToAddress(common.Hex2Bytes(source3))
	addr4 = common.BytesToAddress(common.Hex2Bytes(source4))
	addr5 = common.BytesToAddress(common.Hex2Bytes(source5))
	addr0 = common.BytesToAddress(common.Hex2Bytes(source0))

	gasLimit = types.NewBigInt(10000)

	tx1  = &types.Transaction{Hash: common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4"), Nonce: 3, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr0}
	tx2  = &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d88d"), Nonce: 1, GasPrice: types.NewBigInt(20000), GasLimit: gasLimit, Source: &addr1}
	tx3  = &types.Transaction{Hash: common.HexToHash("d1f1134223133d8ab88897b3ffc68c4797697b4e8603a7fd6a76722e3cc615ae"), Nonce: 1, GasPrice: types.NewBigInt(17000), GasLimit: gasLimit, Source: &addr2}
	tx4  = &types.Transaction{Hash: common.HexToHash("b4f213b67242f9439d62549fc128e98efe21b935b4a211b52b9b0b1812a57165"), Nonce: 1, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr3} //
	tx5  = &types.Transaction{Hash: common.HexToHash("80aa134ea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7123"), Nonce: 4, GasPrice: types.NewBigInt(11000), GasLimit: gasLimit, Source: &addr0}
	tx6  = &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d31a"), Nonce: 3, GasPrice: types.NewBigInt(21000), GasLimit: gasLimit, Source: &addr1}
	tx7  = &types.Transaction{Hash: common.HexToHash("d1f1134223133d8ab88897b3ffc68c4797697b4e8603a7fd6a76722e3cc617fa"), Nonce: 2, GasPrice: types.NewBigInt(9000), GasLimit: gasLimit, Source: &addr2}
	tx8  = &types.Transaction{Hash: common.HexToHash("3761a47f2b6745f1fefff25d529d18bd92ca460892f929b749e3995c4baac2d2"), Nonce: 1, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr0}
	tx9  = &types.Transaction{Hash: common.HexToHash("6d0edf5dc9d37e79d248b0f31796cfed580604b4ca1bcdd5aa696da6765a6054"), Nonce: 2, GasPrice: types.NewBigInt(9000), GasLimit: gasLimit, Source: &addr0}
	tx10 = &types.Transaction{Hash: common.HexToHash("49892838a63742cc522ad7a8c8be0f4360b13e83062a808a042c0b65b1fa096a"), Nonce: 1, GasPrice: types.NewBigInt(11000), GasLimit: gasLimit, Source: &addr0}
	tx11 = &types.Transaction{Hash: common.HexToHash("e41fe4ff98d0fc7df69686e79fa920bdfad6180d5162ce5324863f580522980a"), Nonce: 3, GasPrice: types.NewBigInt(11000), GasLimit: gasLimit, Source: &addr0}
	tx12 = &types.Transaction{Hash: common.HexToHash("b57b9520513eac56dc83af561d606340b8ac041b97f1741ccd11fc9c0cc098bd"), Nonce: 5, GasPrice: types.NewBigInt(8000), GasLimit: gasLimit, Source: &addr4}
	tx13 = &types.Transaction{Hash: common.HexToHash("1a375c639553f66d0ae4316bde2fc82a7b04a688ec63df04d63ff7f2b8d467ca"), Nonce: 1, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr5}
	tx14 = &types.Transaction{Hash: common.HexToHash("ca1896f3507580ef6f3c43d76bb097540f9281c5529c968f3e8f7328276ffe11"), Nonce: 1, GasPrice: types.NewBigInt(21000), GasLimit: gasLimit, Source: &addr1}
	tx15 = &types.Transaction{Hash: common.HexToHash("ba2c2944f27aeaa03ef97b42909b43e0ead02cf08d0c20433dda1a2e8b3c2e5a"), Nonce: 1, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr5}

	//txadd  = &types.Transaction{Hash: common.HexToHash("ba2c2944f27aeaa03ef97b42909b43e0ead02cf08d0c20433dda1a2e8b3c2e54"), Nonce: 2, GasPrice: 21000, Source: &addr1}
)

func printQueue() {
	for _, tx := range container.queue {
		fmt.Printf("[printQueue]: source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
	}
}

func printPending() {

	for _, list := range container.pending.waitingMap {
		for it := list.IterAtPosition(0); it.Next(); {
			tx := it.Value().(*orderByNonceTx).item
			fmt.Printf("[printPending map]: source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
		}
	}

}

var container *simpleContainer

func execute(t *testing.T, tx types.Transaction) {
	fmt.Printf("executing transacition : source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
	BlockChainImpl.(*FullBlockChain).latestStateDB.SetNonce(*tx.Source, tx.Nonce)
}

func Test_push(t *testing.T) {
	t1 := &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d881"), Nonce: 1, GasPrice: types.NewBigInt(20000), GasLimit: gasLimit, Source: &addr1}
	t2 := &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d882"), Nonce: 1, GasPrice: types.NewBigInt(19999), GasLimit: gasLimit, Source: &addr1}
	t3 := &types.Transaction{Hash: common.HexToHash("d3b14a7bab3c68e9369d0e433e5be9a514e843593f0f149cb0906e7bc085d883"), Nonce: 2, GasPrice: types.NewBigInt(20000), GasLimit: gasLimit, Source: &addr1}

	err := initContext4Test()
	fmt.Println("make sure the intrinsicGas check is disabled in the simple_container.go")
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}

	container = newSimpleContainer(10, 3, BlockChainImpl)

	_ = container.push(t1)
	_ = container.push(t2)
	_ = container.push(t3)

	rs := make([]*types.Transaction,3)
	for i, tx := range container.asSlice(3) {
		rs[i] = tx
		fmt.Printf("[asSlice] : source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
	}
	if rs[0].Hash != t1.Hash {
		t.Error("push test fail")
	}
	if rs[1].Hash != t3.Hash {
		t.Error("push test fail")
	}

	container = newSimpleContainer(10, 3, BlockChainImpl)

	_ = container.push(t2)
	_ = container.push(t1)
	_ = container.push(t3)

	rs = make([]*types.Transaction,3)
	for i, tx := range container.asSlice(3) {
		rs[i] = tx
		fmt.Printf("[asSlice] : source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
	}
	if rs[0].Hash != t1.Hash {
		t.Error("push test fail")
	}
	if rs[1].Hash != t3.Hash {
		t.Error("push test fail")
	}

}


func Test_simpleContainer_forEach(t *testing.T) {
	err := initContext4Test()
	fmt.Println("make sure the intrinsicGas check is disabled in the simple_container.go")
	if err != nil {
		t.Fatalf("failed to initContext4Test")
	}

	container = newSimpleContainer(10, 3, BlockChainImpl)

	tx22 := &types.Transaction{Hash: common.HexToHash("ba2c2944f27aeaa03ef97b42909b43e0ead02cf08d0c20433dda1a2e8b3c2e5a"), Nonce: 1, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr5}
	tx23 := &types.Transaction{Hash: common.HexToHash("ba2c2944f27aeaa03ef97b42909b43e0ead02cf08d0c20433dda1a2e8b3c2e5b"), Nonce: 1, GasPrice: types.NewBigInt(9999), GasLimit: gasLimit, Source: &addr5}
	tx24 := &types.Transaction{Hash: common.HexToHash("ba2c2944f27aeaa03ef97b42909b43e0ead02cf08d0c20433dda1a2e8b3c2e5c"), Nonce: 2, GasPrice: types.NewBigInt(10000), GasLimit: gasLimit, Source: &addr5}
	_ = container.push(tx22)
	_ = container.push(tx23)
	_ = container.push(tx24)

	for _, tx := range container.asSlice(10) {
		fmt.Printf("[asSlice1] : source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
	}

	txs := []*types.Transaction{
		tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12, tx13, tx14, tx15,
	}

	for _, tx := range txs {
		// this error can be ignored
		_ = container.push(tx)
	}
	for _, tx := range container.asSlice(10) {
		fmt.Printf("[asSlice] : source = %x, nonce = %d, gas = %d \n", tx.Source, tx.Nonce, tx.GasPrice)
	}

	printPending()
	printQueue()
	executed := packBlocks(t)
	fmt.Println(len(executed))
	printPending()
	printQueue()

}

//idealTxs := []*types.Transaction{
//	tx14, tx3, tx10,
//}
//container.eachForPack(func(tx *types.Transaction) bool {
//	txsFromPending = append(txsFromPending, tx)
//	return len(txsFromPending) < testTxCountPerBlock
//})
//
//for _,tx := range txsFromPending {
//	execute(t,*tx)
//}
//
//if !reflect.DeepEqual(idealTxs, txsFromPending) {
//	t.Error("foreach err，txs doesn't match")
//}
//

func packBlocks(t *testing.T) []*types.Transaction {
	result := make([]*types.Transaction, 0, testTxCountPerBlock)
	for {
		txsFromPending := packBlock(t)
		result = append(result, txsFromPending...)
		if len(txsFromPending) == 0 {
			break
		}
	}
	return result
}

func packBlock(t *testing.T) []*types.Transaction {
	txsFromPending := make([]*types.Transaction, 0, testTxCountPerBlock)
	fmt.Println("----next round----")
	txsFromPending = make([]*types.Transaction, 0, testTxCountPerBlock)
	container.eachForPack(func(tx *types.Transaction) bool {
		txsFromPending = append(txsFromPending, tx)
		return len(txsFromPending) < testTxCountPerBlock
	})
	for _, tx := range txsFromPending {
		execute(t, *tx)
	}
	container.promoteQueueToPending()
	for _, tx := range txsFromPending {
		container.remove(tx.Hash)
	}

	return txsFromPending

}

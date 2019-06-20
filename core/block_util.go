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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/tvm"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/serialize"
	"github.com/zvchain/zvchain/storage/trie"
)

var testTxAccount = []string{"0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103", "0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4",
	"0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402", "0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d",
	"0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5", "0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9",
	"0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03", "0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f",
	"0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc", "0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}

const adminAddr = "0x28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031b"
const miningPoolAddr = "0x28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031c"
const circulatesAddr = "0x28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031d"
const teamFoundationToken = 750000000 * common.TAS
const businessFoundationToken = 250000000 * common.TAS
const miningPoolToken = 425000000 * common.TAS
const circulatesToken = 75000000 * common.TAS

// IsTestTransaction is used for performance testing. We will not check the nonce if a transaction is sent from
// a testing account.
func IsTestTransaction(tx *types.Transaction) bool {
	if tx == nil || tx.Source == nil {
		return false
	}

	source := tx.Source.Hex()
	for _, testAccount := range testTxAccount {
		if source == testAccount {
			return true
		}
	}
	return false
}

func calcTxTree(txs []*types.Transaction) common.Hash {
	if nil == txs || 0 == len(txs) {
		return common.EmptyHash
	}

	buf := new(bytes.Buffer)

	for _, tx := range txs {
		buf.Write(tx.Hash.Bytes())
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return common.EmptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(receipts); i++ {
		if receipts[i] != nil {
			keybuf.Reset()
			serialize.Encode(keybuf, uint(i))
			encode, _ := serialize.EncodeToBytes(receipts[i])
			trie.Update(keybuf.Bytes(), encode)
		}
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
}

func setupGenesisStateDB(stateDB *account.AccountDB, genesisInfo *types.GenesisInfo) {
	tenThousandTasBi := big.NewInt(0).SetUint64(common.TAS2RA(10000))

	//管理员账户
	stateDB.SetBalance(common.HexToAddress(adminAddr), big.NewInt(0).SetUint64(common.TAS2RA(100000000)))
	stateDB.SetBalance(common.HexToAddress("0x6d880ddbcfb24180901d1ea709bb027cd86f79936d5ed23ece70bd98f22f2d84"), big.NewInt(0).SetUint64(common.TAS2RA(100000000)))

	// FoundationContract
	businessFoundationAddr := setupFoundationContract(stateDB, adminAddr, businessFoundationToken, 0)
	stateDB.SetBalance(*businessFoundationAddr, big.NewInt(0).SetUint64(businessFoundationToken))
	teamFoundationAddr := setupFoundationContract(stateDB, adminAddr, teamFoundationToken, 1)
	stateDB.SetBalance(*teamFoundationAddr, big.NewInt(0).SetUint64(teamFoundationToken))
	stateDB.SetNonce(common.HexToAddress(adminAddr), 2)

	// mining pool and circulates
	stateDB.SetBalance(common.HexToAddress(miningPoolAddr), big.NewInt(0).SetUint64(miningPoolToken))
	stateDB.SetBalance(common.HexToAddress(circulatesAddr), big.NewInt(0).SetUint64(circulatesToken))

	//创世账户
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, tenThousandTasBi)
	}

	// 交易脚本账户
	for _, acc := range testTxAccount {
		stateDB.SetBalance(common.HexToAddress(acc), tenThousandTasBi)
	}
}

func setupFoundationContract(stateDB *account.AccountDB, adminAddr string, totalToken, nonce uint64) *common.Address {
	code := `
import block
import account
class Foundation(object):
    def __init__(self):
        self.admin = "%s"
        self.total_token = %d
        self.withdrawed = 0
        self.first_year_weight = 64
        self.total_weight = 360

    def calculate_released(self):
        period = block.number() // 10000000
        if period > 11:
            period = 11
        weight = 0
        for i in range(period+1):
            weight = weight + self.first_year_weight // (2 ** (i // 3))
        print(weight)
        return self.total_token * weight // self.total_weight

    @register.public(int)
    def withdraw(self, amount):
        if msg.sender != self.admin:
            return
        can_withdraw = self.calculate_released() - self.withdrawed
        if amount > can_withdraw:
            return
        if account.get_balance(this) < amount:
            return
        self.withdrawed += amount
        account.transfer(msg.sender, self.admin)

    @register.public(str)    
    def change_admin(self, admin):
        self.admin = admin

`
	code = fmt.Sprintf(code, adminAddr, totalToken)
	transaction := types.Transaction{}
	addr := common.HexToAddress(adminAddr)
	transaction.Source = &addr
	transaction.Value = &types.BigInt{Int: *big.NewInt(0)}

	controller := tvm.NewController(stateDB, nil, nil, transaction, 0, "./py", nil, nil)
	contract := tvm.Contract{
		Code:         code,
		ContractName: "Foundation",
	}
	jsonBytes, err := json.Marshal(contract)
	if err != nil {
		panic("deploy FoundationContract error")
	}
	contractAddress := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.GetSource()[:], common.Uint64ToByte(nonce))))
	stateDB.CreateAccount(contractAddress)
	stateDB.SetCode(contractAddress, jsonBytes)

	contract.ContractAddress = &contractAddress
	controller.VM.SetGas(500000)
	err = controller.Deploy(&contract)
	if err != nil {
		panic("deploy FoundationContract error")
	}
	return contract.ContractAddress
}

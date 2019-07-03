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
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/serialize"
	"github.com/zvchain/zvchain/storage/trie"
	"github.com/zvchain/zvchain/tvm"
)

var testTxAccount = []string{"0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103", "0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4",
	"0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402", "0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d",
	"0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5", "0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9",
	"0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03", "0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f",
	"0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc", "0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}

const adminAddr = "0x28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031b"         // address of admin who can control foundation contract
const miningPoolAddr = "0x6d880ddbcfb24180901d1ea709bb027cd86f79936d5ed23ece70bd98f22f2d84"    // address of mining pool in pre-distribution
const circulatesAddr = "0xebb50bcade66df3fcb8df1eeeebad6c76332f2aee43c9c11b5cd30187b45f6d3"    // address of circulates in pre-distribution
const userNodeAddress = "0xe30c75b3fd8888f410ac38ec0a07d82dcc613053513855fb4dd6d75bc69e8139"   // address of official reserved user node address
const daemonNodeAddress = "0xae1889182874d8dad3c3e033cde3229a3320755692e37cbe1caab687bf6a1122" // address of official reserved daemon node address
const teamFoundationToken = 750000000 * common.TAS                                             // amount of tokens that belong to team
const businessFoundationToken = 250000000 * common.TAS                                         // amount of tokens that belongs to business
const miningPoolToken = 425000000 * common.TAS                                                 // amount of tokens that belongs to mining pool
const circulatesToken = 75000000 * common.TAS                                                  // amount of tokens that belongs to circulates

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

	// FoundationContract
	businessFoundationAddr := setupFoundationContract(stateDB, adminAddr, businessFoundationToken, 1)
	stateDB.SetBalance(*businessFoundationAddr, big.NewInt(0).SetUint64(businessFoundationToken))
	teamFoundationAddr := setupFoundationContract(stateDB, adminAddr, teamFoundationToken, 2)
	stateDB.SetBalance(*teamFoundationAddr, big.NewInt(0).SetUint64(teamFoundationToken))
	stateDB.SetNonce(common.HexToAddress(adminAddr), 2)

	// mining pool and circulates
	stateDB.SetBalance(common.HexToAddress(miningPoolAddr), big.NewInt(0).SetUint64(miningPoolToken))
	stateDB.SetBalance(common.HexToAddress(circulatesAddr), big.NewInt(0).SetUint64(circulatesToken))

	// genesis balance: just for stakes two roles with minimum required value
	genesisBalance := big.NewInt(0).SetUint64(2 * minimumStake())
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, genesisBalance)
	}
}

func setupFoundationContract(stateDB *account.AccountDB, adminAddr string, totalToken, nonce uint64) *common.Address {
	code := fmt.Sprintf(foundationContract, adminAddr, totalToken)
	transaction := types.Transaction{}
	addr := common.HexToAddress(adminAddr)
	transaction.Source = &addr
	transaction.Value = &types.BigInt{Int: *big.NewInt(0)}
	transaction.GasLimit = &types.BigInt{Int: *big.NewInt(300000)}
	controller := tvm.NewController(stateDB, nil, nil, transaction, 0, "./py", nil)
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

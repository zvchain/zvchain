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

package cli

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"strings"
)

type groupInfoReader interface {
	// GetActivatedGroupsAt gets available groups' seed at the given height
	GetActivatedGroupsAt(height uint64) []types.GroupI
	GetLivedGroupsAt(height uint64) []types.GroupI
	// GetGroupBySeed returns the group info of the given seed
	GetGroupBySeed(seedHash common.Hash) types.GroupI
	// GetGroupHeaderBySeed returns the group header info of the given seed
	GetGroupHeaderBySeed(seedHash common.Hash) types.GroupHeaderI
	Height() uint64

	GroupsAfter(height uint64) []types.GroupI
	ActiveGroupCount() int
	GetLivedGroupsByMember(address common.Address, height uint64) []types.GroupI
}

type groupRoutineChecker interface {
	CurrentEraCheck(address common.Address) (selected bool, seed common.Hash, seedHeight uint64, stage int)
}

type blockReader interface {
	CheckPointAt(h uint64) *types.BlockHeader
}

func getGroupReader() groupInfoReader {
	return core.GroupManagerImpl
}

type rpcBaseImpl struct {
	gr groupInfoReader
	br blockReader
}

// RpcGtasImpl provides rpc service for users to interact with remote nodes
type RpcGtasImpl struct {
	*rpcBaseImpl
	routineChecker groupRoutineChecker
}

func (api *RpcGtasImpl) Namespace() string {
	return "Gzv"
}

func (api *RpcGtasImpl) Version() string {
	return "1"
}

func failErrResult(err string) *ErrorResult {
	return &ErrorResult{
		Message: err,
		Code:    -1,
	}
}

// Tx is user transaction interface, used for sending transaction to the node
func (api *RpcGtasImpl) Tx(txRaw *TxRawData) (string, error) {
	if !validateTxType(txRaw.TxType) {
		return "", fmt.Errorf("not supported txType")
	}

	// Check the address for the specified tx types
	switch txRaw.TxType {
	case types.TransactionTypeTransfer, types.TransactionTypeContractCall, types.TransactionTypeStakeAdd,
		types.TransactionTypeStakeReduce,
		types.TransactionTypeStakeRefund, types.TransactionTypeVoteMinerPool:
		if !common.ValidateAddress(strings.TrimSpace(txRaw.Target)) {
			return "", fmt.Errorf("wrong target address format")
		}
	}
	if !common.ValidateAddress(txRaw.Source) {
		return "", fmt.Errorf("wrong source address")
	}

	trans := txRawToTransaction(txRaw)

	if err := sendTransaction(trans); err != nil {
		return "", err
	}

	return trans.Hash.Hex(), nil
}

// Balance is query balance interface
func (api *RpcGtasImpl) Balance(account string) (float64, error) {
	account = strings.TrimSpace(account)
	if !common.ValidateAddress(account) {
		return 0, fmt.Errorf("Wrong account address format")
	}
	b := core.BlockChainImpl.GetBalance(common.StringToAddress(account))

	balance := common.RA2TAS(b.Uint64())
	return balance, nil
}

// BlockHeight query block height
func (api *RpcGtasImpl) BlockHeight() (uint64, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return height, nil
}

// GroupHeight query group height
func (api *RpcGtasImpl) GroupHeight() (uint64, error) {
	height := core.GroupManagerImpl.Height()
	return height, nil
}

func (api *RpcGtasImpl) GetBlockByHeight(height uint64) (*Block, error) {
	b := core.BlockChainImpl.QueryBlockByHeight(height)
	if b == nil {
		return nil, fmt.Errorf("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return block, nil
}

func (api *RpcGtasImpl) GetBlockByHash(hash string) (*Block, error) {
	hash = strings.TrimSpace(hash)
	if !validateHash(hash) {
		return nil, fmt.Errorf("wrong hash format")
	}
	b := core.BlockChainImpl.QueryBlockByHash(common.HexToHash(hash))
	if b == nil {
		return nil, fmt.Errorf("block not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return block, nil
}

func (api *RpcGtasImpl) GetTxsByBlockHash(hash string) ([]string, error) {
	hash = strings.TrimSpace(hash)
	if !validateHash(hash) {
		return nil, fmt.Errorf("wrong hash format")
	}
	b := core.BlockChainImpl.QueryBlockByHash(common.HexToHash(hash))
	if b == nil {
		return nil, fmt.Errorf("block not exists")
	}
	txs := make([]string, len(b.Transactions))
	for index, tx := range b.Transactions {
		txs[index] = tx.GenHash().Hex()
	}
	return txs, nil
}

func (api *RpcGtasImpl) GetTxsByBlockHeight(height uint64) ([]string, error) {
	b := core.BlockChainImpl.QueryBlockByHeight(height)
	if b == nil {
		return nil, fmt.Errorf("height not exists")
	}
	txs := make([]string, len(b.Transactions))
	for index, tx := range b.Transactions {
		txs[index] = tx.GenHash().Hex()
	}
	return txs, nil
}

func (api *RpcGtasImpl) MinerPoolInfo(addr string, height uint64) (*MinerPoolDetail, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(addr) {
		return nil, fmt.Errorf("Wrong account address format")
	}
	var db types.AccountDB
	var err error
	if height == 0 {
		height = core.BlockChainImpl.Height()
		db, err = core.BlockChainImpl.LatestAccountDB()
	} else {
		db, err = core.BlockChainImpl.AccountDBAt(height)
	}
	if err != nil || db == nil {
		return nil, fmt.Errorf("data is nil")
	}
	miner := core.MinerManagerImpl.GetMiner(common.StringToAddress(addr), types.MinerTypeProposal, height)
	var currentStake uint64 = 0
	var fullStake uint64 = 0
	identity := types.MinerNormal
	if miner != nil {
		currentStake = miner.Stake
		identity = miner.Identity
		if miner.IsMinerPool() {
			fullStake = core.MinerManagerImpl.GetFullMinerPoolStake(height)
		}
	}
	tickets := core.MinerManagerImpl.GetTickets(db, common.StringToAddress(addr))
	dt := &MinerPoolDetail{
		CurrentStake: currentStake,
		FullStake:    fullStake,
		Tickets:      tickets,
		Identity:     uint64(identity),
		ValidTickets: core.MinerManagerImpl.GetValidTicketsByHeight(height),
	}
	return dt, nil
}

func (api *RpcGtasImpl) MinerInfo(addr string, detail string) (*MinerStakeDetails, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(addr) {
		return nil, fmt.Errorf("wrong account address format")
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		detail = addr
	} else {
		if !common.ValidateAddress(strings.TrimSpace(detail)) {
			return nil, fmt.Errorf("wrong account address format")
		}
	}

	mTypeString := func(mt types.MinerType) string {
		if types.IsVerifyRole(mt) {
			return "verifier"
		} else if types.IsProposalRole(mt) {
			return "proposal"
		}
		return "unknown"
	}
	statusString := func(st types.StakeStatus) string {
		if st == types.Staked {
			return "normal"
		} else if st == types.StakeFrozen {
			return "frozen"
		} else if st == types.StakePunishment {
			return "punish"
		}
		return "unknown"
	}
	convertDetails := func(dts []*types.StakeDetail) []*StakeDetail {
		details := make([]*StakeDetail, 0)
		for _, d := range dts {
			dt := &StakeDetail{
				Value:           uint64(common.RA2TAS(d.Value)),
				UpdateHeight:    d.UpdateHeight,
				MType:           mTypeString(d.MType),
				Status:          statusString(d.Status),
				CanReduceHeight: d.DisMissHeight,
			}
			details = append(details, dt)
		}
		return details
	}

	minerDetails := &MinerStakeDetails{}
	morts := make([]*MortGage, 0)
	address := common.StringToAddress(addr)
	proposalInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeProposal)
	if proposalInfo != nil {
		morts = append(morts, NewMortGageFromMiner(proposalInfo))
	}
	verifierInfo := core.MinerManagerImpl.GetLatestMiner(address, types.MinerTypeVerify)
	if verifierInfo != nil {
		morts = append(morts, NewMortGageFromMiner(verifierInfo))
	}
	minerDetails.Overview = morts
	// Get details
	details := core.MinerManagerImpl.GetStakeDetails(address, common.StringToAddress(detail))
	m := make(map[string][]*StakeDetail)
	dts := convertDetails(details)
	m[detail] = dts
	minerDetails.Details = m

	return minerDetails, nil
}

func (api *RpcGtasImpl) TransDetail(h string) (*Transaction, error) {
	h = strings.TrimSpace(h)
	if !validateHash(h) {
		return nil, fmt.Errorf("wrong hash format")
	}
	tx := core.BlockChainImpl.GetTransactionByHash(false, common.HexToHash(h))

	if tx != nil {
		trans := convertTransaction(tx)
		return trans, nil
	}
	return nil, nil
}

func (api *RpcGtasImpl) Nonce(addr string) (uint64, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(addr) {
		return 0, fmt.Errorf("wrong account address format")
	}
	address := common.StringToAddress(addr)
	// user will see the nonce as db nonce +1, so that user can use it directly when send a transaction
	nonce := core.BlockChainImpl.GetNonce(address) + 1
	return nonce, nil
}

func (api *RpcGtasImpl) TxReceipt(h string) (*ExecutedTransaction, error) {
	h = strings.TrimSpace(h)
	if !validateHash(h) {
		return nil, fmt.Errorf("wrong hash format")
	}
	hash := common.HexToHash(h)
	rc := core.BlockChainImpl.GetTransactionPool().GetReceipt(hash)
	if rc != nil {
		tx := core.BlockChainImpl.GetTransactionByHash(false, hash)
		return convertExecutedTransaction(&types.ExecutedTransaction{
			Receipt:     rc,
			Transaction: tx,
		}), nil
	}
	return nil, nil
}

// ViewAccount is used for querying account information
func (api *RpcGtasImpl) ViewAccount(hash string) (*ExplorerAccount, error) {
	hash = strings.TrimSpace(hash)
	if !common.ValidateAddress(hash) {
		return nil, fmt.Errorf("wrong address format")
	}
	accountDb, err := core.BlockChainImpl.LatestAccountDB()
	if err != nil {
		return nil, fmt.Errorf("get status failed")
	}
	if accountDb == nil {
		return nil, nil
	}
	address := common.StringToAddress(hash)
	if !accountDb.Exist(address) {
		return nil, fmt.Errorf("account not Exist!")
	}
	account := &ExplorerAccount{}
	account.Balance = accountDb.GetBalance(address)
	account.Nonce = accountDb.GetNonce(address)
	account.CodeHash = accountDb.GetCodeHash(address).Hex()
	account.Code = string(accountDb.GetCode(address)[:])

	account.Type = 0
	if len(account.Code) > 0 {
		account.Type = 1
		account.StateData = make(map[string]interface{})

		contract := tvm.Contract{}
		err = json.Unmarshal([]byte(account.Code), &contract)
		if err != nil {
			return nil, fmt.Errorf("UnMarshall contract fail!" + err.Error())
		}
		abi := parseABI(contract.Code)
		account.ABI = abi

		iter := accountDb.DataIterator(common.StringToAddress(hash), []byte{})
		for iter.Next() {
			k := string(iter.Key[:])
			v := string(iter.Value[:])
			account.StateData[k] = v

		}
	}
	return account, nil
}

func (api *RpcGtasImpl) QueryAccountData(addr string, key string, count int) (interface{}, error) {
	addr = strings.TrimSpace(addr)
	// input check
	if !common.ValidateAddress(addr) {
		return nil, fmt.Errorf("wrong address format")
	}
	address := common.StringToAddress(addr)

	const MaxCountQuery = 100
	if count <= 0 {
		count = 0
	} else if count > MaxCountQuery {
		count = MaxCountQuery
	}

	chain := core.BlockChainImpl
	state, err := chain.GetAccountDBByHash(chain.QueryTopBlock().Hash)
	if err != nil {
		return nil, err
	}

	var resultData interface{}
	if count == 0 {
		value := state.GetData(address, []byte(key))
		if value != nil {
			tmp := make(map[string]interface{})
			tmp["value"] = string(value)
			resultData = tmp
		}
	} else {
		iter := state.DataIterator(address, []byte(key))
		if iter != nil {
			tmp := make([]map[string]interface{}, 0)
			for iter.Next() {
				k := string(iter.Key[:])
				if !strings.HasPrefix(k, key) {
					continue
				}
				v := string(iter.Value[:])
				item := make(map[string]interface{}, 0)
				item["key"] = k
				item["value"] = v
				tmp = append(tmp, item)
				resultData = tmp
				if len(tmp) >= count {
					break
				}
			}
		}
	}
	if resultData != nil {
		return resultData, nil
	} else {
		return nil, fmt.Errorf("query does not have data")
	}
}

func (api *RpcGtasImpl) GroupCheck(addr string) (*GroupCheckInfo, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(addr) {
		return nil, fmt.Errorf("wrong address format:%s", addr)
	}
	address := common.StringToAddress(addr)
	height := core.BlockChainImpl.Height()
	joinedGroups := api.gr.GetLivedGroupsByMember(address, height)
	jgs := make([]*JoinedGroupInfo, 0)
	for _, jg := range joinedGroups {
		info := &JoinedGroupInfo{
			Seed:          jg.Header().Seed(),
			WorkHeight:    jg.Header().WorkHeight(),
			DismissHeight: jg.Header().DismissHeight(),
		}
		jgs = append(jgs, info)
	}

	selected, seed, sh, stage := api.routineChecker.CurrentEraCheck(address)
	currentInfo := &CurrentEraGroupInfo{
		Selected:   selected,
		GroupSeed:  seed,
		SeedHeight: sh,
	}
	if selected {
		switch stage {
		case 0:
			currentInfo.MinerActionStage = "sent no data"
		case 1:
			currentInfo.MinerActionStage = "has sent encrypted share piece packet"
		case 2:
			currentInfo.MinerActionStage = "has sent mpk packet"
		case 3:
			currentInfo.MinerActionStage = "has sent origin share piece packet"
		default:
			currentInfo.MinerActionStage = fmt.Sprintf("unknown stage:%v", stage)
		}
	}

	return &GroupCheckInfo{JoinedGroups: jgs, CurrentGroupRoutine: currentInfo}, nil
}

func (api *RpcGtasImpl) CheckPointAt(h uint64) (*types.BlockHeader, error) {
	cp := api.br.CheckPointAt(h)
	return cp, nil
}

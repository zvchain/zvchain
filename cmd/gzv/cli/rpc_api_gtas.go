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
	// GetAvailableGroupSeeds gets available groups' seed at the given height
	GetAvailableGroupSeeds(height uint64) []types.SeedI
	// GetGroupBySeed returns the group info of the given seed
	GetGroupBySeed(seedHash common.Hash) types.GroupI
	// GetGroupHeaderBySeed returns the group header info of the given seed
	GetGroupHeaderBySeed(seedHash common.Hash) types.GroupHeaderI
	Height() uint64

	GroupsAfter(height uint64) []types.GroupI
	ActiveGroupCount() int
	GetLivedGroupsByMember(address common.Address, height uint64) []types.GroupI
}

type currentEraStatus interface {
	MinerSelected() bool
	MinerStatus() int
	GroupHeight() uint64
	GroupSeed() common.Hash
}

type groupRoutineChecker interface {
	CurrentEraCheck(address common.Address) (selected bool, seed common.Hash, seedHeight uint64, stage int)
}

func getGroupReader() groupInfoReader {
	return &core.GroupManagerImpl
}

type rpcBaseImpl struct {
	gr groupInfoReader
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

func successResult(data interface{}) (*Result, error) {
	return &Result{
		Message: "success",
		Data:    data,
		Status:  0,
	}, nil
}
func failResult(err string) (*Result, error) {
	return &Result{
		Message: err,
		Data:    nil,
		Status:  -1,
	}, nil
}

// Tx is user transaction interfavlce, used for sending transaction to the node
func (api *RpcGtasImpl) Tx(txRawjson string) (*Result, error) {
	var txRaw = new(txRawData)
	if err := json.Unmarshal([]byte(txRawjson), txRaw); err != nil {
		return failResult(err.Error())
	}
	if !validateTxType(txRaw.TxType) {
		return failResult("Not supported txType")
	}

	// Check the address for the specified tx types
	switch txRaw.TxType {
	case types.TransactionTypeTransfer, types.TransactionTypeContractCall, types.TransactionTypeStakeAdd,
		types.TransactionTypeMinerAbort, types.TransactionTypeStakeReduce, types.TransactionTypeApplyGuardMiner,
		types.TransactionTypeStakeRefund, types.TransactionTypeVoteMinerPool, types.TransactionTypeChangeFundGuardMode:
		if !common.ValidateAddress(strings.TrimSpace(txRaw.Target)) {
			return failResult("Wrong target address format")
		}
	}

	trans := txRawToTransaction(txRaw)

	trans.Hash = trans.GenHash()

	if err := sendTransaction(trans); err != nil {
		return failResult(err.Error())
	}

	return successResult(trans.Hash.Hex())
}

// Balance is query balance interface
func (api *RpcGtasImpl) Balance(account string) (*Result, error) {
	if !common.ValidateAddress(strings.TrimSpace(account)) {
		return failResult("Wrong account address format")
	}
	b := core.BlockChainImpl.GetBalance(common.StringToAddress(account))

	balance := common.RA2TAS(b.Uint64())
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %v ZVC", account, balance),
		Data:    balance,
	}, nil
}

// BlockHeight query block height
func (api *RpcGtasImpl) BlockHeight() (*Result, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return successResult(height)
}

// GroupHeight query group height
func (api *RpcGtasImpl) GroupHeight() (*Result, error) {
	height := core.GroupManagerImpl.Height()
	return successResult(height)
}

func (api *RpcGtasImpl) GetBlockByHeight(height uint64) (*Result, error) {
	b := core.BlockChainImpl.QueryBlockByHeight(height)
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *RpcGtasImpl) GetBlockByHash(hash string) (*Result, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return failResult("Wrong hash format")
	}
	b := core.BlockChainImpl.QueryBlockByHash(common.HexToHash(hash))
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *RpcGtasImpl) MinerPoolInfo(addr string, height uint64) (*Result, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong account address format")
	}
	var db types.AccountDB
	var err error
	if height == 0 {
		height = core.BlockChainImpl.Height()
		db, err = core.BlockChainImpl.LatestStateDB()
	} else {
		db, err = core.BlockChainImpl.GetAccountDBByHeight(height)
	}
	if err != nil || db == nil {
		return failResult("data is nil")
	}
	miner := core.MinerManagerImpl.GetMiner(common.StringToAddress(addr), types.MinerTypeProposal, height)
	if miner == nil{
		msg :=fmt.Sprintf("this miner is nil,addr is %s",addr)
		return failResult(msg)
	}
	if !miner.IsMinerPool() {
		msg :=fmt.Sprintf("this addr is not miner pool,identity is %d",miner.Identity)
		return failResult(msg)
	}
	tickets := core.MinerManagerImpl.GetTickets(db, common.StringToAddress(addr))
	fullStake := core.MinerManagerImpl.GetFullMinerPoolStake(height)
	dt := &MinerPoolDetail{
		CurrentStake: miner.Stake,
		FullStake:    fullStake,
		Tickets:      tickets,
	}
	return successResult(dt)
}

func (api *RpcGtasImpl) TicketsInfo(addr string) (*Result, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong account address format")
	}
	db, err := core.BlockChainImpl.LatestStateDB()
	if err != nil || db == nil {
		return failResult("data is nil")
	}
	tickets := core.MinerManagerImpl.GetTickets(db, common.StringToAddress(addr))
	return successResult(tickets)
}

func (api *RpcGtasImpl) MinerInfo(addr string, detail string) (*Result, error) {
	addr = strings.TrimSpace(addr)
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong account address format")
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		detail = addr
	} else {
		if !common.ValidateAddress(strings.TrimSpace(detail)) {
			return failResult("Wrong detail address format")
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
				Value:         uint64(common.RA2TAS(d.Value)),
				UpdateHeight:  d.UpdateHeight,
				MType:         mTypeString(d.MType),
				Status:        statusString(d.Status),
				DisMissHeight: d.DisMissHeight,
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

	return successResult(minerDetails)
}

func (api *RpcGtasImpl) TransDetail(h string) (*Result, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return failResult("Wrong hash format")
	}
	tx := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(h))

	if tx != nil {
		trans := convertTransaction(tx)
		return successResult(trans)
	}
	return successResult(nil)
}

func (api *RpcGtasImpl) Nonce(addr string) (*Result, error) {
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong account address format")
	}
	address := common.StringToAddress(addr)
	// user will see the nonce as db nonce +1, so that user can use it directly when send a transaction
	nonce := core.BlockChainImpl.GetNonce(address) + 1
	return successResult(nonce)
}

func (api *RpcGtasImpl) TxReceipt(h string) (*Result, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return failResult("Wrong hash format")
	}
	hash := common.HexToHash(h)
	rc := core.BlockChainImpl.GetTransactionPool().GetReceipt(hash)
	if rc != nil {
		tx := core.BlockChainImpl.GetTransactionByHash(false, true, hash)
		return successResult(convertExecutedTransaction(&types.ExecutedTransaction{
			Receipt:     rc,
			Transaction: tx,
		}))
	}
	return failResult("tx not exist")
}

// ViewAccount is used for querying account information
func (api *RpcGtasImpl) ViewAccount(hash string) (*Result, error) {
	if !common.ValidateAddress(strings.TrimSpace(hash)) {
		return failResult("Wrong address format")
	}
	accountDb, err := core.BlockChainImpl.LatestStateDB()
	if err != nil {
		return failResult("Get status failed")
	}
	if accountDb == nil {
		return nil, nil
	}
	address := common.StringToAddress(hash)
	if !accountDb.Exist(address) {
		return failResult("Account not Exist!")
	}
	account := ExplorerAccount{}
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
			return failResult("UnMarshall contract fail!" + err.Error())
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
	return successResult(account)
}

func (api *RpcGtasImpl) QueryAccountData(addr string, key string, count int) (*Result, error) {
	// input check
	if !common.ValidateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong address format")
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
		return failResult(err.Error())
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
		return successResult(resultData)
	} else {
		return failResult("query does not have data")
	}
}

func (api *RpcGtasImpl) GroupCheck(addr string) (*Result, error) {
	if !common.ValidateAddress(addr) {
		return failResult("Wrong address format:" + addr)
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

	return successResult(&GroupCheckInfo{JoinedGroups: jgs, CurrentGroupRoutine: currentInfo})
}

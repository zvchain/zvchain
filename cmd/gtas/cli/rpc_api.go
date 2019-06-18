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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
	"math"
	"math/big"
	"strconv"
)

var BonusLogger taslog.Logger

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

// GtasAPI is a single-method API handler to be returned by test services.
type GtasAPI struct {
}

// Tx is user transaction interface
func (api *GtasAPI) Tx(txRawjson string) (*Result, error) {
	var txRaw = new(txRawData)
	if err := json.Unmarshal([]byte(txRawjson), txRaw); err != nil {
		return failResult(err.Error())
	}

	trans := txRawToTransaction(txRaw)

	trans.Hash = trans.GenHash()

	if err := sendTransaction(trans); err != nil {
		return failResult(err.Error())
	}

	return successResult(trans.Hash.Hex())
}

// Balance is query balance interface
func (api *GtasAPI) Balance(account string) (*Result, error) {
	balance, err := walletManager.getBalance(account)
	if err != nil {
		return nil, err
	}
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %v TAS", account, balance),
		Data:    fmt.Sprintf("%v", balance),
	}, nil
}

// NewWallet is create a new account interface
func (api *GtasAPI) NewWallet() (*Result, error) {
	privKey, addr := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	data["address"] = addr
	return successResult(data)
}

// GetWallets get the wallets of the current node
func (api *GtasAPI) GetWallets() (*Result, error) {
	return successResult(walletManager)
}

// DeleteWallet delete the address of the specified serial number of the local node
func (api *GtasAPI) DeleteWallet(key string) (*Result, error) {
	walletManager.deleteWallet(key)
	return successResult(walletManager)
}

// BlockHeight query block height
func (api *GtasAPI) BlockHeight() (*Result, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return successResult(height)
}

// GroupHeight query group height
func (api *GtasAPI) GroupHeight() (*Result, error) {
	height := core.GroupChainImpl.Height()
	return successResult(height)
}

// ConnectedNodes query the information of the linked node
func (api *GtasAPI) ConnectedNodes() (*Result, error) {

	nodes := network.GetNetInstance().ConnInfo()
	conns := make([]ConnInfo, 0)
	for _, n := range nodes {
		conns = append(conns, ConnInfo{ID: n.ID, IP: n.IP, TCPPort: n.Port})
	}
	return successResult(conns)
}

// TransPool query buffer transaction information
func (api *GtasAPI) TransPool() (*Result, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Hash:   v.Hash.Hex(),
			Source: v.Source.Hex(),
			Target: v.Target.Hex(),
			Value:  strconv.FormatInt(int64(v.Value), 10),
		})
	}

	return successResult(transList)
}

// get transaction by hash
func (api *GtasAPI) GetTransaction(hash string) (*Result, error) {
	transaction := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(hash))
	if transaction == nil {
		return failResult("transaction not exists")
	}
	detail := make(map[string]interface{})
	detail["hash"] = hash
	if transaction.Source != nil {
		detail["source"] = transaction.Source.Hash().Hex()
	}
	if transaction.Target != nil {
		detail["target"] = transaction.Target.Hash().Hex()
	}
	detail["value"] = transaction.Value

	return successResult(detail)
}

func (api *GtasAPI) GetBlockByHeight(height uint64) (*Result, error) {
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

func (api *GtasAPI) GetBlockByHash(hash string) (*Result, error) {
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

func (api *GtasAPI) GetBlocks(from uint64, to uint64) (*Result, error) {
	blocks := make([]*Block, 0)
	var preBH *types.BlockHeader
	for h := from; h <= to; h++ {
		b := core.BlockChainImpl.QueryBlockByHeight(h)
		if b != nil {
			block := convertBlockHeader(b)
			if preBH == nil {
				preBH = core.BlockChainImpl.QueryBlockHeaderByHash(b.Header.PreHash)
			}
			if preBH == nil {
				block.Qn = b.Header.TotalQN
			} else {
				block.Qn = b.Header.TotalQN - preBH.TotalQN
			}
			preBH = b.Header
			blocks = append(blocks, block)
		}
	}
	return successResult(blocks)
}

func (api *GtasAPI) GetTopBlock() (*Result, error) {
	bh := core.BlockChainImpl.QueryTopBlock()
	b := core.BlockChainImpl.QueryBlockByHash(bh.Hash)
	bh = b.Header

	blockDetail := make(map[string]interface{})
	blockDetail["hash"] = bh.Hash.Hex()
	blockDetail["height"] = bh.Height
	blockDetail["pre_hash"] = bh.PreHash.Hex()
	blockDetail["pre_time"] = bh.PreTime().Local().Format("2006-01-02 15:04:05")
	blockDetail["total_qn"] = bh.TotalQN
	blockDetail["cur_time"] = bh.CurTime.Local().Format("2006-01-02 15:04:05")
	blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupID)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	blockDetail["txs"] = len(b.Transactions)
	blockDetail["elapsed"] = bh.Elapsed
	blockDetail["tps"] = math.Round(float64(len(b.Transactions)) / float64(bh.Elapsed))

	blockDetail["tx_pool_count"] = len(core.BlockChainImpl.GetTransactionPool().GetReceived())
	blockDetail["tx_pool_total"] = core.BlockChainImpl.GetTransactionPool().TxNum()
	blockDetail["miner_id"] = mediator.Proc.GetPubkeyInfo().ID.ShortS()
	return successResult(blockDetail)
}

func (api *GtasAPI) WorkGroupNum(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroups(height)
	return successResult(groups)
}

func convertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.ID != nil && len(g.ID) != 0 {
		gmap["group_id"] = groupsig.DeserializeID(g.ID).GetHexString()
		gmap["g_hash"] = g.Header.Hash.Hex()
	}
	gmap["parent"] = groupsig.DeserializeID(g.Header.Parent).GetHexString()
	gmap["pre"] = groupsig.DeserializeID(g.Header.PreGroup).GetHexString()
	gmap["begin_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	gmap["create_height"] = g.Header.CreateHeight
	gmap["create_time"] = g.Header.BeginTime
	gmap["mem_size"] = len(g.Members)
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeID(mem).GetHexString()
		mems = append(mems, memberStr[0:6]+"-"+memberStr[len(memberStr)-6:])
	}
	gmap["members"] = mems
	gmap["extends"] = g.Header.Extends
	return gmap
}

func (api *GtasAPI) GetGroupsAfter(height uint64) (*Result, error) {
	groups := core.GroupChainImpl.GetGroupsAfterHeight(height, math.MaxInt64)

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) GetCurrentWorkGroup() (*Result, error) {
	height := core.BlockChainImpl.Height()
	return api.GetWorkGroup(height)
}

func (api *GtasAPI) GetWorkGroup(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroupsFromChain(height)
	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) MinerQuery(mtype int32) (*Result, error) {
	minerInfo := mediator.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	miner := core.MinerManagerImpl.GetMinerByID(address[:], byte(mtype), nil)
	js, err := json.Marshal(miner)
	if err != nil {
		return &Result{Message: err.Error(), Data: nil}, err
	}
	return &Result{Message: address.Hex(), Data: string(js)}, nil
}

// CastStat cast block statistics
func (api *GtasAPI) CastStat(begin uint64, end uint64) (*Result, error) {
	proposerStat := make(map[string]int32)
	groupStat := make(map[string]int32)

	chain := core.BlockChainImpl
	if end == 0 {
		end = chain.QueryTopBlock().Height
	}

	for h := begin; h < end; h++ {
		b := chain.QueryBlockByHeight(h)
		if b == nil {
			continue
		}
		bh := b.Header
		p := string(bh.Castor)
		if v, ok := proposerStat[p]; ok {
			proposerStat[p] = v + 1
		} else {
			proposerStat[p] = 1
		}
		g := string(bh.GroupID)
		if v, ok := groupStat[g]; ok {
			groupStat[g] = v + 1
		} else {
			groupStat[g] = 1
		}
	}
	pmap := make(map[string]int32)
	gmap := make(map[string]int32)

	for key, v := range proposerStat {
		id := groupsig.DeserializeID([]byte(key))
		pmap[id.GetHexString()] = v
	}
	for key, v := range groupStat {
		id := groupsig.DeserializeID([]byte(key))
		gmap[id.GetHexString()] = v
	}
	ret := make(map[string]map[string]int32)
	ret["proposer"] = pmap
	ret["group"] = gmap
	return successResult(ret)
}

func (api *GtasAPI) MinerInfo(addr string) (*Result, error) {
	morts := make([]MortGage, 0)
	id := common.HexToAddress(addr).Bytes()
	heavyInfo := core.MinerManagerImpl.GetMinerByID(id, types.MinerTypeHeavy, nil)
	if heavyInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(heavyInfo))
	}
	lightInfo := core.MinerManagerImpl.GetMinerByID(id, types.MinerTypeLight, nil)
	if lightInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(lightInfo))
	}
	return successResult(morts)
}

func (api *GtasAPI) NodeInfo() (*Result, error) {
	ni := &NodeInfo{}
	p := mediator.Proc
	ni.ID = p.GetMinerID().GetHexString()
	balance, err := walletManager.getBalance(p.GetMinerID().GetHexString())
	if err != nil {
		return failResult(err.Error())
	}
	ni.Balance = balance
	if !p.Ready() {
		ni.Status = "node not ready"
	} else {
		ni.Status = "running"
		morts := make([]MortGage, 0)
		t := "--"
		heavyInfo := core.MinerManagerImpl.GetMinerByID(p.GetMinerID().Serialize(), types.MinerTypeHeavy, nil)
		if heavyInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(heavyInfo))
			if heavyInfo.AbortHeight == 0 {
				t = "proposal role"
			}
		}
		lightInfo := core.MinerManagerImpl.GetMinerByID(p.GetMinerID().Serialize(), types.MinerTypeLight, nil)
		if lightInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(lightInfo))
			if lightInfo.AbortHeight == 0 {
				t += " verify role"
			}
		}
		ni.NType = t
		ni.MortGages = morts

		wg, ag := p.GetJoinedWorkGroupNums()
		ni.WGroupNum = wg
		ni.AGroupNum = ag

		ni.TxPoolNum = int(core.BlockChainImpl.GetTransactionPool().TxNum())

	}
	return successResult(ni)

}

func (api *GtasAPI) PageGetBlocks(page, limit int) (*Result, error) {
	chain := core.BlockChainImpl
	total := chain.Height() + 1
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}
	if page < 1 {
		page = 1
	}
	i := 0
	num := uint64((page - 1) * limit)
	if total < num {
		return successResult(pageObject)
	}
	b := int64(total - num)

	for i < limit && b >= 0 {
		bh := chain.QueryBlockByHeight(uint64(b))
		b--
		if bh == nil {
			continue
		}
		block := convertBlockHeader(bh)
		pageObject.Data = append(pageObject.Data, block)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) PageGetGroups(page, limit int) (*Result, error) {
	chain := core.GroupChainImpl
	total := chain.Height()
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}

	i := 0
	b := int64(0)
	if page < 1 {
		page = 1
	}
	num := uint64((page - 1) * limit)
	if total < num {
		return successResult(pageObject)
	}
	b = int64(total - num)

	for i < limit && b >= 0 {
		g := chain.GetGroupByHeight(uint64(b))
		b--
		if g == nil {
			continue
		}

		mems := make([]string, 0)
		for _, mem := range g.Members {
			mems = append(mems, groupsig.DeserializeID(mem).ShortS())
		}

		group := &Group{
			Height:        uint64(b + 1),
			ID:            groupsig.DeserializeID(g.ID),
			PreID:         groupsig.DeserializeID(g.Header.PreGroup),
			ParentID:      groupsig.DeserializeID(g.Header.Parent),
			BeginHeight:   g.Header.WorkHeight,
			DismissHeight: g.Header.DismissHeight,
			Members:       mems,
		}
		pageObject.Data = append(pageObject.Data, group)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) BlockDetail(h string) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return successResult(nil)
	}
	bh := b.Header
	block := convertBlockHeader(b)

	preBH := chain.QueryBlockHeaderByHash(bh.PreHash)
	block.Qn = bh.TotalQN - preBH.TotalQN

	castor := block.Castor.GetHexString()

	trans := make([]Transaction, 0)
	bonusTxs := make([]BonusTransaction, 0)
	minerBonus := make(map[string]*MinerBonusBalance)
	uniqueBonusBlockHash := make(map[common.Hash]byte)
	minerVerifyBlockHash := make(map[string][]common.Hash)
	blockVerifyBonus := make(map[common.Hash]uint64)

	minerBonus[castor] = genMinerBalance(block.Castor, bh)

	for _, tx := range b.Transactions {
		if tx.Type == types.TransactionTypeBonus {
			btx := *convertBonusTransaction(tx)
			if st, err := mediator.Proc.MainChain.GetTransactionPool().GetTransactionStatus(tx.Hash); err != nil {
				common.DefaultLogger.Errorf("getTransactions statue error, hash %v, err %v", tx.Hash.Hex(), err)
				btx.StatusReport = "get status error" + err.Error()
			} else {
				if st == types.ReceiptStatusSuccessful {
					btx.StatusReport = "success"
					btx.Success = true
				} else {
					btx.StatusReport = "fail"
				}
			}
			bonusTxs = append(bonusTxs, btx)
			blockVerifyBonus[btx.BlockHash] = btx.Value
			for _, tid := range btx.TargetIDs {
				if _, ok := minerBonus[tid.GetHexString()]; !ok {
					minerBonus[tid.GetHexString()] = genMinerBalance(tid, bh)
				}
				if !btx.Success {
					continue
				}
				if hs, ok := minerVerifyBlockHash[tid.GetHexString()]; ok {
					find := false
					for _, h := range hs {
						if h == btx.BlockHash {
							find = true
							break
						}
					}
					if !find {
						hs = append(hs, btx.BlockHash)
						minerVerifyBlockHash[tid.GetHexString()] = hs
					}
				} else {
					hs = make([]common.Hash, 0)
					hs = append(hs, btx.BlockHash)
					minerVerifyBlockHash[tid.GetHexString()] = hs
				}
			}
			if btx.Success {
				uniqueBonusBlockHash[btx.BlockHash] = 1
			}
		} else {
			trans = append(trans, *convertTransaction(tx))
		}
	}

	mbs := make([]*MinerBonusBalance, 0)
	for id, mb := range minerBonus {
		mb.Explain = ""
		increase := uint64(0)
		if id == castor {
			mb.Proposal = true
			mb.PackBonusTx = len(uniqueBonusBlockHash)
			increase += model.Param.ProposalBonus + uint64(mb.PackBonusTx)*model.Param.PackBonus
			mb.Explain = fmt.Sprintf("proposal, pack %v bouns-txs", mb.PackBonusTx)
		}
		if hs, ok := minerVerifyBlockHash[id]; ok {
			for _, h := range hs {
				increase += blockVerifyBonus[h]
			}
			mb.VerifyBlock = len(hs)
			mb.Explain = fmt.Sprintf("%v, verify %v blocks", mb.Explain, mb.VerifyBlock)
		}
		mb.ExpectBalance = new(big.Int).SetUint64(mb.PreBalance.Uint64() + increase)
		mbs = append(mbs, mb)
	}

	var genBonus *BonusTransaction
	if bonusTx := chain.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes()); bonusTx != nil {
		genBonus = convertBonusTransaction(bonusTx)
	}

	bd := &BlockDetail{
		Block:        *block,
		GenBonusTx:   genBonus,
		Trans:        trans,
		BodyBonusTxs: bonusTxs,
		MinerBonus:   mbs,
		PreTotalQN:   preBH.TotalQN,
	}
	return successResult(bd)
}

func (api *GtasAPI) BlockReceipts(h string) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return failResult("block not found")
	}

	evictedReceipts := make([]*types.Receipt, 0)
	receipts := make([]*types.Receipt, len(b.Transactions))
	for i, tx := range b.Transactions {
		wrapper := chain.GetTransactionPool().GetReceipt(tx.Hash)
		if wrapper != nil {
			receipts[i] = wrapper
		}
	}
	br := &BlockReceipt{EvictedReceipts: evictedReceipts, Receipts: receipts}
	return successResult(br)
}

func (api *GtasAPI) TransDetail(h string) (*Result, error) {
	tx := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(h))

	if tx != nil {
		trans := convertTransaction(tx)
		return successResult(trans)
	}
	return successResult(nil)
}

func (api *GtasAPI) Dashboard() (*Result, error) {
	blockHeight := core.BlockChainImpl.Height()
	groupHeight := core.GroupChainImpl.Height()
	workNum := len(mediator.Proc.GetCastQualifiedGroups(blockHeight))
	nodeResult, _ := api.NodeInfo()
	consResult, _ := api.ConnectedNodes()
	dash := &Dashboard{
		BlockHeight: blockHeight,
		GroupHeight: groupHeight,
		WorkGNum:    workNum,
		NodeInfo:    nodeResult.Data.(*NodeInfo),
		Conns:       consResult.Data.([]ConnInfo),
	}
	return successResult(dash)
}

func (api *GtasAPI) Nonce(addr string) (*Result, error) {
	address := common.HexToAddress(addr)
	// user will see the nonce as db nonce +1, so that user can use it directly when send a transaction
	nonce := core.BlockChainImpl.GetNonce(address) + 1
	return successResult(nonce)
}

func (api *GtasAPI) TxReceipt(h string) (*Result, error) {
	hash := common.HexToHash(h)
	rc := core.BlockChainImpl.GetTransactionPool().GetReceipt(hash)
	if rc != nil {
		tx := core.BlockChainImpl.GetTransactionByHash(false, true, hash)
		return successResult(&core.ExecutedTransaction{
			Receipt:     rc,
			Transaction: tx,
		})
	}
	return failResult("tx not exist")
}

func (api *GtasAPI) RPCSyncBlocks(height uint64, limit int, version int) (*Result, error) {
	chain := core.BlockChainImpl
	v := chain.Version()
	if version != v {
		return failResult(fmt.Sprintf("version not support, expect version %v", v))
	}
	blocks := chain.BatchGetBlocksAfterHeight(height, limit)
	return successResult(blocks)
}

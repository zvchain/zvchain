package common

import (
	"encoding/base64"
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
)

type Fetcher struct {
}
type MortGage struct {
	Stake                uint64             `json:"stake"`
	ApplyHeight          uint64             `json:"apply_height"`
	Type                 string             `json:"type"`
	Status               types.MinerStatus  `json:"miner_status"`
	StatusUpdateHeight   uint64             `json:"status_update_height"`
	Identity             types.NodeIdentity `json:"identity"`
	IdentityUpdateHeight uint64             `json:"identity_update_height"`
}
type Group struct {
	Seed          common.Hash `json:"id"`
	BeginHeight   uint64      `json:"begin_height"`
	DismissHeight uint64      `json:"dismiss_height"`
	Threshold     int32       `json:"threshold"`
	Members       []string    `json:"members"`
	MemSize       int         `json:"mem_size"`
	GroupHeight   uint64      `json:"group_height"`
}

func (api *Fetcher) ExplorerBlockDetail(height uint64) (*models.BlockDetail, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return nil, fmt.Errorf("queryBlock error")
	}
	block := convertBlockHeader(b)

	trans := make([]*models.TempTransaction, 0)

	for _, tx := range b.Transactions {
		trans = append(trans, convertTransaction(types.NewTransaction(tx, tx.GenHash())))
	}

	evictedReceipts := make([]*models.Receipt, 0)

	receipts := make([]*models.Receipt, len(b.Transactions))
	for i, tx := range trans {
		wrapper := chain.GetTransactionPool().GetReceipt(tx.Hash)
		if wrapper != nil {
			modelreceipt := convertReceipt(wrapper)
			receipts[i] = modelreceipt
			tx.Status = modelreceipt.Status
		}
	}

	bd := &models.BlockDetail{
		Block:           *block,
		Trans:           trans,
		EvictedReceipts: evictedReceipts,
		Receipts:        receipts,
	}
	return bd, nil
}

func convertReceipt(receipt *types.Receipt) *models.Receipt {
	modelreceipt := &models.Receipt{
		Status:            uint(receipt.Status),
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Logs:              nil,
		TxHash:            receipt.TxHash.Hex(),
		ContractAddress:   receipt.ContractAddress.AddrPrefixString(),
	}
	return modelreceipt

}

func ConvertGroup(g types.GroupI) *models.Group {

	mems := make([]string, 0)
	for _, mem := range g.Members() {
		memberStr := groupsig.DeserializeID(mem.ID()).GetAddrString()
		mems = append(mems, memberStr)
	}
	gh := g.Header()

	data := &Group{
		Seed:          gh.Seed(),
		BeginHeight:   gh.WorkHeight(),
		DismissHeight: gh.DismissHeight(),
		Threshold:     int32(gh.Threshold()),
		Members:       mems,
		MemSize:       len(mems),
		GroupHeight:   gh.GroupHeight(),
	}
	return dataToGroup(data)

}

func dataToGroup(data *Group) *models.Group {

	group := &models.Group{}
	group.Id = data.Seed.Hex()
	group.WorkHeight = data.BeginHeight
	group.DismissHeight = data.DismissHeight
	group.Threshold = uint64(data.Threshold)
	group.Height = data.GroupHeight

	members := data.Members
	group.Members = make([]string, 0)
	group.MemberCount = uint64(len(members))
	for _, midStr := range members {
		if len(midStr) > 0 {
			group.MembersStr = group.MembersStr + midStr + "\r\n"
		}
	}
	return group
}

func convertBlockHeader(b *types.Block) *models.Block {
	bh := b.Header
	block := &models.Block{
		Height:     bh.Height,
		Hash:       bh.Hash.Hex(),
		PreHash:    bh.PreHash.Hex(),
		CurTime:    bh.CurTime.Local(),
		PreTime:    bh.PreTime().Local(),
		Castor:     groupsig.DeserializeID(bh.Castor).GetAddrString(),
		GroupID:    bh.Group.Hex(),
		TotalQN:    bh.TotalQN,
		TransCount: uint64(len(b.Transactions)),
		//Qn: mediator.Proc.CalcBlockHeaderQN(bh),

	}
	return block
}
func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "proposal node"
	if miner.IsVerifyRole() {
		t = "verify node"
	}
	status := types.MinerStatusPrepare
	if miner.IsActive() {
		status = types.MinerStatusActive
	} else if miner.IsFrozen() {
		status = types.MinerStatusFrozen
	}

	i := types.MinerNormal
	if miner.IsMinerPool() {
		i = types.MinerPool
	} else if miner.IsInvalidMinerPool() {
		i = types.InValidMinerPool
	} else if miner.IsGuard() {
		i = types.MinerGuard
	}
	mg := &MortGage{
		Stake:                uint64(common.RA2TAS(miner.Stake)),
		ApplyHeight:          miner.ApplyHeight,
		Type:                 t,
		Status:               status,
		StatusUpdateHeight:   miner.StatusUpdateHeight,
		Identity:             i,
		IdentityUpdateHeight: miner.IdentityUpdateHeight,
	}
	return mg
}
func (api *Fetcher) ConvertTempTransactionToTransaction(temp *models.TempTransaction) *models.Transaction {

	tran := &models.Transaction{
		Value:     temp.Value,
		Nonce:     temp.Nonce,
		Type:      int32(temp.Type),
		GasLimit:  temp.GasLimit,
		GasPrice:  temp.GasPrice,
		Hash:      temp.Hash.Hex(),
		ExtraData: temp.ExtraData,
		Status:    temp.Status,
	}
	tran.Data = util.ObjectTojson(temp.Data)
	if temp.Source != nil {
		tran.Source = temp.Source.AddrPrefixString()
	}
	if temp.Target != nil {
		tran.Target = temp.Target.AddrPrefixString()
	}
	return tran

}

func convertTransaction(tx *types.Transaction) *models.TempTransaction {
	var (
		gasLimit = uint64(0)
		gasPrice = uint64(0)
		value    = uint64(0)
	)
	if tx.GasLimit != nil {
		gasLimit = tx.GasLimit.Uint64()
	}
	if tx.GasPrice != nil {
		gasPrice = tx.GasPrice.Uint64()
	}
	if tx.Value != nil {
		value = tx.Value.Uint64()
	}
	trans := &models.TempTransaction{
		Hash:      tx.Hash,
		Source:    tx.Source,
		Target:    tx.Target,
		Type:      tx.Type,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		Data:      tx.Data,
		ExtraData: base64.StdEncoding.EncodeToString(tx.ExtraData),
		Nonce:     tx.Nonce,
		Value:     common.RA2TAS(value),
	}
	return trans
}

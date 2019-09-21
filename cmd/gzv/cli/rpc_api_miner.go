package cli

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/core"
	"time"
)

// RpcGzvImpl provides rpc service for users to interact with remote nodes
type RpcMinerImpl struct {
	*rpcBaseImpl
}

func (api *RpcMinerImpl) Namespace() string {
	return "Miner"
}

func (api *RpcMinerImpl) Version() string {
	return "1"
}

type RespMsg struct {
	Code  int8        `json:"code"`
	Count uint64      `json:"count"`
	Data  []BasicInfo `json:"data"`
}

type RespMortMsg struct {
	Code  int8       `json:"code"`
	Count uint64     `json:"count"`
	Data  []MortGage `json:"data"`
}

type BasicInfo struct {
	CurTime       string         `json:"cur_time"`
	ClientVersion string         `json:"client_version"`
	ChainId       uint16         `json:"chain_id"`
	BlockHeight   uint64         `json:"block_height"`
	GroupHeight   uint64         `json:"group_height"`
	Addr          common.Address `json:"addr"`
}

func (rm *RpcMinerImpl) BasicData() (*BasicInfo, error) {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	version := common.GzvVersion
	chainId := globalGzv.config.chainID
	addr := common.BytesToAddress(mediator.Proc.GetMinerID().Serialize())
	blockHeight := core.BlockChainImpl.Height()
	groupHeight := core.GroupManagerImpl.Height()

	basicInfo := &BasicInfo{
		CurTime:       currentTime,
		ClientVersion: version,
		ChainId:       chainId,
		Addr:          addr,
		BlockHeight:   blockHeight,
		GroupHeight:   groupHeight,
	}
	return basicInfo, nil
}

func (rm *RpcMinerImpl) MortData() ([]MortGage, error) {
	_, morts := getMorts(mediator.Proc)
	return morts, nil
}

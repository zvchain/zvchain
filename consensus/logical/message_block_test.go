package logical

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
	"sync/atomic"
	"testing"
	time2 "time"
)

func GenTestBH(param string) types.BlockHeader {
	bh := types.BlockHeader{}
	switch param {
	case "Hash":
		bh.Hash = common.HexToHash("0x01")
	case "Height":
		bh.Height = 10
	case "PreHash":
		bh.PreHash = common.HexToHash("0x02")
	case "Elapsed":
		bh.Elapsed = 100
	case "ProveValue":
		bh.ProveValue = []byte{0, 1, 2}
	case "TotalQN":
		bh.TotalQN = 10
	case "CurTime":
		bh.CurTime = time.TimeToTimeStamp(time2.Now())
	case "Castor":
		bh.Castor = []byte{0, 1}
	case "Group":
		bh.Group = common.HexToHash("0x03")
	case "Signature":
		bh.Signature = []byte{23, 22}
	case "Nonce":
		bh.Nonce = 12
	case "TxTree":
		bh.TxTree = common.HexToHash("0x04")
	case "ReceiptTree":
		bh.ReceiptTree = common.HexToHash("0x05")
	case "StateTree":
		bh.StateTree = common.HexToHash("0x06")
	case "ExtraData":
		bh.ExtraData = []byte{4, 22}
	case "Random":
		bh.Random = []byte{4, 22, 145}
	case "GasFee":
		bh.GasFee = 123
	}
	return bh
}

func GenTestBHHash(param string) common.Hash {
	bh := GenTestBH(param)
	return bh.GenHash()
}

func EmptyBHHash() common.Hash {
	bh := types.BlockHeader{}
	return bh.GenHash()
}

var emptyBHHash = EmptyBHHash()

func NewProcess2() *Processor {
	common.InitConf("./tas_config_test.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	err := middleware.InitMiddleware()
	if err != nil {
		panic(err)
	}
	err = core.InitCore(NewConsensusHelper4Test(groupsig.ID{}), nil)
	core.GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})

	process := &Processor{}
	sk := common.HexToSecKey(getAccount().Sk)
	minerInfo, _ := model.NewSelfMinerDO(sk)
	InitConsensus()
	process.Init(minerInfo, common.GlobalConf)
	//hijack some external interface to avoid error
	process.MainChain = &chain4Test{core.BlockChainImpl}
	process.NetServer = &networkServer4Test{process.NetServer}
	return process
}

func TestProcessor_OnMessageCast(t *testing.T) {
	_ = initContext4Test()
	defer clear()

	pt := NewProcessorTest()

	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	//txs := make([]*types.Transaction, 0)
	type args struct {
		msg *model.ConsensusCastMessage
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "Height Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Height"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Height"), emptyBHHash),
		},
		{
			name: "PreHash Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("PreHash"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("PreHash"), emptyBHHash),
		},
		{
			name: "Elapsed Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Elapsed"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Elapsed"), emptyBHHash),
		},
		{
			name: "ProveValue Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("ProveValue"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("ProveValue"), emptyBHHash),
		},
		{
			name: "TotalQN Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("TotalQN"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("TotalQN"), emptyBHHash),
		},
		{
			name: "CurTime Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("CurTime"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("CurTime"), emptyBHHash),
		},
		{
			name: "Castor Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Castor"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Castor"), emptyBHHash),
		},
		{
			name: "Group Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Group"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Group"), emptyBHHash),
		},
		{
			name: "Signature Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Signature"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Signature"), emptyBHHash),
		},
		{
			name: "Nonce Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Nonce"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Nonce"), emptyBHHash),
		},
		{
			name: "TxTree Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("TxTree"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("TxTree"), emptyBHHash),
		},
		{
			name: "ReceiptTree Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("ReceiptTree"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("ReceiptTree"), emptyBHHash),
		},
		{
			name: "StateTree Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("StateTree"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("StateTree"), emptyBHHash),
		},
		{
			name: "ExtraData Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("ExtraData"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("ExtraData"), emptyBHHash),
		},
		{
			name: "Random Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Random"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("Random"), emptyBHHash),
		},
		{
			name: "GasFee Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("GasFee"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v", GenTestBHHash("GasFee"), emptyBHHash),
		},
	}
	p := processorTest
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := p.OnMessageCast(tt.args.msg)
			if msg != nil && msg.Error() != tt.expected {
				t.Errorf("wanted {%s}; got {%s}", tt.expected, msg)
			}
		})
	}
}

func TestProcessor_OnMessageCastRewardSign(t *testing.T) {
	type fields struct {
		mi               *model.SelfMinerDO
		genesisMember    bool
		minerReader      *MinerPoolReader
		blockContexts    *castBlockContexts
		futureVerifyMsgs *FutureMessageHolder
		proveChecker     *proveChecker
		Ticker           *ticker.GlobalTicker
		ready            bool
		MainChain        types.BlockChain
		vrf              atomic.Value
		NetServer        net.NetworkServer
		conf             common.ConfManager
		isCasting        int32
		castVerifyCh     chan *types.BlockHeader
		blockAddCh       chan *types.BlockHeader
		groupReader      *groupReader
		ts               time.TimeService
		rewardHandler    *RewardHandler
	}
	type args struct {
		msg *model.CastRewardTransSignMessage
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Processor{
				mi:               tt.fields.mi,
				genesisMember:    tt.fields.genesisMember,
				minerReader:      tt.fields.minerReader,
				blockContexts:    tt.fields.blockContexts,
				futureVerifyMsgs: tt.fields.futureVerifyMsgs,
				proveChecker:     tt.fields.proveChecker,
				Ticker:           tt.fields.Ticker,
				ready:            tt.fields.ready,
				MainChain:        tt.fields.MainChain,
				vrf:              tt.fields.vrf,
				NetServer:        tt.fields.NetServer,
				conf:             tt.fields.conf,
				isCasting:        tt.fields.isCasting,
				castVerifyCh:     tt.fields.castVerifyCh,
				blockAddCh:       tt.fields.blockAddCh,
				groupReader:      tt.fields.groupReader,
				ts:               tt.fields.ts,
				rewardHandler:    tt.fields.rewardHandler,
			}
			p.OnMessageCastRewardSign(tt.args.msg)
		})
	}
}

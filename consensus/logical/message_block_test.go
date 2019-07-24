package logical

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
	"gopkg.in/fatih/set.v0"
	"strings"
	"testing"
	time2 "time"
)

func GenTestBH(param string, value ...interface{}) types.BlockHeader {
	bh := types.BlockHeader{}
	bh.Elapsed = 1
	switch param {
	case "Hash":
		bh.Hash = common.HexToHash("0x01")
	case "Height":
		bh.Height = 10
		bh.Hash = bh.GenHash()
	case "PreHash":
		bh.PreHash = common.HexToHash("0x02")
		bh.Hash = bh.GenHash()
	case "Elapsed":
		bh.Elapsed = 100
		bh.Hash = bh.GenHash()
	case "ProveValue":
		bh.ProveValue = []byte{0, 1, 2}
		bh.Hash = bh.GenHash()
	case "TotalQN":
		bh.TotalQN = 10
		bh.Hash = bh.GenHash()
	case "CurTime":
		bh.CurTime = time.TimeToTimeStamp(time2.Now())
		bh.Hash = bh.GenHash()
	case "Castor":
		bh.Castor = []byte{0, 1}
		bh.Hash = bh.GenHash()
	case "Group":
		bh.Group = common.HexToHash("0x03")
		bh.Hash = bh.GenHash()
	case "Signature":
		bh.Signature = []byte{23, 22}
		bh.Hash = bh.GenHash()
	case "Nonce":
		bh.Nonce = 12
		bh.Hash = bh.GenHash()
	case "TxTree":
		bh.TxTree = common.HexToHash("0x04")
		bh.Hash = bh.GenHash()
	case "ReceiptTree":
		bh.ReceiptTree = common.HexToHash("0x05")
		bh.Hash = bh.GenHash()
	case "StateTree":
		bh.StateTree = common.HexToHash("0x06")
		bh.Hash = bh.GenHash()
	case "ExtraData":
		bh.ExtraData = []byte{4, 22}
		bh.Hash = bh.GenHash()
	case "Random":
		bh.Random = []byte{4, 22, 145}
		bh.Hash = bh.GenHash()
	case "GasFee":
		bh.GasFee = 123
		bh.Hash = bh.GenHash()
	case "Castor=getMinerId":
		bh.Castor = common.FromHex("0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676")
		bh.Hash = bh.GenHash()
	case "bh.Elapsed<=0":
		bh.Elapsed = -1
		bh.Hash = bh.GenHash()
	case "p.ts.Since(bh.CurTime)<-1":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) + 2
		bh.Hash = bh.GenHash()
	case "block-exists":
		bh.GasFee = 10
		bh.Hash = bh.GenHash()
	case "pre-block-not-exists":
		bh.PreHash = common.HexToHash("0x01")
		bh.Hash = bh.GenHash()
	case "already-cast":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 1
		bh.PreHash = common.HexToHash("0x1234")
		bh.Height = 1
		bh.Hash = bh.GenHash()
	case "already-sign":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 3
		bh.PreHash = common.HexToHash("0x02")
		bh.Height = 2
		bh.Hash = bh.GenHash()
	case "cast-illegal":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 3
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 3
		bh.Castor = common.Hex2Bytes("0x0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	}
	return bh
}

func GenTestBHHash(param string) common.Hash {
	bh := GenTestBH(param)
	return bh.Hash
}

func EmptyBHHash() common.Hash {
	bh := types.BlockHeader{}
	bh.Elapsed = 1
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Height"), emptyBHHash, GenTestBHHash("Height")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("PreHash"), emptyBHHash, GenTestBHHash("PreHash")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Elapsed"), emptyBHHash, GenTestBHHash("Elapsed")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("ProveValue"), emptyBHHash, GenTestBHHash("ProveValue")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("TotalQN"), emptyBHHash, GenTestBHHash("TotalQN")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("CurTime"), emptyBHHash, GenTestBHHash("CurTime")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Castor"), emptyBHHash, GenTestBHHash("Castor")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Group"), emptyBHHash, GenTestBHHash("Group")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Signature"), emptyBHHash, GenTestBHHash("Signature")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Nonce"), emptyBHHash, GenTestBHHash("Nonce")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("TxTree"), emptyBHHash, GenTestBHHash("TxTree")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("ReceiptTree"), emptyBHHash, GenTestBHHash("ReceiptTree")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("StateTree"), emptyBHHash, GenTestBHHash("StateTree")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("ExtraData"), emptyBHHash, GenTestBHHash("ExtraData")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Random"), emptyBHHash, GenTestBHHash("Random")),
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
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("GasFee"), emptyBHHash, GenTestBHHash("GasFee")),
		},
		{
			name: "Castor=getMinerId",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Castor=getMinerId"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("Castor=getMinerId"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "ignore self message",
		},
		{
			name: "bh.Elapsed<=0",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("bh.Elapsed<=0"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("bh.Elapsed<=0"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("elapsed error %v", -1),
		},
		{
			name: "p.ts.Since(bh.CurTime)<-1",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("p.ts.Since(bh.CurTime)<-1"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("p.ts.Since(bh.CurTime)<-1"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block too early",
		},
		{
			name: "block-exists",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("block-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("block-exists"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block onchain already",
		},
		{
			name: "pre-block-not-exists",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("pre-block-not-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("pre-block-not-exists"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "parent block did not received",
		},
		{
			name: "already-cast",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("already-cast"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("already-cast"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "the block of this height has been cast",
		},
		{
			name: "already-sign",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("already-sign"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("already-sign"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block signed",
		},
		{
			name: "cast-illegal",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("cast-illegal"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("cast-illegal"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block onchain already",
		},
	}
	p := processorTest
	p.groupReader.cache.Add(common.HexToHash("0x00"), &verifyGroup{memIndex: map[string]int{
		"0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676": 0,
	}, members: []*member{&member{}}})
	// for already-cast
	p.blockContexts.addCastedHeight(1, common.HexToHash("0x1234"))
	// for already-sign
	vcx := &VerifyContext{}
	vcx.castHeight = 2
	vcx.signedBlockHashs = set.New(set.ThreadSafe)
	vcx.signedBlockHashs.Add(GenTestBHHash("already-sign"))
	p.blockContexts.addVctx(vcx)
	//
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := p.OnMessageCast(tt.args.msg)
			if msg != nil && !strings.Contains(msg.Error(), tt.expected) {
				t.Errorf("wanted {%s}; got {%s}", tt.expected, msg)
			}
		})
	}
}

/*
miner addr:  0x00000000
miner addr:  0x00000001
miner addr:  0x00000002
miner addr:  0x00000003
miner addr:  0x00000004*/

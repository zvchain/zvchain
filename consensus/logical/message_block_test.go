package logical

import (
	"fmt"
	"strings"
	"testing"
	time2 "time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"gopkg.in/fatih/set.v0"
)

const goodCastor = "0000000100000000000000000000000000000000000000000000000000000000"
const inActiveCastor = "0000000200000000000000000000000000000000000000000000000000000000"
const goodPreHash = "0x73462515b5be97b6a4c66dc6acd46a64b442655bd61560525695cadb0c71572d"
const otherGroup = "0x01"

var now = time2.Now()

func GenTestBH(param string, value ...interface{}) types.BlockHeader {

	bh := types.BlockHeader{}
	bh.Elapsed = 1
	bh.CurTime = time.TimeToTimeStamp(now) - 3
	//bh.PreHash = common.HexToHash("0x03")
	bh.Height = 3
	proveString := "03db08597ecb8270a371018a1e4a4cd811938a33e2ca0f89e1d5dff038b7d9f99fd8891b000e06ac3abdf22ac962a5628c07d5bb38451dcdcb2ab07ce0fd7e6c77684b97e8adac2c1f7d5986bba22de4bd"
	bh.Castor = common.Hex2Bytes(goodCastor)
	bh.ProveValue = common.FromHex(proveString)
	bh.Random = common.Hex2Bytes("0320325")
	bh.PreHash = common.HexToHash(goodPreHash)
	bh.TotalQN = 5

	switch param {
	case "ok":
		bh.CurTime = time.TimeToTimeStamp(now) - 40
		bh.Hash = bh.GenHash()
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
		bh.CurTime = time.TimeToTimeStamp(now)
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
		bh.Elapsed = -2
		bh.Height = 10
		bh.Hash = bh.GenHash()
	case "p.ts.Since(bh.CurTime)<-1":
		bh.CurTime = time.TimeToTimeStamp(now) + 3
		bh.Hash = bh.GenHash()
	case "block-exists":
		bh = types.BlockHeader{}
		bh.Elapsed = 1
		bh.GasFee = 10
		bh.Hash = bh.GenHash()
		//bh.Hash = common.HexToHash(existBlockHash)
	case "pre-block-not-exists":
		bh.PreHash = common.HexToHash("0x01")
		bh.Hash = bh.GenHash()
	case "already-cast":
		bh.CurTime = time.TimeToTimeStamp(now) - 1
		bh.PreHash = common.HexToHash("0x1234")
		bh.Height = 1
		bh.Hash = bh.GenHash()
	case "already-sign":
		bh.CurTime = time.TimeToTimeStamp(now) - 3
		bh.PreHash = common.HexToHash("0x02")
		bh.Height = 2
		bh.Hash = bh.GenHash()
	case "cast-illegal":
		bh.Castor = common.Hex2Bytes(inActiveCastor)
		bh.Hash = bh.GenHash()
	case "slot-is-nil":
		bh.CurTime = time.TimeToTimeStamp(now) - 3
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 3
		bh.Hash = bh.GenHash()
	case "not-in-verify-group":
		bh.CurTime = time.TimeToTimeStamp(now)
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 3
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "sender-not-in-verify-group":
		bh.CurTime = time.TimeToTimeStamp(now)
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 4
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "receive-before-proposal":
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 4
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "already-sign-bigger-weight":
		bh.PreHash = common.HexToHash("0x02")
		bh.Height = 6
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "height-casted":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 7
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "has-signed":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 8
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "to51":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 9
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "hash-error":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 10
		bh.Castor = common.Hex2Bytes(goodCastor)
		bh.Hash = bh.GenHash()
	case "group-wrong":
		bh.CurTime = time.TimeToTimeStamp(now) - 40
		bh.Group = common.HexToHash(otherGroup)
		bh.Hash = bh.GenHash()
	case "qn-error":
		bh.TotalQN = 1
		bh.CurTime = time.TimeToTimeStamp(now) - 40
		bh.Hash = bh.GenHash()
	case "prove_wrong":
		bh.CurTime = time.TimeToTimeStamp(now) - 41
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

func TestProcessor_OnMessageCast(t *testing.T) {
	_ = initContext4Test()
	defer clear()

	pt := NewProcessorTest()
	pt.verifyContext.slots[pt.blockHeader.Hash] = &SlotContext{BH: pt.blockHeader, gSignGenerator: model.NewGroupSignGenerator(2)}
	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	//txs := make([]*types.Transaction, 0)
	type args struct {
		msg *model.ConsensusCastMessage
	}
	tests := []struct {
		name     string
		args     args
		expected string
		prepare  func()
		clean    func()
	}{
		{
			name: "ok",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH:        GenTestBH("ok"),
					ProveHash: common.HexToHash(goodPreHash),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("ok"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "success",
			clean: func() {
				bl := GenTestBH("ok")
				if processorTest.blockContexts.getVctxByHeight(bl.Height) != nil {
					processorTest.blockContexts.getVctxByHeight(bl.Height).signedBlockHashs.Remove(bl.Hash)
				}
			},
		},
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
		//{
		//	name: "Castor=getMinerId",
		//	args: args{
		//		msg: &model.ConsensusCastMessage{
		//			BH: GenTestBH("Castor=getMinerId"),
		//			ProveHash: common.HexToHash(goodPreHash),
		//			BaseSignedMessage: model.BaseSignedMessage{
		//				SI: model.GenSignData(GenTestBHHash("Castor=getMinerId"), pt.ids[1], pt.msk[1]),
		//			},
		//		},
		//	},
		//	expected: "ignore self message",
		//},
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
			expected: fmt.Sprintf("elapsed error %v", -2),
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
			expected: "miner can't cast at height",
		},
		{
			name: "group-wrong",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("group-wrong"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("group-wrong"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "calc verify group not equal",
		},

		{
			name: "slot-max-less",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH:        GenTestBH("ok"),
					ProveHash: common.HexToHash(goodPreHash),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("ok"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "comming block weight less than min block weight",
			prepare: func() {
				bh1 := GenTestBH("Hash")
				bh2 := GenTestBH("Height")
				bh3 := GenTestBH("PreHash")
				bh4 := GenTestBH("Elapsed")
				bh5 := GenTestBH("ProveValue")

				slots := make(map[common.Hash]*SlotContext)
				slots[bh1.Hash] = &SlotContext{BH: &bh1, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh2.Hash] = &SlotContext{BH: &bh2, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh3.Hash] = &SlotContext{BH: &bh3, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh4.Hash] = &SlotContext{BH: &bh4, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh5.Hash] = &SlotContext{BH: &bh5, gSignGenerator: model.NewGroupSignGenerator(2)}
				verifyContext := &VerifyContext{
					prevBH:           &types.BlockHeader{},
					castHeight:       0,
					group:            pt.verifyGroup,
					expireTime:       0,
					consensusStatus:  svWorking,
					slots:            slots,
					proposers:        make(map[string]common.Hash),
					signedBlockHashs: set.New(set.ThreadSafe),
				}
				bl := GenTestBH("ok")
				processorTest.blockContexts.heightVctxs.Add(bl.Height, verifyContext)
			},
		},
		{
			name: "slot-max-more",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH:        GenTestBH("ok"),
					ProveHash: common.HexToHash(goodPreHash),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("ok"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "success",
			prepare: func() {
				bh1 := GenTestBH("Hash")
				bh2 := GenTestBH("Height")
				bh3 := GenTestBH("PreHash")
				bh4 := GenTestBH("Elapsed")
				bh5 := GenTestBH("ProveValue")
				bh1.TotalQN = 4

				slots := make(map[common.Hash]*SlotContext)
				slots[bh1.Hash] = &SlotContext{BH: &bh1, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh2.Hash] = &SlotContext{BH: &bh2, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh3.Hash] = &SlotContext{BH: &bh3, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh4.Hash] = &SlotContext{BH: &bh4, gSignGenerator: model.NewGroupSignGenerator(2)}
				slots[bh5.Hash] = &SlotContext{BH: &bh5, gSignGenerator: model.NewGroupSignGenerator(2)}
				verifyContext := &VerifyContext{
					prevBH:           &types.BlockHeader{},
					castHeight:       0,
					group:            pt.verifyGroup,
					expireTime:       0,
					consensusStatus:  svWorking,
					slots:            slots,
					proposers:        make(map[string]common.Hash),
					signedBlockHashs: set.New(set.ThreadSafe),
				}
				bl := GenTestBH("ok")
				processorTest.blockContexts.heightVctxs.Add(bl.Height, verifyContext)
			},
			clean: func() {
				bl := GenTestBH("ok")
				processorTest.blockContexts.getVctxByHeight(bl.Height).signedBlockHashs.Remove(bl.Hash)
			},
		},
		{
			name: "qn-error",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("qn-error"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("qn-error"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "vrf verify block fail, err=qn error",
		},
		{
			name: "prove_wrong",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("prove_wrong"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("prove_wrong"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "check prove hash fail",
		},
	}
	p := processorTest
	// set up group info
	p.groupReader.cache.Add(common.HexToHash("0x00"), &verifyGroup{
		header: &GroupHanderTest{},
		memIndex: map[string]int{
			"zv7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676": 0,
		}, members: []*member{&member{}}})
	p.groupReader.cache.Add(common.HexToHash(otherGroup), &verifyGroup{
		header: &GroupHanderTest{},
		memIndex: map[string]int{
			"zv7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676": 0,
		}, members: []*member{&member{}}})

	p.groupReader.skStore.StoreGroupSignatureSeckey(common.HexToHash("0x00"), pt.sk[0], common.MaxUint64)
	p.groupReader.skStore.StoreGroupSignatureSeckey(common.HexToHash(otherGroup), pt.sk[1], common.MaxUint64)

	// for already-cast
	p.blockContexts.addCastedHeight(1, common.HexToHash("0x1234"))
	// for already-sign
	vcx := &VerifyContext{}
	vcx.castHeight = 2
	vcx.signedBlockHashs = set.New(set.ThreadSafe)
	vcx.signedBlockHashs.Add(GenTestBHHash("already-sign"))
	p.blockContexts.addVctx(vcx)
	// for cast-illegal
	p.minerReader = newMinerPoolReader(p, NewMinerPoolTest(pt.mpk, pt.ids, pt.verifyGroup))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			msg := p.OnMessageCast(tt.args.msg)
			if msg != nil && !strings.Contains(msg.Error(), tt.expected) {
				t.Errorf("wanted {%s}; got {%s}", tt.expected, msg)
			}

			if msg == nil && tt.expected != "success" {
				t.Errorf("wanted {%s}; got success", tt.expected)
			}

			if tt.clean != nil {
				tt.clean()
			}

		})
	}
}

func TestProcessor_OnMessageVerify(t *testing.T) {
	_ = initContext4Test()
	defer clear()

	pt := NewProcessorTest()
	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	//txs := make([]*types.Transaction, 0)
	type args struct {
		msg *model.ConsensusVerifyMessage
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "block-exists",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("block-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block already on chain",
		},
		{
			name: "slot-is-nil",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("slot-is-nil"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "slot is nil",
		},
		{
			name: "Castor=getMinerId",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("Castor=getMinerId"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "ignore self message",
		},
		{
			name: "not-in-verify-group",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("not-in-verify-group"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "don't belong to verifyGroup",
		},
		{
			name: "sender-not-in-verify-group",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("sender-not-in-verify-group"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[8], pt.msk[1]),
					},
				},
			},
			expected: "sender doesn't belong the verifyGroup",
		},
		{
			name: "bh.Elapsed<=0",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("bh.Elapsed<=0"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "elapsed error",
		},
		{
			name: "p.ts.Since(bh.CurTime)<-1",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("p.ts.Since(bh.CurTime)<-1"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block too early",
		},
		{
			name: "receive-before-proposal",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("receive-before-proposal"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "verify context is nil, cache msg",
		},
		{
			name: "receive-before-proposal",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("receive-before-proposal"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "verify context is nil, cache msg",
		},
		{
			name: "already-sign-bigger-weight",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("already-sign-bigger-weight"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "have signed a higher qn block",
		},
		{
			name: "pre-block-not-exists",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("pre-block-not-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "pre not on chain",
		},
		{
			name: "height-casted",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("height-casted"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("height-casted"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "the block of this height has been cast",
		},
		{
			name: "has-signed",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("has-signed"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("has-signed"), pt.ids[1], pt.msk[1]),
					},
					RandomSign: groupsig.Sign(pt.msk[1], []byte{1}),
				},
			},
			expected: "duplicate message",
		},
		{
			name: "to51",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("to51"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("to51"), pt.ids[1], pt.msk[1]),
					},
					RandomSign: groupsig.Sign(pt.msk[1], []byte{1}),
				},
			},
			expected: "",
		},
		{
			name: "hash-error",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: emptyBHHash,
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("hash-error"), pt.ids[1], pt.msk[1]),
					},
					RandomSign: groupsig.Sign(pt.msk[1], []byte{1}),
				},
			},
			expected: "msg genHash",
		},
	}
	p := processorTest
	p.groupReader.cache.Add(common.HexToHash("0x00"), &verifyGroup{memIndex: map[string]int{
		"zv7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676": 1,
	}, members: []*member{&member{}}})
	// for block-exists
	testBH1 := GenTestBH("block-exists")
	p.blockContexts.attachVctx(&testBH1, &VerifyContext{})
	testBH2 := GenTestBH("slot-is-nil")
	p.blockContexts.attachVctx(&testBH2, &VerifyContext{})
	testBH3 := GenTestBH("Castor=getMinerId")
	p.blockContexts.attachVctx(&testBH3, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH3.Hash: {castor: groupsig.DeserializeID(testBH3.Castor)}},
	})
	testBH4 := GenTestBH("not-in-verify-group")
	p.blockContexts.attachVctx(&testBH4, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH4.Hash: {BH: &testBH4, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header: GroupHeaderTest{},
		},
		ts: p.ts,
	})
	testBH5 := GenTestBH("sender-not-in-verify-group")
	p.blockContexts.attachVctx(&testBH5, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH5.Hash: {BH: &testBH5, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 1},
		},
		ts: p.ts,
	})
	testBH6 := GenTestBH("bh.Elapsed<=0")
	p.blockContexts.attachVctx(&testBH6, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH6.Hash: {BH: &testBH6, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 1},
		},
		ts: p.ts,
	})

	testBH7 := GenTestBH("p.ts.Since(bh.CurTime)<-1")
	p.blockContexts.attachVctx(&testBH7, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH7.Hash: {BH: &testBH7, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 1},
		},
		ts: p.ts,
	})

	testBH8 := GenTestBH("already-sign-bigger-weight")
	vctx := &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH8.Hash: {BH: &testBH8, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 1},
		},
		ts:               p.ts,
		signedBlockHashs: set.New(set.ThreadSafe),
		castHeight:       testBH8.Height,
	}
	p.blockContexts.attachVctx(&testBH8, vctx)
	copyTestBH8 := testBH8
	copyTestBH8.TotalQN = 10000
	vctx.markSignedBlock(&copyTestBH8)
	testBH9 := GenTestBH("pre-block-not-exists")
	p.blockContexts.attachVctx(&testBH9, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH9.Hash: {BH: &testBH9, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 1, pt.ids[1].GetAddrString(): 1},
		},
		ts:     p.ts,
		prevBH: genBlockHeader(),
	})
	testBH10 := GenTestBH("height-casted")
	p.blockContexts.attachVctx(&testBH10, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH10.Hash: {BH: &testBH10, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 1, pt.ids[1].GetAddrString(): 1},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")},
	})
	p.blockContexts.recentCasted.Add(testBH10.Height, &castedBlock{height: testBH10.Height, preHash: testBH10.PreHash})
	testBH11 := GenTestBH("has-signed")
	gsg := model.NewGroupSignGenerator(2)
	gsg.AddWitness(pt.ids[1], groupsig.Sign(pt.msk[1], GenTestBHHash("has-signed").Bytes()))
	p.blockContexts.attachVctx(&testBH11, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH11.Hash: {BH: &testBH11, gSignGenerator: gsg}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{{pt.ids[1], pt.mpk[1]}},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 0, pt.ids[1].GetAddrString(): 0},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501"), Random: []byte{1}},
	})
	testBH12 := GenTestBH("to51")
	rsg := model.NewGroupSignGenerator(2)
	rsg.AddWitnessForce(pt.ids[8], groupsig.Sign(pt.msk[2], []byte{1}))
	p.blockContexts.attachVctx(&testBH12, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH12.Hash: {BH: &testBH12, gSignGenerator: model.NewGroupSignGenerator(1), rSignGenerator: rsg}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{{pt.ids[1], pt.mpk[1]}},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 0, pt.ids[1].GetAddrString(): 0},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501"), Random: []byte{1}},
	})
	testBH13 := GenTestBH("hash-error")
	testBH13.Hash = emptyBHHash
	rsg1 := model.NewGroupSignGenerator(2)
	rsg1.AddWitnessForce(pt.ids[8], groupsig.Sign(pt.msk[2], []byte{1}))
	p.blockContexts.attachVctx(&testBH13, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH13.Hash: {BH: &testBH13, gSignGenerator: model.NewGroupSignGenerator(1), rSignGenerator: rsg1}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{{pt.ids[1], pt.mpk[1]}},
			memIndex: map[string]int{p.GetMinerID().GetAddrString(): 0, pt.ids[1].GetAddrString(): 0},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501"), Random: []byte{1}},
	})
	// for cast-illegal
	p.minerReader = newMinerPoolReader(p, NewMinerPoolTest(pt.mpk, pt.ids, pt.verifyGroup))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := p.OnMessageVerify(tt.args.msg)
			if msg != nil && !strings.Contains(msg.Error(), tt.expected) {
				t.Errorf("wanted {%s}; got {%s}", tt.expected, msg)
			}
		})
	}
}

type GroupHeaderTest struct{}

func (GroupHeaderTest) Seed() common.Hash {
	return common.HexToHash("0x00")
}

func (GroupHeaderTest) WorkHeight() uint64 {
	panic("implement me")
}

func (GroupHeaderTest) DismissHeight() uint64 {
	panic("implement me")
}

func (GroupHeaderTest) PublicKey() []byte {
	return []byte{}
}

func (GroupHeaderTest) Threshold() uint32 {
	panic("implement me")
}

func (GroupHeaderTest) GroupHeight() uint64 {
	panic("implement me")
}

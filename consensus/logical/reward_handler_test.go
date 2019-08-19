package logical

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"gopkg.in/fatih/set.v0"
	"testing"

	"github.com/zvchain/zvchain/consensus/model"
)

type GroupHanderTest struct {
}

func (g *GroupHanderTest) Seed() common.Hash {
	return common.Hash{}
}
func (g *GroupHanderTest) WorkHeight() uint64 {
	return uint64(0)
}
func (g *GroupHanderTest) DismissHeight() uint64 {
	return uint64(0)
}
func (g *GroupHanderTest) PublicKey() []byte {
	return make([]byte, 0)
}
func (g *GroupHanderTest) Threshold() uint32 {
	return uint32(5)
}
func (g *GroupHanderTest) GroupHeight() uint64 {
	return uint64(0)
}

type ProcessorTest struct {
	ProcessorInterface

	sk    []groupsig.Seckey
	pk    []groupsig.Pubkey
	ids   []groupsig.ID
	msk   []groupsig.Seckey
	mpk   []groupsig.Pubkey
	sigs  []groupsig.Signature
	sigs2 []groupsig.Signature

	blockHeader   *types.BlockHeader
	blockHeader2  *types.BlockHeader
	verifyGroup   *verifyGroup
	verifyContext *VerifyContext
}

func NewProcessorTest() *ProcessorTest {
	pt := &ProcessorTest{}

	n := 9
	k := 5

	// [1]
	r := base.NewRand()
	sk := make([]groupsig.Seckey, n)
	pk := make([]groupsig.Pubkey, n)
	ids := make([]groupsig.ID, n)
	mpk := make([]groupsig.Pubkey, n)

	for i := 0; i < n; i++ {
		sk[i] = *groupsig.NewSeckeyFromRand(r.Deri(i))
		pk[i] = *groupsig.NewPubkeyFromSeckey(sk[i])
		err := ids[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			panic("")
		}
	}

	shares := make([][]groupsig.Seckey, n)
	for i := 0; i < n; i++ {
		shares[i] = make([]groupsig.Seckey, n)
		msec := sk[i].GetMasterSecretKey(k)

		for j := 0; j < n; j++ {
			err := shares[i][j].Set(msec, &ids[j])
			if err != nil {
				panic("")
			}
		}
	}

	//for i := 0; i < n; i++ {
	//	shares[0][i]
	//}

	msk := make([]groupsig.Seckey, n)
	shareVec := make([]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			shareVec[i] = shares[i][j]
		}
		msk[j] = *groupsig.AggregateSeckeys(shareVec)
		mpk[j] = *groupsig.NewPubkeyFromSeckey(msk[j])
	}

	sigs := make([]groupsig.Signature, n)
	for i := 0; i < n; i++ {
		sigs[i] = groupsig.Sign(msk[i], common.BytesToHash([]byte("test")).Bytes())
	}
	sigs2 := make([]groupsig.Signature, n)
	for i := 0; i < n; i++ {
		sigs2[i] = groupsig.Sign(msk[1], common.BytesToHash([]byte("test")).Bytes())
	}
	//gpk := groupsig.AggregatePubkeys(pk)

	var gsig *groupsig.Signature
	for m := k; m <= n; m++ {
		sigVec := make([]groupsig.Signature, m)
		idVec := make([]groupsig.ID, m)

		for i := 0; i < m; i++ {
			sigVec[i] = sigs[i]
			idVec[i] = ids[i]
		}
		gsig = groupsig.RecoverSignature(sigVec, idVec)
	}

	pt.sk = sk
	pt.pk = pk
	pt.ids = ids
	pt.msk = msk
	pt.mpk = mpk
	pt.sigs = sigs
	pt.sigs2 = sigs2

	// [2]
	signArray := [2]groupsig.Signature{groupsig.Sign(pt.sk[0], common.BytesToHash([]byte("test")).Bytes()), *gsig}
	aggSign := groupsig.AggregateSigs(signArray[:])
	pt.blockHeader = &types.BlockHeader{
		Hash:      common.BytesToHash([]byte("test")),
		Castor:    pt.ids[0].Serialize(),
		Signature: aggSign.Serialize(),
	}
	pt.blockHeader2 = &types.BlockHeader{
		Hash:      common.BytesToHash([]byte("test")),
		Castor:    pt.ids[0].Serialize(),
		Signature: aggSign.Serialize(),
		Group:     common.HexToHash("0x2"),
	}

	// [3]
	mems := make([]*member, n)
	memIndex := make(map[string]int)
	for i := 0; i < n; i++ {
		mems[i] = &member{id: groupsig.DeserializeID(pt.ids[i].Serialize()), pk: groupsig.DeserializePubkeyBytes(pt.mpk[i].Serialize())}
		memIndex[mems[i].id.GetHexString()] = i
	}
	pt.verifyGroup = &verifyGroup{
		header:   &groupHeader{},
		memIndex: memIndex,
		members:  mems,
	}

	// [4]
	pt.verifyContext = &VerifyContext{
		prevBH:          &types.BlockHeader{},
		castHeight:      0,
		group:           pt.verifyGroup,
		expireTime:      0,
		consensusStatus: svWorking,
		slots:           make(map[common.Hash]*SlotContext),
		//ts:               time.TSInstance,
		//createTime:       time.TSInstance.Now(),
		proposers:        make(map[string]common.Hash),
		signedBlockHashs: set.New(set.ThreadSafe),
	}
	pt.verifyContext.slots[pt.blockHeader.Hash] = &SlotContext{
		BH:                  pt.blockHeader,
		castor:              pt.ids[0],
		pSign:               groupsig.Sign(pt.sk[0], common.BytesToHash([]byte("test")).Bytes()),
		slotStatus:          slWaiting,
		gSignGenerator:      model.NewGroupSignGenerator(int(k)),
		rSignGenerator:      model.NewGroupSignGenerator(int(k)),
		signedRewardTxHashs: set.New(set.ThreadSafe),
	}
	for i := 0; i < n; i++ {
		pt.verifyContext.slots[pt.blockHeader.Hash].AcceptVerifyPiece(pt.ids[i], pt.sigs[i], pt.sigs[i])
	}

	return pt
}

func (pt *ProcessorTest) GetMinerID() groupsig.ID {
	return pt.ids[0]
}

func (*ProcessorTest) GetRewardManager() types.RewardManager {
	return core.NewRewardManager()
}

func (pt *ProcessorTest) GetBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	if common.HexToHash("0x1") == hash {
		return nil
	} else if common.HexToHash("0x2") == hash {
		return pt.blockHeader2
	}
	return pt.blockHeader
}

func (pt *ProcessorTest) GetVctxByHeight(height uint64) *VerifyContext {
	return pt.verifyContext
}

func (pt *ProcessorTest) GetGroupBySeed(seed common.Hash) *verifyGroup {
	if common.HexToHash("0x0") == seed || common.HexToHash("0x03") == seed {
		return pt.verifyGroup
	}
	return nil
}

func (pt *ProcessorTest) GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey {
	return pt.msk[0]
}

func (*ProcessorTest) AddTransaction(tx *types.Transaction) (bool, error) {
	fmt.Println("AddTransaction")
	return true, nil
}

func (*ProcessorTest) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	fmt.Println("SendCastRewardSign")
}

func (*ProcessorTest) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	fmt.Println("SendCastRewardSignReq")
}

func _OnMessageCastRewardSign(pt *ProcessorTest, rh *RewardHandler, t *testing.T) {
	type fields struct {
		processor        ProcessorInterface
		futureRewardReqs *FutureMessageHolder
	}
	type args struct {
		msg *model.CastRewardTransSignMessage
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "ok",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0xf412782d9dc9fe7542ea13aa6153df1faea9e710e9b469ca90a88bf5e09efc06"), pt.ids[4], pt.msk[4]),
					},
				},
			},
		},
		{
			name: "block not exist",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0xf412782d9dc9fe7542ea13aa6153df1faea9e710e9b469ca90a88bf5e09efc06"), pt.ids[1], pt.msk[1]),
					},
					BlockHash: common.HexToHash("0x1"),
				},
			},
		},
		{
			name: "group not exist",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0xf412782d9dc9fe7542ea13aa6153df1faea9e710e9b469ca90a88bf5e09efc06"), pt.ids[2], pt.msk[2]),
					},
					BlockHash: common.HexToHash("0x2"),
				},
			},
		},
		{
			name: "data sign error 1",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0x0"), pt.ids[3], pt.msk[3]),
					},
				},
			},
		},
		{
			name: "data sign error 2",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0xf412782d9dc9fe7542ea13aa6153df1faea9e710e9b469ca90a88bf5e09efc06"), pt.ids[5], pt.msk[4]),
					},
				},
			},
		},
		{
			name: "data sign error 3",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0xf412782d9dc9fe7542ea13aa6153df1faea9e710e9b469ca90a88bf5e09efc06"), pt.ids[4], pt.msk[5]),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		//rh := &RewardHandler{
		//	processor:        tt.fields.processor,
		//	futureRewardReqs: tt.fields.futureRewardReqs,
		//}
		err := rh.OnMessageCastRewardSign(tt.args.msg)
		switch tt.name {
		case "ok":
			if err != nil {
				t.Error(tt.name)
			}
		case "block not exist",
			"group not exist",
			"data sign error 1",
			"data sign error 2",
			"data sign error 3":
			if err == nil {
				t.Error(tt.name)
			}
		default:
			panic("")
		}
	}
}

func TestRewardHandler_OnMessageCastRewardSign(t *testing.T) {
	pt := NewProcessorTest()
	rh := &RewardHandler{
		processor:        pt,
		futureRewardReqs: NewFutureMessageHolder(),
	}
	slotContext := pt.GetVctxByHeight(0).slots[pt.GetBlockHeaderByHash(common.Hash{}).Hash]
	slotContext.setSlotStatus(slSuccess)
	rh.reqRewardTransSign(pt.GetVctxByHeight(0), pt.GetBlockHeaderByHash(common.Hash{}))
	_OnMessageCastRewardSign(pt, rh, t)
}

func _OnMessageCastRewardSignReq(pt *ProcessorTest, rh *RewardHandler, t *testing.T) {
	type fields struct {
		processor        ProcessorInterface
		futureRewardReqs *FutureMessageHolder
	}
	type args struct {
		msg *model.CastRewardTransSignReqMessage
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "ok",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.Hash{}, pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "block not exist",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.Hash{}, pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
						BlockHash: common.HexToHash("0x1"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "group not exist 1",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.Hash{}, pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
						BlockHash: common.HexToHash("0x2"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "group not exist 2",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.Hash{}, pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
						Group:     common.HexToHash("0x1"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "group not exist 3",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.Hash{}, pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0xb0109da3ecdf66ad2b134afa9e0c05f10ac1680a67d9bfd4c35339bac21e98fc"),
						BlockHash: common.HexToHash("0x2"),
						Group:     common.HexToHash("0x2"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "sign data error 1",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash(""), pt.ids[0], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "sign data error 2",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash(""), pt.ids[1], pt.msk[0]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "sign data error 3",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0x1"), pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "tx hash not exist",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash("0x1"), pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x1"),
					},
					SignedPieces: pt.sigs,
				},
			},
		},
		{
			name: "ids error",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash(""), pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{1, 1, 1, 1, 1, 1, 1, 1, 1},
						TxHash:    common.HexToHash("0x7a3273d8ea535b1007076ecd4c01f0b54763b7d2d1352adf7c5b3a3af599039e"),
					},
					SignedPieces: pt.sigs2,
				},
			},
		},
		{
			name: "signed pieces error",
			fields: fields{
				processor:        pt,
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignReqMessage{
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(common.HexToHash(""), pt.ids[1], pt.msk[1]),
					},
					Reward: types.Reward{
						TargetIds: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8},
						TxHash:    common.HexToHash("0x70676b767052302f7cead4c232bdd1159194023d9ea06c16e2f4a0fda7d7e1b3"),
					},
					SignedPieces: pt.sigs2,
				},
			},
		},
	}
	for _, tt := range tests {
		//rh := &RewardHandler{
		//	processor:        tt.fields.processor,
		//	futureRewardReqs: tt.fields.futureRewardReqs,
		//}
		err := rh.OnMessageCastRewardSignReq(tt.args.msg)
		switch tt.name {
		case "ok":
			if err != nil {
				t.Error(tt.name)
			}
		case "block not exist",
			"group not exist 1",
			"group not exist 2",
			"group not exist 3",
			"sign data error 1",
			"sign data error 2",
			"sign data error 3",
			"tx hash not exist",
			"ids error",
			"signed pieces error":
			if err == nil {
				t.Error(tt.name)
			}
		default:
			panic("")
		}
	}
}

func TestRewardHandler_OnMessageCastRewardSignReq_bug(t *testing.T) {
	pt := NewProcessorTest()
	rh := &RewardHandler{
		processor:        pt,
		futureRewardReqs: NewFutureMessageHolder(),
	}
	_OnMessageCastRewardSignReq(pt, rh, t)
}

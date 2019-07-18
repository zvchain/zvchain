package logical

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"testing"

	"github.com/zvchain/zvchain/consensus/model"
)

type ProcessorTest struct {

}

func (*ProcessorTest) GetMinerID() groupsig.ID {
	panic("implement me")
}

func (*ProcessorTest) GetRewardManager() types.RewardManager {
	panic("implement me")
}

func (*ProcessorTest) GetBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	return nil
}

func (*ProcessorTest) GetVctxByHeight(height uint64) *VerifyContext {
	panic("implement me")
}

func (*ProcessorTest) GetGroupBySeed(seed common.Hash) *verifyGroup {
	panic("implement me")
}

func (*ProcessorTest) GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey {
	panic("implement me")
}

func (*ProcessorTest) AddTransaction(tx *types.Transaction) (bool, error) {
	panic("implement me")
}

func (*ProcessorTest) AddMessage(hash common.Hash, msg interface{}) {
	panic("implement me")
}

func (*ProcessorTest) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	panic("implement me")
}

func TestRewardHandler_OnMessageCastRewardSign(t *testing.T) {
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
			name: "test1",
			fields: fields{
				processor: &ProcessorTest{

				},
				futureRewardReqs: NewFutureMessageHolder(),
			},
			args: args{
				msg: &model.CastRewardTransSignMessage {

				},
			},
		},
	}
	for _, tt := range tests {
		rh := &RewardHandler{
			processor:        tt.fields.processor,
			futureRewardReqs: tt.fields.futureRewardReqs,
		}
		rh.OnMessageCastRewardSign(tt.args.msg)
	}
}

func TestRewardHandler_OnMessageCastRewardSignReq(t *testing.T) {
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		rh := &RewardHandler{
			processor:        tt.fields.processor,
			futureRewardReqs: tt.fields.futureRewardReqs,
		}
		rh.OnMessageCastRewardSignReq(tt.args.msg)
	}
}

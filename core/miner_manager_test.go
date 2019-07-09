package core

import (
	json2 "encoding/json"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"testing"
)

func TestMinerManager_MaxStake(t *testing.T) {
	maxs := []uint64{2500000, 4245283, 5803571, 7203389, 8467741, 9615384, 10661764, 10915492, 11148648, 11363636, 11562500,
		11746987, 11918604, 11797752, 11684782, 11578947, 11479591, 11386138, 11298076, 11098130, 10909090, 10730088,
		10560344, 10399159, 10245901}
	for i := 0; i <= 30; i++ {
		var cur = i
		if i >= len(maxs) {
			cur = len(maxs) - 1
		}
		max := maximumStake(uint64(i * 5000000))
		if max != maxs[cur]*common.TAS {
			t.Errorf("max stake wanted:%d, got %d", maxs[cur]*common.TAS, max)
		}
	}
}

type mOperMsg struct {
	opType   int8
	operator *common.Address
	target   *common.Address
	value    *big.Int
	data     []byte
}

func (msg *mOperMsg) OpType() int8 {
	return msg.opType
}

func (msg *mOperMsg) Operator() *common.Address {
	return msg.operator
}

func (msg *mOperMsg) OpTarget() *common.Address {
	return msg.target
}

func (msg *mOperMsg) Amount() *big.Int {
	return msg.value
}

func (msg *mOperMsg) Payload() []byte {
	return msg.data
}

func genMOperMsg(source, target *common.Address, typ int8, value uint64, data []byte) *mOperMsg {
	return &mOperMsg{
		operator: source,
		target:   target,
		value:    new(big.Int).SetUint64(value),
		data:     data,
		opType:   typ,
	}
}

type mOperContext struct {
	source        *common.Address
	target        *common.Address
	mType         types.MinerType
	stakeAddValue uint64
	originBalance uint64
	reduceValue   uint64
}

var (
	src    = common.HexToAddress("0x123")
	target = common.HexToAddress("0x456")
	ctx    = &mOperContext{
		source:        &src,
		target:        &target,
		mType:         types.MinerTypeProposal,
		stakeAddValue: 2000 * common.TAS,
		originBalance: 3000 * common.TAS,
		reduceValue:   1000 * common.TAS,
	}
	accountDB types.AccountDB
)

func setup() {
	err := initContext4Test()
	if err != nil {
		panic("init fail " + err.Error())
	}
	db := BlockChainImpl.LatestStateDB()
	db.AddBalance(src, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(target, new(big.Int).SetUint64(ctx.originBalance))
	accountDB = db
}

func testStakeAddFromSelf(ctx *mOperContext, t *testing.T) {
	var mpks = &types.MinerPks{
		MType: ctx.mType,
		Pk:    common.FromHex("0x215fdace84c59a6d86e1cbe4238c3e4a5d7a6e07f6d4c5603399e573cc05a32617faae51cfd3fce7c84447522e52a1439f46fc5adb194240325fcb800a189ae129ebca2b59999a9ecd16e03184e7fe578418b20cbcdc02129adc79bf090534a80fb9076c3518ae701477220632008fc67981e2a1be97a160a2f9b5804f9b280f"),
		VrfPk: common.FromHex("0x7bc1cb6798543feb524456276d9b26014ddfb5cd757ac6063821001b50679bcf"),
	}

	bs, err := types.EncodePayload(mpks)
	if err != nil {
		t.Fatalf("encode payload error:%v", err)
	}
	stakeAddMsg := genMOperMsg(ctx.target, ctx.target, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)

	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}

	miner, err := getMiner(accountDB, *ctx.target, ctx.mType)
	if miner == nil {
		t.Errorf("get miner nil")
	}
	t.Logf("minerstatus after stake for self:%v %v", miner.Stake, miner.Status)
	if !miner.IsActive() {
		t.Error("miner should be active")
	}
}

func testStakeAddFromOthers(ctx *mOperContext, t *testing.T) {
	var mpks = &types.MinerPks{
		MType: ctx.mType,
	}

	bs, err := types.EncodePayload(mpks)
	if err != nil {
		t.Fatalf("encode payload error:%v", err)
	}
	stakeAddMsg := genMOperMsg(ctx.source, ctx.target, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)
	source := *stakeAddMsg.operator

	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}

	balance2 := accountDB.GetBalance(source)
	t.Logf("operator balance after stake-add:%v", balance2)

	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if miner == nil {
		t.Errorf("get miner nil")
	}

	t.Logf("minerstatus after stake from others:%v %v", miner.Stake, miner.Status)

}

func testMinerAbort(ctx *mOperContext, t *testing.T) {
	msg := genMOperMsg(ctx.target, ctx.target, types.TransactionTypeMinerAbort, 0, []byte{byte(ctx.mType)})
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, msg, 1)
	if err != nil {
		t.Fatalf("execute miner abort msg error:%v", err)
	}
	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if miner == nil {
		t.Errorf("get miner nil")
	}
	if !miner.IsPrepare() {
		t.Errorf("abort fail, status %v %v", miner.Status, miner.StatusUpdateHeight)
	}
	t.Logf("miner status after abort %v %v", miner.Stake, miner.Status)
}

func testStakeReduce(ctx *mOperContext, t *testing.T) {
	msg := genMOperMsg(ctx.target, ctx.target, types.TransactionTypeStakeReduce, ctx.reduceValue, []byte{byte(ctx.mType)})
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, msg, 1)
	if err != nil {
		t.Fatalf("execute miner abort msg error:%v", err)
	}
	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if miner == nil {
		t.Errorf("get miner nil")
	}
	if miner.Stake != ctx.stakeAddValue-ctx.reduceValue {
		t.Errorf("stake error expect %v, infact %v", ctx.stakeAddValue-ctx.reduceValue, miner.Stake)
	}

	details := MinerManagerImpl.GetStakeDetails(*ctx.target, *ctx.target)
	t.Log(detailString(details))
	t.Logf("miner status after reduce %v %v", miner.Stake, miner.Status)
}

func testStakeRefund(ctx *mOperContext, t *testing.T) {
	msg := genMOperMsg(ctx.target, ctx.target, types.TransactionTypeStakeRefund, 0, []byte{byte(ctx.mType)})
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, msg, 10000000)
	if err != nil {
		t.Fatalf("execute miner abort msg error:%v", err)
	}
	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if miner == nil {
		t.Errorf("get miner nil")
	}

	details := MinerManagerImpl.GetStakeDetails(*ctx.target, *ctx.target)
	t.Log(detailString(details))
	t.Logf("miner status after reduce %v %v", miner.Stake, miner.Status)
}

func TestMinerManager_ExecuteOperation_StakeAddForOthers(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromOthers(ctx, t)
	srcBalance := accountDB.GetBalance(*ctx.source)
	if srcBalance.Uint64()+ctx.stakeAddValue != ctx.originBalance {
		t.Errorf("src balance error after stake")
	}
}

func TestMinerManager_ExecuteOperation_StakeAddForSelf(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromSelf(ctx, t)
	targetBalance := accountDB.GetBalance(*ctx.target)
	if targetBalance.Uint64()+ctx.stakeAddValue != ctx.originBalance {
		t.Errorf("src balance error after stake")
	}

	totalStake := getProposalTotalStake(accountDB.AsAccountDBTS())
	if totalStake != ctx.stakeAddValue {
		t.Errorf("totalStake should be zero,infact is %v", totalStake)
	}
}

func detailString(data interface{}) string {
	json, _ := json2.MarshalIndent(data, "", "\t")
	return string(json)
}

func TestMinerManager_GetAllStakeDetails_StakeAdd(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromSelf(ctx, t)
	details := MinerManagerImpl.GetStakeDetails(*ctx.target, *ctx.target)
	t.Log(detailString(details))

	t.Log("===========stakes from others===============")
	testStakeAddFromOthers(ctx, t)
	details = MinerManagerImpl.GetStakeDetails(*ctx.target, *ctx.source)
	t.Log(detailString(details))

	totalStake := getProposalTotalStake(accountDB.AsAccountDBTS())

	t.Logf("total stake:%v", totalStake)
	if totalStake != ctx.stakeAddValue*2 {
		t.Errorf("total stake error: expect %v, infact %v", ctx.stakeAddValue*2, totalStake)
	}

	t.Log("==============all details")
	allDetails := MinerManagerImpl.GetAllStakeDetails(*ctx.target)
	t.Log(detailString(allDetails))
}

func TestMinerManager_ExecuteOperation_MinerAbort(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromSelf(ctx, t)

	testMinerAbort(ctx, t)
}

func TestMinerManager_ExecuteOperation_StakeReduce(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromSelf(ctx, t)

	totalStake := getProposalTotalStake(accountDB.AsAccountDBTS())
	if totalStake != ctx.stakeAddValue {
		t.Errorf("totalStake should be zero,infact is %v", totalStake)
	}

	testStakeReduce(ctx, t)
	totalStake = getProposalTotalStake(accountDB.AsAccountDBTS())
	if totalStake != ctx.stakeAddValue-ctx.reduceValue {
		t.Errorf("totalStake should be zero,infact is %v", totalStake)
	}
}

func TestMinerManager_ExecuteOperation_StakeRefund(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromSelf(ctx, t)
	totalStake := getProposalTotalStake(accountDB.AsAccountDBTS())
	if totalStake != ctx.stakeAddValue {
		t.Errorf("totalStake should be zero,infact is %v", totalStake)
	}

	testStakeReduce(ctx, t)
	totalStake = getProposalTotalStake(accountDB.AsAccountDBTS())
	if totalStake != ctx.stakeAddValue-ctx.reduceValue {
		t.Errorf("totalStake should be zero,infact is %v", totalStake)
	}

	testStakeRefund(ctx, t)
	balance := accountDB.GetBalance(*ctx.target)
	if balance.Uint64() != ctx.originBalance-(ctx.stakeAddValue-ctx.reduceValue) {
		t.Errorf("balance not equal to origin")
	}
}

func TestMinerManager_GetLatestMiner(t *testing.T) {
	setup()
	defer clear()

	testStakeAddFromSelf(ctx, t)

	miner := MinerManagerImpl.GetLatestMiner(*ctx.target, ctx.mType)
	t.Log(detailString(miner))
}

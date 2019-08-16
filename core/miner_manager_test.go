package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"testing"
)

func TestMinerManager_MaxStake(t *testing.T) {
	maxs := []uint64{2500000, 3500000, 4500000, 5500000, 6000000, 6500000,
		7000000, 7250000, 7500000, 7750000, 7875000, 8000000, 8125000}
	for i := 0; i <= 30; i++ {
		var cur = i
		if i >= len(maxs) {
			cur = len(maxs) - 1
		}
		max := maximumStake(uint64(i * 10000000))
		if max != maxs[cur]*common.ZVC {
			t.Errorf("max stake wanted:%d, got %d", maxs[cur]*common.ZVC, max)
		}
	}
}


var fullStake uint64 = 2500000 * common.ZVC

type mOperMsg struct {
	opType   int8
	operator *common.Address
	target   *common.Address
	value    *big.Int
	data     []byte
}

func (msg *mOperMsg)GetExtraData()[]byte{
	return nil
}

func (msg *mOperMsg)GetHash()common.Hash{
	return common.Hash{}
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
	src    = common.StringToAddress("zv123")
	target = common.StringToAddress("zv456")
	guardNode1 = common.StringToAddress("zv01111")
	guardNode2 = common.StringToAddress("zv02222")
	guardNode3 = common.StringToAddress("zv03333")
	guardNode4 = common.StringToAddress("zv04444")
	guardNode5 = common.StringToAddress("zv05555")
	guardNode6 = common.StringToAddress("zv06666")
	guardNode7 = common.StringToAddress("zv07777")
	guardNode8 = common.StringToAddress("zv08888")
	minerPool = common.StringToAddress("zv09999")
	ctx    = &mOperContext{
		source:        &src,
		target:        &target,
		mType:         types.MinerTypeProposal,
		stakeAddValue: 2000 * common.ZVC,
		originBalance: 300000000 * common.ZVC,
		reduceValue:   1000 * common.ZVC,
	}
	accountDB types.AccountDB
)

func setup() {
	err := initContext4Test()
	if err != nil {
		panic("init fail " + err.Error())
	}
	db, error := BlockChainImpl.LatestStateDB()
	if error != nil {
		panic("init fail " + err.Error())
	}
	db.AddBalance(src, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(target, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode1, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode2, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode3, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode4, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode5, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode6, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode7, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(guardNode8, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(minerPool, new(big.Int).SetUint64(ctx.originBalance))
	accountDB = db
}

func TestInit(t *testing.T){
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)
	for _,addr := range types.ExtractGuardNodes{
		fd,_:=getFundGuardNode(accountDB,addr)
		if fd == nil{
			t.Fatalf("except got value,but got nil")
		}
		if !fd.isFundGuard(){
			t.Fatalf("except %v,but got %v",FundGuardNode,fd.Type)
		}
	}
}

func TestStakeSelf(t *testing.T){
	setup()
	defer clear()
	testStakeSelfProposal(t)
	testStakeSelfVerify(t)
	bla := accountDB.GetBalance(src)
	if bla.Uint64() != ctx.originBalance - 1000 * common.ZVC{
		t.Fatalf("balance error")
	}
}

func TestNormalApplyGuardNode(t *testing.T){
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)
	testFullStakeFromSelf(ctx,t)
	var height uint64 = 0
	testApplyGuardNode(t,true,height)
	dt := getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod /2 {
		t.Fatalf("except height = %v,but got %v",adjustWeightPeriod /2,dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB,*ctx.source){
		t.Fatalf("except in full stake guard node,but got nil")
	}
	testApplyGuardNode(t,true,height)
	dt = getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod  {
		t.Fatalf("except height = %v,but got %v",adjustWeightPeriod ,dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB,*ctx.source){
		t.Fatalf("except in full stake guard node,but got nil")
	}
	testApplyGuardNode(t,true,height)
	dt = getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod  {
		t.Fatalf("except height = %v,but got %v",adjustWeightPeriod ,dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB,*ctx.source){
		t.Fatalf("except in full stake guard node,but got nil")
	}
	ctx.mType = types.MinerTypeVerify
	testApplyGuardNode(t,false,height)
	dt = getStakeDetail()
	if dt != nil{
		t.Fatalf("except nil,but got value")
	}
}

func testStakeSelfProposal(t *testing.T){
	ctx.stakeAddValue = 100 * common.ZVC
	ctx.source = &src
	ctx.target = &src
	ctx.mType = types.MinerTypeProposal
	testStakeAddFromSelf(ctx,t)

	total := getTotalStake()
	if total!= 0{
		t.Fatalf("except 0,but got %v",total)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil{
		t.Fatalf("error is %v",err)
	}
	if !miner.IsPrepare(){
		t.Fatalf("except prepared,but got %v",miner.Status)
	}
	if miner.Stake != ctx.stakeAddValue{
		t.Fatalf("except %v,but got %v",ctx.stakeAddValue,miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != ctx.stakeAddValue{
		t.Fatalf("except %v,but got %v",ctx.stakeAddValue,dt.Value)
	}

	ctx.stakeAddValue = 400 * common.ZVC
	testStakeAddFromSelf(ctx,t)
	total = getTotalStake()
	if total!= 500*common.ZVC{
		t.Fatalf("except %v,but got %v",500*common.ZVC,total)
	}
	miner, err = getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil{
		t.Fatalf("error is %v",err)
	}
	if !miner.IsActive(){
		t.Fatalf("except prepared,but got %v",miner.Status)
	}
	if miner.Stake != 500 * common.ZVC{
		t.Fatalf("except %v,but got %v",500*common.ZVC,miner.Stake)
	}
	dt = getStakeDetail()
	if dt.Value != 500 * common.ZVC{
		t.Fatalf("except %v,but got %v",500*common.ZVC,dt.Value)
	}
}

func testStakeSelfVerify(t *testing.T){
	ctx.stakeAddValue = 100 * common.ZVC
	ctx.source = &src
	ctx.target = &src
	ctx.mType = types.MinerTypeVerify
	testStakeAddFromSelf(ctx,t)
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil{
		t.Fatalf("error is %v",err)
	}
	if !miner.IsPrepare(){
		t.Fatalf("except prepared,but got %v",miner.Status)
	}
	if miner.Stake != ctx.stakeAddValue{
		t.Fatalf("except %v,but got %v",ctx.stakeAddValue,miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != ctx.stakeAddValue{
		t.Fatalf("except %v,but got %v",ctx.stakeAddValue,dt.Value)
	}
	ctx.stakeAddValue = 400 * common.ZVC
	testStakeAddFromSelf(ctx,t)
	miner, err = getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil{
		t.Fatalf("error is %v",err)
	}
	if !miner.IsActive(){
		t.Fatalf("except prepared,but got %v",miner.Status)
	}
	if miner.Stake != 500 * common.ZVC{
		t.Fatalf("except %v,but got %v",500*common.ZVC,miner.Stake)
	}
	dt = getStakeDetail()
	if dt.Value != 500 * common.ZVC{
		t.Fatalf("except %v,but got %v",500*common.ZVC,dt.Value)
	}

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

	stakeAddMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)
	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
}

func testFullStakeFromSelf(ctx *mOperContext, t *testing.T) {
	ctx.source = &src
	ctx.stakeAddValue = 2500000 * common.ZVC
	ctx.target = &src
	var mpks = &types.MinerPks{
		MType: ctx.mType,
		Pk:    common.FromHex("0x215fdace84c59a6d86e1cbe4238c3e4a5d7a6e07f6d4c5603399e573cc05a32617faae51cfd3fce7c84447522e52a1439f46fc5adb194240325fcb800a189ae129ebca2b59999a9ecd16e03184e7fe578418b20cbcdc02129adc79bf090534a80fb9076c3518ae701477220632008fc67981e2a1be97a160a2f9b5804f9b280f"),
		VrfPk: common.FromHex("0x7bc1cb6798543feb524456276d9b26014ddfb5cd757ac6063821001b50679bcf"),
	}
	bs, err := types.EncodePayload(mpks)
	if err != nil {
		t.Fatalf("encode payload error:%v", err)
	}
	stakeAddMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)
	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil{
		t.Fatalf("error is %v",err)
	}
	if !miner.IsActive(){
		t.Fatalf("except prepared,but got %v",miner.Status)
	}
	if miner.Stake != 2500000 * common.ZVC{
		t.Fatalf("except %v,but got %v",2500000*common.ZVC,miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != 2500000 * common.ZVC{
		t.Fatalf("except %v,but got %v",2500000*common.ZVC,dt.Value)
	}
	bla :=accountDB.GetBalance(src)
	if bla.Uint64() != ctx.originBalance - 2500000 * common.ZVC{
		t.Fatalf("except %v,but got %v",ctx.originBalance - 2500000 * common.ZVC,bla.Uint64())
	}
}

func testApplyGuardNode(t *testing.T,success bool,height uint64){
	applyMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeApplyGuardMiner, 0,nil)
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, height)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil{
		t.Fatalf("error is %v",err)
	}
	if success{
		if !miner.IsGuard(){
			t.Fatalf("except %v,but got %v",types.MinerGuard,miner.Identity)
		}
		dt := getStakeDetail()
		if dt.Value != fullStake{
			t.Fatalf("except %v,but got %v",fullStake,dt.Value)
		}
	}
}

func getTotalStake()uint64{
	return getProposalTotalStake(accountDB)
}

func getStakeDetail()*stakeDetail{
	detailKey := getDetailKey(*ctx.source, ctx.mType, types.Staked)
	detail,_ := getDetail(accountDB,*ctx.target, detailKey)
	return detail
}




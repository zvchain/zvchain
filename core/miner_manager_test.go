package core

import (
	json2 "encoding/json"
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
	guardNode1 = common.HexToAddress("0x01")
	guardNode2 = common.HexToAddress("0x02")
	guardNode3 = common.HexToAddress("0x03")
	guardNode4 = common.HexToAddress("0x04")
	guardNode5 = common.HexToAddress("0x05")
	guardNode6 = common.HexToAddress("0x06")
	guardNode7 = common.HexToAddress("0x07")
	guardNode8 = common.HexToAddress("0x08")
	minerPool = common.HexToAddress("0x09")
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
	stakeAddMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)
	source := *stakeAddMsg.operator

	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}

	balance2 := accountDB.GetBalance(source)
	t.Logf("operator balance after stake-add:%v", balance2)

	miner, _ := getMiner(accountDB, *ctx.source, ctx.mType)
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

func testStakeReduce(ctx *mOperContext, height uint64,t *testing.T) {
	msg := genMOperMsg(ctx.target, ctx.target, types.TransactionTypeStakeReduce, ctx.reduceValue, []byte{byte(ctx.mType)})
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, msg, height)
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
	totalStake := getProposalTotalStake(accountDB.AsAccountDBTS())
	details := MinerManagerImpl.GetStakeDetails(*ctx.target, *ctx.target)
	t.Log(detailString(details))

	t.Log("===========stakes from others===============")
	testStakeAddFromOthers(ctx, t)
	details = MinerManagerImpl.GetStakeDetails(*ctx.target, *ctx.source)
	t.Log(detailString(details))

	totalStake = getProposalTotalStake(accountDB.AsAccountDBTS())

	t.Logf("total stake:%v", totalStake)
	if totalStake != ctx.stakeAddValue {
		t.Errorf("total stake error: expect %v, infact %v", ctx.stakeAddValue*2, totalStake)
	}

	t.Log("==============all details")
	allDetails := MinerManagerImpl.GetAllStakeDetails(*ctx.target)
	t.Log(detailString(allDetails))
}


func TestMinerManager_GuardInvalid(t *testing.T){
	setup()
	defer clear()
	geneMinerPoolWithExartNode(t)

	miner, _ := getMiner(accountDB, minerPool, ctx.mType)
	if miner == nil || !miner.IsMinerPool(){
		t.Fatalf("except miner pool,but not")
	}
	testAdminCancel(t)

	miner, _ = getMiner(accountDB, minerPool, ctx.mType)
	if !miner.IsInvalidMinerPool(){
		t.Fatalf("except invalid miner pool,but not")
	}

	exartGuardAddr := common.HexToAddress(types.ExtractGuardNodes[0])
	vote,err := getVoteInfo(accountDB,exartGuardAddr)
	if err !=nil{
		t.Fatalf("error")
	}
	if vote != nil{
		t.Fatalf("except vote is nil,but got value")
	}

	key := getTicketsKey(minerPool)
	totalTickets := getTotalTickets(accountDB.AsAccountDBTS(),key)

	if totalTickets != 7{
		t.Fatalf("except 7 tickets,but got %d",totalTickets)
	}
}

func TestMinerManager_Vote(t *testing.T){
	setup()
	defer clear()
	geneMinerPool(t)
	info,err := getGuardMinerNodeInfo(accountDB.AsAccountDBTS())
	if err !=nil{
		t.Fatalf("except no error,but got error")
	}
	if info.Len!= 8{
		t.Fatalf("except got 8 ,but got %d",info.Len)
	}
	if info.BeginIndex!=0{
		t.Fatalf("except got 0 ,but got %d",info.BeginIndex)
	}

	miner, _ := getMiner(accountDB, minerPool, ctx.mType)
	if miner == nil || !miner.IsMinerPool(){
		t.Fatalf("except miner pool,but not")
	}
}

func geneMinerPoolWithExartNode(t *testing.T){
	MinerManagerImpl.genGuardNodes(accountDB)

	ctx.stakeAddValue = 2500000 * common.ZVC
	ctx.target = &guardNode1
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode2
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode3
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode4
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode5
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode6
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode7
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}

	addExart := common.HexToAddress(types.ExtractGuardNodes[0])
	testVote(&addExart,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
}

func geneMinerPool(t *testing.T){
	ctx.stakeAddValue = 2500000 * common.ZVC
	ctx.target = &guardNode1
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode2
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode3
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode4
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode5
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode6
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode7
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
	ctx.target = &guardNode8
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)
	testVote(ctx.target,&minerPool,t)
	miner, _ = getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}
}


func TestMinerManager_ApplyGuardNode(t *testing.T){
	setup()
	defer clear()
	ctx.stakeAddValue = 2500000 * common.ZVC
	testStakeAddFromSelf(ctx, t)
	testApplyGuardNode(t)

	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)

	if !miner.IsGuard(){
		t.Fatalf("except guard node,but not")
	}

	detailKey := getDetailKey(*ctx.target, ctx.mType, types.Staked)
	detail,err := getDetail(accountDB,*ctx.target, detailKey)
	if err !=nil{
		t.Fatalf("error")
	}
	if detail.DisMissHeight!= halfOfYearBlocks{
		t.Fatalf("except dismiss height is %v,but got %v",halfOfYearBlocks,detail.DisMissHeight)
	}

	vf,err := getVoteInfo(accountDB,*ctx.target)
	if err != nil{
		t.Fatalf("error")
	}
	if vf == nil{
		t.Fatalf("except not nil,but got nil")
	}
	empty := common.Address{}
	if vf.Target != empty{
		t.Fatalf("except empty addr,but got %s",vf.Target)
	}
	if vf.Last!= 1{
		t.Fatalf("except got 1,but got %d",vf.Last)
	}
}

func testApplyGuardNode(t *testing.T){
	applyMsg := genMOperMsg(ctx.target, ctx.target, types.TransactionTypeApplyGuardMiner, 0,nil)
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
}


func testAdminCancel(t *testing.T){
	source := common.HexToAddress(types.MiningPoolAddr)
	target = common.HexToAddress(types.ExtractGuardNodes[0])
	applyMsg := genMOperMsg(&source, &target, types.TransactionTypeCancelGuard, 0,nil)
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
}


func testVote(source,target *common.Address,t *testing.T){
	applyMsg := genMOperMsg(source, target, types.TransactionTypeVoteMinerPool, 0,nil)
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
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

	testStakeReduce(ctx, 1,t)
	totalStake = getProposalTotalStake(accountDB.AsAccountDBTS())
	if totalStake != ctx.stakeAddValue-ctx.reduceValue {
		t.Errorf("totalStake should be zero,infact is %v", totalStake)
	}

	ctx.stakeAddValue = 1000 * common.ZVC
	testStakeReduce(ctx, 1,t)

	miner, _ := getMiner(accountDB, *ctx.target, ctx.mType)
	if !miner.IsPrepare(){
		t.Fatalf("except perpared,but not")
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

	testStakeReduce(ctx,1, t)
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

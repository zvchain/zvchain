package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
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

func (msg *mOperMsg) GetExtraData() []byte {
	return nil
}

func (msg *mOperMsg) GetHash() common.Hash {
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
	height        uint64
}

var (
	src        = common.StringToAddress("zv123")
	target     = common.StringToAddress("zv456")
	normal1    = common.StringToAddress("zv11223344")
	normal2    = common.StringToAddress("zv111111")
	normal3    = common.StringToAddress("zv222222")
	normal4    = common.StringToAddress("zv333333")
	normal5    = common.StringToAddress("zv444444")
	guardNode1 = common.StringToAddress("zv01111")
	guardNode2 = common.StringToAddress("zv02222")
	guardNode3 = common.StringToAddress("zv03333")
	guardNode4 = common.StringToAddress("zv04444")
	guardNode5 = common.StringToAddress("zv05555")
	guardNode6 = common.StringToAddress("zv06666")
	guardNode7 = common.StringToAddress("zv07777")
	guardNode8 = common.StringToAddress("zv08888")
	minerPool  = common.StringToAddress("zv09999")
	ctx        = &mOperContext{
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
	db.AddBalance(types.ExtractGuardNodes[0], new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(types.ExtractGuardNodes[1], new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(normal1, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(normal2, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(normal3, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(normal4, new(big.Int).SetUint64(ctx.originBalance))
	db.AddBalance(normal5, new(big.Int).SetUint64(ctx.originBalance))
	accountDB = db
}

func TestInit(t *testing.T) {
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)
	for _, addr := range types.ExtractGuardNodes {
		fd, _ := getFundGuardNode(accountDB, addr)
		if fd == nil {
			t.Fatalf("except got value,but got nil")
		}
		if !fd.isFundGuard() {
			t.Fatalf("except %v,but got %v", fundGuardNodeType, fd.Type)
		}
		if fd.FundModeType != common.SIXAddSix {
			t.Fatalf("except 1,but got %v", fd.FundModeType)
		}
	}
}

func TestInsteadStake(t *testing.T) {
	setup()
	defer clear()
	ctx.source = &src
	ctx.target = &target
	testStakeFromOther(t, false)

	ctx.source = &src
	ctx.target = &guardNode1
	testStakeFromOther(t, false)

	genePoolMiner(t)
	ctx.stakeAddValue = 100 * common.ZVC
	ctx.source = &src
	ctx.target = &minerPool
	testStakeFromOther(t, true)

	ctx.source = &minerPool
	ctx.target = &src
	ctx.stakeAddValue = 100 * common.ZVC
	testStakeFromOther(t, false)

	ctx.source = &types.MiningPoolAddr
	ctx.target = &normal1
	ctx.stakeAddValue = 100 * common.ZVC
	testStakeFromOther(t, true)
}

func TestInvalidMinerPoolAction(t *testing.T) {
	setup()
	defer clear()
	genePoolMiner(t)
	accountDB.(*account.AccountDB).Commit(true)
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod+1000)
	miner, err := getMiner(accountDB, minerPool, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsInvalidMinerPool() {
		t.Fatalf("except invalid miner pool,but got %v", miner.Identity)
	}
	ctx.source = &minerPool
	ctx.target = &minerPool
	ctx.stakeAddValue = 100 * common.ZVC
	testStakeAddFromSelf(t, false)

	ctx.source = &src
	ctx.target = &minerPool
	testStakeFromOther(t, false)

	ctx.source = &minerPool
	ctx.target = &minerPool
	testStakeAbort(t, false)
}

func TestStakeMax(t *testing.T) {
	setup()
	defer clear()
	ctx.source = &src
	ctx.target = &src
	testFullStakeFromSelf(t)

	ctx.stakeAddValue = 100 * common.ZVC
	testStakeAddFromSelf(t, false)
	genePoolMiner(t)
	miner, err := getMiner(accountDB, minerPool, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsMinerPool() {
		t.Fatalf("except miner pool,but got %v", miner.Identity)
	}

	ctx.source = &guardNode1
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode2
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode3
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode4
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode5
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode6
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode7
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &guardNode8
	ctx.target = &minerPool
	testFullStakeFromOther(t)

	ctx.source = &minerPool
	ctx.target = &minerPool
	ctx.stakeAddValue = 100 * common.ZVC
	testStakeAddFromSelf(t, false)
}

func TestStakeSelf(t *testing.T) {
	setup()
	defer clear()
	testStakeSelfProposal(t)
	testStakeSelfVerify(t)
	bla := accountDB.GetBalance(src)
	if bla.Uint64() != ctx.originBalance-1000*common.ZVC {
		t.Fatalf("balance error")
	}
}

func TestRepeatVote(t *testing.T) {
	setup()
	defer clear()
	ctx.source = &normal2
	ctx.target = &normal2
	testFullStakeFromSelf(t)
	var height uint64 = 0
	testApplyGuardNode(t, true, height)

	ctx.target = &normal3
	ctx.height = 10
	testVote(t, true)

	ctx.height = 20
	testVote(t, false)

	ctx.height = adjustWeightPeriod/2 + 10
	testVote(t, true)

	ctx.source = &normal1
	ctx.target = &guardNode1
	testVote(t, false)

	genePoolMiner(t)
	ctx.source = &normal1
	ctx.target = &normal1
	ctx.stakeAddValue = 100 * common.ZVC
	testStakeAddFromSelf(t, true)

	ctx.source = &normal1
	ctx.target = &minerPool
	testVote(t, false)
}

func TestVoteOther(t *testing.T) {
	setup()
	defer clear()
	geneGuardNodes(t)
	ctx.height = 10000
	ctx.source = &guardNode1
	ctx.target = &normal1
	testVote(t, true)
	vf, err := getVoteInfo(accountDB, *ctx.source)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if vf.Target != normal1 {
		t.Fatalf("except target is %v,but got target is %v", normal1, vf.Target)
	}

	ctx.source = &guardNode2
	ctx.target = &normal1
	testVote(t, true)

	ctx.source = &guardNode3
	ctx.target = &normal2
	testVote(t, true)

	ctx.source = &guardNode4
	ctx.target = &normal2
	testVote(t, true)

	totalTickets := getTickets(accountDB, normal1)
	if totalTickets != 2 {
		t.Fatalf("except 2 ,but got %d", totalTickets)
	}

	totalTickets = getTickets(accountDB, normal2)
	if totalTickets != 2 {
		t.Fatalf("except 2 ,but got %d", totalTickets)
	}
	ctx.height = adjustWeightPeriod / 2
	ctx.source = &guardNode1
	ctx.target = &normal2
	testVote(t, true)

	totalTickets = getTickets(accountDB, normal1)
	if totalTickets != 1 {
		t.Fatalf("except 1 ,but got %d", totalTickets)
	}

	totalTickets = getTickets(accountDB, normal2)
	if totalTickets != 3 {
		t.Fatalf("except 3 ,but got %d", totalTickets)
	}

}

func TestInvalid(t *testing.T) {
	setup()
	defer clear()
	genePoolMiner(t)
	accountDB.(*account.AccountDB).Commit(true)
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod+1000)
	miner, err := getMiner(accountDB, minerPool, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsInvalidMinerPool() {
		t.Fatalf("except miner is invalid miner pool,but got %v", miner.Identity)
	}
	totalTickets := getTickets(accountDB, minerPool)
	if totalTickets != 0 {
		t.Fatalf("except 0 ,but got %d", totalTickets)
	}
	isFullGuard := isInFullStakeGuardNode(accountDB, guardNode1)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode2)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode3)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode4)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode5)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode6)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode7)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
	isFullGuard = isInFullStakeGuardNode(accountDB, guardNode8)
	if isFullGuard {
		t.Fatalf("need not in full guard,but in guard")
	}
}

func TestNotFullGuardNode(t *testing.T) {
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)

	ctx.source = &guardNode1
	ctx.target = &guardNode1
	ctx.height = 6000000
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, ctx.height)
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsGuard() {
		t.Fatalf("except miner is guard,but got %v", miner.Type)
	}
	accountDB.(*account.AccountDB).Commit(true)
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod+1000)
	detail := getStakeDetail()
	if detail.MarkNotFullHeight != adjustWeightPeriod+1000 {
		t.Fatalf("except height = %v,but got %v", adjustWeightPeriod+1000, detail.MarkNotFullHeight)
	}
	accountDB.(*account.AccountDB).Commit(true)
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod+stakeBuffer+2000)

	detail = getStakeDetail()

	miner, err = getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsNormal() {
		t.Fatalf("except miner is normal,but got %v", miner.Type)
	}

	isFullGuard := isInFullStakeGuardNode(accountDB, *ctx.source)
	if isFullGuard {
		t.Fatalf("except not in full guard node,but got in full guard node")
	}
}

func TestScan(t *testing.T) {
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)
	ctx.source = &types.ExtractGuardNodes[0]
	testChangeFundMode(t, 0, true)
	accountDB.(*account.AccountDB).Commit(true)
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod/2+1000)
	fd, _ := getFundGuardNode(accountDB, types.ExtractGuardNodes[0])
	if !fd.isNormal() {
		t.Fatalf("except normal,but got %v", fd.Type)
	}
	scanned := hasScanedSixAddFiveFundGuards(accountDB)
	if !scanned {
		t.Fatalf("except true,but got false")
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsNormal() {
		t.Fatalf("except miner is normal,but got %v", miner.Type)
	}
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod+1000)
	scanned = hasScanedSixAddSixFundGuards(accountDB)
	if !scanned {
		t.Fatalf("except true,but got false")
	}
	for _, addr := range types.ExtractGuardNodes {
		fd, _ := getFundGuardNode(accountDB, addr)
		if !fd.isNormal() {
			t.Fatalf("except normal,but got %v", fd.FundModeType)
		}
	}
	ctx.source = &guardNode1
	ctx.target = &guardNode1
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, 0)
	miner, err = getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsGuard() {
		t.Fatalf("except miner is guard,but got %v", miner.Type)
	}

	accountDB.(*account.AccountDB).Commit(true)
	MinerManagerImpl.GuardNodesCheck(accountDB, adjustWeightPeriod+1000)

	isFullGuard := isInFullStakeGuardNode(accountDB, *ctx.source)
	if isFullGuard {
		t.Fatalf("except not in full guard node,but got in full guard node")
	}
}

func TestChangeFundMode(t *testing.T) {
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)
	ctx.source = &types.ExtractGuardNodes[0]
	testChangeFundMode(t, 0, true)

	ctx.source = &types.ExtractGuardNodes[0]
	testChangeFundMode(t, 2, false)

	ctx.source = &guardNode1
	testChangeFundMode(t, 1, false)

	ctx.height = 100
	ctx.source = &types.ExtractGuardNodes[0]
	testChangeFundMode(t, 1, true)

	ctx.height = adjustWeightPeriod/2 + 1
	ctx.source = &types.ExtractGuardNodes[0]
	testChangeFundMode(t, 1, false)
}

func TestFundApplyGuardNode(t *testing.T) {
	setup()
	defer clear()
	MinerManagerImpl.genFundGuardNodes(accountDB)
	ctx.source = &types.ExtractGuardNodes[0]
	ctx.target = &types.ExtractGuardNodes[0]

	testFullStakeFromSelf(t)
	var height uint64 = 0
	testApplyGuardNode(t, true, height)
	dt := getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod/2 {
		t.Fatalf("except height = %v,but got %v", adjustWeightPeriod/2, dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB, *ctx.source) {
		t.Fatalf("except in full stake guard node,but got nil")
	}
	fd, err := getFundGuardNode(accountDB, *ctx.source)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if fd == nil && !fd.isFullStakeGuardNode() {
		t.Fatalf("except full gurad node,but got not")
	}
}

func TestNormalApplyGuardNode(t *testing.T) {
	setup()
	defer clear()
	ctx.source = &src
	ctx.target = &src
	MinerManagerImpl.genFundGuardNodes(accountDB)
	testFullStakeFromSelf(t)
	var height uint64 = 0
	testApplyGuardNode(t, true, height)
	dt := getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod/2 {
		t.Fatalf("except height = %v,but got %v", adjustWeightPeriod/2, dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB, *ctx.source) {
		t.Fatalf("except in full stake guard node,but got nil")
	}
	testApplyGuardNode(t, true, height)
	dt = getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod {
		t.Fatalf("except height = %v,but got %v", adjustWeightPeriod, dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB, *ctx.source) {
		t.Fatalf("except in full stake guard node,but got nil")
	}
	testApplyGuardNode(t, false, height)
	dt = getStakeDetail()
	if dt.DisMissHeight != adjustWeightPeriod {
		t.Fatalf("except height = %v,but got %v", adjustWeightPeriod, dt.DisMissHeight)
	}
	if !isInFullStakeGuardNode(accountDB, *ctx.source) {
		t.Fatalf("except in full stake guard node,but got nil")
	}
	ctx.mType = types.MinerTypeVerify
	testApplyGuardNode(t, false, height)
	dt = getStakeDetail()
	if dt != nil {
		t.Fatalf("except nil,but got value")
	}
}

func TestFailVote(t *testing.T) {
	setup()
	defer clear()
	ctx.source = &guardNode1
	ctx.target = &guardNode1
	testFullStakeFromSelf(t)

	testVote(t, false)
}

func testStakeSelfProposal(t *testing.T) {
	ctx.stakeAddValue = 100 * common.ZVC
	ctx.source = &src
	ctx.target = &src
	ctx.mType = types.MinerTypeProposal
	testStakeAddFromSelf(t, true)

	total := getTotalStake()
	if total != 0 {
		t.Fatalf("except 0,but got %v", total)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsPrepare() {
		t.Fatalf("except prepared,but got %v", miner.Status)
	}
	if miner.Stake != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, dt.Value)
	}

	ctx.stakeAddValue = 400 * common.ZVC
	testStakeAddFromSelf(t, true)
	total = getTotalStake()
	if total != 500*common.ZVC {
		t.Fatalf("except %v,but got %v", 500*common.ZVC, total)
	}
	miner, err = getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsActive() {
		t.Fatalf("except prepared,but got %v", miner.Status)
	}
	if miner.Stake != 500*common.ZVC {
		t.Fatalf("except %v,but got %v", 500*common.ZVC, miner.Stake)
	}
	dt = getStakeDetail()
	if dt.Value != 500*common.ZVC {
		t.Fatalf("except %v,but got %v", 500*common.ZVC, dt.Value)
	}
}

func testStakeSelfVerify(t *testing.T) {
	ctx.stakeAddValue = 100 * common.ZVC
	ctx.source = &src
	ctx.target = &src
	ctx.mType = types.MinerTypeVerify
	testStakeAddFromSelf(t, true)
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsPrepare() {
		t.Fatalf("except prepared,but got %v", miner.Status)
	}
	if miner.Stake != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, dt.Value)
	}
	ctx.stakeAddValue = 400 * common.ZVC
	testStakeAddFromSelf(t, true)
	miner, err = getMiner(accountDB, *ctx.source, ctx.mType)

	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsActive() {
		t.Fatalf("except prepared,but got %v", miner.Status)
	}
	if miner.Stake != 500*common.ZVC {
		t.Fatalf("except %v,but got %v", 500*common.ZVC, miner.Stake)
	}
	dt = getStakeDetail()
	if dt.Value != 500*common.ZVC {
		t.Fatalf("except %v,but got %v", 500*common.ZVC, dt.Value)
	}

}

func testStakeAddFromSelf(t *testing.T, needSuccess bool) {
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
	if needSuccess {
		if err != nil {
			t.Fatalf("execute stake add msg error:%v", err)
		}
	} else {
		if err == nil {
			t.Fatalf("except err is nil,but got error")
		}
	}
}

func testStakeAbort(t *testing.T, success bool) {
	stakeAbortMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeMinerAbort, 0, []byte{byte(ctx.mType)})
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, stakeAbortMsg, 0)
	if success {
		if err != nil {
			t.Fatalf("execute stake abort msg error:%v", err)
		}
	} else {
		if err == nil {
			t.Fatalf("except got err,but got nil")
		}
	}
}

func testStakeFromOther(t *testing.T, success bool) {
	var mpks = &types.MinerPks{
		MType: ctx.mType,
	}
	bs, err := types.EncodePayload(mpks)
	if err != nil {
		t.Fatalf("encode payload error:%v", err)
	}
	stakeAddMsg := genMOperMsg(ctx.source, ctx.target, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)
	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if success {
		if err != nil {
			t.Fatalf("execute stake add msg error:%v", err)
		}
	} else {
		if err == nil {
			t.Fatalf("except got err,but got nil")
		}
	}
}

func testFullStakeFromOther(t *testing.T) {
	ctx.stakeAddValue = 2500000 * common.ZVC
	var mpks = &types.MinerPks{
		MType: ctx.mType,
	}
	bs, err := types.EncodePayload(mpks)
	if err != nil {
		t.Fatalf("encode payload error:%v", err)
	}
	stakeAddMsg := genMOperMsg(ctx.source, ctx.target, types.TransactionTypeStakeAdd, ctx.stakeAddValue, bs)
	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, 0)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if miner.Stake != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, dt.Value)
	}
}

func testFullStakeFromSelf(t *testing.T) {
	ctx.stakeAddValue = 2500000 * common.ZVC
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
	_, err = MinerManagerImpl.ExecuteOperation(accountDB, stakeAddMsg, ctx.height)
	if err != nil {
		t.Fatalf("execute stake add msg error:%v", err)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if !miner.IsActive() {
		t.Fatalf("except prepared,but got %v", miner.Status)
	}
	if miner.Stake != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, miner.Stake)
	}
	dt := getStakeDetail()
	if dt.Value != ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.stakeAddValue, dt.Value)
	}
	bla := accountDB.GetBalance(*ctx.source)
	if bla.Uint64() != ctx.originBalance-ctx.stakeAddValue {
		t.Fatalf("except %v,but got %v", ctx.originBalance-ctx.stakeAddValue, bla.Uint64())
	}
}

func geneGuardNodes(t *testing.T) {
	ctx.source = &guardNode1
	ctx.target = &guardNode1
	testFullStakeFromSelf(t)
	var height uint64 = 0
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode2
	ctx.target = &guardNode2
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode3
	ctx.target = &guardNode3
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode4
	ctx.target = &guardNode4
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode5
	ctx.target = &guardNode5
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode6
	ctx.target = &guardNode6
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode7
	ctx.target = &guardNode7
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)

	ctx.source = &guardNode8
	ctx.target = &guardNode8
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
}

func genePoolMiner(t *testing.T) {
	ctx.source = &guardNode1
	ctx.target = &guardNode1
	testFullStakeFromSelf(t)
	var height uint64 = 0
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode2
	ctx.target = &guardNode2
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode3
	ctx.target = &guardNode3
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode4
	ctx.target = &guardNode4
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode5
	ctx.target = &guardNode5
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode6
	ctx.target = &guardNode6
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode7
	ctx.target = &guardNode7
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	ctx.source = &guardNode8
	ctx.target = &guardNode8
	testFullStakeFromSelf(t)
	testApplyGuardNode(t, true, height)
	ctx.target = &minerPool
	testVote(t, true)

	totalTickets := getTickets(accountDB, minerPool)
	if totalTickets != 8 {
		t.Fatalf("except 8 ,but got %d", totalTickets)
	}
}

func testChangeFundMode(t *testing.T, tp byte, needSuccess bool) {
	var err error
	var fd *fundGuardNode
	applyMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeChangeFundGuardMode, 0, []byte{tp})
	_, err = MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, ctx.height)
	if !needSuccess {
		if err == nil {
			t.Fatalf("except got err,but got nil")
		}
	} else {
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		fd, err = getFundGuardNode(accountDB, *ctx.source)

		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if fd == nil {
			t.Fatalf("except vote false ,but got vote true")
		}
		if fd.FundModeType != common.FundModeType(tp) {
			t.Fatalf("except got %v,but got %v", common.FundModeType(tp), fd.FundModeType)
		}
	}

}

func testVote(t *testing.T, needSuccess bool) {
	applyMsg := genMOperMsg(ctx.source, ctx.target, types.TransactionTypeVoteMinerPool, 0, nil)
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, ctx.height)
	if err != nil && needSuccess {
		t.Fatalf("error = %v", err)
	}
	vote, err := getVoteInfo(accountDB, *ctx.source)
	if needSuccess {
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if vote == nil {
			t.Fatalf("except vote false ,but got vote true")
		}
	}
}

func testApplyGuardNode(t *testing.T, success bool, height uint64) {
	applyMsg := genMOperMsg(ctx.source, ctx.source, types.TransactionTypeApplyGuardMiner, 0, nil)
	_, err := MinerManagerImpl.ExecuteOperation(accountDB, applyMsg, height)
	if err != nil && success {
		t.Fatalf("execute stake add msg error:%v", err)
	}
	miner, err := getMiner(accountDB, *ctx.source, ctx.mType)
	if err != nil {
		t.Fatalf("error is %v", err)
	}
	if success {
		if !miner.IsGuard() {
			t.Fatalf("except %v,but got %v", types.MinerGuard, miner.Identity)
		}
		dt := getStakeDetail()
		if dt.Value != fullStake {
			t.Fatalf("except %v,but got %v", fullStake, dt.Value)
		}
	}
}

func getTotalStake() uint64 {
	return getProposalTotalStake(accountDB)
}

func getStakeDetail() *stakeDetail {
	detailKey := getDetailKey(*ctx.source, ctx.mType, types.Staked)
	detail, _ := getDetail(accountDB, *ctx.target, detailKey)
	return detail
}

//   Copyright (C) 2019 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
)

const (
	MinMinerStake                = 500 * common.ZVC     // minimal token of miner can stake
	initMaxMinerStakeAddAmount   = 1000000 * common.ZVC //init stake adjust amount of token
	maxMinerStakeAddAdjustPeriod = 30000000             //
	initMaxMinerStake            = 2500000 * common.ZVC
	MaxMinerStakeAdjustPeriod    = 10000000 // maximal token of miner can stake

	stakeAdjustTimes     = 12 // stake adjust times
	initMinerPoolTickets = 8  // init miner pool need tickets
	minMinerPoolTickets  = 1  // minimal miner pool need tickets
	minerPoolReduceCount = 2  // every reduce tickets count
)

type FundGuardType byte

const (
	normalNodeType FundGuardType = iota
	fullStakeGuardNodeType
	fundGuardNodeType
)

// minimumStake shows miner can stake the min value
func minimumStake() uint64 {
	return MinMinerStake
}

// maximumStake shows miner can stake the max value
func maximumStake(height uint64) uint64 {
	canStake := uint64(initMaxMinerStake)
	period := height / MaxMinerStakeAdjustPeriod
	if height > stakeAdjustTimes*MaxMinerStakeAdjustPeriod {
		period = stakeAdjustTimes
		height = stakeAdjustTimes * MaxMinerStakeAdjustPeriod
	}
	for i := uint64(0); i < period; i++ {
		canStake += initMaxMinerStakeAddAmount >> (i * MaxMinerStakeAdjustPeriod / maxMinerStakeAddAdjustPeriod)
	}
	return canStake
}

// miner pool valid tickets
func getValidTicketsByHeight(height uint64) uint64 {
	reduce := height / (adjustWeightPeriod * 3)
	reduceTickets := reduce * minerPoolReduceCount
	if initMinerPoolTickets <= reduceTickets {
		return minMinerPoolTickets
	}
	return initMinerPoolTickets - reduceTickets
}

type fundGuardNode struct {
	Type         FundGuardType
	Height       uint64              // Operation height
	FundModeType common.FundModeType //default 6+6
}

type fundGuardNodeDetail struct {
	Address common.Address
	*fundGuardNode
}

type stakeDetail struct {
	Value             uint64 // Stake operation amount
	Height            uint64 // Operation height
	DisMissHeight     uint64 // Stake end height
	MarkNotFullHeight uint64 // Mark the height when stake is not full
}

type voteInfo struct {
	Target common.Address // vote target addr,default empty adress
	Height uint64         // Operation height
}

func NewFundGuardNode() *fundGuardNode {
	return &fundGuardNode{
		Type:         fundGuardNodeType,
		Height:       0,
		FundModeType: common.SIXAddSix,
	}
}
func (f *fundGuardNode) isSixAddSix() bool {
	return f.FundModeType == common.SIXAddSix
}

func (f *fundGuardNode) isSixAddFive() bool {
	return f.FundModeType == common.SIXAddFive
}

func (f *fundGuardNode) isFundGuard() bool {
	return f.Type == fundGuardNodeType
}

func (f *fundGuardNode) isNormal() bool {
	return f.Type == normalNodeType
}

func (f *fundGuardNode) isFullStakeGuardNode() bool {
	return f.Type == fullStakeGuardNodeType
}

func checkCanVote(voteHeight, currentHeight uint64) bool {
	if voteHeight == 0 {
		return true
	}
	voteRound := voteHeight / (adjustWeightPeriod / 2)
	currentRound := currentHeight / (adjustWeightPeriod / 2)

	return currentRound > voteRound
}

func newVoteInfo() *voteInfo {
	return &voteInfo{
		Target: common.Address{},
		Height: 0,
	}
}

func getDetailKey(address common.Address, typ types.MinerType, status types.StakeStatus) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(common.PrefixDetail)
	buf.Write(address.Bytes())
	buf.WriteByte(byte(typ))
	buf.WriteByte(byte(status))
	return buf.Bytes()
}

func parseDetailKey(key []byte) (common.Address, types.MinerType, types.StakeStatus) {
	reader := bytes.NewReader(key)

	detail := make([]byte, len(common.PrefixDetail))
	n, err := reader.Read(detail)
	if err != nil || n != len(common.PrefixDetail) {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	addrBytes := make([]byte, len(common.Address{}))
	n, err = reader.Read(addrBytes)
	if err != nil || n != len(addrBytes) {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	mtByte, err := reader.ReadByte()
	if err != nil {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	stByte, err := reader.ReadByte()
	if err != nil {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	return common.BytesToAddress(addrBytes), types.MinerType(mtByte), types.StakeStatus(stByte)
}

func setPks(miner *types.Miner, pks *types.MinerPks) *types.Miner {
	if len(pks.Pk) > 0 {
		miner.PublicKey = pks.Pk
	}
	if len(pks.VrfPk) > 0 {
		miner.VrfPublicKey = pks.VrfPk
	}

	return miner
}

// checkCanActivate if status can be set to types.MinerStatusActive
func checkCanActivate(miner *types.Miner) bool {
	// pks not completed
	if !miner.PksCompleted() {
		return false
	}
	// If the stake up to the lower bound, then activate the miner
	return checkLowerBound(miner)
}

func checkUpperBound(miner *types.Miner, height uint64) bool {
	return miner.Stake <= maximumStake(height)
}

func isFullStake(stake, height uint64) bool {
	return stake == maximumStake(height)
}

func checkMinerPoolUpperBound(miner *types.Miner, height uint64) bool {
	return miner.Stake <= getFullMinerPoolStake(height)
}

func getFullMinerPoolStake(height uint64) uint64 {
	return maximumStake(height) * getValidTicketsByHeight(height)
}

func checkLowerBound(miner *types.Miner) bool {
	return miner.Stake >= minimumStake()
}

func getMinerKey(typ types.MinerType) []byte {
	buf := bytes.NewBuffer(common.PrefixMiner)
	buf.WriteByte(byte(typ))
	return buf.Bytes()
}

func getPoolKey(prefix []byte, address common.Address) []byte {
	buf := bytes.NewBuffer(prefix)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func getFundGuardKey(prefix []byte, address common.Address) []byte {
	buf := bytes.NewBuffer(prefix)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func getMiner(db types.AccountDB, address common.Address, mType types.MinerType) (*types.Miner, error) {
	data := db.GetData(address, getMinerKey(mType))
	if data != nil && len(data) > 0 {
		var miner types.Miner
		err := msgpack.Unmarshal(data, &miner)
		if err != nil {
			return nil, err
		}
		return &miner, nil
	}
	return nil, nil
}

func setMiner(db types.AccountDB, miner *types.Miner) error {
	bs, err := msgpack.Marshal(miner)
	if err != nil {
		return err
	}
	db.SetData(common.BytesToAddress(miner.ID), getMinerKey(miner.Type), bs)
	return nil
}

func parseDetail(value []byte) (*stakeDetail, error) {
	var detail stakeDetail
	err := msgpack.Unmarshal(value, &detail)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func getDetail(db types.AccountDB, address common.Address, detailKey []byte) (*stakeDetail, error) {
	data := db.GetData(address, detailKey)
	if data != nil && len(data) > 0 {
		return parseDetail(data)
	}
	return nil, nil
}

func setDetail(db types.AccountDB, address common.Address, detailKey []byte, sd *stakeDetail) error {
	bs, err := msgpack.Marshal(sd)
	if err != nil {
		return err
	}
	db.SetData(address, detailKey, bs)
	return nil
}

func getTotalTickets(db types.AccountDB, key []byte) uint64 {
	totalTicketsBytes := db.GetData(common.MinerPoolTicketsAddr, key)
	totalTickets := uint64(0)
	if len(totalTicketsBytes) > 0 {
		totalTickets = common.ByteToUInt64(totalTicketsBytes)
	}
	return totalTickets
}

func getTicketsKey(address common.Address) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(common.KeyTickets)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func getProposalTotalStake(db types.AccountDB) uint64 {
	totalStakeBytes := db.GetData(common.MinerPoolAddr, common.KeyPoolProposalTotalStake)
	totalStake := uint64(0)
	if len(totalStakeBytes) > 0 {
		totalStake = common.ByteToUInt64(totalStakeBytes)
	}
	return totalStake
}

func setFundGuardNode(db types.AccountDB, address common.Address, fn *fundGuardNode) error {
	bs, err := msgpack.Marshal(fn)
	if err != nil {
		return err
	}
	key := getFundGuardKey(common.KeyGuardNodes, address)
	db.SetData(common.FundGuardNodeAddr, key, bs)
	return nil
}

func addFundGuardPool(db types.AccountDB, address common.Address) error {
	fg := NewFundGuardNode()
	err := setFundGuardNode(db, address, fg)
	return err
}

func getFundGuardNode(db types.AccountDB, address common.Address) (*fundGuardNode, error) {
	key := getFundGuardKey(common.KeyGuardNodes, address)
	bts := db.GetData(common.FundGuardNodeAddr, key)
	if bts == nil {
		return nil, nil
	}
	var fn fundGuardNode
	err := msgpack.Unmarshal(bts, &fn)
	if err != nil {
		return nil, err
	}
	return &fn, nil
}

func hasScanedSixAddFiveFundGuards(db types.AccountDB) bool {
	bts := db.GetData(common.ScanAllFundGuardStatusAddr, common.KeyScanSixAddFiveNodes)
	if bts == nil {
		return false
	}
	return true
}

func markScanedSixAddFiveFundGuards(db types.AccountDB) {
	db.SetData(common.ScanAllFundGuardStatusAddr, common.KeyScanSixAddFiveNodes, []byte{1})
}

func hasScanedSixAddSixFundGuards(db types.AccountDB) bool {
	bts := db.GetData(common.ScanAllFundGuardStatusAddr, common.KeyScanSixAddSixNodes)
	if bts == nil {
		return false
	}
	return true
}

func markScanedSixAddSixFundGuards(db types.AccountDB) {
	db.SetData(common.ScanAllFundGuardStatusAddr, common.KeyScanSixAddSixNodes, []byte{1})
}

func updateFundGuardMode(db types.AccountDB, fn *fundGuardNode, address common.Address, mode common.FundModeType, height uint64) error {
	fn.Height = height
	fn.FundModeType = mode
	err := setFundGuardNode(db, address, fn)
	return err
}

func updateFundGuardPoolStatus(db types.AccountDB, address common.Address, fnType FundGuardType, height uint64) error {
	fn, err := getFundGuardNode(db, address)
	if err != nil {
		return nil
	}
	if fn == nil {
		return fmt.Errorf("fund  guard info is nil,addr is %s", address.String())
	}
	fn.Height = height
	fn.Type = fnType
	err = setFundGuardNode(db, address, fn)
	return err
}

func addFullStakeGuardPool(db types.AccountDB, address common.Address) {
	key := getFundGuardKey(common.KeyGuardNodes, address)
	db.SetData(common.FullStakeGuardNodeAddr, key, []byte{1})
}

func isInFullStakeGuardNode(db types.AccountDB, address common.Address) bool {
	key := getFundGuardKey(common.KeyGuardNodes, address)
	return db.GetData(common.FullStakeGuardNodeAddr, key) != nil
}

func removeFullStakeGuardNodeFromPool(db types.AccountDB, address common.Address) {
	key := getFundGuardKey(common.KeyGuardNodes, address)
	db.RemoveData(common.FullStakeGuardNodeAddr, key)
}

func setVoteInfo(db types.AccountDB, address common.Address, vf *voteInfo) error {
	bs, err := msgpack.Marshal(vf)
	if err != nil {
		return err
	}
	db.SetData(address, common.KeyVote, bs)
	return nil
}

func delVoteInfo(db types.AccountDB, address common.Address) {
	db.RemoveData(address, common.KeyVote)
}

func initVoteInfo(db types.AccountDB, address common.Address) (*voteInfo, error) {
	vote := newVoteInfo()
	bs, err := msgpack.Marshal(vote)
	if err != nil {
		return nil, err
	}
	db.SetData(address, common.KeyVote, bs)
	return vote, nil
}

func getVoteInfo(db types.AccountDB, address common.Address) (*voteInfo, error) {
	data := db.GetData(address, common.KeyVote)
	if data == nil {
		return nil, nil
	}
	var vf voteInfo
	err := msgpack.Unmarshal(data, &vf)
	if err != nil {
		return nil, err
	}
	return &vf, nil
}

func guardNodeExpired(db types.AccountDB, address common.Address, height uint64, isFundGuardNode bool) error {
	miner, err := getMiner(db, address, types.MinerTypeProposal)
	if err != nil {
		return err
	}
	if miner == nil {
		return fmt.Errorf("guard invalid find miner is nil,addr is %s", address.String())
	}
	miner.UpdateIdentity(types.MinerNormal, height)
	err = setMiner(db, miner)
	if err != nil {
		return err
	}
	// fund guard node is not in this pool,only full stake node in this pool
	if !isFundGuardNode {
		removeFullStakeGuardNodeFromPool(db, address)
		log.CoreLogger.Infof("full stake expired,addr is %s,height = %v", address.String(), height)
	} else {
		log.CoreLogger.Infof("fund stake expired,addr is %s,height = %v", address.String(), height)
	}
	vf, err := getVoteInfo(db, address)
	if err != nil {
		return err
	}
	if vf != nil {
		delVoteInfo(db, address)
		var empty = common.Address{}
		if vf.Target != empty {
			mop := newReduceTicketsOp(db, vf.Target, height)
			ret := mop.Transition()
			if ret.err != nil {
				return ret.err
			}
		}
	}
	return nil
}

func subProposalTotalStake(db types.AccountDB, subStake uint64) {
	totalStake := getProposalTotalStake(db)
	// Must not happen
	if totalStake < subStake {
		panic("total stake less than sub stake")
	}
	newTotalStake := totalStake - subStake
	db.SetData(common.MinerPoolAddr, common.KeyPoolProposalTotalStake, common.Uint64ToByte(newTotalStake))
	log.CoreLogger.Infof("total stake changed,reduce total stake,total stake is %v",newTotalStake)
}

func removeFromPool(db types.AccountDB, minerType types.MinerType, address common.Address, stake uint64) {
	var key []byte
	if types.IsProposalRole(minerType) {
		key = getPoolKey(common.PrefixPoolProposal, address)
		totalStakeBytes := db.GetData(common.MinerPoolAddr, common.KeyPoolProposalTotalStake)
		totalStake := uint64(0)
		if len(totalStakeBytes) > 0 {
			totalStake = common.ByteToUInt64(totalStakeBytes)
		}
		if totalStake < stake {
			panic(fmt.Errorf("totalStake less than stake: %v %v", totalStake, stake))
		}
		newTotalStake := totalStake - stake
		db.SetData(common.MinerPoolAddr, common.KeyPoolProposalTotalStake, common.Uint64ToByte(newTotalStake))
		log.CoreLogger.Infof("total stake changed,remove from pool,total stake is %v",newTotalStake)
	} else if types.IsVerifyRole(minerType) {
		key = getPoolKey(common.PrefixPoolVerifier, address)

	}
	log.CoreLogger.Infof("remove from pool,addr is %s,type is %d", address.String(), minerType)
	db.RemoveData(common.MinerPoolAddr, key)
}

func addProposalTotalStake(db types.AccountDB, addStake uint64) {
	totalStake := getProposalTotalStake(db)
	// Must not happen
	if addStake+totalStake < totalStake {
		panic(fmt.Errorf("total stake overflow:%v %v", addStake, totalStake))
	}
	newTotalStake := addStake+totalStake
	db.SetData(common.MinerPoolAddr, common.KeyPoolProposalTotalStake, common.Uint64ToByte(newTotalStake))
	log.CoreLogger.Infof("total stake changed,add stake,current total stake is %v",newTotalStake)
}

func removeDetail(db types.AccountDB, address common.Address, detailKey []byte) {
	db.RemoveData(address, detailKey)
}

func initMiner(op *stakeAddOp) *types.Miner {
	miner := &types.Miner{
		ID:          op.addTarget.Bytes(),
		Stake:       op.value,
		ApplyHeight: op.height,
		Type:        op.minerType,
		Status:      types.MinerStatusPrepare,
	}
	return miner
}

func addToPool(db types.AccountDB, minerType types.MinerType, address common.Address, addStake uint64) {
	var key []byte
	if types.IsProposalRole(minerType) {
		key = getPoolKey(common.PrefixPoolProposal, address)
		addProposalTotalStake(db, addStake)
	} else if types.IsVerifyRole(minerType) {
		key = getPoolKey(common.PrefixPoolVerifier, address)

	}
	db.SetData(common.MinerPoolAddr, key, []byte{1})
}

func addTicket(db types.AccountDB, address common.Address) uint64 {
	key := getTicketsKey(address)
	totalTickets := getTotalTickets(db, key)
	totalTickets += 1
	db.SetData(common.MinerPoolTicketsAddr, key, common.Uint64ToByte(totalTickets))
	return totalTickets
}

func subTicket(db types.AccountDB, address common.Address) uint64 {
	key := getTicketsKey(address)
	totalTickets := getTotalTickets(db, key)
	if totalTickets < 1 {
		totalTickets = 0
	} else {
		totalTickets -= 1
	}
	db.SetData(common.MinerPoolTicketsAddr, key, common.Uint64ToByte(totalTickets))
	return totalTickets
}

func getTickets(db types.AccountDB, address common.Address) uint64 {
	key := getTicketsKey(address)
	return getTotalTickets(db, key)
}

func processVote(op *voteMinerPoolOp, vf *voteInfo) (error, bool,types.ReceiptStatus) {
	var err error
	if vf == nil {
		vf, err = initVoteInfo(op.accountDB, op.source)
		if err != nil {
			return err, false,types.RSFail
		}
	}

	oldTarget := vf.Target
	// set vote false
	err = voteMinerPool(op.accountDB, vf, op.source, op.targetAddr, op.height)
	if err != nil {
		return err, false,types.RSFail
	}
	var totalTickets uint64 = 0
	var empty = common.Address{}
	// vote target is old target
	if oldTarget == op.targetAddr && oldTarget != empty {
		totalTickets = getTickets(op.accountDB, op.targetAddr)
	} else {
		if oldTarget != empty {
			//reduce ticket first
			mop := newReduceTicketsOp(op.accountDB, oldTarget, op.height)
			ret := mop.Transition()
			if ret.err != nil {
				return ret.err, false,ret.transitionStatus
			}
		}
		// add tickets count
		totalTickets = addTicket(op.accountDB, op.targetAddr)
	}
	log.CoreLogger.Infof("vote success,source = %s,target =%s,current tickets = %d,height = %v", op.source, op.targetAddr, totalTickets, op.height)
	isFull := isFullTickets(totalTickets, op.height)
	return nil, isFull,types.RSSuccess
}

func isFullTickets(totalTickets uint64, height uint64) bool {
	needTickets := getValidTicketsByHeight(height)
	return totalTickets >= needTickets
}

func voteMinerPool(db types.AccountDB, vf *voteInfo, source, targetAddress common.Address, height uint64) error {
	vf.Height = height
	vf.Target = targetAddress
	bs, err := msgpack.Marshal(vf)
	if err != nil {
		return err
	}
	db.SetData(source, common.KeyVote, bs)
	return nil
}

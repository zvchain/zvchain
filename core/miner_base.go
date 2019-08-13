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
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

var (
	prefixMiner               = []byte("minfo")
	prefixDetail              = []byte("dt")
	prefixPoolProposal        = []byte("p")
	prefixPoolVerifier        = []byte("v")
	keyPoolProposalTotalStake = []byte("totalstake")
	keyVote 				  = []byte("votekey")
	keyTickets 				  = []byte("tickets")
	keyIndexOfGuardMiner 	  = []byte("gmindex")
	keyGuardMinerInfos 	  	  = []byte("gminfos")
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
	reduce := height / threeYearBlocks
	needTickets := initMinerPoolTickets - (reduce * minerPoolReduceCount)
	if needTickets < minMinerPoolTickets {
		return minMinerPoolTickets
	}
	return needTickets
}

// Special account address
// Need to access by AccountDBTS for concurrent situations
var (
	minerPoolAddr           = common.BigToAddress(big.NewInt(1)) // The Address storing total stakes of each roles and addresses of all active nodes
	rewardStoreAddr         = common.BigToAddress(big.NewInt(2)) // The Address storing the block hash corresponding to the reward transaction
    minerPoolTicketsAddr    = common.BigToAddress(big.NewInt(3)) // The Address storing all miner pool tickets
	guardMinerNodeIndexAddr = common.BigToAddress(big.NewInt(4)) // The Address storing current guard miner node index
	guardMinerNodeInfoAddr  = common.BigToAddress(big.NewInt(5)) // The Address storing all guard miners length and beginIndex

)

var punishmentDetailAddr = common.BigToAddress(big.NewInt(0))

type guardMinerInfo struct{
	BeginIndex  uint64
	Len         uint64
}

func newGuardMinerInfo()*guardMinerInfo{
	return &guardMinerInfo{
		BeginIndex:0,
		Len:0,
	}
}

type stakeDetail struct {
	Value               uint64  // Stake operation amount
	Height              uint64  // Operation height
	DisMissHeight       uint64  // Stake end height
	MarkNotFullHeight   uint64 // mark the height when stake is not full
}

type voteInfo struct {
	Target       common.Address
	Height		 uint64
	Last         byte
	UpdateHeight uint64
}

func NewVoteInfo(height uint64)*voteInfo{
	return &voteInfo{
		Target:common.Address{},
		Height:height,
		UpdateHeight:height,
		Last:1,
	}
}

func getGuardMinerIndexKey(index uint64)[]byte{
	buf := bytes.NewBuffer([]byte{})
	buf.Write(keyIndexOfGuardMiner)
	buf.Write(common.UInt64ToByte(index))
	return buf.Bytes()
}

func getDetailKey(address common.Address, typ types.MinerType, status types.StakeStatus) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(prefixDetail)
	buf.Write(address.Bytes())
	buf.WriteByte(byte(typ))
	buf.WriteByte(byte(status))
	return buf.Bytes()
}

func parseDetailKey(key []byte) (common.Address, types.MinerType, types.StakeStatus) {
	reader := bytes.NewReader(key)

	detail := make([]byte, len(prefixDetail))
	n, err := reader.Read(detail)
	if err != nil || n != len(prefixDetail) {
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

func isFullStake(stake, height uint64)bool{
	return stake == maximumStake(height)
}

func checkMinerPoolUpperBound(miner *types.Miner, height uint64) bool {
	return miner.Stake <= maximumStake(height)*getValidTicketsByHeight(height)
}

func checkLowerBound(miner *types.Miner) bool {
	return miner.Stake >= minimumStake()
}

func getMinerKey(typ types.MinerType) []byte {
	buf := bytes.NewBuffer(prefixMiner)
	buf.WriteByte(byte(typ))
	return buf.Bytes()
}

func getPoolKey(prefix []byte, address common.Address) []byte {
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

func setMiner(db types.AccountDB,miner *types.Miner)error{
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

func setDetail(db types.AccountDB,address common.Address, detailKey []byte, sd *stakeDetail) error {
	bs, err := msgpack.Marshal(sd)
	if err != nil {
		return err
	}
	db.SetData(address, detailKey, bs)
	return nil
}


func getTotalTickets(db types.AccountDBTS,key []byte)uint64{
	totalTicketsBytes := db.GetDataSafe(minerPoolTicketsAddr, key)
	totalTickets := uint64(0)
	if len(totalTicketsBytes) > 0 {
		totalTickets = common.ByteToUInt64(totalTicketsBytes)
	}
	return totalTickets
}


func getTicketsKey(address common.Address) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(keyTickets)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func getProposalTotalStake(db types.AccountDBTS) uint64 {
	totalStakeBytes := db.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
	totalStake := uint64(0)
	if len(totalStakeBytes) > 0 {
		totalStake = common.ByteToUInt64(totalStakeBytes)
	}
	return totalStake
}

type baseOperation struct {
	*transitionContext
	minerType types.MinerType
	minerPool types.AccountDBTS
	db        types.AccountDB
	msg       types.MinerOperationMessage
	height    uint64
}

func newBaseOperation(db types.AccountDB, msg types.MinerOperationMessage, height uint64, tc *transitionContext) *baseOperation {
	return &baseOperation{
		transitionContext: tc,
		db:                db,
		minerPool:         db.AsAccountDBTS(),
		msg:               msg,
		height:            height,
	}
}

func (op *baseOperation) opProposalRole() bool {
	return types.IsProposalRole(op.minerType)
}
func (op *baseOperation) opVerifyRole() bool {
	return types.IsVerifyRole(op.minerType)
}


func setVoteInfo(db types.AccountDB,address common.Address,vf *voteInfo)error{
	bs, err := msgpack.Marshal(vf)
	if err != nil {
		return err
	}
	db.SetData(address, keyVote,bs)
	return nil
}

func delVoteInfo(db types.AccountDB,address common.Address){
	db.RemoveData(address,keyVote)
}

func initVoteInfo(db types.AccountDB,address common.Address)error{
	vote := NewVoteInfo(0)
	bs, err := msgpack.Marshal(vote)
	if err != nil {
		return err
	}
	db.SetData(address, keyVote,bs)
	return nil
}

func getVoteInfo(db types.AccountDB,address common.Address)(*voteInfo,error){
	data := db.GetData(address,keyVote)
	if data == nil{
		return nil,nil
	}
	var vf voteInfo
	err := msgpack.Unmarshal(data, &vf)
	if err != nil {
		return nil,err
	}
	return &vf,nil
}

func (op *baseOperation) addTicket(address common.Address)uint64{
	key := getTicketsKey(address)
	totalTickets := getTotalTickets(op.minerPool,key)
	totalTickets+=1
	op.minerPool.SetDataSafe(minerPoolTicketsAddr, key, common.Uint64ToByte(totalTickets))
	return totalTickets
}

func (op *baseOperation) getTickets(address common.Address)uint64{
	key := getTicketsKey(address)
	return getTotalTickets(op.minerPool,key)
}

func (op *baseOperation) subTicket(address common.Address)uint64{
	key := getTicketsKey(address)
	totalTickets := getTotalTickets(op.minerPool,key)
	if totalTickets < 1 {
		totalTickets = 0
	}else{
		totalTickets -=1
	}
	op.minerPool.SetDataSafe(minerPoolTicketsAddr, key, common.Uint64ToByte(totalTickets))
	return totalTickets
}

func getGuardMinerNodeInfo(db types.AccountDBTS)(*guardMinerInfo,error){
	bytes := db.GetDataSafe(guardMinerNodeInfoAddr,keyGuardMinerInfos)
	var gm guardMinerInfo
	var err error
	if bytes == nil{
		gm = *newGuardMinerInfo()
	}else{
		err = msgpack.Unmarshal(bytes,&gm)
		if err != nil{
			return nil,err
		}
	}
	return &gm,nil
}

func setGuardMinerNodeInfo(db types.AccountDBTS,gm *guardMinerInfo)error{
	bytes,err := msgpack.Marshal(gm)
	if err != nil{
		return err
	}
	db.SetDataSafe(guardMinerNodeInfoAddr,keyGuardMinerInfos,bytes)
	return nil
}

func delGuardMinerIndex(db types.AccountDBTS,index uint64){
	indexKey := getGuardMinerIndexKey(index)
	db.RemoveDataSafe(guardMinerNodeIndexAddr,indexKey)
}

func getGuardMinerIndex(db types.AccountDBTS,index uint64)(*common.Address,error){
	indexKey := getGuardMinerIndexKey(index)
	bytes := db.GetDataSafe(guardMinerNodeIndexAddr,indexKey)
	if bytes == nil{
		return nil,nil
	}
	addr := common.BytesToAddress(bytes)
	return &addr,nil
}

func setGuardMinerIndex(db types.AccountDBTS,address common.Address,index uint64)error{
	indexKey := getGuardMinerIndexKey(index)
	db.SetDataSafe(guardMinerNodeIndexAddr,indexKey,address.Bytes())
	return nil
}

func (op *baseOperation)addGuardMinerInfo(address common.Address,disMissHeight uint64)error{
	gm,err := getGuardMinerNodeInfo(op.minerPool)
	if err != nil{
		return err
	}
	gm.Len+=1
	err = setGuardMinerNodeInfo(op.minerPool,gm)
	if err != nil{
		return err
	}
	err = setGuardMinerIndex(op.minerPool,address,gm.BeginIndex)
	if err != nil{
		return err
	}
	return nil
}


func (op *baseOperation) voteMinerPool(source common.Address,targetAddress common.Address)error{
	vf,err := getVoteInfo(op.db,source)
	if err != nil{
		return err
	}
	if vf!= nil && vf.Last > 0 {
		vf.Last = 0
		vf.UpdateHeight = op.height
		vf.Target = targetAddress

		bs, err := msgpack.Marshal(vf)
		if err != nil {
			return err
		}
		op.db.SetData(source,keyVote,bs)
	}
	return nil
}


func (op *baseOperation) isFullTickets(address common.Address,totalTickets uint64)bool{
	needTickets := getValidTicketsByHeight(op.height)
	return totalTickets >= needTickets
}


func (op *baseOperation) addToPool(address common.Address, addStake uint64) {
	var key []byte
	if op.opProposalRole() {
		key = getPoolKey(prefixPoolProposal, address)
		op.addProposalTotalStake(addStake)
	} else if op.opVerifyRole() {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.minerPool.SetDataSafe(minerPoolAddr, key, []byte{1})
}

func (op *baseOperation) addProposalTotalStake(addStake uint64) {
	totalStake := getProposalTotalStake(op.minerPool)
	// Must not happen
	if addStake+totalStake < totalStake {
		panic(fmt.Errorf("total stake overflow:%v %v", addStake, totalStake))
	}
	op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(addStake+totalStake))
}

func guardNodeExpired(db types.AccountDB,address common.Address,height uint64){
	miner,err := getMiner(db, address, types.MinerTypeProposal)
	if err != nil{
		Logger.Error(err)
		return
	}
	if miner == nil{
		Logger.Errorf("guard invalid find miner is nil,addr is %s",address.Hex())
		return
	}
	miner.UpdateIdentity(types.MinerNormal,height)
	err = setMiner(db,miner)
	if err != nil{
		Logger.Error(err)
		return
	}
	vf,err := getVoteInfo(db,address)
	if err != nil{
		Logger.Error(err)
		return
	}
	if vf == nil{
		Logger.Errorf("find guard node vote info is nil,addr is %s",address.Hex())
		return
	}
	delVoteInfo(db,address)
	var empty = common.Address{}
	if vf.Target != empty{
		mop:=newReduceTicketsOp(db,vf.Target,address,height)
		mop.ParseTransaction()
		ret := mop.Transition()
		if ret.err!= nil{
			Logger.Error(ret.err)
		}
	}
}

func (op *baseOperation) subProposalTotalStake(subStake uint64) {
	totalStake := getProposalTotalStake(op.minerPool)
	// Must not happen
	if totalStake < subStake {
		panic("total stake less than sub stake")
	}
	op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-subStake))
}

func (op *baseOperation) removeFromPool(address common.Address, stake uint64) {
	var key []byte
	if op.opProposalRole() {
		key = getPoolKey(prefixPoolProposal, address)
		totalStakeBytes := op.minerPool.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
		totalStake := uint64(0)
		if len(totalStakeBytes) > 0 {
			totalStake = common.ByteToUInt64(totalStakeBytes)
		}
		if totalStake < stake {
			panic(fmt.Errorf("totalStake less than stake: %v %v", totalStake, stake))
		}
		op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-stake))
	} else if op.opVerifyRole() {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.minerPool.RemoveDataSafe(minerPoolAddr, key)
}

func (op *baseOperation) getDetail(address common.Address, detailKey []byte) (*stakeDetail, error) {
	return getDetail(op.db, address, detailKey)
}

func (op *baseOperation) setDetail(address common.Address, detailKey []byte, sd *stakeDetail) error {
	err := setDetail(op.db,address,detailKey,sd)
	return err
}

func (op *baseOperation) removeDetail(address common.Address, detailKey []byte) {
	op.db.RemoveData(address, detailKey)
}

func (op *baseOperation) getMiner(address common.Address) (*types.Miner, error) {
	return getMiner(op.db, address, op.minerType)
}

func (op *baseOperation) setMiner(miner *types.Miner) error {
	return setMiner(op.db,miner)
}

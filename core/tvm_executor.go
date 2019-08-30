//   Copyright (C) 2018 ZVChain
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
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/zvchain/zvchain/core/group"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/tvm"
)

const (
	TransactionGasCost     uint64 = 400
	CodeBytePrice                 = 19073 //1.9073486328125
	CodeBytePricePrecision        = 10000
	MaxCastBlockTime              = time.Second * 2
	adjustGasPricePeriod          = 30000000
	adjustGasPriceTimes           = 3
	initialMinGasPrice            = 500
)

var (
	ProposerPackageTime = MaxCastBlockTime
	GasLimitForPackage  = uint64(GasLimitPerBlock)
	IgnoreVmCall        = false
)

var (
	errGasPriceTooLow   = fmt.Errorf("gas price too low")
	errGasTooLow        = fmt.Errorf("gas too low")
	errBalanceNotEnough = fmt.Errorf("balance not enough")
	errNonceError       = fmt.Errorf("nonce error")
)

// stateTransition define some functions on state transition
type stateTransition interface {
	ParseTransaction() error // Parse the input transaction
	Transition() *result     // Do the transition
	GasUsed() *big.Int       // Total gas use during the transition
}

type checkpointUpdator interface {
	updateVotes(db types.AccountDB, bh *types.BlockHeader)
}

// Process after all transactions executed
type statePostProcessor func(db types.AccountDB, bh *types.BlockHeader)

func newStateTransition(db types.AccountDB, tx *types.Transaction, bh *types.BlockHeader) stateTransition {
	base := newTransitionContext(db, tx, bh, bh.Height)
	base.intrinsicGasUsed = intrinsicGas(tx)
	base.gasUsed = base.intrinsicGasUsed
	return getOpByType(base, tx.Type)
}

func getOpByType(base *transitionContext, txType int8) stateTransition {
	switch txType {
	case types.TransactionTypeTransfer:
		return &txTransfer{transitionContext: base}
	case types.TransactionTypeContractCreate:
		return &contractCreator{transitionContext: base}
	case types.TransactionTypeContractCall:
		return &contractCaller{transitionContext: base}
	case types.TransactionTypeStakeAdd:
		return &stakeAddOp{transitionContext: base}
	case types.TransactionTypeMinerAbort:
		return &minerAbortOp{transitionContext: base}
	case types.TransactionTypeVoteMinerPool:
		return &voteMinerPoolOp{transitionContext: base}
	case types.TransactionTypeStakeReduce:
		return &stakeReduceOp{transitionContext: base}
	case types.TransactionTypeApplyGuardMiner:
		return &applyGuardMinerOp{transitionContext: base}
	case types.TransactionTypeStakeRefund:
		return &stakeRefundOp{transitionContext: base}
	case types.TransactionTypeChangeFundGuardMode:
		return &changeFundGuardMode{transitionContext: base}
	case types.TransactionTypeGroupPiece, types.TransactionTypeGroupMpk, types.TransactionTypeGroupOriginPiece:
		return &groupOperator{transitionContext: base}
	default:
		return &unSupported{typ: txType}
	}
}

type transitionContext struct {
	accountDB        types.AccountDB
	bh               *types.BlockHeader
	msg              types.TxMessage
	intrinsicGasUsed *big.Int
	gasUsed          *big.Int
	height           uint64
}

func (tc *transitionContext) GasUsed() *big.Int {
	return tc.gasUsed
}

type result struct {
	cumulativeGasUsed *big.Int
	transitionStatus  types.ReceiptStatus
	err               error
	logs              []*types.Log   // Generated when calls contract
	contractAddress   common.Address // Generated when creates contract
}

func newResult() *result {
	return &result{
		transitionStatus:  types.RSSuccess,
		cumulativeGasUsed: new(big.Int).SetUint64(0),
	}
}

func (r *result) setError(err error, status types.ReceiptStatus) {
	r.err = err
	r.transitionStatus = status
}

func newTransitionContext(db types.AccountDB, tx types.TxMessage, bh *types.BlockHeader, height uint64) *transitionContext {
	return &transitionContext{accountDB: db, msg: tx, bh: bh, height: height}
}

func checkState(db types.AccountDB, tx *types.Transaction, height uint64) error {
	if !validGasPrice(&tx.GasPrice.Int, height) {
		return errGasPriceTooLow
	}
	gasLimitFee := new(types.BigInt).Mul(tx.GasLimit.Value(), tx.GasPrice.Value())
	if !db.CanTransfer(*tx.Source, gasLimitFee) {
		return errBalanceNotEnough
	}
	if !validateNonce(db, tx) {
		return errNonceError
	}
	return nil
}

// unSupported encounters an unknown type
type unSupported struct {
	typ int8
}

func (op *unSupported) Operation() error {
	return fmt.Errorf("unSupported tx type %v", op.typ)
}

func (op *unSupported) Source() common.Address {
	return common.Address{}
}

func (op *unSupported) Target() common.Address {
	return common.Address{}
}

func (op *unSupported) ParseTransaction() error {
	return fmt.Errorf("unSupported tx type %v", op.typ)
}

func (op *unSupported) GasUsed() *big.Int {
	return &big.Int{}
}
func (op *unSupported) Transition() *result {
	return nil
}

type txTransfer struct {
	*transitionContext
	target common.Address
	source common.Address
	value  *big.Int
}

func (ss *txTransfer) ParseTransaction() error {
	ss.target = *ss.msg.OpTarget()
	ss.value = ss.msg.Amount()
	ss.source = *ss.msg.Operator()
	return nil
}

func (ss *txTransfer) Transition() *result {
	ret := newResult()
	if needTransfer(ss.value) {
		if ss.accountDB.CanTransfer(ss.source, ss.value) {
			ss.accountDB.Transfer(ss.source, ss.target, ss.value)
		} else {
			ret.setError(errBalanceNotEnough, types.RSBalanceNotEnough)
		}
	}

	return ret
}

// minerStakeOperator handles all transactions related to group create
type groupOperator struct {
	*transitionContext
	groupOp group.Operation // Real group operation interface
}

func (ss *groupOperator) ParseTransaction() error {
	ss.groupOp = GroupManagerImpl.NewOperation(ss.accountDB, ss.msg, ss.bh.Height)
	return ss.groupOp.ParseTransaction()
}

func (ss *groupOperator) Transition() *result {
	ret := newResult()
	err := ss.groupOp.Operation()
	if err != nil {
		ret.setError(err, types.RSFail)
	}
	return ret
}

type contractCreator struct {
	*transitionContext
	source common.Address
}

func (ss *contractCreator) ParseTransaction() error {
	ss.source = *ss.msg.Operator()
	return nil
}

func (ss *contractCreator) Transition() *result {
	ret := newResult()
	controller := tvm.NewController(ss.accountDB, BlockChainImpl, ss.bh, ss.msg, ss.intrinsicGasUsed.Uint64(), MinerManagerImpl)
	contractAddress, txErr := createContract(ss.accountDB, ss.msg)
	if txErr != nil {
		ret.setError(txErr, types.RSFail)
	} else {
		contract := tvm.LoadContract(contractAddress)
		isTransferSuccess := transfer(ss.accountDB, ss.source, *contract.ContractAddress, ss.msg.Amount())
		if !isTransferSuccess {
			ret.setError(fmt.Errorf("balance not enough ,address is %v", ss.source.AddrPrefixString()), types.RSBalanceNotEnough)
		} else {
			_, logs, err := controller.Deploy(contract)
			ret.logs = logs
			if err != nil {
				if err.Code == types.TVMGasNotEnoughError {
					ret.setError(fmt.Errorf(err.Message), types.RSGasNotEnoughError)
				} else {
					ret.setError(fmt.Errorf(err.Message), types.RSTvmError)
				}
			} else {
				Logger.Debugf("Contract create success! Tx hash:%s, contract addr:%s", ss.msg.GetHash().Hex(), contractAddress.AddrPrefixString())
			}
		}
	}
	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(ss.msg.GetGasLimitOriginal(), gasLeft)
	ss.gasUsed = allUsed
	ret.contractAddress = contractAddress
	return ret
}

type contractCaller struct {
	*transitionContext
}

func (ss *contractCaller) ParseTransaction() error {
	return nil
}

func (ss *contractCaller) QueryData(state types.AccountDB, address string, key string, count int) map[string]string {
	hexAddr := common.StringToAddress(address)
	result := make(map[string]string)
	if count == 0 {
		value := state.GetData(hexAddr, []byte(key))
		if value != nil {
			result[key] = string(value)
			fmt.Println("key:", key, "value:", string(value))
		}
	} else {
		iter := state.DataIterator(hexAddr, []byte(key))
		if iter != nil {
			for iter.Next() {
				k := string(iter.Key[:])
				if !strings.HasPrefix(k, key) {
					continue
				}
				v := string(iter.Value[:])
				result[k] = v
				fmt.Println("key:", k, "value:", v)
				count--
				if count <= 0 {
					break
				}
			}
		}
	}
	return result
}


func (ss *contractCaller) Transition() *result {
	ret := newResult()
	controller := tvm.NewController(ss.accountDB, BlockChainImpl, ss.bh, ss.msg, ss.intrinsicGasUsed.Uint64(), MinerManagerImpl)
	contract := tvm.LoadContract(*ss.msg.OpTarget())
	if contract.Code == "" {
		ret.setError(fmt.Errorf("no code at the given address %v", ss.msg.OpTarget().AddrPrefixString()), types.RSNoCodeError)
	} else {
		isTransferSuccess := transfer(ss.accountDB, *ss.msg.Operator(), *contract.ContractAddress, ss.msg.Amount())
		if !isTransferSuccess {
			ret.setError(fmt.Errorf("balance not enough ,address is %v", ss.msg.Operator().AddrPrefixString()), types.RSBalanceNotEnough)
		} else {
			_, logs, err := controller.ExecuteAbiEval(ss.msg.Operator(), contract, string(ss.msg.Payload()))
			ret.logs = logs
			if err != nil {
				if err.Code == types.TVMCheckABIError {
					ret.setError(fmt.Errorf(err.Message), types.RSAbiError)
				} else if err.Code == types.TVMGasNotEnoughError {
					ret.setError(fmt.Errorf(err.Message), types.RSGasNotEnoughError)
				} else {
					ret.setError(fmt.Errorf(err.Message), types.RSTvmError)
				}
			} else {
				Logger.Debugf("Contract call success! contract addr:%s，abi is %s", contract.ContractAddress.AddrPrefixString(), string(ss.msg.Payload()))
				Logger.Debugf("QueryData: %v, gas: %v", ss.QueryData(ss.accountDB, contract.ContractAddress.AddrPrefixString(), "", 100), controller.GetGasLeft())
			}
		}
	}

	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(ss.msg.GetGasLimitOriginal(), gasLeft)
	ss.gasUsed = allUsed
	return ret
}

type rewardExecutor struct {
	*transitionContext
	blockHash   common.Hash
	blockHeight uint64
	targets     []common.Address
	reward      *big.Int
	packFee     *big.Int
	proposal    common.Address
}

func (ss *rewardExecutor) ParseTransaction() error {
	rm := BlockChainImpl.GetRewardManager()

	_, targets, blockHash, packFee, err := rm.ParseRewardTransaction(ss.msg)
	if err != nil {
		return err
	}

	ss.blockHash = blockHash

	// Reward for each target address
	ss.reward = ss.msg.Amount()
	ss.packFee = packFee

	ss.targets = make([]common.Address, 0)
	for _, tid := range targets {
		ss.targets = append(ss.targets, common.BytesToAddress(tid))
	}

	// Check if the corresponding block exists
	if bh := BlockChainImpl.QueryBlockHeaderByHash(ss.blockHash); bh == nil {
		return fmt.Errorf("block not exist：%v", ss.blockHash.Hex())
	} else {
		ss.blockHeight = bh.Height
	}
	// Check if there is a reward transaction of the same block already executed
	if rm.HasRewardedOfBlock(ss.blockHash, ss.accountDB) {
		return fmt.Errorf("reward transaction already executed:%v", ss.blockHash.Hex())
	}
	ss.proposal = common.BytesToAddress(ss.bh.Castor)
	return nil
}

func (ss *rewardExecutor) Transition() *result {
	ret := newResult()
	// Add the balance of the target addresses for verifying the block
	// Including the verifying reward and gas fee share
	for _, addr := range ss.targets {
		ss.accountDB.AddBalance(addr, ss.reward)
	}

	// Add the balance of proposer with pack fee for packing the reward tx
	ss.accountDB.AddBalance(ss.proposal, ss.packFee)

	// Mark reward tx of the block has been executed
	BlockChainImpl.GetRewardManager().MarkBlockRewarded(ss.blockHash, ss.msg.GetHash(), ss.accountDB)
	return ret
}

type TVMExecutor struct {
	bc    types.BlockChain
	procs []statePostProcessor
}

func NewTVMExecutor(bc types.BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc:    bc,
		procs: make([]statePostProcessor, 0),
	}
}

func (executor *TVMExecutor) addPostProcessor(proc statePostProcessor) {
	executor.procs = append(executor.procs, proc)
}

func doTransition(accountDB types.AccountDB, ss stateTransition) *result {
	snapshot := accountDB.Snapshot()
	ret := ss.Transition()
	if ret.err != nil {
		// Revert any state changes when error occurs
		accountDB.RevertToSnapshot(snapshot)
	}
	return ret
}

func applyStateTransition(accountDB types.AccountDB, tx *types.Transaction, bh *types.BlockHeader) (*result, error) {
	var ret *result

	// Reward tx is treated different from others
	if tx.IsReward() {
		executor := &rewardExecutor{transitionContext: newTransitionContext(accountDB, tx, bh, bh.Height)}
		// Reward tx should be removed from pool and not contained in the block when parse error
		if err := executor.ParseTransaction(); err != nil {
			return nil, err
		}
		ret = doTransition(accountDB, executor)
	} else {
		// Check state related condition on the non-reward tx type
		// Should be removed from pool and not contained in the block when error
		if err := checkState(accountDB, tx, bh.Height); err != nil {
			return nil, err
		}
		ss := newStateTransition(accountDB, tx, bh)

		// pre consume the gas limit for the normal transaction types
		gasLimitFee := new(big.Int).Mul(tx.GasLimit.Value(), tx.GasPrice.Value())
		accountDB.SubBalance(*tx.Source, gasLimitFee)

		// Shouldn't return when ParseTransaction error for ddos risk concern
		if err := ss.ParseTransaction(); err != nil {
			ret = newResult()
			ret.setError(err, types.RSParseFail)
		} else {
			ret = doTransition(accountDB, ss)
		}

		// refund the gas left
		if tx.GasLimit.Cmp(ss.GasUsed()) > 0 {
			refund := new(big.Int).Sub(tx.GasLimit.Value(), ss.GasUsed())
			accountDB.AddBalance(*tx.Source, refund.Mul(refund, tx.GasPrice.Value()))
		}
		ret.cumulativeGasUsed = ss.GasUsed()
	}
	return ret, nil
}

// Execute executes all types transactions and returns the receipts
func (executor *TVMExecutor) Execute(accountDB *account.AccountDB, bh *types.BlockHeader, txs []*types.Transaction, pack bool, ts *common.TimeStatCtx) (state common.Hash, evits []common.Hash, executed txSlice, recps []*types.Receipt, gasFee uint64, err error) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make(txSlice, 0)
	evictedTxs := make([]common.Hash, 0)
	castor := common.BytesToAddress(bh.Castor)
	rm := executor.bc.GetRewardManager().(*rewardManager)
	totalGasUsed := uint64(0)
	isCall  := false
	for _, tx := range txs {
		if tx.Type == 2{
			isCall = true
		}
		if pack && time.Since(beginTime).Seconds() > float64(ProposerPackageTime) {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", tx.Hash.Hex())
			break
		}

		if pack && totalGasUsed >= GasLimitForPackage {
			Logger.Warnf("exceeds the block gas limit GasLimitForPackage :%v %v ", totalGasUsed, GasLimitForPackage)
			break
		}

		snapshot := accountDB.Snapshot()

		if isCall{
			state = accountDB.IntermediateRoot(true)
			Logger.Infof("=============call1 hash = %s \n",state.Hex())
		}
		// Apply transaction
		ret, err := applyStateTransition(accountDB, tx, bh)
		if err != nil {
			Logger.Errorf("apply transaction error and will be removed: type=%v, hash=%v, source=%v, err=%v", tx.Type, tx.Hash.Hex(), tx.Source, err)
			// transaction will be remove from pool when error happens
			evictedTxs = append(evictedTxs, tx.Hash)
			continue
		}
		if ret.err != nil {
			Logger.Errorf("apply transaction error: type=%v, hash=%v, source=%v, err=%v", tx.Type, tx.Hash.Hex(), tx.Source, ret.err)
		}

		if isCall{
			state = accountDB.IntermediateRoot(true)
			Logger.Infof("=============call2 hash = %s \n",state.Hex())
		}

		// Accumulate gas fee
		cumulativeGas := uint64(0)
		if ret.cumulativeGasUsed != nil {
			cumulativeGas = ret.cumulativeGasUsed.Uint64()
			totalGasUsed += cumulativeGas
			if totalGasUsed > GasLimitPerBlock {
				// Revert snapshot in case total gas used exceeds the limit and break the loop
				// The tx just executed won't be packed into the block
				accountDB.RevertToSnapshot(snapshot)
				Logger.Warnf("revert to snapshot because total gas exceeds:%v %v ", totalGasUsed, GasLimitPerBlock)
				break
			}

			fee := big.NewInt(0).Mul(ret.cumulativeGasUsed, tx.GasPrice.Value())
			gasFee += fee.Uint64()
		}

		// Set nonce of the source
		if tx.Source != nil {
			accountDB.SetNonce(*tx.Source, tx.Nonce)
		}

		// New receipt
		idx := len(transactions)
		transactions = append(transactions, tx)
		receipt := types.NewReceipt(nil, ret.transitionStatus, cumulativeGas)
		for _, log := range ret.logs {
			log.TxIndex = uint(idx)
		}
		receipt.Logs = ret.logs
		receipt.TxHash = tx.Hash
		receipt.ContractAddress = ret.contractAddress
		receipt.TxIndex = uint16(idx)
		receipt.Height = bh.Height
		receipts = append(receipts, receipt)
		//errs[i] = err

	}
	//ts.AddStat("executeLoop", time.Since(b))
	castorTotalRewards := rm.calculateGasFeeCastorRewards(gasFee)
	castorTotalRewards += rm.calculateCastorRewards(bh.Height)
	deamonNodeRewards := rm.daemonNodesRewards(bh.Height)
	if deamonNodeRewards != 0 {
		accountDB.AddBalance(types.DaemonNodeAddress, big.NewInt(0).SetUint64(deamonNodeRewards))
	}
	userNodesRewards := rm.userNodesRewards(bh.Height)
	if userNodesRewards != 0 {
		accountDB.AddBalance(types.UserNodeAddress, big.NewInt(0).SetUint64(userNodesRewards))
	}

	if isCall{
		state = accountDB.IntermediateRoot(true)
		Logger.Infof("=============call3 hash = %s \n",state.Hex())
	}
	accountDB.AddBalance(castor, big.NewInt(0).SetUint64(castorTotalRewards))

	if isCall{
		state = accountDB.IntermediateRoot(true)
		Logger.Infof("=============call4 hash = %s \n",state.Hex())
	}

	for _, proc := range executor.procs {
		proc(accountDB, bh)
	}

	if isCall{
		state = accountDB.IntermediateRoot(true)
		Logger.Infof("=============call5 hash = %s \n",state.Hex())
	}

	state = accountDB.IntermediateRoot(true)
	if isCall{
		Logger.Infof("=============call6 hash = %s \n",state.Hex())
	}
	//Logger.Debugf("castor reward at %v, %v %v %v %v", bh.Height, castorTotalRewards, gasFee, rm.daemonNodesRewards(bh.Height), rm.userNodesRewards(bh.Height))
	return state, evictedTxs, transactions, receipts, gasFee, nil
}

func validateNonce(accountDB types.AccountDB, transaction *types.Transaction) bool {
	if transaction.Type == types.TransactionTypeReward {
		return true
	}
	nonce := accountDB.GetNonce(*transaction.Source)
	if transaction.Nonce != nonce+1 {
		Logger.Infof("Tx nonce error! Hash:%s,Source:%s,expect nonce:%d,real nonce:%d ", transaction.Hash.Hex(), transaction.Source.AddrPrefixString(), nonce+1, transaction.Nonce)
		return false
	}
	return true
}

func createContract(accountDB types.AccountDB, transaction types.TxMessage) (common.Address, error) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Operator()[:], common.Uint64ToByte(transaction.GetNonce()))))

	if accountDB.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, fmt.Errorf("contract address conflict")
	}
	accountDB.CreateAccount(contractAddr)
	accountDB.SetCode(contractAddr, transaction.Payload())
	accountDB.SetNonce(contractAddr, 1)
	return contractAddr, nil
}

func validGasPrice(gasPrice *big.Int, height uint64) bool {
	times := height / adjustGasPricePeriod
	if times > adjustGasPriceTimes {
		times = adjustGasPriceTimes
	}
	if gasPrice.Cmp(big.NewInt(0).SetUint64(initialMinGasPrice<<times)) < 0 {
		return false
	}
	return true
}

func needTransfer(amount *big.Int) bool {
	if amount.Sign() <= 0 {
		return false
	}
	return true
}

func transfer(accountDB types.AccountDB, source common.Address, target common.Address, amount *big.Int) bool {
	if !needTransfer(amount) {
		return true
	}
	if accountDB.CanTransfer(source, amount) {
		accountDB.Transfer(source, target, amount)
		return true
	}
	return false
}

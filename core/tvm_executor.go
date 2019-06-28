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
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/vm"
	"github.com/zvchain/zvchain/tvm"
)

const (
	TransactionGasCost   uint64 = 1000
	CodeBytePrice               = 0.3814697265625
	MaxCastBlockTime            = time.Second * 3
	adjustGasPricePeriod        = 30000000
	adjustGasPriceTimes         = 3
	initialMinGasPrice          = 200
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
}

func newStateTransition(db vm.AccountDB, tx *types.Transaction, bh *types.BlockHeader) stateTransition {
	base := newTransitionContext(db, tx, bh)
	switch tx.Type {
	case types.TransactionTypeTransfer:
		return &txTransfer{transitionContext: base}
	case types.TransactionTypeContractCreate:
		return &contractCreator{transitionContext: base}
	case types.TransactionTypeContractCall:
		return &contractCaller{transitionContext: base}
	case types.TransactionTypeReward:
		return &rewardExecutor{transitionContext: base}
	case types.TransactionTypeStakeAdd, types.TransactionTypeMinerAbort, types.TransactionTypeStakeReduce, types.TransactionTypeStakeRefund:
		return &minerStakeOperator{transitionContext: base}
	default:
		return &unSupported{typ: tx.Type}
	}
	return nil
}

type transitionContext struct {
	// Input
	accountDB vm.AccountDB
	bh        *types.BlockHeader
	tx        *types.Transaction
	source    common.Address

	// Output
	intrinsicGasUsed *big.Int
}

type result struct {
	cumulativeGas    *big.Int
	transitionStatus types.ReceiptStatus
	err              error
	logs             []*types.Log   // Generated when calls contract
	contractAddress  common.Address // Generated when creates contract
}

func newResult() *result {
	return &result{
		transitionStatus: types.RSSuccess,
	}
}

func (r *result) setError(err error, status types.ReceiptStatus) {
	r.err = err
	r.transitionStatus = status
}

func newTransitionContext(db vm.AccountDB, tx *types.Transaction, bh *types.BlockHeader) *transitionContext {
	return &transitionContext{accountDB: db, tx: tx, source: *tx.Source, bh: bh}
}

func checkState(db vm.AccountDB, tx *types.Transaction, height uint64) error {
	if !validGasPrice(&tx.GasPrice.Int, height) {
		return errGasPriceTooLow
	}
	gasLimitFee := new(types.BigInt).Mul(tx.GasLimit.Value(), tx.GasPrice.Value())
	if !canTransfer(db, *tx.Source, gasLimitFee) {
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

func (op *unSupported) ParseTransaction() error {
	return fmt.Errorf("unSupported tx type %v", op.typ)
}

func (op *unSupported) Validate() error {
	return fmt.Errorf("unSupported tx type %v", op.typ)
}

func (op *unSupported) Transition() *result {
	return nil
}

type txTransfer struct {
	*transitionContext
	target common.Address
	value  *big.Int
}

func (ss *txTransfer) ParseTransaction() error {
	ss.target = *ss.tx.Target
	ss.value = ss.tx.Value.Value()
	return nil
}

func (ss *txTransfer) Transition() *result {
	ret := newResult()
	if canTransfer(ss.accountDB, ss.source, ss.value) {
		transfer(ss.accountDB, ss.source, ss.target, ss.value)
	} else {
		ret.setError(errBalanceNotEnough, types.RSBalanceNotEnough)
	}
	ret.cumulativeGas = ss.intrinsicGasUsed
	return ret
}

// minerStakeOperator handles all transactions related to miner stake
type minerStakeOperator struct {
	*transitionContext
	minerOp mOperation // Real miner operation interface
}

func (ss *minerStakeOperator) ParseTransaction() error {
	ss.minerOp = newOperation(ss.accountDB, ss.tx, ss.bh.Height)
	return ss.minerOp.ParseTransaction()
}

func (ss *minerStakeOperator) Transition() *result {
	ret := newResult()
	err := ss.minerOp.Operation()
	if err != nil {
		ret.setError(err, types.RSFail)
	}
	ret.cumulativeGas = ss.intrinsicGasUsed
	return ret
}

type contractCreator struct {
	*transitionContext
}

func (ss *contractCreator) ParseTransaction() error {
	return nil
}

func (ss *contractCreator) Transition() *result {
	ret := newResult()
	controller := tvm.NewController(ss.accountDB, BlockChainImpl, ss.bh, ss.tx, ss.intrinsicGasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl)
	contractAddress, txErr := createContract(ss.accountDB, ss.tx)
	if txErr != nil {
		ret.setError(txErr, types.RSFail)
	} else {
		contract := tvm.LoadContract(contractAddress)
		err := controller.Deploy(contract)
		if err != nil {
			ret.setError(err, types.RSTvmError)
		} else {
			Logger.Debugf("Contract create success! Tx hash:%s, contract addr:%s", ss.tx.Hash.Hex(), contractAddress.Hex())
		}
	}
	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(ss.tx.GasLimit.Value(), gasLeft)
	ret.cumulativeGas = allUsed
	ret.contractAddress = contractAddress
	return ret
}

type contractCaller struct {
	*transitionContext
}

func (ss *contractCaller) ParseTransaction() error {
	return nil
}

func (ss *contractCaller) Transition() *result {
	ret := newResult()
	tx := ss.tx

	controller := tvm.NewController(ss.accountDB, BlockChainImpl, ss.bh, tx, ss.intrinsicGasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl)
	contract := tvm.LoadContract(*tx.Target)
	if contract.Code == "" {
		ret.setError(fmt.Errorf("no code at the given address %v", tx.Target.Hex()), types.RSTvmError)
	} else {
		_, logs, err := controller.ExecuteABI(tx.Source, contract, string(tx.Data))
		ret.logs = logs
		if err != nil {
			if err.Code == types.SysABIJSONError {
				ret.setError(fmt.Errorf(err.Message), types.RSAbiError)
			} else {
				ret.setError(fmt.Errorf(err.Message), types.RSTvmError)
			}
		} else if canTransfer(ss.accountDB, ss.source, tx.Value.Value()) {
			transfer(ss.accountDB, ss.source, *contract.ContractAddress, tx.Value.Value())
		} else {
			ret.setError(errBalanceNotEnough, types.RSBalanceNotEnough)
		}
	}
	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(tx.GasLimit.Value(), gasLeft)
	ret.cumulativeGas = allUsed

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

	_, targets, blockHash, packFee, err := rm.ParseRewardTransaction(ss.tx)
	if err != nil {
		return err
	}

	ss.blockHash = blockHash

	// Reward for each target address
	ss.reward = ss.tx.Value.Value()
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
	if rm.contain(ss.blockHash.Bytes(), ss.accountDB) {
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
	ret.cumulativeGas = new(big.Int)

	// Add the balance of proposer with pack fee for packing the reward tx
	ss.accountDB.AddBalance(ss.proposal, ss.packFee)

	// Mark reward tx of the block has been executed
	BlockChainImpl.GetRewardManager().put(ss.blockHash.Bytes(), ss.tx.Hash.Bytes(), ss.accountDB)
	return ret
}

type TVMExecutor struct {
	bc BlockChain
}

func NewTVMExecutor(bc BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc: bc,
	}
}

func applyStateTransition(accountDB vm.AccountDB, tx *types.Transaction, bh *types.BlockHeader) (*result, error) {
	ss := newStateTransition(accountDB, tx, bh)
	var err error

	// Check state related condition on the non-reward tx type
	if !tx.IsReward() {
		if err = checkState(accountDB, tx, bh.Height); err != nil {
			return nil, err
		}
	}

	if err = ss.ParseTransaction(); err != nil {
		Logger.Errorf("state transition parse error:tx %v", tx.Hash.Hex())
		return nil, err
	}

	// pre consume the gas limit for the normal transaction types
	if tx.Source != nil {
		gasLimitFee := new(big.Int).Mul(tx.GasLimit.Value(), tx.GasPrice.Value())
		accountDB.SubBalance(*tx.Source, gasLimitFee)
	}

	// Create the snapshot, and the stateDB will roll back to the the snapshot if error occurs
	// during transaction process
	snapshot := accountDB.Snapshot()
	ret := ss.Transition()
	if ret.err != nil {
		// Revert any state changes when error occurs
		accountDB.RevertToSnapshot(snapshot)
		Logger.Errorf("state transition error:tx %v", tx.Hash.Hex())
	}

	// refund the gas left
	if tx.Source != nil && tx.GasLimit.Cmp(ret.cumulativeGas) > 0 {
		refund := new(big.Int).Sub(tx.GasLimit.Value(), ret.cumulativeGas)
		accountDB.AddBalance(*tx.Source, refund.Mul(refund, tx.GasPrice.Value()))
	}
	return ret, nil
}

// Execute executes all types transactions and returns the receipts
func (executor *TVMExecutor) Execute(accountDB *account.AccountDB, bh *types.BlockHeader, txs []*types.Transaction, pack bool, ts *common.TimeStatCtx) (state common.Hash, evits []common.Hash, executed []*types.Transaction, recps []*types.Receipt, gasFee uint64, err error) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	castor := common.BytesToAddress(bh.Castor)
	rm := executor.bc.GetRewardManager()

	for _, tx := range txs {
		if pack && time.Since(beginTime).Seconds() > float64(MaxCastBlockTime) {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", tx.Hash.Hex())
			break
		}

		// Apply transaction
		ret, err := applyStateTransition(accountDB, tx, bh)
		if err != nil {
			Logger.Errorf("apply transaction error: type=%v, hash=%v, source=%v, err=%v", tx.Type, tx.Hash.Hex(), tx.Source, err)
			// transaction will be remove from pool when error happens
			evictedTxs = append(evictedTxs, tx.Hash)
			continue
		}

		// Accumulate gas fee
		fee := big.NewInt(0).Mul(ret.cumulativeGas, tx.GasPrice.Value())
		gasFee += fee.Uint64()

		// Set nonce of the source
		if tx.Source != nil {
			accountDB.SetNonce(*tx.Source, tx.Nonce)
		}

		// New receipt
		idx := len(transactions)
		transactions = append(transactions, tx)
		receipt := types.NewReceipt(nil, ret.transitionStatus, ret.cumulativeGas.Uint64())
		receipt.Logs = ret.logs
		receipt.TxHash = tx.Hash
		receipt.ContractAddress = ret.contractAddress
		receipt.TxIndex = uint16(idx)
		receipt.Height = bh.Height
		receipts = append(receipts, receipt)
		//errs[i] = err

	}

	// Accumulate reward with the share of all gas fee for castor
	castorTotalRewards := rm.calculateGasFeeCastorRewards(gasFee)

	// Calculate rewards with the specified height
	if rm.reduceBlockRewards(bh.Height, accountDB) {
		castorTotalRewards += rm.calculateCastorRewards(bh.Height)
		accountDB.AddBalance(common.HexToAddress(daemonNodeAddress), big.NewInt(0).SetUint64(rm.daemonNodesRewards(bh.Height)))
		accountDB.AddBalance(common.HexToAddress(userNodeAddress), big.NewInt(0).SetUint64(rm.userNodesRewards(bh.Height)))
	}
	accountDB.AddBalance(castor, big.NewInt(0).SetUint64(castorTotalRewards))

	state = accountDB.IntermediateRoot(true)
	return state, evictedTxs, transactions, receipts, gasFee, nil
}

func validateNonce(accountDB vm.AccountDB, transaction *types.Transaction) bool {
	if transaction.Type == types.TransactionTypeReward || IsTestTransaction(transaction) {
		return true
	}
	nonce := accountDB.GetNonce(*transaction.Source)
	if transaction.Nonce != nonce+1 {
		Logger.Infof("Tx nonce error! Hash:%s,Source:%s,expect nonce:%d,real nonce:%d ", transaction.Hash.Hex(), transaction.Source.Hex(), nonce+1, transaction.Nonce)
		return false
	}
	return true
}

func createContract(accountDB vm.AccountDB, transaction *types.Transaction) (common.Address, error) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(transaction.Nonce))))

	if accountDB.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, fmt.Errorf("contract address conflict")
	}
	accountDB.CreateAccount(contractAddr)
	accountDB.SetCode(contractAddr, transaction.Data)
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

func canTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	if amount.Sign() == -1 {
		return false
	}
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	// Escape if amount is zero
	if amount.Sign() <= 0 {
		return
	}
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

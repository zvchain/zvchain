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
	"bytes"
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
	errGasPriceTooLow = fmt.Errorf("gas price too low")
	errGasTooLow		= fmt.Errorf("gas too low")
	errBalanceNotEnough = fmt.Errorf("balance not enough")
	errNonceError		= fmt.Errorf("nonce error")
)

type transitionStatus int
const (
	tsSuccess				transitionStatus = iota
	tsOperationFail
	tsBalanceNotEnough
	tsAbiError
	tsTvmFail
)

// stateTransition define some functions on state transition
type stateTransition interface {
	Validate() error         // Validate the input args
	ParseTransaction() error // Parse the input transaction
	Transition() *result       // Do the transition
}

func newStateTransition(db vm.AccountDB, tx *types.Transaction, bh *types.BlockHeader) stateTransition {
	base := newTransitionContext(db, tx, bh)
	return nil
}

type transitionContext struct {
	// Input
	accountDB 	vm.AccountDB
	bh 			*types.BlockHeader
	tx 			*types.Transaction
	source 		common.Address

	// Output
	intrinsicGasUsed *big.Int
}

type result struct {
	cumulativeGas *big.Int
	transitionStatus 		transitionStatus
	err 		error
	logs 		[]*types.Log
	contractAddress common.Address
}
func newResult() *result {
	return &result{
		transitionStatus: tsSuccess,
	}
}

func (r *result) setError(err error, status transitionStatus)  {
    r.err = err
    r.transitionStatus = status
}

func newTransitionContext(db vm.AccountDB, tx *types.Transaction, bh *types.BlockHeader) *transitionContext {
	return &transitionContext{accountDB: db, tx:tx, source:*tx.Source, bh:bh}
}

func (ctx *transitionContext) checkNormTx() error {
	tx := ctx.tx
	if tx.GasLimit == nil {
		return fmt.Errorf("gas limit is nil")
	}
	if !tx.GasLimit.IsUint64() {
		return fmt.Errorf("gas limit is not uint64")
	}
	if tx.GasPrice == nil {
		return fmt.Errorf("gas price is nil")
	}
	if !tx.GasPrice.IsUint64() {
		return fmt.Errorf("gas price is not uint64")
	}
	if !validGasPrice(&tx.GasPrice.Int, ctx.bh.Height) {
		return errGasPriceTooLow
	}
	intriGas, err := intrinsicGas(tx)
	if err != nil {
		return errGasTooLow
	}
	ctx.intrinsicGasUsed = intriGas
	gasLimitFee := new(types.BigInt).Mul(tx.GasLimit.Value(), tx.GasPrice.Value())
	if !canTransfer(ctx.accountDB, *tx.Source, gasLimitFee) {
		return errBalanceNotEnough
	}
	if !validateNonce(ctx.accountDB, tx) {
		return errNonceError
	}
	return nil
}

type txTransfer struct {
	*transitionContext
	target common.Address
	value 	*big.Int
}

func (ss *txTransfer) Validate() error {
	tx := ss.tx
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	if tx.Value == nil || !tx.Value.IsUint64() {
		return fmt.Errorf("value format error")
	}
	if err := ss.checkNormTx(); err != nil {
		return err
	}
	return nil
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
		ret.setError(errBalanceNotEnough, tsBalanceNotEnough)
	}
	ret.cumulativeGas = ss.intrinsicGasUsed
	return ret
}

type minerStakeOperator struct {
	*transitionContext
	minerOp mOperation
}

func (ss *minerStakeOperator) Validate() error {
	ss.minerOp = newOperation(ss.accountDB, ss.tx, ss.bh.Height)
	if err := ss.checkNormTx(); err != nil {
		return err
	}
	return ss.minerOp.Validate()
}

func (ss *minerStakeOperator) ParseTransaction() error {
	return ss.minerOp.ParseTransaction()
}

func (ss *minerStakeOperator) Transition() *result {
	ret := newResult()
	err := ss.minerOp.Operation()
	if err != nil {
		ret.setError(err, tsOperationFail)
	}
	ret.cumulativeGas = ss.intrinsicGasUsed
	return ret
}

type contractCreator struct {
	*transitionContext
}

func (ss *contractCreator) Validate() error {
	tx := ss.tx
	if tx.Target != nil {
		return fmt.Errorf("contract create tx shouldn't have target")
	}
	if len(tx.Data) == 0 {
		return fmt.Errorf("no codes")
	}
	if err := ss.checkNormTx(); err != nil {
		return err
	}
	return nil
}

func (ss *contractCreator) ParseTransaction() error {
	return nil
}

func (ss *contractCreator) Transition() *result {
	ret := newResult()
	controller := tvm.NewController(ss.accountDB, BlockChainImpl, ss.bh, ss.tx, ss.intrinsicGasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl)
	contractAddress, txErr := createContract(ss.accountDB, ss.tx)
	if txErr != nil {
		ret.setError(txErr, tsOperationFail)
	} else {
		contract := tvm.LoadContract(contractAddress)
		err := controller.Deploy(contract)
		if err != nil {
			ret.setError(err, tsTvmFail)
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

func (ss *contractCaller) Validate() error {
	tx := ss.tx
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	if len(tx.Data) == 0 {
		return fmt.Errorf("data is empty")
	}
	if err := ss.checkNormTx(); err != nil {
		return err
	}
	return nil
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
		ret.setError(fmt.Errorf("no code at the given address %v", tx.Target.Hex()), tsTvmFail)
	} else {
		_, logs, err := controller.ExecuteABI(tx.Source, contract, string(tx.Data))
		ret.logs = logs
		if err != nil {
			if err.Code == types.SysABIJSONError {
				ret.setError(fmt.Errorf(err.Message), tsAbiError)
			} else {
				ret.setError(fmt.Errorf(err.Message), tsTvmFail)
			}
		} else if canTransfer(ss.accountDB, ss.source, tx.Value.Value()) {
			transfer(ss.accountDB, ss.source, *contract.ContractAddress, tx.Value.Value())
		} else {
			ret.setError(errBalanceNotEnough, tsBalanceNotEnough)
		}
	}
	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(tx.GasLimit.Value(), gasLeft)
	ret.cumulativeGas = allUsed

	return ret
}

type rewardExecutor struct {
	*transitionContext
	blockHash common.Hash
}

func (ss *rewardExecutor) Validate() error {
	if len(ss.tx.Data) != len(common.Hash{}) {
		return fmt.Errorf("data length should be 32bit")
	}
	if len(ss.tx.ExtraData) == 0 {
		return fmt.Errorf("extra data empty")
	}
	return nil
}

func (ss *rewardExecutor) ParseTransaction() error {
	rm := BlockChainImpl.GetRewardManager()
	ss.blockHash = rm.parseRewardBlockHash(ss.tx)
	if !BlockChainImpl.HasBlock(ss.blockHash) {
		return fmt.Errorf("block not exist：%v", ss.blockHash.Hex())
	}
	if rm.contain(ss.blockHash.Bytes(), ss.accountDB) {
		return fmt.Errorf("reward transaction already executed:%v", ss.blockHash.Hex())
	}
	return nil
}

func (ss *rewardExecutor) Transition() *result {
	ret := newResult()
	reader := bytes.NewReader(ss.tx.ExtraData)
	groupID := make([]byte, common.GroupIDLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupID); n != common.GroupIDLength {
		Logger.Errorf("TVMExecutor Read GroupID Fail")
		ret.setError(fmt.Errorf("read groupId fail"), tsOperationFail)
		return ret
	}
	for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
		if n != common.AddressLength {
			Logger.Errorf("TVMExecutor Reward Addr Size:%d Invalid", n)
			ret.setError(fmt.Errorf("read address fail"), tsOperationFail)
			return ret
		}
		address := common.BytesToAddress(addr)
		ss.accountDB.AddBalance(address, ss.tx.Value.Value())
	}
	ret.cumulativeGas = new(big.Int)
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
	if err = ss.Validate(); err != nil {
		Logger.Errorf("state transition validate error:tx %v", tx.Hash.Hex())
		return nil, err
	}
	if err = ss.ParseTransaction(); err != nil {
		Logger.Errorf("state transition parse error:tx %v", tx.Hash.Hex())
		return nil, err
	}

	// pre consume the gas limit for the normal transaction types
	if !tx.IsReward() {
		gasLimitFee := new(big.Int).Mul(tx.GasLimit.Value(), tx.GasPrice.Value())
		accountDB.SubBalance(*tx.Source, gasLimitFee)
	}

	snapshot := accountDB.Snapshot()
	ret := ss.Transition()
	if ret.err != nil {
		accountDB.RevertToSnapshot(snapshot)
		Logger.Errorf("state transition error:tx %v", tx.Hash.Hex())
	}

	// refund the gas left
	if tx.GasLimit.Uint64()-ret.cumulativeGas > 0 {
		refund := new(big.Int).Sub(tx.GasLimit.Value(), new(big.Int).SetUint64(ret.cumulativeGas))
		accountDB.AddBalance(*tx.Source, refund)
	}
	return ret, nil
}

// Execute executes all types transactions and returns the receipts
func (executor *TVMExecutor) Execute(accountdb *account.AccountDB, bh *types.BlockHeader, txs []*types.Transaction, pack bool, ts *common.TimeStatCtx) (state common.Hash, evits []common.Hash, executed []*types.Transaction, recps []*types.Receipt, gasFee uint64, err error) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	castor := common.BytesToAddress(bh.Castor)
	rm := executor.bc.GetRewardManager()
	var castorTotalRewards uint64
	for _, tx := range txs {
		if pack && time.Since(beginTime).Seconds() > float64(MaxCastBlockTime) {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", tx.Hash.Hex())
			break
		}

		ret, err := applyStateTransition(accountdb, tx, bh)
		if err != nil {
			Logger.Errorf("apply transaction error: type=%v, hash=%v, source=%v, err=%v", tx.Type, tx.Hash.Hex(), tx.Source, err)
			evictedTxs = append(evictedTxs, tx.Hash)
			continue
		}

		fee := big.NewInt(0)
		fee := big.NewInt(0).Mul(, transaction.GasPrice.Value())
		accountdb.SubBalance(*transaction.Source, fee)
		gasFee += fee.Uint64()

		var (
			contractAddress   common.Address
			logs              []*types.Log
			gasUsed           *types.BigInt
			cumulativeGasUsed uint64
			executeError      *types.TransactionError
			status            int = types.Success
		)

		if !transaction.IsReward() {
			if err = transaction.BoundCheck(); err != nil {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			if !validGasPrice(&transaction.GasPrice.Int, bh.Height) {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			intriGas, err := intrinsicGas(transaction)
			if err != nil {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			gasLimitFee := new(types.BigInt).Mul(transaction.GasLimit.Value(), transaction.GasPrice.Value())
			success := checkGasFeeIsEnough(accountdb, *transaction.Source, gasLimitFee)
			if !success {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			gasUsed = intriGas
			if !executor.validateNonce(accountdb, transaction) {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			if canTransfer(accountdb, *transaction.Source, new(big.Int).Add(transaction.Value.Value(), gasLimitFee)) {
				switch transaction.Type {
				case types.TransactionTypeTransfer:
					cumulativeGasUsed = executor.executeTransferTx(accountdb, transaction, castor, gasUsed)
				case types.TransactionTypeContractCreate:
					executeError, contractAddress, cumulativeGasUsed = executor.executeContractCreateTx(accountdb, transaction, castor, bh, gasUsed)
				case types.TransactionTypeContractCall:
					success, _, logs, cumulativeGasUsed = executor.executeContractCallTx(accountdb, transaction, castor, bh, gasUsed)
				case types.TransactionTypeStakeAdd:
					success, cumulativeGasUsed = executor.executeMinerApplyTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeMinerAbort:
					success, cumulativeGasUsed = executor.executeMinerAbortTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeStakeReduce:
					success, cumulativeGasUsed = executor.executeMinerRefundTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeMinerCancelStake:
					success, cumulativeGasUsed = executor.executeMinerCancelStakeTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeStakeRefund:
					success, cumulativeGasUsed = executor.executeMinerStakeTx(accountdb, transaction, bh.Height, castor, gasUsed)
				}
			} else {
				cumulativeGasUsed = intriGas.Uint64()
				executeError = types.TxErrorBalanceNotEnoughErr
			}
			fee := big.NewInt(0)
			fee = fee.Mul(fee.SetUint64(cumulativeGasUsed), transaction.GasPrice.Value())
			accountdb.SubBalance(*transaction.Source, fee)
			gasFee += fee.Uint64()
		} else {
			executeError = executor.executeRewardTx(accountdb, transaction, castor)
			Logger.Debugf("executed rewards tx, success: %t", executeError != nil)
			if executeError == nil {
				b := BlockChainImpl.QueryBlockByHash(common.BytesToHash(transaction.Data))
				if b != nil {
					castorTotalRewards += rm.CalculatePackedRewards(b.Header.Height)
				}
			} else {
				evictedTxs = append(evictedTxs, transaction.Hash)
				// Failed reward tx should not be included in block
				continue
			}
		}
		if executeError != nil {
			status = executeError.Code
		}
		idx := len(transactions)
		transactions = append(transactions, transaction)
		receipt := types.NewReceipt(nil, status, cumulativeGasUsed)
		receipt.Logs = logs
		receipt.TxHash = transaction.Hash
		receipt.ContractAddress = contractAddress
		receipt.TxIndex = uint16(idx)
		receipt.Height = bh.Height
		receipts = append(receipts, receipt)
		//errs[i] = err
		if transaction.Source != nil {
			accountdb.SetNonce(*transaction.Source, transaction.Nonce)
		}

	}
	//ts.AddStat("executeLoop", time.Since(b))
	castorTotalRewards += rm.CalculateGasFeeCastorRewards(gasFee)
	if rm.reduceBlockRewards(bh.Height, accountdb) {
		castorTotalRewards += rm.CalculateCastorRewards(bh.Height)
		accountdb.AddBalance(common.HexToAddress(daemonNodeAddress),
			big.NewInt(0).SetUint64(rm.daemonNodesRewards(bh.Height)))
		accountdb.AddBalance(common.HexToAddress(userNodeAddress),
			big.NewInt(0).SetUint64(rm.userNodesRewards(bh.Height)))
	}
	accountdb.AddBalance(castor, big.NewInt(0).SetUint64(castorTotalRewards))

	state = accountdb.IntermediateRoot(true)
	return state, evictedTxs, transactions, receipts, gasFee, nil
}

func validateNonce(accountdb vm.AccountDB, transaction *types.Transaction) bool {
	if transaction.Type == types.TransactionTypeReward || IsTestTransaction(transaction) {
		return true
	}
	nonce := accountdb.GetNonce(*transaction.Source)
	if transaction.Nonce != nonce+1 {
		Logger.Infof("Tx nonce error! Hash:%s,Source:%s,expect nonce:%d,real nonce:%d ", transaction.Hash.Hex(), transaction.Source.Hex(), nonce+1, transaction.Nonce)
		return false
	}
	return true
}


func createContract(accountdb vm.AccountDB, transaction *types.Transaction) (common.Address, error) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(transaction.Nonce))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, fmt.Errorf("contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)
	return contractAddr, nil
}

// intrinsicGas means transaction consumption intrinsic gas
func intrinsicGas(transaction *types.Transaction) (gasUsed *big.Int, err error) {
	gas := uint64(float32(len(transaction.Data)+len(transaction.ExtraData)) * CodeBytePrice)
	gasBig := new(big.Int).SetUint64(TransactionGasCost + gas)
	if transaction.GasLimit.Cmp(gasBig) < 0 {
		return nil, fmt.Errorf("gas not enough")
	}
	return new(big.Int).SetUint64(gas), nil
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

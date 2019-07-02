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

const TransactionGasCost uint64 = 400
const CodeBytePrice = 0.3814697265625
const MaxCastBlockTime = time.Millisecond * 2000
const adjustGasPricePeriod = 30000000
const adjustGasPriceTimes = 3
const initialMinGasPrice = 500

var (
	ProposerPackageTime = MaxCastBlockTime
	GasLimitForPackage  = uint64(GasLimitPerBlock)
)

type TVMExecutor struct {
	bc BlockChain
}

func NewTVMExecutor(bc BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc: bc,
	}
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
	totalGasUsed := uint64(0)
	for _, transaction := range txs {
		if pack && time.Since(beginTime).Seconds() > float64(MaxCastBlockTime) {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.Hex())
			break
		}

		if pack && totalGasUsed >= GasLimitForPackage {
			Logger.Infof("exceeds the block gas limit GasLimitForPackage ")
			break
		}

		var (
			contractAddress   common.Address
			logs              []*types.Log
			gasUsed           *types.BigInt
			cumulativeGasUsed uint64
			executeError      *types.TransactionError
			status            = types.Success
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
			if canTransfer(accountdb, *transaction.Source, transaction.Value.Value(), gasLimitFee) {
				snapshot := accountdb.Snapshot()

				switch transaction.Type {
				case types.TransactionTypeTransfer:
					cumulativeGasUsed = executor.executeTransferTx(accountdb, transaction, castor, gasUsed)
				case types.TransactionTypeContractCreate:
					executeError, contractAddress, cumulativeGasUsed = executor.executeContractCreateTx(accountdb, transaction, castor, bh, gasUsed)
				case types.TransactionTypeContractCall:
					executeError, logs, cumulativeGasUsed = executor.executeContractCallTx(accountdb, transaction, castor, bh, gasUsed)
				case types.TransactionTypeMinerApply:
					_, executeError, cumulativeGasUsed = executor.executeMinerApplyTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeMinerAbort:
					_, cumulativeGasUsed = executor.executeMinerAbortTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeMinerRefund:
					_, cumulativeGasUsed = executor.executeMinerRefundTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeMinerCancelStake:
					_, cumulativeGasUsed = executor.executeMinerCancelStakeTx(accountdb, transaction, bh.Height, castor, gasUsed)
				case types.TransactionTypeMinerStake:
					_, executeError, cumulativeGasUsed = executor.executeMinerStakeTx(accountdb, transaction, bh.Height, castor, gasUsed)
				}

				totalGasUsed += cumulativeGasUsed
				if totalGasUsed > GasLimitPerBlock {
					accountdb.RevertToSnapshot(snapshot)
					Logger.Infof("RevertToSnapshot happens: totalGasUsed is %d ,cumulativeGasUsed is %d", totalGasUsed, cumulativeGasUsed)
					break
				}

			} else {
				cumulativeGasUsed = intriGas.Uint64()
				executeError = types.TxErrorBalanceNotEnoughErr
				totalGasUsed += cumulativeGasUsed
				if totalGasUsed > GasLimitPerBlock {
					Logger.Infof("totalGasUsed is %d ,cumulativeGasUsed is %d", totalGasUsed, cumulativeGasUsed)
					break
				}
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

func (executor *TVMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
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

func (executor *TVMExecutor) executeTransferTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, gasUsed *types.BigInt) (cumulativeGasUsed uint64) {
	amount := transaction.Value.Value()
	transfer(accountdb, *transaction.Source, *transaction.Target, amount)
	cumulativeGasUsed = gasUsed.Uint64()

	return cumulativeGasUsed
}

func (executor *TVMExecutor) executeContractCreateTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, bh *types.BlockHeader, gasUsed *types.BigInt) (err *types.TransactionError, contractAddress common.Address, cumulativeGasUsed uint64) {
	var txErr *types.TransactionError

	gasLimit := transaction.GasLimit
	controller := tvm.NewController(accountdb, BlockChainImpl, bh, transaction, gasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl, GroupChainImpl)
	snapshot := controller.AccountDB.Snapshot()
	contractAddress, txErr = createContract(accountdb, transaction)
	if txErr != nil {
		Logger.Debugf("ContractCreate tx %s execute error:%s ", transaction.Hash.Hex(), txErr.Message)
		controller.AccountDB.RevertToSnapshot(snapshot)
	} else {
		contract := tvm.LoadContract(contractAddress)
		err := controller.Deploy(contract)
		if err != nil {
			txErr = types.NewTransactionError(types.TVMExecutedError, err.Error())
			controller.AccountDB.RevertToSnapshot(snapshot)
			Logger.Debugf("Contract deploy failed! Tx hash:%s, contract addr:%s errorCode:%d errorMsg%s",
				transaction.Hash.Hex(), contractAddress.Hex(), types.TVMExecutedError, err.Error())
		} else {
			Logger.Debugf("Contract create success! Tx hash:%s, contract addr:%s", transaction.Hash.Hex(), contractAddress.Hex())
		}
	}
	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(gasLimit.Value(), gasLeft)
	cumulativeGasUsed = allUsed.Uint64()

	Logger.Debugf("TVMExecutor Execute ContractCreate Transaction %s", transaction.Hash.Hex())
	return txErr, contractAddress, cumulativeGasUsed
}

func (executor *TVMExecutor) executeContractCallTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, bh *types.BlockHeader, gasUsed *types.BigInt) (err *types.TransactionError, logs []*types.Log, cumulativeGasUsed uint64) {
	transferAmount := transaction.Value.Value()

	gasLimit := transaction.GasLimit

	controller := tvm.NewController(accountdb, BlockChainImpl, bh, transaction, gasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl, GroupChainImpl)
	contract := tvm.LoadContract(*transaction.Target)
	if contract.Code == "" {
		err = types.NewTransactionError(types.TxErrorCodeNoCode, fmt.Sprintf(types.NoCodeErrorMsg, *transaction.Target))

	} else {
		snapshot := controller.AccountDB.Snapshot()
		_, _, logs, err = controller.ExecuteAbiEval(transaction.Source, contract, string(transaction.Data))
		if err != nil {
			controller.AccountDB.RevertToSnapshot(snapshot)
		} else {
			transfer(accountdb, *transaction.Source, *contract.ContractAddress, transferAmount)
		}
	}
	gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
	allUsed := new(big.Int).Sub(gasLimit.Value(), gasLeft)
	cumulativeGasUsed = allUsed.Uint64()

	Logger.Debugf("TVMExecutor Execute ContractCall Transaction %s,success:%t", transaction.Hash.Hex(), err != nil)
	return err, logs, cumulativeGasUsed
}

func (executor *TVMExecutor) executeRewardTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address) (err *types.TransactionError) {
	if executor.bc.GetRewardManager().contain(transaction.Data, accountdb) == false {
		Logger.Debugf("executeRewardTx: blockhash: %s, value: %d", common.Bytes2Hex(transaction.Data),
			transaction.Value.Value().Uint64())
		reader := bytes.NewReader(transaction.ExtraData)
		groupID := make([]byte, common.GroupIDLength)
		addr := make([]byte, common.AddressLength)
		if n, _ := reader.Read(groupID); n != common.GroupIDLength {
			Logger.Errorf("TVMExecutor Read GroupID Fail")
			return types.TxErrorFailedErr
		}
		for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
			if n != common.AddressLength {
				Logger.Errorf("TVMExecutor Reward Addr Size:%d Invalid", n)
				break
			}
			address := common.BytesToAddress(addr)
			accountdb.AddBalance(address, transaction.Value.Value())
		}
		executor.bc.GetRewardManager().put(transaction.Data, transaction.Hash[:], accountdb)
	} else {
		return types.TxErrorFailedErr
	}
	return err
}

func (executor *TVMExecutor) executeMinerApplyTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool, err *types.TransactionError, cumulativeGasUsed uint64) {
	Logger.Debugf("Execute miner apply tx:%s,source: %v\n", transaction.Hash.Hex(), transaction.Source.Hex())
	success = false

	var miner = MinerManagerImpl.Transaction2Miner(transaction)
	miner.ID = transaction.Source[:]
	amount := new(big.Int).SetUint64(miner.Stake)
	mExist := MinerManagerImpl.GetMinerByID(transaction.Source[:], miner.Type, accountdb)
	if !canTransfer(accountdb, *transaction.Source, amount, new(big.Int).SetUint64(0)) {
		Logger.Error("execute MinerApply balance not enough!")
		err = types.TxErrorBalanceNotEnoughErr
		return
	}
	if mExist != nil {
		if mExist.Status != types.MinerStatusNormal {
			if (mExist.Stake + miner.Stake) < MinerManagerImpl.MinStake() {
				Logger.Debugf("TVMExecutor Execute MinerApply Fail((mExist.Stake + miner.Stake) < MinStake) Source:%s Height:%d", transaction.Source.Hex(), height)
				return
			}
			snapshot := accountdb.Snapshot()
			if MinerManagerImpl.activateAndAddStakeMiner(miner, accountdb, height) &&
				MinerManagerImpl.AddStakeDetail(miner.ID, miner, miner.Stake, accountdb) {
				accountdb.SubBalance(*transaction.Source, amount)
				Logger.Debugf("TVMExecutor Execute MinerApply success(activate) Source %s", transaction.Source.Hex())
				success = true
			} else {
				accountdb.RevertToSnapshot(snapshot)
			}
		} else {
			Logger.Debugf("TVMExecutor Execute MinerApply Fail(Already Exist) Source %s", transaction.Source.Hex())
		}
		return
	}
	miner.ApplyHeight = height
	miner.Status = types.MinerStatusNormal
	if MinerManagerImpl.addMiner(transaction.Source[:], miner, accountdb) > 0 &&
		MinerManagerImpl.AddStakeDetail(miner.ID, miner, miner.Stake, accountdb) {
		accountdb.SubBalance(*transaction.Source, amount)
		Logger.Debugf("TVMExecutor Execute MinerApply Success Source:%s Height:%d", transaction.Source.Hex(), height)
		success = true
	}

	return success, err, cumulativeGasUsed
}

func (executor *TVMExecutor) executeMinerStakeTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool, err *types.TransactionError, cumulativeGasUsed uint64) {
	Logger.Debugf("Execute miner Stake tx:%s,source: %v\n", transaction.Hash.Hex(), transaction.Source.Hex())
	success = false

	var _type, id, value = MinerManagerImpl.Transaction2MinerParams(transaction)
	amount := new(big.Int).SetUint64(value)

	mExist := MinerManagerImpl.GetMinerByID(id, _type, accountdb)
	cumulativeGasUsed = gasUsed.Uint64()

	if !canTransfer(accountdb, *transaction.Source, amount, new(big.Int).SetUint64(0)) {
		Logger.Error("execute Miner Stake balance not enough!")
		err = types.TxErrorBalanceNotEnoughErr
		return
	}

	if mExist == nil {
		success = false
		Logger.Debugf("TVMExecutor Execute Miner Stake Fail(Do not exist this Miner) Source:%s Height:%d", transaction.Source.Hex(), height)
	} else {
		snapshot := accountdb.Snapshot()
		if MinerManagerImpl.AddStake(mExist.ID, mExist, value, accountdb, height) && MinerManagerImpl.AddStakeDetail(transaction.Source[:], mExist, value, accountdb) {
			Logger.Debugf("TVMExecutor Execute MinerUpdate Success Source:%s Height:%d", transaction.Source.Hex(), height)
			accountdb.SubBalance(*transaction.Source, amount)
			success = true
		} else {
			accountdb.RevertToSnapshot(snapshot)
		}
	}

	return success, err, cumulativeGasUsed
}

func (executor *TVMExecutor) executeMinerCancelStakeTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool, cumulativeGasUsed uint64) {
	Logger.Debugf("Execute miner cancel pledge tx:%s,source: %v\n", transaction.Hash.Hex(), transaction.Source.Hex())
	success = false

	var _type, id, value = MinerManagerImpl.Transaction2MinerParams(transaction)

	mExist := MinerManagerImpl.GetMinerByID(id, _type, accountdb)
	if mExist == nil {
		Logger.Debugf("TVMExecutor Execute MinerCancelStake Fail(Can not find miner) Source %s", transaction.Source.Hex())
		return
	}
	cumulativeGasUsed = gasUsed.Uint64()
	snapshot := accountdb.Snapshot()
	if MinerManagerImpl.CancelStake(transaction.Source[:], mExist, value, accountdb, height) &&
		MinerManagerImpl.ReduceStake(mExist.ID, mExist, value, accountdb, height) {
		success = true
		Logger.Debugf("TVMExecutor Execute MinerCancelStake success Source %s", transaction.Source.Hex())
	} else {
		Logger.Debugf("TVMExecutor Execute MinerCancelStake Fail(CancelStake or ReduceStake error) Source %s", transaction.Source.Hex())
		accountdb.RevertToSnapshot(snapshot)
	}
	return
}

func (executor *TVMExecutor) executeMinerAbortTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool, cumulativeGasUsed uint64) {
	success = false

	cumulativeGasUsed = gasUsed.Uint64()
	if transaction.Data != nil {
		success = MinerManagerImpl.abortMiner(transaction.Source[:], transaction.Data[0], height, accountdb)
	}

	Logger.Debugf("TVMExecutor Execute MinerAbort Tx %s,Source:%s, Success:%t", transaction.Hash.Hex(), transaction.Source.Hex(), success)
	return success, cumulativeGasUsed
}

func (executor *TVMExecutor) executeMinerRefundTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool, cumulativeGasUsed uint64) {
	success = false

	cumulativeGasUsed = gasUsed.Uint64()
	var _type, id, _ = MinerManagerImpl.Transaction2MinerParams(transaction)
	mexist := MinerManagerImpl.GetMinerByID(id, _type, accountdb)
	if mexist != nil {
		snapShot := accountdb.Snapshot()
		defer func() {
			if !success {
				accountdb.RevertToSnapshot(snapShot)
			}
		}()
		if mexist.Type == types.MinerTypeHeavy {
			latestCancelPledgeHeight := MinerManagerImpl.GetLatestCancelStakeHeight(transaction.Source[:], mexist, accountdb)
			if height > latestCancelPledgeHeight+10 || (mexist.Status == types.MinerStatusAbort && height > mexist.AbortHeight+10) {
				value, ok := MinerManagerImpl.RefundStake(transaction.Source.Bytes(), mexist, accountdb)
				if !ok {
					success = false
					return
				}
				amount := new(big.Int).SetUint64(value)
				accountdb.AddBalance(*transaction.Source, amount)
				Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Success %s", transaction.Source.Hex())
				success = true
			} else {
				Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Fail(Refund height less than abortHeight+10) Hash%s", transaction.Source.Hex())
			}
		} else if mexist.Type == types.MinerTypeLight {
			value, ok := MinerManagerImpl.RefundStake(transaction.Source.Bytes(), mexist, accountdb)
			if !ok {
				success = false
				return
			}
			amount := new(big.Int).SetUint64(value)
			accountdb.AddBalance(*transaction.Source, amount)
			Logger.Debugf("TVMExecutor Execute MinerRefund Light Success %s,Type:%s", transaction.Source.Hex())
			success = true
		} else {
			Logger.Debugf("TVMExecutor Execute MinerRefund Fail(No such miner type) %s", transaction.Source.Hex())
			return
		}
	} else {
		Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Not Exist Or Not Abort) %s", transaction.Source.Hex())
	}

	return success, cumulativeGasUsed
}

func createContract(accountdb *account.AccountDB, transaction *types.Transaction) (common.Address, *types.TransactionError) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(transaction.Nonce))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, types.NewTransactionError(types.TxErrorCodeContractAddressConflict, "contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)
	return contractAddr, nil
}

// intrinsicGas means transaction consumption intrinsic gas
func intrinsicGas(transaction *types.Transaction) (gasUsed *types.BigInt, err error) {
	gas := uint64(float32(len(transaction.Data)+len(transaction.ExtraData)) * CodeBytePrice)
	gas = TransactionGasCost + gas
	gasBig := types.NewBigInt(gas)
	if transaction.GasLimit.Cmp(gasBig.Value()) < 0 {
		return nil, fmt.Errorf("gas not enough")
	}
	return types.NewBigInt(gas), nil
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

func checkGasFeeIsEnough(db vm.AccountDB, addr common.Address, gasFee *big.Int) bool {
	if db.GetBalance(addr).Cmp(gasFee) < 0 {
		return false
	}
	return true
}

func canTransfer(db vm.AccountDB, addr common.Address, amount *big.Int, gasFee *big.Int) bool {
	totalAmount := new(big.Int).Add(amount, gasFee)
	return db.GetBalance(addr).Cmp(totalAmount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	// Escape if amount is zero
	if amount.Sign() == 0 {
		return
	}
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

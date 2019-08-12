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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

// validator is responsible for the info validation of the given transaction
type validator func() error

// intrinsicGas means transaction consumption intrinsic gas
func intrinsicGas(transaction *types.Transaction) *big.Int {
	gas := uint64((len(transaction.Data) + len(transaction.ExtraData)) * CodeBytePrice / CodeBytePricePrecision)
	gasBig := new(big.Int).SetUint64(TransactionGasCost + gas)
	return gasBig
}

// commonValidate performs the validations on all of transactions
func commonValidate(tx *types.Transaction) error {
	if !tx.Hash.IsValid() {
		return ErrHash
	}
	size := 0
	if tx.Data != nil {
		size += len(tx.Data)
	}
	if tx.ExtraData != nil {
		size += len(tx.ExtraData)
	}
	if size > txMaxSize {
		return fmt.Errorf("tx size(%v) should not larger than %v", size, txMaxSize)
	}

	if tx.Hash != tx.GenHash() {
		return ErrHash
	}

	if tx.Sign == nil {
		return fmt.Errorf("tx sign nil")
	}
	return nil
}

// gasValidate does gas related validations.
// Only reward transactions doesn't need to do this
func gasValidate(tx *types.Transaction) error {
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
	if tx.GasLimit.Uint64() > gasLimitPerTransaction {
		return fmt.Errorf("gas limit too high")
	}
	// Check if the gasLimit less than the intrinsic gas
	intrinsicGas := intrinsicGas(tx)
	if tx.GasLimit.Cmp(intrinsicGas) < 0 {
		return fmt.Errorf("gas limit too low")
	}

	return nil
}

func valueValidate(tx *types.Transaction) error {
	if tx.Value == nil {
		return fmt.Errorf("value is nil")
	}
	if !tx.Value.IsUint64() {
		return fmt.Errorf("value is not uint64")
	}
	return nil
}

func sourceRecover(tx *types.Transaction) error {
	var sign = common.BytesToSign(tx.Sign)
	if sign == nil {
		return fmt.Errorf("BytesToSign fail, sign=%v", tx.Sign)
	}
	msg := tx.Hash.Bytes()
	pk, err := sign.RecoverPubkey(msg)
	if err != nil {
		return err
	}
	src := pk.GetAddress()
	tx.Source = &src
	return nil
}

// stateValidate performs state related validation
// Nonce validate delay to push to the container
// All state related validation have to performed again when apply transactions because the state may be have changed
func stateValidate(tx *types.Transaction) error {
	accountDB, err := BlockChainImpl.LatestStateDB()
	if err != nil {
		return fmt.Errorf("fail get last state db,error = %v", err.Error())
	}
	gasLimitFee := new(types.BigInt).Mul(tx.GasPrice.Value(), tx.GasLimit.Value())
	balance := accountDB.GetBalance(*tx.Source)
	src := tx.Source.Hex()
	if gasLimitFee.Cmp(balance) > 0 {
		return fmt.Errorf("balance not enough for paying gas, %v", src)
	}
	// Check gas price related to height
	if !validGasPrice(tx.GasPrice.Value(), BlockChainImpl.Height()) {
		return fmt.Errorf("gas price below the lower bound")
	}
	return nil
}

func transferValidator(tx *types.Transaction) error {
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	return valueValidate(tx)
}

func minerTypeCheck(mt types.MinerType) error {
	if !types.IsProposalRole(mt) && !types.IsVerifyRole(mt) {
		return fmt.Errorf("unknown miner type %v", mt)
	}
	return nil
}

func stakeAddValidator(tx *types.Transaction) error {
	if len(tx.Data) == 0 {
		return fmt.Errorf("data is empty")
	}
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	pks, err := types.DecodePayload(tx.Data)
	if err != nil {
		return err
	}
	if err := minerTypeCheck(pks.MType); err != nil {
		return err
	}
	return valueValidate(tx)
}

func minerAbortValidator(tx *types.Transaction) error {
	if len(tx.Data) != 1 {
		return fmt.Errorf("data length should be 1")
	}
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	if err := minerTypeCheck(types.MinerType(tx.Data[0])); err != nil {
		return err
	}
	return nil
}

func stakeReduceValidator(tx *types.Transaction) error {
	if len(tx.Data) != 1 {
		return fmt.Errorf("data length should be 1")
	}
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	if err := minerTypeCheck(types.MinerType(tx.Data[0])); err != nil {
		return err
	}
	if tx.Value == nil {
		return fmt.Errorf("value is nil")
	}
	if !tx.Value.IsUint64() {
		return fmt.Errorf("value is not uint64")
	}
	if tx.Value.Uint64() == 0 {
		return fmt.Errorf("value is 0")
	}
	return nil
}

func stakeRefundValidator(tx *types.Transaction) error {
	if len(tx.Data) != 1 {
		return fmt.Errorf("data length should be 1")
	}
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	if err := minerTypeCheck(types.MinerType(tx.Data[0])); err != nil {
		return err
	}
	return nil
}

func applyGuardValidator(tx *types.Transaction) error {
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	return nil
}

func voteMinerPoolValidator(tx *types.Transaction) error {
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	return nil
}


func groupValidator(tx *types.Transaction) error {
	if len(tx.Data) == 0 {
		return fmt.Errorf("data is empty")
	}
	if tx.Target != nil {
		return fmt.Errorf("target should be nil")
	}
	return nil
}

func contractCreateValidator(tx *types.Transaction) error {
	if len(tx.Data) == 0 {
		return fmt.Errorf("data is empty")
	}
	if tx.Target != nil {
		return fmt.Errorf("target should be nil")
	}
	return valueValidate(tx)
}

func contractCallValidator(tx *types.Transaction) error {
	if len(tx.Data) == 0 {
		return fmt.Errorf("data is empty")
	}
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	return valueValidate(tx)
}

func rewardValidate(tx *types.Transaction) error {
	// Data stores the block hash which is 32bit
	if len(tx.Data) != len(common.Hash{}) {
		return fmt.Errorf("data length should be 32bit")
	}
	// ExtraData stores details which cannot be nil
	if len(tx.ExtraData) == 0 {
		return fmt.Errorf("extra data empty")
	}
	if err := valueValidate(tx); err != nil {
		return err
	}
	if ok, err := BlockChainImpl.GetConsensusHelper().VerifyRewardTransaction(tx); !ok {
		return err
	}
	return nil
}

// getValidator returns the corresponding validator of the given transaction
func getValidator(tx *types.Transaction) validator {
	return func() error {
		var err error
		// Common validations
		if err = commonValidate(tx); err != nil {
			return err
		}
		// Reward tx type
		if tx.IsReward() {
			return rewardValidate(tx)
		} else {
			// Validate gas
			if err = gasValidate(tx); err != nil {
				return err
			}
			switch tx.Type {
			case types.TransactionTypeTransfer:
				err = transferValidator(tx)
			case types.TransactionTypeContractCreate:
				err = contractCreateValidator(tx)
			case types.TransactionTypeContractCall:
				err = contractCallValidator(tx)
			case types.TransactionTypeStakeAdd:
				err = stakeAddValidator(tx)
			case types.TransactionTypeMinerAbort:
				err = minerAbortValidator(tx)
			case types.TransactionTypeStakeReduce:
				err = stakeReduceValidator(tx)
			case types.TransactionTypeStakeRefund:
				err = stakeRefundValidator(tx)
			case types.TransactionTypeApplyGuardMiner:
				err = applyGuardValidator(tx)
			case types.TransactionTypeVoteMinerPool:
				err = voteMinerPoolValidator(tx)
			case types.TransactionTypeGroupPiece, types.TransactionTypeGroupMpk, types.TransactionTypeGroupOriginPiece:
				err = groupValidator(tx)
			default:
				err = fmt.Errorf("no such kind of tx")
			}
			if err != nil {
				return err
			}
			// Recover source at last for performance concern
			if err := sourceRecover(tx); err != nil {
				return err
			}
			//check the abort tx's source and target
			if tx.Type == types.TransactionTypeMinerAbort {
				if bytes.Compare(tx.Target.Bytes(), tx.Source.Bytes()) != 0 {
					return fmt.Errorf("could not abort for other node")
				}
			}
			if tx.Type == types.TransactionTypeVoteMinerPool {
				if bytes.Compare(tx.Target.Bytes(), tx.Source.Bytes()) == 0 {
					return fmt.Errorf("could not vote myself")
				}
			}
			// Validate state
			if err := stateValidate(tx); err != nil {
				return err
			}
		}
		return nil
	}
}

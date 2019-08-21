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
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"io"
)

const codecVersion = 1

func marshalTx(tx *types.RawTransaction) ([]byte, error) {
	return msgpack.Marshal(tx)
}

func unmarshalTx(data []byte) (*types.RawTransaction, error) {
	var tx types.RawTransaction
	err := msgpack.Unmarshal(data, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func encodeBlockTransactions(b *types.Block) ([]byte, error) {
	dataBuf := bytes.NewBuffer([]byte{})
	// Write the version number
	dataBuf.Write(common.UInt16ToByte(uint16(codecVersion)))
	// Write the count of transactions
	dataBuf.Write(common.UInt16ToByte(uint16(len(b.Transactions))))
	// Write each transaction length and transaction data
	if len(b.Transactions) > 0 {
		txBuf := bytes.NewBuffer([]byte{})
		for _, tx := range b.Transactions {
			txBytes, err := marshalTx(tx.RawTransaction)
			if err != nil {
				return nil, err
			}
			// Write each transaction length
			dataBuf.Write(common.UInt16ToByte(uint16(len(txBytes))))
			txBuf.Write(txBytes)
		}
		// Finally write transaction data
		dataBuf.Write(txBuf.Bytes())
	}

	return dataBuf.Bytes(), nil
}

func decodeBlockTransactions(data []byte) ([]*types.RawTransaction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	txs := make([]*types.RawTransaction, 0)
	dataReader := bytes.NewReader(data)

	twoBytes := make([]byte, 2)
	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	version := common.ByteToUInt16(twoBytes)
	if version != codecVersion {
		return nil, fmt.Errorf("version error")
	}

	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	txNum := common.ByteToUInt16(twoBytes)
	if txNum == 0 {
		return txs, nil
	}

	lenBytes := make([]byte, txNum*2)
	_, err := io.ReadFull(dataReader, lenBytes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(txNum); i++ {
		txLen := common.ByteToUInt16(lenBytes[2*i : 2*(i+1)])
		txBytes := make([]byte, txLen)
		_, err := io.ReadFull(dataReader, txBytes)
		if err != nil {
			return nil, err
		}
		tx, err := unmarshalTx(txBytes)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func decodeTransaction(idx int, data []byte) (*types.RawTransaction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	dataReader := bytes.NewReader(data)

	twoBytes := make([]byte, 2)
	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	version := common.ByteToUInt16(twoBytes)
	if version != codecVersion {
		return nil, fmt.Errorf("version error")
	}

	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	txNum := common.ByteToUInt16(twoBytes)
	if txNum == 0 {
		return nil, fmt.Errorf("txNum is zero")
	}

	lenBytes := make([]byte, txNum*2)
	_, err := io.ReadFull(dataReader, lenBytes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(txNum); i++ {
		txLen := common.ByteToUInt16(lenBytes[2*i : 2*(i+1)])
		txBytes := make([]byte, txLen)
		_, err := io.ReadFull(dataReader, txBytes)
		if err != nil {
			return nil, err
		}
		if idx == i {
			tx, err := unmarshalTx(txBytes)
			if err != nil {
				return nil, err
			}
			return tx, nil
		}
	}
	return nil, fmt.Errorf("tx not found")
}

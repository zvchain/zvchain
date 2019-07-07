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

package group

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

const (
	dataVersion         = 1
	dataTypePiece       = 1
	dataTypeMpk         = 2
	dataTypeOriginPiece = 3
)

var (
	originPieceReqKey = common.Hex2Bytes("originPieceRequired") //the key for marking origin piece required in current seed
	groupDataKey          = common.Hex2Bytes("group")               //the key for saving group data in levelDb
)

// Store implements GroupStoreReader
type Store struct {
	//chain chainReader
	db *account.AccountDB
}

func NewStore(db *account.AccountDB) types.GroupStoreReader {
	return &Store{db}
}

//	返回指定seed的piece数据
//  共识通过此接口获取所有piece数据，生成自己的签名私钥和签名公钥
func (s *Store) GetEncryptedPiecePackets(seed types.SeedI) ([]types.EncryptedSharePiecePacket, error) {
	seedAdder := common.HashToAddress(seed.Seed())
	rs := make([]types.EncryptedSharePiecePacket, 0, 100) //max group member number is 100
	prefix := getKeyPrefix(dataTypePiece)
	iter := s.db.DataIterator(seedAdder, prefix)
	if iter == nil {
		return nil, errors.New("no pieces uploaded for this seed")
	}
	for iter.Next() {

		var data EncryptedSharePiecePacketImpl
		err := msgpack.Unmarshal(iter.Value, &data)
		if err != nil {
			return nil, err
		}
		rs = append(rs, &data)
	}

	return rs, nil
}

// 查询指定sender 和seed 是否有piece数据
func (s *Store) HasSentEncryptedPiecePacket(sender []byte, seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypePiece, common.BytesToAddress(sender))
	//return s.db.Exist(seedAdder, key.toByte())
	return s.db.GetData(seedAdder, key.toByte()) != nil
}

func (s *Store) HasSentMpkPacket(sender []byte, seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypeMpk, common.BytesToAddress(sender))
	return s.db.GetData(seedAdder, key.toByte()) != nil
}

// 返回所有的建组数据
// 共识在校验是否建组成时调用
func (s *Store) GetMpkPackets(seed types.SeedI) ([]types.MpkPacket, error) {
	seedAdder := common.HashToAddress(seed.Seed())
	prefix := getKeyPrefix(dataTypeMpk)
	iter := s.db.DataIterator(seedAdder, prefix)
	mpks := make([]types.MpkPacket, 0)
	for iter.Next() {
		var data MpkPacketImpl
		err := msgpack.Unmarshal(iter.Value, &data)
		if err != nil {
			return nil, err
		}
		mpks = append(mpks, &data)
	}
	return mpks, nil
}

// 返回origin piece 是否需要的标志
func (s *Store) IsOriginPieceRequired(seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	return s.db.GetData(seedAdder, originPieceReqKey) != nil
}

// 获取所有明文分片
func (s *Store) GetOriginPiecePackets(seed types.SeedI) ([]types.OriginSharePiecePacket, error) {
	seedAdder := common.HashToAddress(seed.Seed())
	prefix := getKeyPrefix(dataTypeOriginPiece)
	iter := s.db.DataIterator(seedAdder, prefix)
	pieces := make([]types.OriginSharePiecePacket, 0, 100)
	for iter.Next() {
		var data OriginSharePiecePacketImpl
		err := msgpack.Unmarshal(iter.Value, &data)
		if err != nil {
			return nil, err
		}
		pieces = append(pieces, &data)
	}
	return pieces, nil
}

func (s *Store) HasSentOriginPiecePacket(sender []byte, seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypeOriginPiece, common.BytesToAddress(sender))
	return s.db.GetData(seedAdder, key.toByte()) != nil
}

// Get available group infos at the given height
func (s *Store) GetAvailableGroupInfos(h uint64) []types.GroupI {
	rs := make([]types.GroupI,0)
	iter := s.db.DataIterator(common.GroupActiveAddress,[]byte{})
	if iter == nil {
		return nil
	}
	for iter.Next() {
		var life groupLife
		err := msgpack.Unmarshal(iter.Value, &life)
		if err != nil {
			return nil
		}
		if life.begin <= h && h <= life.end {
			key := iter.Key
			byteInfo := s.db.GetData(common.BytesToAddress(key),groupDataKey)
			info := &Group{}
			err := msgpack.Unmarshal(byteInfo, &life)
			if err != nil {
				return nil
			}
			rs = append(rs, info)
		}
	}
	return rs
}

func (s *Store) GetGroupInfoBySeed(seed types.SeedI) types.GroupI {
	byteData := s.db.GetData(common.HashToAddress(seed.Seed()),groupDataKey)
	if byteData != nil {

		var data Group
		err := msgpack.Unmarshal(byteData, &data)
		if err != nil {
			return nil
		}
		return &data
	}
	return nil
}

type txDataKey struct {
	version  byte
	dataType byte
	source   common.Address
}

func newTxDataKey(dataType byte, source common.Address) *txDataKey {
	return &txDataKey{dataVersion, dataType, source}
}

func (t *txDataKey) toByte() []byte {
	return keyToByte(t)
}

func getKeyPrefix(dataType byte) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(dataVersion)
	buf.WriteByte(dataType)
	return buf.Bytes()
}

func keyLength() int {
	return 2 + common.AddressLength
}

// keyToByte
func keyToByte(key *txDataKey) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(key.version)
	buf.WriteByte(key.dataType)
	buf.Write(key.source.Bytes())
	return buf.Bytes()
}

// byteToKey parse the byte key to struct. key will be like version+dataType+address
func byteToKey(bs []byte) (key *txDataKey, err error) {
	totalLen := keyLength()
	if len(bs) != totalLen {
		return nil, fmt.Errorf("length error")
	}
	version := bs[0]
	if version != dataVersion {
		return nil, fmt.Errorf("version error %v", version)
	}
	dType := bs[1]
	source := bs[2 : 2+common.HashLength]

	key = &txDataKey{version, dType, common.BytesToAddress(source)}
	return
}

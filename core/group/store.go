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
)

const (
	dataVersion         = 1
	dataTypePiece       = 1
	dataTypeMpk         = 2
	dataTypeOriginPiece = 3
)

var (
	originPieceReqKey = common.Hex2Bytes("originPieceRequired") //the key for marking origin piece required in current seed
	groupDataKey      = common.Hex2Bytes("group")               //the key for saving group data in levelDb
)

// Store implements GroupStoreReader
type Store struct {
	chain    chainReader
	poolImpl pool
}

func NewStore(chain chainReader, p pool) types.GroupStoreReader {
	return &Store{chain, p}
}

// GetEncryptedPiecePackets returns all uploaded encrypted share piece with given seed
func (s *Store) GetEncryptedPiecePackets(seed types.SeedI) ([]types.EncryptedSharePiecePacket, error) {
	seedAdder := common.HashToAddress(seed.Seed())
	rs := make([]types.EncryptedSharePiecePacket, 0, 100) //max group member number is 100
	prefix := getKeyPrefix(dataTypePiece)
	iter := s.chain.LatestStateDB().DataIterator(seedAdder, prefix)
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

// HasSentEncryptedPiecePacket checks if sender sent the encrypted piece to chain
func (s *Store) HasSentEncryptedPiecePacket(sender []byte, seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypePiece, common.BytesToAddress(sender))
	//return s.db.Exist(seedAdder, key.toByte())
	return s.chain.LatestStateDB().GetData(seedAdder, key.toByte()) != nil
}

// HasSentEncryptedPiecePacket checks if sender sent the mpk to chain
func (s *Store) HasSentMpkPacket(sender []byte, seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypeMpk, common.BytesToAddress(sender))
	return s.chain.LatestStateDB().GetData(seedAdder, key.toByte()) != nil
}

// GetMpkPackets return all mpk with given seed
func (s *Store) GetMpkPackets(seed types.SeedI) ([]types.MpkPacket, error) {
	seedAdder := common.HashToAddress(seed.Seed())
	prefix := getKeyPrefix(dataTypeMpk)
	iter := s.chain.LatestStateDB().DataIterator(seedAdder, prefix)
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

// IsOriginPieceRequired checks if group members should upload origin piece
func (s *Store) IsOriginPieceRequired(seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	return s.chain.LatestStateDB().GetData(seedAdder, originPieceReqKey) != nil
}

// GetOriginPiecePackets returns all origin pieces with given seed
func (s *Store) GetOriginPiecePackets(seed types.SeedI) ([]types.OriginSharePiecePacket, error) {
	seedAdder := common.HashToAddress(seed.Seed())
	prefix := getKeyPrefix(dataTypeOriginPiece)
	iter := s.chain.LatestStateDB().DataIterator(seedAdder, prefix)
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

// HasSentOriginPiecePacket checks if sender sent the origin piece to chain
func (s *Store) HasSentOriginPiecePacket(sender []byte, seed types.SeedI) bool {
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypeOriginPiece, common.BytesToAddress(sender))
	return s.chain.LatestStateDB().GetData(seedAdder, key.toByte()) != nil
}

// GetAvailableGroupSeeds returns all activeList groups at the given height
func (s *Store) GetAvailableGroupSeeds(height uint64) []types.SeedI {
	gls := s.poolImpl.getActives(s.chain, height)
	if gls != nil {
		rs := make([]types.SeedI, 0, len(gls))
		for _, v := range gls {
			rs = append(rs, v)
		}
		return rs
	}
	return nil
}

// GetGroupBySeed returns group with given seed
func (s *Store) GetGroupBySeed(seedHash common.Hash) types.GroupI {
	return s.poolImpl.get(s.chain.LatestStateDB(), seedHash)
}

// GetGroupBySeed returns group header with given seed
func (s *Store) GetGroupHeaderBySeed(seedHash common.Hash) types.GroupHeaderI {
	g := s.poolImpl.get(s.chain.LatestStateDB(), seedHash)
	if g == nil {
		return nil
	}
	return g.Header()
}

// IsMinerInLiveGroup returns the count of living groups which contains the given miner address
func (s *Store) MinerLiveGroupCount(addr common.Address, height uint64) int {
	return s.poolImpl.minerLiveGroupCount(s.chain, addr, height)
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

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
	"fmt"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

const (
	dataVersion         = 1
	dataTypePiece       = 1
	dataTypeMpk         =2
	//dataTypeSign        = 3
	dataTypeOriginPiece = 3


)

type Store struct {
	chain chainReader
	db *account.AccountDB
}



//	返回指定seed的piece数据
//  共识通过此接口获取所有piece数据，生成自己的签名私钥和签名公钥

func (s *Store)GetEncryptedPiecePackets(seed types.SeedI) ([]types.EncryptedSharePiecePacket, error){
	seedAdder := common.HashToAddress(seed.Seed())
	rs := make([]types.EncryptedSharePiecePacket,0,100)		//max group member number is 100
	prefix := getKeyPrefix(dataTypePiece)
	iter := s.db.DataIterator(seedAdder,prefix )
	for iter.Next() {

		var data EncryptedSharePiecePacketImp
		err := msgpack.Unmarshal(iter.Value, &data)
		if err != nil {
			return nil, err
		}
		rs  = append(rs ,&data)
	}

	return rs, nil
}

// 查询指定sender 和seed 是否有piece数据
func (s *Store)HasSentEncryptedPiecePacket(sender []byte, seed types.SeedI) bool{
	seedAdder := common.HashToAddress(seed.Seed())
	key := newTxDataKey(dataTypePiece,common.BytesToAddress(sender))
	return s.db.GetData(seedAdder,key.toByte()) == nil
}

// 返回所有的建组数据
// 共识在校验是否建组成时调用
func (s *Store)GetPieceAndMpkPackets(seed types.SeedI) (types.FullPacket, error){
	seedAdder := common.HashToAddress(seed.Seed())
	//prefix1 := getKeyPrefix(dataTypePiece)
	prefix := getKeyPrefix(dataTypeMpk)
	iter := s.db.DataIterator(seedAdder,prefix )
	//mpks := make([]*MpkPacketImpl,0)
	for iter.Next() {
		//sender := common.BytesToAddress(iter.Key[len(prefix):])
		var data MpkPacketImpl
		err := msgpack.Unmarshal(iter.Value, &data)
		if err != nil {
			return nil, err
		}

	}

	return nil, nil
}


// 返回origin piece 是否需要的标志
func (s *Store)IsOriginPieceRequired(seed types.SeedI) bool{

	return false
}
// 获取所有明文分片
func (s *Store)GetAllOriginPiecePackets(seed types.SeedI) ([]types.OriginSharePiecePacket, error){

	return nil, nil
}
// Get available group infos at the given height
func (s *Store)GetAvailableGroupInfos(h uint64) []types.GroupI{
	return nil
}




type txDataKey struct {
	version byte
	dataType byte
	source common.Address
}

func newTxDataKey(dataType byte, source common.Address) *txDataKey {
	return &txDataKey{dataVersion,dataType,source}
}

func (t *txDataKey)toByte() []byte {
	return keyToByte(t)
}

func getKeyPrefix(dataType byte) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(dataVersion)
	buf.WriteByte(dataType)
	return buf.Bytes()
}

func keyLength() int {
	return  2 +common.AddressLength
}

// keyToByte
func keyToByte(key *txDataKey) []byte{
	//totalLen := keyLength()
	//rs := make([]byte,totalLen)
	//rs = append(rs, key.version)
	//rs = append(rs, key.dataType)
	//rs = append(rs, key.source.Bytes()...)
	//return rs

	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(key.version)
	buf.WriteByte(key.dataType)
	buf.Write(key.source.Bytes())
	return buf.Bytes()
}

// byteToKey parse the byte key to struct. key will be like version+dataType+address
func byteToKey(bs []byte) ( key *txDataKey, err error) {
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

	key = &txDataKey{version,dType,common.BytesToAddress(source)}
	return
}
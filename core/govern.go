//   Copyright (C) 2020 ZVChain
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
	"sync"
)

var (
	blackStoreAddr = common.BytesToAddress([]byte("black-store"))
	blackPrefix    = []byte("list-")
)

type governManagerI interface {
	addBlacks(db types.AccountDB, addrs []common.Address) error
	removeBlacks(db types.AccountDB, addrs []common.Address) error
	isBlack(db types.AccountDB, address common.Address) bool
}

func getGovernInstance() governManagerI {
	return BlockChainImpl.governer
}

type blackUpdateTx struct {
	*transitionContext
	blackOp *types.BlackOperator
	gov     governManagerI
}

func genSignDataBytes(opType byte, addrs []common.Address) []byte {
	buff := new(bytes.Buffer)
	buff.WriteByte(opType)
	for _, addr := range addrs {
		buff.Write(addr.Bytes())
	}
	return common.Sha256(buff.Bytes())
}

func (ss *blackUpdateTx) ParseTransaction() error {
	b, err := types.DecodeBlackOperator(ss.msg.Payload())
	if err != nil {
		return err
	}
	ss.blackOp = b

	// get sign data
	signBytes := genSignDataBytes(ss.blackOp.OpType, ss.blackOp.Addrs)

	// load guard nodes
	guardNodes, err := MinerManagerImpl.GetAllGuardNodeAddrs(ss.accountDB)
	if err != nil {
		return err
	}
	// signs enough
	var threshold = len(guardNodes)/2 + 1
	if len(ss.blackOp.Signs) < threshold {
		return fmt.Errorf("not enough guard node signs, receive %v, expect %v", len(ss.blackOp.Signs), threshold)
	}

	isGuard := func(addr common.Address) bool {
		for _, g := range guardNodes {
			if g == addr {
				return true
			}
		}
		return false
	}

	// check guards and sign
	for _, sig := range ss.blackOp.Signs {
		var sign = common.BytesToSign(sig)
		if sign == nil {
			return fmt.Errorf("decode sign fail, sign=%v", sign)
		}
		pk, err := sign.RecoverPubkey(signBytes)
		if err != nil {
			return err
		}
		src := pk.GetAddress()
		if !isGuard(src) {
			return fmt.Errorf("sign %v is not from a guard node", common.ToHex(sig))
		}
	}

	return nil
}

func (ss *blackUpdateTx) Transition() *result {
	ret := newResult()
	if ss.blackOp.OpType == 1 {
		getGovernInstance().removeBlacks(ss.accountDB, ss.blackOp.Addrs)
	} else {
		getGovernInstance().addBlacks(ss.accountDB, ss.blackOp.Addrs)
	}
	return ret
}

type governManager struct {
	blacks map[common.Address]struct{}
	root   common.Hash
	lock   sync.RWMutex
}

func newGovernManager() *governManager {
	return &governManager{
		blacks: make(map[common.Address]struct{}),
	}
}

func (gm *governManager) loadBlacks(db types.AccountDB) {
	iter := db.DataIterator(blackStoreAddr, blackPrefix)
	for iter.Next() {
		if !bytes.HasPrefix(iter.Key, blackPrefix) {
			break
		}
		addr := common.BytesToAddress(iter.Key[len(blackPrefix):])
		gm.blacks[addr] = struct{}{}
	}
}

func (gm *governManager) isBlack(db types.AccountDB, addr common.Address) bool {
	obj := db.GetStateObject(blackStoreAddr)
	if obj == nil {
		return false
	}
	gm.lock.Lock()
	defer gm.lock.Unlock()

	if obj.GetRootHash() != gm.root {
		gm.root = obj.GetRootHash()
		gm.loadBlacks(db)
		Logger.Infof("load blacklist size %v", len(gm.blacks))
	}

	_, ok := gm.blacks[addr]
	return ok
}

func (gm *governManager) genKey(addr common.Address) []byte {
	buf := bytes.Buffer{}
	buf.Write(blackPrefix)
	buf.Write(addr.Bytes())
	return buf.Bytes()
}

func (gm *governManager) addBlacks(db types.AccountDB, addrs []common.Address) error {
	for _, addr := range addrs {
		db.SetData(blackStoreAddr, gm.genKey(addr), []byte{1})
	}
	return nil
}

func (gm *governManager) removeBlacks(db types.AccountDB, addrs []common.Address) error {
	for _, addr := range addrs {
		db.RemoveData(blackStoreAddr, gm.genKey(addr))
	}
	return nil
}

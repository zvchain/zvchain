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

package consensustest

import (
	"errors"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/group"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/types"
	"testing"
)

func initCore() error {
	common.InitConf("/Users/pxf/go_lib/src/github.com/zvchain/zvchain/local_test/zv.ini")
	var err error
	// Initialization middlewarex
	middleware.InitMiddleware()
	log.Init()

	skString := "0x04a3ef44d60158ac0fc63a13ad6beb0d9db463631868f085790c278dfa42d5ab3fd6d23ce0ac6f2236f37920954c86992e0db428e9237901ec274ec48d2626125ec3521378d09a7018edf052812220835a1f7e8315c6e188acaca69376f2acbf17"

	sk := common.HexToSecKey(skString)
	minerInfo, err := model.NewSelfMinerDO(sk)
	if err != nil {
		return err
	}
	helper := mediator.NewConsensusHelper(minerInfo.ID)

	err = core.InitCore(helper, nil)
	if err != nil {
		return err
	}
	ok := mediator.ConsensusInit(minerInfo, common.GlobalConf)
	if !ok {
		return errors.New("consensus module error")
	}
	return nil
}

func addBlock(chain *core.FullBlockChain, h uint64) (types.AddBlockResult, error) {
	b := chain.QueryBlockByHeight(h)
	if b == nil {
		return -1, fmt.Errorf("block is nil %v", h)
	}
	pre := chain.QueryBlockHeaderByHash(b.Header.PreHash)
	if pre == nil {
		return -1, fmt.Errorf("pre is nil %v", b.Header.PreHash.Hex())
	}
	if err := chain.UpdateLatestBlock(pre); err != nil {
		return -1, err
	}
	group.GroupRoutine.UpdateContext(pre)
	return chain.AddBlock(b)
}

func TestFullBlockChain_AddBlockOnChain(t *testing.T) {
	if err := initCore(); err != nil {
		t.Fatal(err)
	}
	chain := core.BlockChainImpl

	h := uint64(2486176)
	ret, err := addBlock(chain, h)
	t.Log(ret, err)

	h = uint64(2486177)
	ret, err = addBlock(chain, h)
	t.Log(ret, err)

}

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

package logical

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"io"
)

type member struct {
	id groupsig.ID
	pk groupsig.Pubkey
}

type groupHeader struct {
	seed          common.Hash
	workHeight    uint64
	dismissHeight uint64
	gpk           groupsig.Pubkey
	threshold     uint32
	groupHeight   uint64
}

func (gh *groupHeader) Seed() common.Hash {
	return gh.seed
}

func (gh *groupHeader) WorkHeight() uint64 {
	return gh.workHeight
}

func (gh *groupHeader) DismissHeight() uint64 {
	return gh.dismissHeight
}

func (gh *groupHeader) PublicKey() []byte {
	return gh.gpk.Serialize()
}

func (gh *groupHeader) Threshold() uint32 {
	return gh.threshold
}

func (gh *groupHeader) GroupHeight() uint64 {
	return gh.groupHeight
}

type verifyGroup struct {
	header   *groupHeader
	members  []*member
	memIndex map[string]int
}

func convertGroupHeaderI(gh types.GroupHeaderI) *groupHeader {
	return &groupHeader{
		seed:          gh.Seed(),
		workHeight:    gh.WorkHeight(),
		dismissHeight: gh.DismissHeight(),
		gpk:           groupsig.DeserializePubkeyBytes(gh.PublicKey()),
		threshold:     gh.Threshold(),
		groupHeight:   gh.GroupHeight(),
	}
}

func convertGroupI(g types.GroupI) *verifyGroup {
	mems := make([]*member, len(g.Members()))
	memIndex := make(map[string]int)
	for i, mem := range g.Members() {
		mems[i] = &member{id: groupsig.DeserializeID(mem.ID()), pk: groupsig.DeserializePubkeyBytes(mem.PK())}
		memIndex[mems[i].id.GetAddrString()] = i
	}
	return &verifyGroup{
		header:   convertGroupHeaderI(g.Header()),
		memIndex: memIndex,
		members:  mems,
	}
}

func (vg *verifyGroup) getMembers() []groupsig.ID {
	ids := make([]groupsig.ID, len(vg.members))
	for i, mem := range vg.members {
		ids[i] = mem.id
	}
	return ids
}

func (vg *verifyGroup) hasMember(id groupsig.ID) bool {
	if _, ok := vg.memIndex[id.GetAddrString()]; ok {
		return true
	}
	return false
}

func (vg *verifyGroup) getMemberIndex(id groupsig.ID) int {
	if v, ok := vg.memIndex[id.GetAddrString()]; ok {
		return v
	}
	return -1
}

func (vg *verifyGroup) getMemberAt(idx int) *member {
	if idx < 0 || idx >= len(vg.members) {
		return nil
	}
	return vg.members[idx]
}

func (vg *verifyGroup) getMemberPubkey(id groupsig.ID) groupsig.Pubkey {
	if i, ok := vg.memIndex[id.GetAddrString()]; ok {
		return vg.members[i].pk
	}
	return groupsig.Pubkey{}
}

func (vg *verifyGroup) memberSize() int {
	return len(vg.members)
}

type skStorage interface {
	io.Closer
	GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey
	StoreGroupSignatureSeckey(seed common.Hash, sk groupsig.Seckey, expireHeight uint64)
}

type groupInfoReader interface {
	// GetActivatedGroupSeeds gets activated groups' seed at the given height
	GetActivatedGroupsAt(height uint64) []types.GroupI
	// GetLivedGroupsAt gets lived groups
	GetLivedGroupsAt(height uint64) []types.GroupI
	// GetGroupBySeed returns the group info of the given seed
	GetGroupBySeed(seedHash common.Hash) types.GroupI
	// GetGroupHeaderBySeed returns the group header info of the given seed
	GetGroupHeaderBySeed(seedHash common.Hash) types.GroupHeaderI
	Height() uint64
	// GetGroupSkipCountsAt gets group skip counts at the given height
	GetGroupSkipCountsAt(h uint64, groups []types.GroupI) (map[common.Hash]uint16, error)
}

type groupGetter func(seed common.Hash) types.GroupI

type groupReader struct {
	skStore skStorage
	cache   *lru.Cache
	reader  groupInfoReader
}

func newGroupReader(infoReader groupInfoReader, skReader skStorage) *groupReader {
	return &groupReader{
		skStore: skReader,
		reader:  infoReader,
		cache:   common.MustNewLRUCache(200),
	}
}

func (gr *groupReader) getGroupHeaderBySeed(seed common.Hash) *groupHeader {
	g := gr.getGroupBySeed(seed)
	if g == nil {
		return nil
	}
	return g.header
}

func (gr *groupReader) getGroupBySeed(seed common.Hash) *verifyGroup {
	return gr.tryToGetOrConvert(seed, gr.reader.GetGroupBySeed)
}

func (gr *groupReader) tryToGetOrConvert(seed common.Hash, getter groupGetter) *verifyGroup {
	if v, ok := gr.cache.Get(seed); !ok {
		g := getter(seed)
		if g == nil {
			return nil
		}
		gi := convertGroupI(g)
		gr.cache.ContainsOrAdd(gi.header.seed, gi)
		return gi
	} else {
		return v.(*verifyGroup)
	}
}

// getActivatedGroupsByHeight gets all activated groups at the given height
func (gr *groupReader) getActivatedGroupsByHeight(h uint64) []*verifyGroup {
	gs := gr.reader.GetActivatedGroupsAt(h)
	vgs := make([]*verifyGroup, len(gs))
	for i, gi := range gs {
		vgs[i] = gr.tryToGetOrConvert(gi.Header().Seed(), func(seed common.Hash) types.GroupI {
			return gi
		})
	}
	return vgs
}

// getGroupSkipCountsByHeight gets skip counts of activated group seeds at the given height
func (gr *groupReader) getGroupSkipCountsByHeight(h uint64) map[common.Hash]uint16 {
	activeGroups := gr.reader.GetActivatedGroupsAt(h)
	skipInfos, err := gr.reader.GetGroupSkipCountsAt(h, activeGroups)
	if err != nil {
		stdLogger.Panicf("get group skip info fail at %v, err %v", h, err)
		return nil
	}
	return skipInfos
}

func (gr *groupReader) getLivedGroupsByHeight(h uint64) []*verifyGroup {
	gs := gr.reader.GetLivedGroupsAt(h)
	vgs := make([]*verifyGroup, len(gs))
	for i, gi := range gs {
		vgs[i] = gr.tryToGetOrConvert(gi.Header().Seed(), func(seed common.Hash) types.GroupI {
			return gi
		})
	}
	return vgs
}

func (gr *groupReader) getGroupSignatureSeckey(seed common.Hash) groupsig.Seckey {
	return gr.skStore.GetGroupSignatureSeckey(seed)
}

func (gr *groupReader) Height() uint64 {
	return gr.reader.Height()
}

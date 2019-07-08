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
)

type member struct {
	id groupsig.ID
	pk groupsig.Pubkey
}

type verifyGroup struct {
	header   types.GroupHeaderI
	members  []*member
	memIndex map[string]int
}

func convertGroupI(g types.GroupI) *verifyGroup {
	mems := make([]*member, len(g.Members()))
	memIndex := make(map[string]int)
	for i, mem := range g.Members() {
		mems[i] = &member{id: groupsig.DeserializeID(mem.ID()), pk: groupsig.DeserializePubkeyBytes(mem.PK())}
		memIndex[mems[i].id.GetHexString()] = i
	}
	return &verifyGroup{
		header:   g.Header(),
		memIndex: memIndex,
		members:  mems,
	}
}

func (vg *verifyGroup) getMembers() []groupsig.ID {
	ids := make([]groupsig.ID, len(vg.members))
	for _, mem := range vg.members {
		ids = append(ids, mem.id)
	}
	return ids
}

func (vg *verifyGroup) hasMember(id groupsig.ID) bool {
	if _, ok := vg.memIndex[id.GetHexString()]; ok {
		return true
	}
	return false
}

func (vg *verifyGroup) getMemberIndex(id groupsig.ID) int {
	return vg.memIndex[id.GetHexString()]
}

func (vg *verifyGroup) getMemberAt(idx int) *member {
	if idx < 0 || idx >= len(vg.members) {
		return nil
	}
	return vg.members[idx]
}

func (vg *verifyGroup) getMemberPubkey(id groupsig.ID) groupsig.Pubkey {
	if i, ok := vg.memIndex[id.GetHexString()]; ok {
		return vg.members[i].pk
	}
	return groupsig.Pubkey{}
}

func (vg *verifyGroup) memberSize() int {
	return len(vg.members)
}

type skReader interface {
	GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey
}

type groupReader struct {
	reader   types.GroupStoreReader
	skReader skReader
	cache    *lru.Cache
}

func newGroupReader(reader types.GroupStoreReader, skReader skReader) *groupReader {
	return &groupReader{
		reader:   reader,
		skReader: skReader,
		cache:    common.MustNewLRUCache(50),
	}
}

func (gr *groupReader) getGroupHeaderBySeed(seed common.Hash) types.GroupHeaderI {
	if v, ok := gr.cache.Get(seed); ok {
		return v.(*verifyGroup).header
	}
	g := gr.reader.GetGroupHeaderBySeed(seed)
	if g != nil {
		gr.cache.ContainsOrAdd(seed, &verifyGroup{header: g})
	}
	return g
}

func (gr *groupReader) getGroupBySeed(seed common.Hash) *verifyGroup {
	if v, ok := gr.cache.Get(seed); ok {
		g := v.(*verifyGroup)
		if len(g.members) > 0 {
			return g
		}
	}
	g := gr.reader.GetGroupBySeed(seed)
	if g != nil {
		gi := convertGroupI(g)
		gr.cache.ContainsOrAdd(gi.header.Seed(), gi)
		return gi
	}
	return nil
}

func (gr *groupReader) getAvailableGroupsByHeight(h uint64) []*verifyGroup {
	gs := gr.reader.GetAvailableGroups(h)
	gis := make([]*verifyGroup, len(gs))
	if gs != nil {
		for i, g := range gs {
			gi := convertGroupI(g)
			gis[i] = gi
			gr.cache.ContainsOrAdd(g.Header().Seed(), gi)
		}
	}
	return gis
}

func (gr *groupReader) getGroupSignatureSeckey(seed common.Hash) groupsig.Seckey {
	return gr.skReader.GetGroupSignatureSeckey(seed)
}

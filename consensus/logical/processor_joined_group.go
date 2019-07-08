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

package logical

//
//import (
//	"crypto/rand"
//	"encoding/json"
//	"github.com/syndtr/goleveldb/leveldb/filter"
//	"github.com/syndtr/goleveldb/leveldb/opt"
//	"io/ioutil"
//	"os"
//	"strings"
//	"sync"
//
//	lru "github.com/hashicorp/golang-lru"
//	"github.com/zvchain/zvchain/common"
//	"github.com/zvchain/zvchain/consensus/groupsig"
//	"github.com/zvchain/zvchain/consensus/model"
//	"github.com/zvchain/zvchain/storage/tasdb"
//)
//
//// key suffix definition when store the verifyGroup infos to db
//const (
//	suffixSignKey = "_signKey"
//	suffixGInfo   = "_gInfo"
//)
//
//// JoinedGroup stores verifyGroup-related infos the current node joins in.
//// Note that, nodes outside the verifyGroup don't care the infos
//type JoinedGroup struct {
//	GroupID groupsig.ID        // Group ID
//	SignKey groupsig.Seckey    // Miner signature private key related to the verifyGroup
//	GroupPK groupsig.Pubkey    // Group public key (backup, which can be got from the global verifyGroup)
//	Members groupsig.PubkeyMap // Group related public keys of all members
//	gHash   common.Hash
//	lock    sync.RWMutex
//}
//
//type joinedGroupStore struct {
//	GroupID groupsig.ID        // Group ID
//	GroupPK groupsig.Pubkey    // Group public key (backup, which can be taken from the global verifyGroup)
//	Members groupsig.PubkeyMap // Group member signature public key
//}
//
//func newJoindGroup(n *GroupNode, gHash common.Hash) *JoinedGroup {
//	gpk := n.GetGroupPubKey()
//	joinedGroup := &JoinedGroup{
//		GroupPK: gpk,
//		SignKey: n.getSignSecKey(),
//		Members: make(groupsig.PubkeyMap, 0),
//		GroupID: *groupsig.NewIDFromPubkey(gpk),
//		gHash:   gHash,
//	}
//	return joinedGroup
//}
//
//func signKeySuffix(gid groupsig.ID) []byte {
//	return []byte(gid.GetHexString() + suffixSignKey)
//}
//
//func gInfoSuffix(gid groupsig.ID) []byte {
//	return []byte(gid.GetHexString() + suffixGInfo)
//}
//
//func (jg *JoinedGroup) addMemSignPK(mem groupsig.ID, signPK groupsig.Pubkey) {
//	jg.lock.Lock()
//	defer jg.lock.Unlock()
//	jg.Members[mem.GetHexString()] = signPK
//}
//
//func (jg *JoinedGroup) memSignPKSize() int {
//	jg.lock.RLock()
//	defer jg.lock.RUnlock()
//	return len(jg.Members)
//}
//
//// getMemSignPK get the signature public key of a member of the verifyGroup
//func (jg *JoinedGroup) getMemSignPK(mid groupsig.ID) (pk groupsig.Pubkey, ok bool) {
//	jg.lock.RLock()
//	defer jg.lock.RUnlock()
//	pk, ok = jg.Members[mid.GetHexString()]
//	return
//}
//
//func (jg *JoinedGroup) getMemberMap() groupsig.PubkeyMap {
//	jg.lock.RLock()
//	defer jg.lock.RUnlock()
//	m := make(groupsig.PubkeyMap, 0)
//	for key, pk := range jg.Members {
//		m[key] = pk
//	}
//	return m
//}
//
//// BelongGroups stores all verifyGroup-related infos which is important to the members
//type BelongGroups struct {
//	cache    *lru.Cache
//	priKey   common.PrivateKey
//	dirty    int32
//	store    *tasdb.LDBDatabase
//	storeDir string
//	initMu   sync.Mutex
//}
//
//func NewBelongGroups(file string, priKey common.PrivateKey) *BelongGroups {
//	return &BelongGroups{
//		dirty:    0,
//		priKey:   priKey,
//		storeDir: file,
//	}
//}
//
//func (bg *BelongGroups) initStore() {
//	bg.initMu.Lock()
//	defer bg.initMu.Unlock()
//
//	if bg.ready() {
//		return
//	}
//	options := &opt.Options{
//		OpenFilesCacheCapacity:        10,
//		WriteBuffer:                   32 * opt.MiB, // Two of these are used internally
//		Filter:                        filter.NewBloomFilter(10),
//		CompactionTableSize:           4 * opt.MiB,
//		CompactionTableSizeMultiplier: 2,
//		CompactionTotalSize:           16 * opt.MiB,
//		BlockSize:                     1 * opt.MiB,
//	}
//	db, err := tasdb.NewLDBDatabase(bg.storeDir, options)
//	if err != nil {
//		stdLogger.Errorf("newLDBDatabase fail, file=%v, err=%v\n", bg.storeDir, err.Error())
//		return
//	}
//
//	bg.store = db
//	bg.cache = common.MustNewLRUCache(30)
//}
//
//func (bg *BelongGroups) ready() bool {
//	return bg.cache != nil && bg.store != nil
//}
//
//func (bg *BelongGroups) storeSignKey(jg *JoinedGroup) {
//	if !bg.ready() {
//		return
//	}
//	pubKey := bg.priKey.GetPubKey()
//	ct, err := pubKey.Encrypt(rand.Reader, jg.SignKey.Serialize())
//	if err != nil {
//		stdLogger.Errorf("encrypt signkey fail, err=%v", err.Error())
//		return
//	}
//	bg.store.Put(signKeySuffix(jg.GroupID), ct)
//}
//
//func (bg *BelongGroups) storeGroupInfo(jg *JoinedGroup) {
//	if !bg.ready() {
//		return
//	}
//	st := joinedGroupStore{
//		GroupID: jg.GroupID,
//		GroupPK: jg.GroupPK,
//		Members: jg.getMemberMap(),
//	}
//	bs, err := json.Marshal(st)
//	if err != nil {
//		stdLogger.Errorf("marshal joinedGroup fail, err=%v", err)
//	} else {
//		bg.store.Put(gInfoSuffix(jg.GroupID), bs)
//	}
//}
//
//func (bg *BelongGroups) storeJoinedGroup(jg *JoinedGroup) {
//	bg.storeSignKey(jg)
//	bg.storeGroupInfo(jg)
//}
//
//func (bg *BelongGroups) loadJoinedGroup(gid groupsig.ID) *JoinedGroup {
//	if !bg.ready() {
//		return nil
//	}
//	jg := new(JoinedGroup)
//	jg.Members = make(groupsig.PubkeyMap, 0)
//	// Load signature private key
//	bs, err := bg.store.Get(signKeySuffix(gid))
//	if err != nil {
//		stdLogger.Errorf("get signKey fail, gid=%v, err=%v", gid.ShortS(), err.Error())
//		return nil
//	}
//	m, err := bg.priKey.Decrypt(rand.Reader, bs)
//	if err != nil {
//		stdLogger.Errorf("decrypt signKey fail, err=%v", err.Error())
//		return nil
//	}
//	jg.SignKey.Deserialize(m)
//
//	// Load verifyGroup information
//	infoBytes, err := bg.store.Get(gInfoSuffix(gid))
//	if err != nil {
//		stdLogger.Errorf("get gInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
//		return jg
//	}
//	if err := json.Unmarshal(infoBytes, jg); err != nil {
//		stdLogger.Errorf("unmarsal gInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
//		return jg
//	}
//	return jg
//}
//
//func fileExists(f string) bool {
//	// os.Stat gets file information
//	_, err := os.Stat(f)
//	if err != nil {
//		if os.IsExist(err) {
//			return true
//		}
//		return false
//	}
//	return true
//}
//func isDir(path string) bool {
//	s, err := os.Stat(path)
//	if err != nil {
//		return false
//	}
//	return s.IsDir()
//}
//
//func (bg *BelongGroups) joinedGroup2DBIfConfigExists(file string) bool {
//	if !fileExists(file) || isDir(file) {
//		return false
//	}
//	stdLogger.Debugf("load belongGroups from %v", file)
//	data, err := ioutil.ReadFile(file)
//	if err != nil {
//		stdLogger.Errorf("load file %v fail, err %v", file, err.Error())
//		return false
//	}
//	var gs []*JoinedGroup
//	err = json.Unmarshal(data, &gs)
//	if err != nil {
//		stdLogger.Errorf("unmarshal belongGroup store file %v fail, err %v", file, err.Error())
//		return false
//	}
//	n := 0
//	bg.initStore()
//	for _, jg := range gs {
//		if bg.getJoinedGroup(jg.GroupID) == nil {
//			n++
//			bg.addJoinedGroup(jg)
//		}
//	}
//	stdLogger.Debugf("joinedGroup2DBIfConfigExists belongGroups size %v", n)
//	return true
//}
//
//func (bg *BelongGroups) getJoinedGroup(id groupsig.ID) *JoinedGroup {
//	if !bg.ready() {
//		bg.initStore()
//	}
//	v, ok := bg.cache.Get(id.GetHexString())
//	if ok {
//		return v.(*JoinedGroup)
//	}
//	jg := bg.loadJoinedGroup(id)
//	if jg != nil {
//		bg.cache.Add(jg.GroupID.GetHexString(), jg)
//	}
//	return jg
//}
//
//func (bg *BelongGroups) addMemSignPk(uid groupsig.ID, gid groupsig.ID, signPK groupsig.Pubkey) (*JoinedGroup, bool) {
//	if !bg.ready() {
//		bg.initStore()
//	}
//	jg := bg.getJoinedGroup(gid)
//	if jg == nil {
//		return nil, false
//	}
//
//	if _, ok := jg.getMemSignPK(uid); !ok {
//		jg.addMemSignPK(uid, signPK)
//		bg.storeGroupInfo(jg)
//		return jg, true
//	}
//	return jg, false
//}
//
//func (bg *BelongGroups) addJoinedGroup(jg *JoinedGroup) {
//	if !bg.ready() {
//		bg.initStore()
//	}
//	newBizLog("addJoinedGroup").debug("add gid=%v", jg.GroupID.ShortS())
//	bg.cache.Add(jg.GroupID.GetHexString(), jg)
//	bg.storeJoinedGroup(jg)
//}
//
//func (bg *BelongGroups) leaveGroups(gids []groupsig.ID) {
//	if !bg.ready() {
//		return
//	}
//	for _, gid := range gids {
//		bg.cache.Remove(gid.GetHexString())
//		bg.store.Delete(gInfoSuffix(gid))
//	}
//}
//
//func (bg *BelongGroups) close() {
//	if !bg.ready() {
//		return
//	}
//	bg.cache = nil
//	bg.store.Close()
//}
//
//func (p *Processor) genBelongGroupStoreFile() string {
//	storeFile := p.conf.GetString(ConsensusConfSection, "groupstore", "")
//	if strings.TrimSpace(storeFile) == "" {
//		storeFile = "groupstore" + p.conf.GetString("instance", "index", "")
//	}
//	return storeFile
//}
//
//// getMemberSignPubKey get the signature public key of the member in the verifyGroup
//func (p Processor) getMemberSignPubKey(gmi *model.GroupMinerID) (pk groupsig.Pubkey, ok bool) {
//	if jg := p.belongGroups.getJoinedGroup(gmi.Gid); jg != nil {
//		pk, ok = jg.getMemSignPK(gmi.UID)
//		if !ok && !p.GetMinerID().IsEqual(gmi.UID) {
//			p.askSignPK(gmi)
//		}
//	}
//	return
//}
//
//// joinGroup join a verifyGroup (a miner ID can join multiple groups)
////			gid : verifyGroup ID (not dummy id)
////			sk: user's verifyGroup member signature private key
//func (p *Processor) joinGroup(g *JoinedGroup) {
//	stdLogger.Infof("begin Processor(%v)::joinGroup, gid=%v...\n", p.getPrefix(), g.GroupID.ShortS())
//	if !p.IsMinerGroup(g.GroupID) {
//		p.belongGroups.addJoinedGroup(g)
//	}
//	return
//}
//
//// getSignKey get the signature private key of the miner in a certain verifyGroup
//func (p Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
//	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
//		return jg.SignKey
//	}
//	return groupsig.Seckey{}
//}
//
//// IsMinerGroup detecting whether a verifyGroup is a miner's ingot verifyGroup
//// (a miner can participate in multiple groups)
//func (p *Processor) IsMinerGroup(gid groupsig.ID) bool {
//	return p.belongGroups.getJoinedGroup(gid) != nil
//}
//
//func (p *Processor) askSignPK(gmi *model.GroupMinerID) {
//	if !addSignPkReq(gmi.UID) {
//		return
//	}
//	msg := &model.ConsensusSignPubkeyReqMessage{
//		GroupID: gmi.Gid,
//	}
//	ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
//	if msg.GenSign(ski, msg) {
//		newBizLog("AskSignPK").debug("ask sign pk message, receiver %v, gid %v", gmi.UID.ShortS(), gmi.Gid.ShortS())
//		p.NetServer.AskSignPkMessage(msg, gmi.UID)
//	}
//}

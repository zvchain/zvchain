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
//// OnMessageCreateGroupPing handles Ping request from parent nodes
//// It only happens when current node is chosen to join a new verifyGroup
//func (p *Processor) OnMessageCreateGroupPing(msg *model.CreateGroupPingMessage) {
//	blog := newBizLog("OMCGPing")
//	var err error
//	defer func() {
//		if err != nil {
//			blog.error("from %v, gid %v, pingId %v, height=%v, won't pong, err=%v", msg.SI.GetID(), msg.FromGroupID, msg.PingID, msg.BaseHeight, err)
//		} else {
//			blog.debug("from %v, gid %v, pingId %v, height=%v, pong!", msg.SI.GetID(), msg.FromGroupID, msg.PingID, msg.BaseHeight)
//		}
//	}()
//	pk := GetMinerPK(msg.SI.GetID())
//	if pk == nil {
//		return
//	}
//	if msg.VerifySign(*pk) {
//		top := p.MainChain.Height()
//		if top <= msg.BaseHeight {
//			err = fmt.Errorf("localheight is %v, not enough", top)
//			return
//		}
//		pongMsg := &model.CreateGroupPongMessage{
//			PingID: msg.PingID,
//			Ts:     time.Now(),
//		}
//		group := p.GetGroup(msg.FromGroupID)
//		if group == nil {
//			err = fmt.Errorf("verifyGroup is nil:groupID=%v", msg.FromGroupID)
//			return
//		}
//		gb := &net.GroupBrief{
//			Gid:    msg.FromGroupID,
//			MemIds: group.GetMembers(),
//		}
//		if pongMsg.GenSign(p.getDefaultSeckeyInfo(), pongMsg) {
//			p.NetServer.SendGroupPongMessage(pongMsg, gb)
//		} else {
//			err = fmt.Errorf("gen sign fail")
//		}
//	} else {
//		err = fmt.Errorf("verify sign fail")
//	}
//}
//
//// OnMessageCreateGroupPong handles Pong response from new verifyGroup candidates
//// It only happens among the parent verifyGroup nodes
//func (p *Processor) OnMessageCreateGroupPong(msg *model.CreateGroupPongMessage) {
//	blog := newBizLog("OMCGPong")
//	var err error
//	defer func() {
//		blog.debug("from %v, pingId %v, got pong, ret=%v", msg.SI.GetID(), msg.PingID, err)
//	}()
//
//	ctx := p.groupManager.getContext()
//	if ctx == nil {
//		err = fmt.Errorf("creatingGroupCtx is nil")
//		return
//	}
//	if ctx.pingID != msg.PingID {
//		err = fmt.Errorf("pingId not equal, expect=%v, got=%v", p.groupManager.creatingGroupCtx.pingID, msg.PingID)
//		return
//	}
//	pk := GetMinerPK(msg.SI.GetID())
//	if pk == nil {
//		return
//	}
//
//	if msg.VerifySign(*pk) {
//		add, got := ctx.addPong(p.MainChain.Height(), msg.SI.GetID())
//		err = fmt.Errorf("size %v", got)
//		if add {
//			p.groupManager.checkReqCreateGroupSign(p.MainChain.Height())
//		}
//	} else {
//		err = fmt.Errorf("verify sign fail")
//	}
//}
//
//// OnMessageCreateGroupRaw triggered when receives raw verifyGroup-create message from other nodes of the parent verifyGroup
//// It check and sign the verifyGroup-create message for the requester
////
//// Before the formation of the new verifyGroup, the parent verifyGroup needs to reach a consensus on the information of the new verifyGroup
//// which transited by ConsensusCreateGroupRawMessage.
//func (p *Processor) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) {
//	blog := newBizLog("OMCGR")
//
//	gh := msg.GInfo.GI.GHeader
//	blog.debug("Proc(%v) begin, gHash=%v sender=%v", p.getPrefix(), gh.Hash, msg.SI.SignMember)
//
//	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
//		return
//	}
//	parentGid := msg.GInfo.GI.ParentID()
//
//	gpk, ok := p.getMemberSignPubKey(model.NewGroupMinerID(parentGid, msg.SI.SignMember))
//	if !ok {
//		blog.error("getMemberSignPubKey not ok, ask id %v", parentGid)
//		return
//	}
//
//	if !msg.VerifySign(gpk) {
//		return
//	}
//	if gh.Hash != gh.GenHash() || gh.Hash != msg.SI.DataHash {
//		blog.error("hash diff expect %v, receive %v", gh.GenHash(), gh.Hash)
//		return
//	}
//
//	tlog := newHashTraceLog("OMCGR", gh.Hash, msg.SI.GetID())
//	if ok, err := p.groupManager.onMessageCreateGroupRaw(msg); ok {
//		signMsg := &model.ConsensusCreateGroupSignMessage{
//			Launcher: msg.SI.SignMember,
//			GHash:    gh.Hash,
//		}
//		ski := p.getInGroupSeckeyInfo(parentGid)
//		if signMsg.GenSign(ski, signMsg) {
//			tlog.log("SendCreateGroupSignMessage id=%v", p.getPrefix())
//			blog.debug("OMCGR SendCreateGroupSignMessage... ")
//			p.NetServer.SendCreateGroupSignMessage(signMsg, parentGid)
//		} else {
//			blog.error("SendCreateGroupSignMessage sign fail, ski=%v, %v", ski.ID, ski.SK)
//		}
//
//	} else {
//		tlog.log("groupManager.onMessageCreateGroupRaw fail, err:%v", err.Error())
//	}
//}
//
//// OnMessageCreateGroupSign receives sign message from other members after ConsensusCreateGroupRawMessage was sent
//// during the new-verifyGroup-info consensus process
//func (p *Processor) OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage) {
//	blog := newBizLog("OMCGS")
//
//	blog.debug("Proc(%v) begin, gHash=%v, sender=%v", p.getPrefix(), msg.GHash, msg.SI.SignMember)
//	if p.GetMinerID().IsEqual(msg.SI.SignMember) {
//		return
//	}
//
//	if msg.GenHash() != msg.SI.DataHash {
//		blog.error("hash diff")
//		return
//	}
//
//	ctx := p.groupManager.getContext()
//	if ctx == nil {
//		blog.warn("context is nil")
//		return
//	}
//	mpk, ok := p.getMemberSignPubKey(model.NewGroupMinerID(ctx.parentInfo.GroupID, msg.SI.SignMember))
//	if !ok {
//		blog.error("getMemberSignPubKey not ok, ask id %v", ctx.parentInfo.GroupID)
//		return
//	}
//	if !msg.VerifySign(mpk) {
//		return
//	}
//	if ok, err := p.groupManager.onMessageCreateGroupSign(msg); ok {
//		gpk := ctx.parentInfo.GroupPK
//		if !groupsig.VerifySig(gpk, msg.SI.DataHash.Bytes(), ctx.gInfo.GI.Signature) {
//			blog.error("Proc(%v) verify verifyGroup sign fail", p.getPrefix())
//			return
//		}
//		initMsg := &model.ConsensusGroupRawMessage{
//			GInfo: *ctx.gInfo,
//		}
//
//		blog.debug("Proc(%v) send verifyGroup init Message", p.getPrefix())
//		ski := p.getDefaultSeckeyInfo()
//		if initMsg.GenSign(ski, initMsg) && ctx.getStatus() != sendInit {
//			tlog := newHashTraceLog("OMCGS", msg.GHash, msg.SI.GetID())
//			tlog.log("collecting pieces,SendGroupInitMessage")
//			p.NetServer.SendGroupInitMessage(initMsg)
//			ctx.setStatus(sendInit)
//
//		} else {
//			blog.error("genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
//		}
//
//	} else {
//		blog.error("fail, err=%v", err)
//	}
//}
//
//// OnMessageGroupInit receives new-verifyGroup-info messages from parent nodes and starts the verifyGroup formation process
//// That indicates the current node is chosen to be a member of the new verifyGroup
//func (p *Processor) OnMessageGroupInit(msg *model.ConsensusGroupRawMessage) {
//	blog := newBizLog("OMGI")
//	gHash := msg.GInfo.GroupHash()
//	gis := &msg.GInfo.GI
//	gh := gis.GHeader
//
//	blog.debug("proc(%v) begin, sender=%v, gHash=%v...", p.getPrefix(), msg.SI.GetID(), gHash)
//	tlog := newHashTraceLog("OMGI", gHash, msg.SI.GetID())
//
//	if msg.SI.DataHash != msg.GenHash() || gh.Hash != gh.GenHash() {
//		return
//	}
//
//	// Non-verifyGroup members do not follow the follow-up process
//	if !msg.MemberExist(p.GetMinerID()) {
//		return
//	}
//
//	groupContext := p.joiningGroups.GetGroup(gHash)
//	if groupContext != nil && groupContext.GetGroupStatus() != GisInit {
//		blog.debug("already handle, status=%v", groupContext.GetGroupStatus())
//		return
//	}
//
//	topHeight := p.MainChain.QueryTopBlock().Height
//	if gis.ReadyTimeout(topHeight) {
//		return
//	}
//
//	var candidates []groupsig.ID
//	cands, ok, err := p.groupManager.checkGroupInfo(&msg.GInfo)
//	if !ok {
//		blog.debug("verifyGroup header illegal, err=%v", err)
//		return
//	}
//	candidates = cands
//
//	tlog.logStart("%v", "")
//
//	groupContext = p.joiningGroups.ConfirmGroupFromRaw(msg, candidates, p.mi)
//	if groupContext == nil {
//		// hold it for now
//		panic("Processor::OMGI failed, ConfirmGroupFromRaw return nil.")
//	}
//
//	// Establish a verifyGroup network at local
//	p.NetServer.BuildGroupNet(gHash.Hex(), msg.GInfo.Mems)
//
//	gs := groupContext.GetGroupStatus()
//	blog.debug("joining verifyGroup(%v) status=%v.", gHash, gs)
//
//	// Use CAS operation to make sure the logical below executed once
//	if groupContext.StatusTransfrom(GisInit, GisSendSharePiece) {
//
//		// Generate secret sharing
//		shares := groupContext.GenSharePieces()
//
//		spm := &model.ConsensusSharePieceMessage{
//			GHash: gHash,
//		}
//		ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
//		spm.SI.SignMember = p.GetMinerID()
//		spm.MemCnt = int32(msg.GInfo.MemberSize())
//
//		// Send each node a different piece
//		for id, piece := range shares {
//			if id != "0x0" && piece.IsValid() {
//				spm.Dest.SetHexString(id)
//				spm.Share = piece
//				if spm.GenSign(ski, spm) {
//					blog.debug("piece to ID(%v), gHash=%v, share=%v, pub=%v.", spm.Dest, gHash, spm.Share.Share, spm.Share.Pub)
//					tlog.log("sharepiece to %v", spm.Dest)
//					blog.debug("call network service SendKeySharePiece...")
//					p.NetServer.SendKeySharePiece(spm)
//				} else {
//					blog.error("genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
//				}
//
//			} else {
//				blog.error("GenSharePieces data not isValid.")
//			}
//		}
//	}
//
//	return
//}
//
//// handleSharePieceMessage handles a piece information from other nodes
//// It has two sources:
//// One is that shared with each other during the verifyGroup formation process.
//// The other is the response obtained after actively requesting from the other party.
//func (p *Processor) handleSharePieceMessage(blog *bizLog, gHash common.Hash, share *model.SharePiece, si *model.SignData, response bool) (recover bool, err error) {
//	blog.debug("gHash=%v, sender=%v, response=%v", gHash, si.GetID(), response)
//	defer func() {
//		blog.debug("recover %v, err %v", recover, err)
//	}()
//
//	gc := p.joiningGroups.GetGroup(gHash)
//	if gc == nil {
//		err = fmt.Errorf("failed, receive SHAREPIECE msg but gc=nil.gHash=%v", gHash.Hex())
//		return
//	}
//	if gc.gInfo.GroupHash() != gHash {
//		err = fmt.Errorf("failed, gisHash diff")
//		return
//	}
//
//	pk := GetMinerPK(si.GetID())
//	if pk == nil {
//		err = fmt.Errorf("miner pk is nil, id=%v", si.GetID())
//		return
//	}
//	if !si.VerifySign(*pk) {
//		err = fmt.Errorf("miner sign verify fail")
//		return
//	}
//
//	gh := gc.gInfo.GI.GHeader
//
//	topHeight := p.MainChain.QueryTopBlock().Height
//
//	if !response && gc.gInfo.GI.ReadyTimeout(topHeight) {
//		err = fmt.Errorf("ready timeout, readyHeight=%v, now=%v", gh.ReadyHeight, topHeight)
//		return
//	}
//
//	result := gc.PieceMessage(si.GetID(), share)
//	waitPieceIds := make([]string, 0)
//	for _, mem := range gc.gInfo.Mems {
//		if !gc.node.hasPiece(mem) {
//			waitPieceIds = append(waitPieceIds, mem)
//			if len(waitPieceIds) >= 10 {
//				break
//			}
//		}
//	}
//
//	mtype := "OMSP"
//	if response {
//		mtype = "OMSPResponse"
//	}
//	tlog := newHashTraceLog(mtype, gHash, si.GetID())
//	tlog.log("number of pieces received %v, collecting slices %v, missing %v etc.", gc.node.groupInitPool.GetSize(), result == 1, waitPieceIds)
//
//	// All piece collected
//	if result == 1 {
//		recover = true
//		jg := gc.GetGroupInfo()
//		p.joinGroup(jg)
//
//		if jg.GroupPK.IsValid() && jg.SignKey.IsValid() {
//			ski := model.NewSecKeyInfo(p.mi.GetMinerID(), jg.SignKey)
//			// 1. Broadcast the verifyGroup-related public key to other members
//			if gc.StatusTransfrom(GisSendSharePiece, GisSendSignPk) {
//				msg := &model.ConsensusSignPubKeyMessage{
//					GroupID: jg.GroupID,
//					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
//					GHash:   gHash,
//					MemCnt:  int32(gc.gInfo.MemberSize()),
//				}
//				if !msg.SignPK.IsValid() {
//					// hold it for now
//					panic("signPK is InValid")
//				}
//				if msg.GenSign(ski, msg) {
//					tlog.log("SendSignPubKey %v", p.getPrefix())
//					p.NetServer.SendSignPubKey(msg)
//				} else {
//					err = fmt.Errorf("genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
//					return
//				}
//			}
//			// 2. Broadcast the complete verifyGroup information that has been initialized
//			if !response && gc.StatusTransfrom(GisSendSignPk, GisSendInited) {
//				msg := &model.ConsensusGroupInitedMessage{
//					GHash:        gHash,
//					GroupPK:      jg.GroupPK,
//					GroupID:      jg.GroupID,
//					CreateHeight: gh.CreateHeight,
//					ParentSign:   gc.gInfo.GI.Signature,
//					MemCnt:       int32(gc.gInfo.MemberSize()),
//					MemMask:      gc.generateMemberMask(),
//				}
//				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
//
//				if msg.GenSign(ski, msg) {
//					tlog.log("BroadcastGroupInfo %v", jg.GroupID)
//					p.NetServer.BroadcastGroupInfo(msg)
//				} else {
//					err = fmt.Errorf("genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
//					return
//				}
//			}
//		} else {
//			err = fmt.Errorf("Processor::%v failed, aggr key error", mtype)
//			return
//		}
//	}
//	return
//}
//
//// OnMessageSharePiece handles sharepiece message received from other members during the verifyGroup formation process.
//func (p *Processor) OnMessageSharePiece(spm *model.ConsensusSharePieceMessage) {
//	blog := newBizLog("OMSP")
//
//	p.handleSharePieceMessage(blog, spm.GHash, &spm.Share, &spm.SI, false)
//	return
//}
//
//// OnMessageSignPK handles verifyGroup-related public key messages received from other members
//// Simply stores the public key for future use
//func (p *Processor) OnMessageSignPK(spkm *model.ConsensusSignPubKeyMessage) {
//	blog := newBizLog("OMSPK")
//	tlog := newHashTraceLog("OMSPK", spkm.GHash, spkm.SI.GetID())
//
//	blog.debug("proc(%v) begin , sender=%v, gHash=%v, gid=%v...", p.getPrefix(), spkm.SI.GetID(), spkm.GHash, spkm.GroupID)
//
//	if spkm.GenHash() != spkm.SI.DataHash {
//		blog.error("spkm hash diff")
//		return
//	}
//
//	if !spkm.VerifySign(spkm.SignPK) {
//		blog.error("miner sign verify fail")
//		return
//	}
//
//	removeSignPkRecord(spkm.SI.GetID())
//
//	jg, ret := p.belongGroups.addMemSignPk(spkm.SI.GetID(), spkm.GroupID, spkm.SignPK)
//
//	if jg != nil {
//		blog.debug("after SignPKMessage exist mem sign pks=%v, ret=%v", jg.memSignPKSize(), ret)
//		tlog.log("signed public keys received count %v", jg.memSignPKSize())
//		for mem, pk := range jg.getMemberMap() {
//			blog.debug("signPKS: %v, %v", mem, pk.GetHexString())
//		}
//	}
//
//	return
//}
//
//// OnMessageSignPKReq receives verifyGroup-related public key request from other members and
//// responses own public key
//func (p *Processor) OnMessageSignPKReq(msg *model.ConsensusSignPubkeyReqMessage) {
//	blog := newBizLog("OMSPKR")
//	sender := msg.SI.GetID()
//	var err error
//	defer func() {
//		blog.debug("sender=%v, gid=%v, result=%v", sender, msg.GroupID, err)
//	}()
//
//	jg := p.belongGroups.getJoinedGroup(msg.GroupID)
//	if jg == nil {
//		err = fmt.Errorf("failed, local node not found joinedGroup with verifyGroup id=%v", msg.GroupID)
//		return
//	}
//
//	pk := GetMinerPK(sender)
//	if pk == nil {
//		err = fmt.Errorf("get minerPK is nil, id=%v", sender)
//		return
//	}
//	if !msg.VerifySign(*pk) {
//		err = fmt.Errorf("verifySign fail, pk=%v, sign=%v", pk.GetHexString(), msg.SI.DataSign.GetHexString())
//		return
//	}
//	if !jg.SignKey.IsValid() {
//		err = fmt.Errorf("invalid sign secKey, id=%v, sk=%v", p.GetMinerID(), jg.SignKey)
//		return
//	}
//
//	resp := &model.ConsensusSignPubKeyMessage{
//		GHash:   jg.gHash,
//		GroupID: msg.GroupID,
//		SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
//	}
//	ski := model.NewSecKeyInfo(p.GetMinerID(), jg.SignKey)
//	if resp.GenSign(ski, resp) {
//		blog.debug("answer signPKReq Message, receiver %v, gid %v", sender, msg.GroupID)
//		p.NetServer.AnswerSignPkMessage(resp, sender)
//	} else {
//		err = fmt.Errorf("gen Sign fail, ski=%v,%v", ski.ID, ski.SK.GetHexString())
//	}
//}
//
//func (p *Processor) acceptGroup(staticGroup *StaticGroupInfo) {
//	add := p.globalGroups.AddStaticGroup(staticGroup)
//	blog := newBizLog("acceptGroup")
//	blog.debug("Add to Global static groups, result=%v, groups=%v.", add, p.globalGroups.GetGroupSize())
//	if staticGroup.MemExist(p.GetMinerID()) {
//		p.prepareForCast(staticGroup)
//	}
//}
//
//// OnMessageGroupInited is a network-wide node processing function.
//// The entire network node receives a verifyGroup of initialized completion messages from all of the members in the verifyGroup
//// and when 51% of the same message received from the verifyGroup members, the verifyGroup will be added on chain
//func (p *Processor) OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage) {
//	blog := newBizLog("OMGIED")
//	gHash := msg.GHash
//
//	blog.debug("proc(%v) begin, sender=%v, gHash=%v, gid=%v, gpk=%v...", p.getPrefix(),
//		msg.SI.GetID(), gHash, msg.GroupID, msg.GroupPK)
//	tlog := newHashTraceLog("OMGIED", gHash, msg.SI.GetID())
//
//	if msg.SI.DataHash != msg.GenHash() {
//		blog.error("grm gis hash diff")
//		return
//	}
//
//	// The verifyGroup already added on chain before because of synchronization process
//	g := p.GroupChain.GetGroupByID(msg.GroupID.Serialize())
//	if g != nil {
//		blog.debug("verifyGroup already onchain")
//		p.globalGroups.removeInitedGroup(gHash)
//		p.joiningGroups.Clean(gHash)
//		return
//	}
//
//	pk := GetMinerPK(msg.SI.GetID())
//	if !msg.VerifySign(*pk) {
//		blog.error("verify sign fail, id=%v, pk=%v, sign=%v", msg.SI.GetID(), pk.GetHexString(), msg.SI.DataSign.GetHexString())
//		return
//	}
//
//	initedGroup := p.globalGroups.GetInitedGroup(msg.GHash)
//	if initedGroup == nil {
//		gInfo, err := p.groupManager.recoverGroupInitInfo(msg.CreateHeight, msg.MemMask)
//		if err != nil {
//			blog.error("recover verifyGroup info fail, err %v", err)
//			return
//		}
//		if gInfo.GroupHash() != msg.GHash {
//			blog.error("groupHeader hash error, expect %v, receive %v", gInfo.GroupHash().Hex(), msg.GHash.Hex())
//			return
//		}
//		gInfo.GI.Signature = msg.ParentSign
//		initedGroup = createInitedGroup(gInfo)
//		blog.debug("add inited verifyGroup")
//	}
//	// Check the time window, deny messages out of date
//	if initedGroup.gInfo.GI.ReadyTimeout(p.MainChain.Height()) {
//		blog.warn("verifyGroup ready timeout, gid=%v", msg.GroupID)
//		return
//	}
//
//	parentID := initedGroup.gInfo.GI.ParentID()
//	parentGroup := p.GetGroup(parentID)
//	if parentGroup == nil {
//		blog.error("verifyGroup is nil:groupID=%v", parentID)
//		return
//	}
//
//	gpk := parentGroup.GroupPK
//	if !groupsig.VerifySig(gpk, msg.GHash.Bytes(), msg.ParentSign) {
//		blog.error("verify parent groupsig fail! gHash=%v", gHash)
//		return
//	}
//	if !initedGroup.gInfo.GI.Signature.IsEqual(msg.ParentSign) {
//		blog.error("signature differ, old %v, new %v", initedGroup.gInfo.GI.Signature.GetHexString(), msg.ParentSign.GetHexString())
//		return
//	}
//	initedGroup = p.globalGroups.generator.addInitedGroup(initedGroup)
//
//	result := initedGroup.receive(msg.SI.GetID(), msg.GroupPK)
//
//	waitIds := make([]string, 0)
//	for _, mem := range initedGroup.gInfo.Mems {
//		if !initedGroup.hasReceived(mem) {
//			waitIds = append(waitIds, mem)
//			if len(waitIds) >= 10 {
//				break
//			}
//		}
//	}
//
//	tlog.log("ret:%v,number of messages received %v, number of messages required %v, missing %v etc.", result, initedGroup.receiveSize(), initedGroup.threshold, waitIds)
//
//	switch result {
//	case InitSuccess: // Receive the same message in the verifyGroup >= threshold, can add on chain
//		staticGroup := newSGIFromStaticGroupSummary(msg.GroupID, msg.GroupPK, initedGroup)
//		gh := staticGroup.getGroupHeader()
//		blog.debug("SUCCESS accept a new verifyGroup, gHash=%v, gid=%v, workHeight=%v, dismissHeight=%v.", gHash, msg.GroupID, gh.WorkHeight, gh.DismissHeight)
//
//		p.groupManager.addGroupOnChain(staticGroup)
//		p.globalGroups.removeInitedGroup(gHash)
//		p.joiningGroups.Clean(gHash)
//
//	case InitFail: // The verifyGroup is initialized abnormally and cannot be recovered
//		tlog.log("initialization failed")
//		p.globalGroups.removeInitedGroup(gHash)
//	}
//	return
//}
//
//// OnMessageSharePieceReq receives share piece request from other members
//// It happens in the case that the current node didn't heard from the other part during the piece-sharing with each other process.
//func (p *Processor) OnMessageSharePieceReq(msg *model.ReqSharePieceMessage) {
//	blog := newBizLog("OMSPR")
//	blog.debug("gHash=%v, sender=%v", msg.GHash, msg.SI.GetID())
//
//	pk := GetMinerPK(msg.SI.GetID())
//	if pk == nil || !msg.VerifySign(*pk) {
//		blog.error("verify sign fail")
//		return
//	}
//	gc := p.joiningGroups.GetGroup(msg.GHash)
//	if gc == nil {
//		blog.warn("gc is nil")
//		return
//	}
//	if gc.sharePieceMap == nil {
//		blog.warn("sharePiece map is nil")
//		return
//	}
//	piece := gc.sharePieceMap[msg.SI.GetID().GetHexString()]
//
//	pieceMsg := &model.ResponseSharePieceMessage{
//		GHash: msg.GHash,
//		Share: piece,
//	}
//	if pieceMsg.GenSign(p.getDefaultSeckeyInfo(), pieceMsg) {
//		blog.debug("response share piece to %v, gHash=%v, share=%v", msg.SI.GetID(), msg.GHash, piece.Share)
//		p.NetServer.ResponseSharePiece(pieceMsg, msg.SI.GetID())
//	}
//}
//
//// OnMessageSharePieceResponse receives share piece message from other member after requesting
//func (p *Processor) OnMessageSharePieceResponse(msg *model.ResponseSharePieceMessage) {
//	blog := newBizLog("OMSPRP")
//
//	p.handleSharePieceMessage(blog, msg.GHash, &msg.Share, &msg.SI, true)
//	return
//}

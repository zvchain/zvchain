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

package net

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/network"
)

// ConsensusHandler used for handling consensus-related messages from network
type ConsensusHandler struct {
	processor MessageProcessor
}

var MessageHandler = new(ConsensusHandler)

func (c *ConsensusHandler) Init(proc MessageProcessor) {
	c.processor = proc
	initStateMachines()
}

func (c *ConsensusHandler) Processor() MessageProcessor {
	return c.processor
}

func (c *ConsensusHandler) ready() bool {
	return c.processor != nil && c.processor.Ready()
}

// Handle is the main entrance for handling messages.
// It assigns different types of messages to different processor handlers for processing according to the code field
func (c *ConsensusHandler) Handle(sourceID string, msg network.Message) error {
	code := msg.Code
	body := msg.Body

	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	if !c.ready() {
		log.Printf("message ingored because processor not ready. code=%v\n", code)
		return fmt.Errorf("processor not ready yet")
	}
	switch code {
	case network.GroupInitMsg:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return e
		}

		GroupInsideMachines.GetMachine(m.GInfo.GI.GetHash().Hex(), len(m.GInfo.Mems)).transform(newStateMsg(code, m, sourceID))
	case network.KeyPieceMsg:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GHash.Hex(), int(m.MemCnt)).transform(newStateMsg(code, m, sourceID))
		logger.Infof("SharepieceMsg receive from:%v, gHash:%v", sourceID, m.GHash.Hex())
	case network.SignPubkeyMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GHash.Hex(), int(m.MemCnt)).transform(newStateMsg(code, m, sourceID))
		logger.Infof("SignPubKeyMsg receive from:%v, gHash:%v, groupId:%v", sourceID, m.GHash.Hex(), m.GroupID.GetHexString())
	case network.GroupInitDoneMsg:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return e
		}
		logger.Infof("Rcv GroupInitDoneMsg from:%s,gHash:%s, groupId:%v", sourceID, m.GHash.Hex(), m.GroupID.GetHexString())

		GroupInsideMachines.GetMachine(m.GHash.Hex(), int(m.MemCnt)).transform(newStateMsg(code, m, sourceID))

	case network.CastVerifyMsg:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return e
		}
		c.processor.OnMessageCast(m)
	case network.VerifiedCastMsg:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageVerify(m)

	case network.CreateGroupaRaw:
		m, e := unMarshalConsensusCreateGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupRawMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupRaw(m)
		return nil
	case network.CreateGroupSign:
		m, e := unMarshalConsensusCreateGroupSignMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupSign(m)
		return nil
	case network.CastRewardSignReq:
		m, e := unMarshalCastRewardReqMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard CastRewardSignReqMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCastRewardSignReq(m)
	case network.CastRewardSignGot:
		m, e := unMarshalCastRewardSignMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard CastRewardSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCastRewardSign(m)
	case network.AskSignPkMsg:
		m, e := unMarshalConsensusSignPKReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalConsensusSignPKReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSignPKReq(m)
	case network.AnswerSignPkMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSignPK(m)

	case network.GroupPing:
		m, e := unMarshalCreateGroupPingMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalCreateGroupPingMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageCreateGroupPing(m)
	case network.GroupPong:
		m, e := unMarshalCreateGroupPongMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalCreateGroupPongMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageCreateGroupPong(m)

	case network.ReqSharePiece:
		m, e := unMarshalSharePieceReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalSharePieceReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSharePieceReq(m)

	case network.ResponseSharePiece:
		m, e := unMarshalSharePieceResponseMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalSharePieceResponseMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSharePieceResponse(m)

	case network.ReqProposalBlock:
		m, e := unmarshalReqProposalBlockMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unmarshalReqProposalBlockMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageReqProposalBlock(m, sourceID)

	case network.ResponseProposalBlock:
		m, e := unmarshalResponseProposalBlockMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unmarshalResponseProposalBlockMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageResponseProposalBlock(m)

	}

	return nil
}

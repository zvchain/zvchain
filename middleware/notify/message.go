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

package notify

import (
	"github.com/zvchain/zvchain/middleware/types"
)

type BlockOnChainSuccMessage struct {
	Block *types.Block
}

func (m *BlockOnChainSuccMessage) GetRaw() []byte {
	return []byte{}
}
func (m *BlockOnChainSuccMessage) GetData() interface{} {
	return m.Block
}

type GroupMessage struct {
	Group *types.Group
}

func (m *GroupMessage) GetRaw() []byte {
	return []byte{}
}
func (m *GroupMessage) GetData() interface{} {
	return m.Group
}

// DefaultMessage is a default implementation of the Message interface.
// It can meet most of demands abort chain event
type DefaultMessage struct {
	body            []byte
	source          string
	chainID         uint16
	protocalVersion uint16
}

func (m *DefaultMessage) GetRaw() []byte {
	panic("implement me")
}

func (m *DefaultMessage) GetData() interface{} {
	return m.Body
}

func (m *DefaultMessage) Body() []byte {
	return m.body
}

func (m *DefaultMessage) Source() string {
	return m.source
}
func (m *DefaultMessage) ChainID() uint16 {
	return m.chainID
}
func (m *DefaultMessage) ProtocalVersion() uint16 {
	return m.protocalVersion
}

func NewDefaultMessage(body []byte, from string, chainID, protocal uint16) *DefaultMessage {
	return &DefaultMessage{body: body, source: from, chainID: chainID, protocalVersion: protocal}
}

func AsDefault(message Message) *DefaultMessage {
	return message.(*DefaultMessage)
}

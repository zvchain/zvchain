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

package tvm

import (
	"github.com/zvchain/zvchain/common"
	"math/big"
)

type minerOpMsg struct {
	source  *common.Address
	target  *common.Address
	value   *big.Int
	payload []byte
	typ     int8
}

func (msg *minerOpMsg) OpType() int8 {
	return msg.typ
}

func (msg *minerOpMsg) Operator() *common.Address {
	return msg.source
}

func (msg *minerOpMsg) OpTarget() *common.Address {
	return msg.target
}

func (msg *minerOpMsg) Amount() *big.Int {
	return msg.value
}

func (msg *minerOpMsg) Payload() []byte {
	return msg.payload
}

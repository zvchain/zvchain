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

package group

import (
	"bytes"
	"github.com/zvchain/zvchain/middleware/types"
	"reflect"
	"testing"
)

type senderI2 interface {
	types.SenderI
}

type senderImpl struct {
	sender []byte
}

func (s *senderImpl) Sender() []byte {
	return s.sender
}

func findSender1(senderArray interface{}, sender []byte) (bool, types.SenderI) {
	value := reflect.ValueOf(senderArray)
	for i := 0; i < value.Len(); i++ {
		v := value.Index(i)
		senderI := v.Interface().(types.SenderI)
		if bytes.Equal(senderI.Sender(), sender) {
			return true, senderI
		}
	}
	return false, nil
}

func TestFindSender(t *testing.T) {
	senders := make([]senderI2, 0)
	senders = append(senders, &senderImpl{sender: []byte{1}})
	senders = append(senders, &senderImpl{sender: []byte{2}})

	ok, s := findSender1(senders, []byte{1})
	t.Log(ok, s)
}

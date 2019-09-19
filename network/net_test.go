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

package network

import (
	"math/rand"
	"testing"
	"time"
)

func TestRandomPerm(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomNodes := r.Perm(10)

	t.Logf("node:%v", randomNodes)
	if len(randomNodes) != 10 {
		t.Fatalf("size is wrony")
	}

}

func TestDecodeMessage(t *testing.T) {
	if !InitTestNetwork() {
		t.Fatalf("init network failed")
	}

	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(int(MessageType_MessagePing), 1024, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := netCore.decodeMessage(p)

	t.Logf("type :%v,size :%v,err : %v", msgType, packetSize, err)

	if err == nil {
		t.Fatalf("decode error:%v", err)
	}

}

func Test_HandleMessagePanic(t *testing.T) {
	if !InitTestNetwork() {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(int(1024), 1024, pdata)

	p.addRecvData(packet.Bytes())

	err := netCore.handleMessage(p)

	t.Logf("err:%v", err)

}

func TestDecodeMessage2(t *testing.T) {
	if !InitTestNetwork() {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	for i := 0; i < 1024; i++ {
		pdata[i] = byte(i % 256)
	}
	packet := encodePacket(int(MessageType_MessageData), 1024, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := netCore.decodeMessage(p)

	t.Logf("type :%v,size :%v,err:%v", msgType, packetSize, err)

	if err == nil {
		t.Fatalf("decode error is nil")
	}
}

func TestHandleMessageUnknownMessage(t *testing.T) {
	if !InitTestNetwork() {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	for i := 0; i < 1024; i++ {
		pdata[i] = byte(i % 256)
	}
	packet := encodePacket(int(1024), 1024, pdata)

	p.addRecvData(packet.Bytes())

	err := netCore.handleMessage(p)

	t.Logf("err:%v \n", err)

	if err == nil {
		t.Fatalf("decode error is nil")
	}
}

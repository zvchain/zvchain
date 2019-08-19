package network

import (
	"fmt"
	"testing"
)

func TestDecodeMessage(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}

	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(int(MessageType_MessagePing), 1024, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := netCore.decodeMessage(p)

	fmt.Printf("type :%v,size :%v,err:%v", msgType, packetSize, err)

	if err == nil {
		t.Fatalf("decode error:%v", err)
	}

}

func TestHandleMessagePanic(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(int(1024), 1024, pdata)

	p.addRecvData(packet.Bytes())

	err := netCore.handleMessage(p)

	fmt.Printf("err:%v", err)

}

func TestDecodeMessage2(t *testing.T) {
	if InitTestNetwork() == false {
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

	fmt.Printf("type :%v,size :%v,err:%v", msgType, packetSize, err)

	if err == nil {
		t.Fatalf("decode error:%v", err)
	}
}

func TestHandleMessageUnknownMessage(t *testing.T) {
	if InitTestNetwork() == false {
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

	fmt.Printf("err:%v \n", err)

	if err == nil {
		t.Fatalf("decode error:%v", err)
	}
}

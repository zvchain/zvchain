package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/zvchain/zvchain/common"
)

func TestPeerAuth(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}

	toID := NewNodeID(netServerInstance.config.NodeIDHex)
	content := genPeerAuthContext(netServerInstance.config.PK, netServerInstance.config.SK, toID)

	result, verifyID := content.Verify()
	if !result || verifyID != netServerInstance.config.NodeIDHex {
		t.Fatalf("PeerAuth verify failed,result:%v,PK:%v,verifyPK:%v", result, netServerInstance.config.NodeIDHex, verifyID)
	}

}

func InitTestNetwork() bool {
	SK, _ := common.GenerateKey("")
	PK := SK.GetPubKey()
	ID := PK.GetAddress()
	Seeds := make([]string, 0)
	netCfg := NetworkConfig{IsSuper: false,
		TestMode:        true,
		NatAddr:         "",
		NatPort:         0,
		SeedAddr:        "",
		NodeIDHex:       ID.Hex(),
		ChainID:         0,
		ProtocolVersion: common.ProtocolVersion,
		SeedIDs:         Seeds,
		PK:              PK.Hex(),
		SK:              SK.Hex(),
	}

	err := Init(nil, nil, netCfg)

	return err == nil
}

func TestDecodePacketNil(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	p.addRecvData(nil)

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err == nil {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 0 {
		t.Fatalf("msgType wrong")
	}

	if packetSize != 0 {
		t.Fatalf("packetSize wrong")
	}
}

func TestDecodePacket2BuffersEq8(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(2, 1024, pdata)

	p.addRecvData(packet.Bytes()[0:4])
	p.addRecvData(packet.Bytes()[4:8])
	p.addRecvData(packet.Bytes()[8:])

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err != nil {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 2 {
		t.Fatalf("msgType wrong")
	}

	if packetSize != 1024+8 {
		t.Fatalf("packetSize wrong")
	}
}

func TestDecodePacket2BuffersLess8(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(2, 1024, pdata)

	p.addRecvData(packet.Bytes()[0:4])
	p.addRecvData(packet.Bytes()[4:7])

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v,remain size:%v\n", msgType, packetSize, p.getDataSize())
	if err != errPacketTooSmall {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 0 {
		t.Fatalf("msgType wrong")
	}

	if p.getDataSize() != 7 {
		t.Fatalf("packetSize wrong")
	}

}

func TestDecodePacket3Buffers2BuffersLess8(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(2, 1024, pdata)

	p.addRecvData(packet.Bytes()[0:4])
	p.addRecvData(packet.Bytes()[4:7])
	p.addRecvData(packet.Bytes()[7:])

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err != nil {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 2 {
		t.Fatalf("msgType wrong")
	}

	if packetSize != 1024+8 {
		t.Fatalf("packetSize wrong")
	}

}

func TestDecodePacketSmall(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(2, 1024, pdata)

	p.addRecvData(packet.Bytes()[:512])

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err != errPacketTooSmall {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 0 {
		t.Fatalf("msgType wrong")
	}

	if packetSize != 0 {
		t.Fatalf("packetSize wrong")
	}

}

func TestDecodePacket16M(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)
	dataSize := uint32(16 * 1024 * 1024)
	pdata := make([]byte, dataSize)

	packet := encodePacket(2, dataSize, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err != nil {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 2 {
		t.Fatalf("msgType wrong")
	}

	if uint32(packetSize) != dataSize+8 {
		t.Fatalf("packetSize wrong")
	}

}

func TestDecodePacketOver16M(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)
	dataSize := uint32(18 * 1024 * 1024)
	pdata := make([]byte, dataSize)

	packet := encodePacket(2, dataSize, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err == nil {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 0 {
		t.Fatalf("msgType wrong")
	}

	if uint32(packetSize) != 0 {
		t.Fatalf("packetSize wrong")
	}

}

func TestDecodePacketOverflow(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(2, math.MaxUint32-1, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v\n", msgType, packetSize)
	if err != errBadPacket {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 0 {
		t.Fatalf("msgType wrong")
	}

	if packetSize != 0 {
		t.Fatalf("packetSize wrong")
	}

}

func TestDecodePacketBigBuffer(t *testing.T) {
	if InitTestNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(netCore.ID, 0)

	pdata := make([]byte, 1024)

	packet := encodePacket(100122, 512, pdata)

	p.addRecvData(packet.Bytes())

	msgType, packetSize, _, _, err := p.decodePacket()

	fmt.Printf("type :%v,size %v,p.getDataSize():%v\n", msgType, packetSize, p.getDataSize())
	if err != nil {
		t.Fatalf("decode error:%v", err)
	}
	if msgType != 100122 {
		t.Fatalf("msgType wrong")
	}

	if packetSize != 512+8 {
		t.Fatalf("packetSize wrong")
	}

	if p.getDataSize() != 512 {
		t.Fatalf("data size wrong")
	}

}

func encodePacket(pType int, len uint32, pdata []byte) *bytes.Buffer {

	length := len
	bufferSize := int(length + PacketHeadSize)
	if bufferSize > 64*1024*1024 {
		bufferSize = 64 * 1024 * 1024
	}
	b := netCore.bufferPool.getBuffer(bufferSize)

	err := binary.Write(b, binary.BigEndian, uint32(pType))
	if err != nil {
		return nil
	}
	err = binary.Write(b, binary.BigEndian, uint32(length))
	if err != nil {
		return nil
	}

	b.Write(pdata)
	return b
}

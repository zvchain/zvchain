package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"math"
	"testing"
)

func TestPeerAuth(t *testing.T) {
	SK, _ := common.GenerateKey("")
	PK := SK.GetPubKey()
	ID := PK.GetAddress()

	content := genPeerAuthContext(PK.Hex(), SK.Hex(), nil)

	result, verifyID := content.Verify()
	if !result || verifyID != ID.Hex() {
		t.Fatalf("PeerAuth verify failed,result:%v,PK:%v,verifyPK:%v", result, ID.Hex(), verifyID)
	}

}

func InitNetwork() bool {
	SK, _ := common.GenerateKey("")
	PK := SK.GetPubKey()
	ID := PK.GetAddress()
	Seeds := make([]string, 0, 0)
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
	if err != nil {
		return false
	}
	return true
}

func TestDecodePacket(t *testing.T) {
	if InitNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(NewNodeID(""), 0)

	pdata := make([]byte, 1024, 1024)

	packet := encodePacket(2, 1024, pdata)

	p.addRecvData(packet.Bytes())

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
	if InitNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(NewNodeID(""), 0)

	pdata := make([]byte, 1024, 1024)

	packet := encodePacket(2, 2024, pdata)

	p.addRecvData(packet.Bytes())

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

func TestDecodePacketOverflow(t *testing.T) {
	if InitNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(NewNodeID(""), 0)

	pdata := make([]byte, 1024, 1024)

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
	if InitNetwork() == false {
		t.Fatalf("init network failed")
	}
	p := newPeer(NewNodeID(""), 0)

	pdata := make([]byte, 1024, 1024)

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
	if bufferSize > 16*1024*1024 {
		bufferSize = 16 * 1024 * 1024
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

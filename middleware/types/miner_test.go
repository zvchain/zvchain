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

package types

import (
	"bytes"
	"github.com/zvchain/zvchain/common"
	"testing"
)

var mpks = &MinerPks{
	MType: MinerTypeProposal,
	Pk:    common.FromHex("0x215fdace84c59a6d86e1cbe4238c3e4a5d7a6e07f6d4c5603399e573cc05a32617faae51cfd3fce7c84447522e52a1439f46fc5adb194240325fcb800a189ae129ebca2b59999a9ecd16e03184e7fe578418b20cbcdc02129adc79bf090534a80fb9076c3518ae701477220632008fc67981e2a1be97a160a2f9b5804f9b280f"),
	VrfPk: common.FromHex("0x7bc1cb6798543feb524456276d9b26014ddfb5cd757ac6063821001b50679bcf"),
}

func TestEncodePayload(t *testing.T) {

	bs, err := EncodePayload(mpks)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 2+128+32 {
		t.Error("length error")
	}
	t.Log(common.ToHex(bs))
}

func TestDecodePayload(t *testing.T) {
	bs := common.FromHex("0x0101215fdace84c59a6d86e1cbe4238c3e4a5d7a6e07f6d4c5603399e573cc05a32617faae51cfd3fce7c84447522e52a1439f46fc5adb194240325fcb800a189ae129ebca2b59999a9ecd16e03184e7fe578418b20cbcdc02129adc79bf090534a80fb9076c3518ae701477220632008fc67981e2a1be97a160a2f9b5804f9b280f7bc1cb6798543feb524456276d9b26014ddfb5cd757ac6063821001b50679bcf")
	pks, err := DecodePayload(bs)
	if err != nil {
		t.Fatal(err)
	}
	if !IsProposalRole(pks.MType) {
		t.Error("should be proposal type")
	}
	if !bytes.Equal(pks.Pk, mpks.Pk) {
		t.Error("pk error")
	}
	if !bytes.Equal(pks.VrfPk, mpks.VrfPk) {
		t.Error("vrf pk error")
	}
}

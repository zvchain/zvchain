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

package groupsig

import (
	"testing"

	"github.com/zvchain/zvchain/common"
)

/*
**  Creator: pxf
**  Date: 2018/10/16 下午2:39
**  Description:
 */
//
//func TestSignature_MarshalJSON(t *testing.T) {
//	sig := Signature{}
//	sig.SetHexString("0x123")
//	bs := sig.GetHexString()
//	t.Log(string(bs))
//}

func TestVerifySig(t *testing.T) {
	var sign Signature
	sign.SetHexString("0x05ead28fcc43a2d28b212b4c29510b1d9af8da3ec3b75055820b2f0f39abdb9401")
	var gpk Pubkey
	gpk.SetHexString("0x1087cd0d07107598c39bf18a9899dcb4feb34cf9072d1d8a891e58a16bb50c97164c6bc19515d066b1b93d2ab88723ccc912e9f098de0f5b326008c529bd2cf00343c5438788dd8e031cb65f65a05881a508f519b425e7d8b3bf3ea702e60ff5208453ed6f2cf250706058c5616e5c41a4b0a4ad9f4aa68eb0c5a786c43ab2ef")
	var hash = common.HexToHash("0x05cf9c5359bfd6c8487ebb120c4f69cddb95466e75ef386b556ba2ad02247e6e")

	t.Log(VerifySig(gpk, hash.Bytes(), sign))
}

func TestSKRecover(t *testing.T) {
	sks := "0x64b59f9ff74d2143a70d7e3c18edaef5750974bc08e5e34b3c57f8b95ea2a8"
	pks := "0x2295308fd6ba3783fac06d83eb83d4a7cd2850c4eff03451e10288506f7d549026be374d220a07b8de2ad2b953badafd42be7b0e88d947bb26ff264b64c24ed02f53da70cce2c7e55b6486f630723fb19a5ffd35465855028b7439993784a02c238633b93852e17fa2d011501cc89428be1c8619f7dd7ba6b32b38bd7ead09a8"
	signs := "0x0a2706f4fd597a3d0585f22a2be9d4367971fed626182857d77887cd9e39afa001"
	datas := "0xd7b61f93d478db97aecfcdbbc4f22987099b6f062afeaa39b28af6cbf3e5e9a5"

	var (
		sk   Seckey
		pk   Pubkey
		sig  Signature
		data common.Hash
	)

	sk.SetHexString(sks)

	t.Log(len(sk.Serialize()))
	pk.SetHexString(pks)
	sig.SetHexString(signs)
	data = common.HexToHash(datas)

	if data.Hex() != datas {
		t.Errorf("hash diff")
	}

	pkRecover := NewPubkeyFromSeckey(sk)
	if !pkRecover.IsEqual(pk) {
		t.Errorf("pk recover:%v", pkRecover.GetHexString())
	}
}

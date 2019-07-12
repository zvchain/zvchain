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

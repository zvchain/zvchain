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

package groupsig

import (
	"encoding/json"
	"testing"
)

func TestSeckey_MarshalJSON(t *testing.T) {
	var pk = Seckey{}
	pk.SetHexString("0x2cb5f53ef1fed6d682ee1013d188fd3d1995fed14cc2bc19b0dd7593824ee11e")

	jsonBytes, err := json.Marshal(&pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonBytes))

}

func TestSeckey_UnmarshalJSON(t *testing.T) {
	var pk = Seckey{}
	pk.SetHexString("0x2cb5f53ef1fed6d682ee1013d188fd3d1995fed14cc2bc19b0dd7593824ee11e")

	jsonBytes, err := json.Marshal(&pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("size ", len(jsonBytes))
	pk2 := &Seckey{}
	err = json.Unmarshal(jsonBytes, pk2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(pk2.GetHexString())
	if pk2.GetHexString() != pk.GetHexString() {
		t.Errorf("unmarshal error")
	}
}

func TestSeckey_GetHexString(t *testing.T) {
	var sk Seckey
	t.Log(sk)
	t.Log(sk.GetHexString())
	t.Log(len(sk.Serialize()))
}

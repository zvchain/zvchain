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

package base

import (
	"encoding/json"
	"testing"
)

func TestVRFPublicKey_MarshalJSON(t *testing.T) {
	pk := Hex2VRFPublicKey("0x7eb45a50b618678f3649decd693389ab8043e5780e02f2718c1f3de9a50cd03a")

	jsonBytes, err := json.Marshal(&pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonBytes))

}

func TestVRFPublicKey_UnmarshalJSON(t *testing.T) {
	pk := Hex2VRFPublicKey("0x7eb45a50b618678f3649decd693389ab8043e5780e02f2718c1f3de9a50cd03a")

	jsonBytes, err := json.Marshal(&pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonBytes))

	t.Log("size ", len(jsonBytes))
	pk2 := &VRFPublicKey{}
	err = json.Unmarshal(jsonBytes, pk2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(pk2.GetHexString())
	if pk2.GetHexString() != pk.GetHexString() {
		t.Errorf("unmarshal error")
	}
}

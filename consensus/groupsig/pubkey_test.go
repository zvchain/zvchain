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

func TestPubkey_MarshalJSON(t *testing.T) {
	var pk = Pubkey{}
	pk.SetHexString("0x08d8447b6403991f01b15a20fd444034374458ddd5dc49924135a6877bc94c7a15694df8f82f7917ad226bf14ea3e9eaa1ef7cfb37df198b39da463ed2e66965015da2ede2775f3f9b6339808c514c4e2cb935d5d88e63c430e9acd7f1a4b3022bbcad5c09f2c9e22c2da10c72391940a61fcac29e12a4e3fc3bcd63cdca9083")

	jsonBytes, err := json.Marshal(&pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonBytes))

}

func TestPubkey_UnmarshalJSON(t *testing.T) {
	var pk = Pubkey{}
	pk.SetHexString("0x08d8447b6403991f01b15a20fd444034374458ddd5dc49924135a6877bc94c7a15694df8f82f7917ad226bf14ea3e9eaa1ef7cfb37df198b39da463ed2e66965015da2ede2775f3f9b6339808c514c4e2cb935d5d88e63c430e9acd7f1a4b3022bbcad5c09f2c9e22c2da10c72391940a61fcac29e12a4e3fc3bcd63cdca9083")

	jsonBytes, err := json.Marshal(&pk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("size ", len(jsonBytes))
	pk2 := &Pubkey{}
	err = json.Unmarshal(jsonBytes, pk2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(pk2.GetHexString())
	if pk2.GetHexString() != pk.GetHexString() {
		t.Errorf("unmarshal error")
	}
}

func TestPubkey_String(t *testing.T) {
	var pk Pubkey
	pk.SetHexString("0x08d8447b6403991f01b15a20fd444034374458ddd5dc49924135a6877bc94c7a15694df8f82f7917ad226bf14ea3e9eaa1ef7cfb37df198b39da463ed2e66965015da2ede2775f3f9b6339808c514c4e2cb935d5d88e63c430e9acd7f1a4b3022bbcad5c09f2c9e22c2da10c72391940a61fcac29e12a4e3fc3bcd63cdca9083")

	t.Log(pk)
}

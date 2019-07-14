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

func TestAggregatePubkeysWithPk0(t *testing.T) {

	var pubs [7]Pubkey
	pubs[0].SetHexString("0x0edc142dfacaf5462e4ebbf05457be0389088e597c0b3066a048edb8c8fab1d2100ca458ec30c1e4c833102432a6e5305a0c3abfd2720785b1be4302db4d52982daf77384d5b95295f2a2f86b56c0a3fb44ff21e4aa4291c75a4eccbad8037e9178bdc7c8d556acc1094ffe5dd3d45fc0c7b1a30249516834a4bb817801bbb2a")
	pubs[1].SetHexString("0x01f8aa58d9c364d157e565c6d5eae4f58d68a466a687217527b939b2afb8cc800932e3b9bb8dca4d8fe6c8c2537aac7912b6f73e07af37ddb836a80c5d97ecf30c135873e9b0ee973974f788d7c38260d25fb1b948b29fbbf32e3cfa75bd89b72079470b67f796de2ec7f42e28538dfd6f0fee58a571221fc4b0e06b6e47925e")
	pubs[2].SetHexString("0x17bfef8f86e04bcee96317398fab57c9882fb7951df2892e43157c2bdfbd78fc2b336777e7d5879a04db9cb353fdf569fa60df6d075c2754d9d82ff9621b43462abfe08af814197c9119ed52e172378a8f73602758942913417607b3295f2efb300edf1e3e045604ba640d7c12e0fc8e1c23a40bd247548eb8cc542e9d21ae26")
	pubs[3].SetHexString("0x165886e366c30a46da8cd62ada2ed8925efe5f9e32d55a402e34604a0b912fbd28f86b1c74333b51d3b57f6993094a99bf36aab35a77647a9b53ac1d4d40b5f023cd1b8f0028b582ec4b783500255c3fb4d111fb1b683da3d19976058ce96cc627f4c9cb3ff0cdd1126987c1647d469ecb68c2c992a19008f0338f1f64d18f2e")
	pubs[4].SetHexString("0x2249a2dc645bb0f8918ca036258204e90d032c1fcd9a1dbdabad42cf409ab35e10009351c615644fc988dec9c45520e5e18ffcfadb6f06506bb74faa5d01544911fc0fd58bbe96bfa82ce46276691109ed967d5a8a34b8cbac94c2ffd99df042063b789f549dd1eaa617c8f5eaa339f4d843d5c47c380e0391ca435cf2524b91")
	pubs[5].SetHexString("0x14cf05371d1bdf4dda360c2ea233d96b7ba335431020f185eb552242a3e4f7d12628e462b75dd13a5e99e2a2817a792052f8aac13248f16bc68f5e3c6352b94f07c93f33dfad539a6c4bdcd9276ba8df9a312e524e56f7ec4dce6ba2d60298300b992765110942ded5a7befcb9d8e8667dc435917eeeae6ba4edfad24d46ef53")
	pubs[6].SetHexString("0x1dd33de441c82dfae6e79994a2b4fd3cd10dbd07ec7968e7f4b90b33f0c6c40b08afebcf1efcba73cb9a30b4a2c6b8fdc64f19314e44ce05b153360f786dc79e1402b7f9509d117f639dbfeb25dbde236794de606771eda89dc13ca8b3d62fde01626d9cb71d8d67f652d09dfddf13fa49a3011254670abac79faa9f3ec285ca")

	gpk := AggregatePubkeys(pubs[:])

	t.Log(gpk.GetHexString())

	if gpk.GetHexString() != "0x2625c7cceb8646420c6db0c5d6ed08bacaa0f8d3cc0f2b87c28fa6f069c8342a28a162d2ecc22d006d7b4d2b8f729150a0c0bd9392ac60e577ab6c8d29c22b9c1280ee9016bd76b1b6b7b9b64c0395d1c587b5fc23906b8cf61122caa5d628ee0070afa38983cb887bbd1728193ca6534f7f898a9a54fb270e304508dba505d0" {
		t.Errorf("aggregate error")
	}

}

func TestAggregatePubkeysWithMpk(t *testing.T) {

	var pubs [6]Pubkey
	pubs[0].SetHexString("0x0f7187ba150ac8518bc264dab56aa2310afd03e7c5375270e347a050b290a5c629056316de16bc46d3bcfbaf9fef19b497010ddec37a689d7ec066a11168f7f305894c457e25703bf0357b1d475c95d747110fc3b6c18d87bb60e830ac5b6acb26ef82ef3f6d68c2d5c223367d9688b460489ee403a9e0c901fbb7f5452b82ce")
	pubs[1].SetHexString("0x2dff85c1a839ab4a087feeee75af2376cf572f3033e10dc7235411881f58b4fb061dfc1c3b4576bfde93cdd6e27d5a8a8a87dd405c24724a006f8f82a07046be15bd68f79ceaf80a7c3291758446344b5bfe9f5186b6c758bdddfd12f8d4b15a1789459aa3f3da4d3e4685bd93de288fc172cc4d499576bb0ec5fb2f887f0593")
	pubs[2].SetHexString("0x25b7326a3f761404ea1af0668d04d93d1952c5f90ab781e82db1c8e3de355ca519571eb94be01bb6402f39f6c2344754ab49316f7756b5a6eb78ecd3f17ffa7505ae42b114fac3517cd583f3dc2f043f2bb01bacd243edef3902f38d0a81f53c132d3e4bc0fb6061a8420f86fc372191f29feadf8b0541aed976c0b0341c358f")
	pubs[3].SetHexString("0x0bcfbfa3ab0707f8509affe0241b25179ec7efe526bbb403e11432ca0171fe3129fccf9b2f088b1416be4b01813dea60cbcc98c922f63f6cd02813f70139651405a9d298bfe51127363ad04994b141282d435525a32d7433e8aa65a4ae0b4d1508ed4ce572e3414ab8250f2664e5976d7fb3b53e5e86a8e0e8765cbd24ee16cb")
	pubs[4].SetHexString("0x0c7887062ad2f503f0d041e0de2347e6e0a6380608d1f97be0b11993507033d62886524bd7d8c7f535d22f320b1cd2e75cd0818906d23a602d696184b535721b28f49c2027708738dbcd48426e76d30b7e09af2b672b1661216b0ea54d91bf53199e84632e0a4eca2fa45c158f31bf97941c3f675ac64315e15c5f9f878d349f")
	pubs[5].SetHexString("0x26727f91f184f4d6215abdf43a0dd467c2b39116301df16ddb6f0d9169ea4ba222897c76d99718e035e64672c2d335d11197f500c248aee976a9c9ed6f3c7ba41df378e262f32683fa0069b050547b31a8fdee595df504772e7a62b3a600b9e11789011e63080ce49e04ba207af7f3044f0f4e3cfd7f98071a84e1d8904f3f25")

	gpk := AggregatePubkeys(pubs[:])

	t.Log(gpk.GetHexString())

	if gpk.GetHexString() != "0x2625c7cceb8646420c6db0c5d6ed08bacaa0f8d3cc0f2b87c28fa6f069c8342a28a162d2ecc22d006d7b4d2b8f729150a0c0bd9392ac60e577ab6c8d29c22b9c1280ee9016bd76b1b6b7b9b64c0395d1c587b5fc23906b8cf61122caa5d628ee0070afa38983cb887bbd1728193ca6534f7f898a9a54fb270e304508dba505d0" {
		t.Errorf("aggregate error")
	}

}

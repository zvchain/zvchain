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

//
//func TestSignature_MarshalJSON(t *testing.T) {
//	sig := Signature{}
//	sig.SetAddrString("0x123")
//	bs := sig.GetHexString()
//	t.Log(string(bs))
//}

func TestVerifySig(t *testing.T) {
	var sign Signature
	sign.SetHexString("0x0acf827498032e4568bccc0de10dd19866e50dcb3feadb0a9ad1a5123895bfd501")
	var gpk Pubkey
	gpk.SetHexString("0x1519943f30e53e7627865ca7e14495d1619e80c7745ef9d8e70c6cbcc09a51aa13cad69a5f50b6a8a268e7949d91ddd85c75c3e3ebb40e7ef43f60f8d355189411fe95c32828a6230de8f7e242bdeff874109b3ac7a055634bc77087fc3605a2118c4c5f3b560426ea4659486c85313624479b1f2acba48e2796dadacb70491a")
	var hash = common.HexToHash("0x176f7a716d1ee8f2a11cf4dfadf0e49db9a535a1b376cc1d5bd5d722aba5c588")

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

func TestSignature_Serialize(t *testing.T) {
	sign := "0x25eec26c19d282ad0b20ea1870ae5a6a0bbf1b8ae6997a722c9b473197b234a601"

	var sig Signature
	sig.SetHexString(sign)

	t.Log(sig.Serialize())
}

func TestBatchVerifyGroupSig(t *testing.T) {
	signs := make([]Signature, 6)
	signs[0].SetHexString("0x27318035b79b84dd3bdc2a3e5fd6ceadebd3c3dd0c28a16342790faf3257705601")
	signs[1].SetHexString("0x23010a57f862d1759f28971adc742f8a9c46f2bc8b8808466234893d0971b9db00")
	signs[2].SetHexString("0x10d5e48c76fc642d6ec79d9a4847feeb037fdad84c59b19ea6323a8f763bacb400")
	signs[3].SetHexString("0x1600fc41368e4f7624ac7fbb5f3127d5ca8f87a710d41b78b9a455e43fb4d97200")
	signs[4].SetHexString("0x137c03ffeea45a3533e2848ce1a28fb94457d5983bb2e0a5f4e8edc922e444b501")
	signs[5].SetHexString("0x2467ac8a06db45fd89695c52e9e307667fd23d804ba1b534c5807c9905cefb8b01")

	pks := make([]Pubkey, 6)
	pks[0].SetHexString("0x0f7187ba150ac8518bc264dab56aa2310afd03e7c5375270e347a050b290a5c629056316de16bc46d3bcfbaf9fef19b497010ddec37a689d7ec066a11168f7f305894c457e25703bf0357b1d475c95d747110fc3b6c18d87bb60e830ac5b6acb26ef82ef3f6d68c2d5c223367d9688b460489ee403a9e0c901fbb7f5452b82ce")
	pks[1].SetHexString("0x2dff85c1a839ab4a087feeee75af2376cf572f3033e10dc7235411881f58b4fb061dfc1c3b4576bfde93cdd6e27d5a8a8a87dd405c24724a006f8f82a07046be15bd68f79ceaf80a7c3291758446344b5bfe9f5186b6c758bdddfd12f8d4b15a1789459aa3f3da4d3e4685bd93de288fc172cc4d499576bb0ec5fb2f887f0593")
	pks[2].SetHexString("0x25b7326a3f761404ea1af0668d04d93d1952c5f90ab781e82db1c8e3de355ca519571eb94be01bb6402f39f6c2344754ab49316f7756b5a6eb78ecd3f17ffa7505ae42b114fac3517cd583f3dc2f043f2bb01bacd243edef3902f38d0a81f53c132d3e4bc0fb6061a8420f86fc372191f29feadf8b0541aed976c0b0341c358f")
	pks[3].SetHexString("0x0bcfbfa3ab0707f8509affe0241b25179ec7efe526bbb403e11432ca0171fe3129fccf9b2f088b1416be4b01813dea60cbcc98c922f63f6cd02813f70139651405a9d298bfe51127363ad04994b141282d435525a32d7433e8aa65a4ae0b4d1508ed4ce572e3414ab8250f2664e5976d7fb3b53e5e86a8e0e8765cbd24ee16cb")
	pks[4].SetHexString("0x0c7887062ad2f503f0d041e0de2347e6e0a6380608d1f97be0b11993507033d62886524bd7d8c7f535d22f320b1cd2e75cd0818906d23a602d696184b535721b28f49c2027708738dbcd48426e76d30b7e09af2b672b1661216b0ea54d91bf53199e84632e0a4eca2fa45c158f31bf97941c3f675ac64315e15c5f9f878d349f")
	pks[5].SetHexString("0x26727f91f184f4d6215abdf43a0dd467c2b39116301df16ddb6f0d9169ea4ba222897c76d99718e035e64672c2d335d11197f500c248aee976a9c9ed6f3c7ba41df378e262f32683fa0069b050547b31a8fdee595df504772e7a62b3a600b9e11789011e63080ce49e04ba207af7f3044f0f4e3cfd7f98071a84e1d8904f3f25")

	data := common.HexToHash("0xef210f2867870450f28e59bc35e29a573c727c4ab6d53e3ff49c016bd92610d4")

	for i, sig := range signs {
		if !VerifySig(pks[i], data.Bytes(), sig) {
			t.Errorf("verify fail at %v", i)
		}
	}

	var gpk Pubkey
	gpk.SetHexString("0x25a8f08dca5e797f85a56182c096967ec18b1ada3d393b286b2096e42f4784a21ba598334e17694fa30b0928a3837ccbf0605d508f4fea29b5023ac0d86da57d02ab64ce646169c4008cedf141a7e844164a226576ba06b2ca3c2743ddb7737c2ddc52bcfb6fed75fab69517b91109de096c2ca2352f713e2de9e515c7c4b69a")
	var gSig Signature
	gSig.SetHexString("0x25eec26c19d282ad0b20ea1870ae5a6a0bbf1b8ae6997a722c9b473197b234a601")

	if !VerifySig(gpk, data.Bytes(), gSig) {
		t.Errorf("gsign verify fail")
	}

}

func TestRecoverSignatureSigs(t *testing.T) {
	signs := make([]Signature, 6)
	signs[0].SetHexString("0x27318035b79b84dd3bdc2a3e5fd6ceadebd3c3dd0c28a16342790faf3257705601")
	signs[1].SetHexString("0x23010a57f862d1759f28971adc742f8a9c46f2bc8b8808466234893d0971b9db00")
	signs[2].SetHexString("0x10d5e48c76fc642d6ec79d9a4847feeb037fdad84c59b19ea6323a8f763bacb400")
	signs[3].SetHexString("0x1600fc41368e4f7624ac7fbb5f3127d5ca8f87a710d41b78b9a455e43fb4d97200")
	signs[4].SetHexString("0x137c03ffeea45a3533e2848ce1a28fb94457d5983bb2e0a5f4e8edc922e444b501")
	signs[5].SetHexString("0x2467ac8a06db45fd89695c52e9e307667fd23d804ba1b534c5807c9905cefb8b01")

	ids := make([]ID, 6)
	ids[0].SetAddrString("0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4")
	ids[1].SetAddrString("0x586e580e7d3352d617f35189ed4995679729a1a8b53ed8d91e46d7f8970d4737")
	ids[2].SetAddrString("0x69d959d090df8c77adc85b5294871a904b0294eb85fb8251ba903710805d64c2")
	ids[3].SetAddrString("0x806ec4eb2d7a2ba0ebee40e2a39e3e8b1f3a09d91ae78bd7fdf30c77e543f545")
	ids[4].SetAddrString("0xb2e882c6d59b37636d65cd6e023d4f2bd49f25947c37221ac52b3c9b60278813")
	ids[5].SetAddrString("0xc7d83d1e57ac5e2df25df8c569e87f62fc1173039faf3cb30f65d0efab9ecc50")

	gSign := RecoverSignature(signs, ids)

	t.Log(gSign.GetHexString())

	if gSign.GetHexString() != "0x25eec26c19d282ad0b20ea1870ae5a6a0bbf1b8ae6997a722c9b473197b234a601" {
		t.Errorf("recover sign error")
	}

}

func TestSignAndVerify(t *testing.T) {
	var msk Seckey
	var mpk Pubkey
	msk.SetHexString("0x2932e91eb13ef59f0cec914169da092bf5628ba0c3a0e3e1c039dd1951871f1d")
	mpk.SetHexString("0x169dcf2f252322b117c500db3eb8f86c4266aae226b5550838a3f3c5496dfe4b21a082666c202d505acd0a892f7e49d112a45344a5476e0fc85cf8600238b0351d86b3b3a2ed06d8a2a4f7c5d3f772d9a77a54d7d60ec06e92bf717be4f1ee9e0204bcfba515b65c207cdbde5d15d371d8378ef64ef04c1f78e08eceb8b8068a")
	// Generate encrypted share piece

	sig := Sign(msk, []byte("123"))
	if !VerifySig(mpk, []byte("123"), sig) {
		t.Fatal("verify fail")
	}
}

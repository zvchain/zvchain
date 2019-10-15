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
	"fmt"
	"math/big"

	//"math/rand"
	"bytes"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
)

func TestPubkey(t *testing.T) {
	fmt.Printf("\nbegin test pub key...\n")
	t.Log("TestPubkey")
	r := base.NewRand()

	fmt.Printf("size of rand = %v\n.", len(r))
	sec := NewSeckeyFromRand(r.Deri(1))
	if sec == nil {
		t.Fatal("NewSeckeyFromRand")
	}
	pub := NewPubkeyFromSeckey(*sec)
	if pub == nil {
		t.Log("NewPubkeyFromSeckey")
	}
	{
		var pub2 Pubkey
		err := pub2.SetHexString(pub.GetHexString())
		if err != nil || !pub.IsEqual(pub2) {
			t.Log("pub != pub2")
		}
	}
	{
		var pub2 Pubkey
		err := pub2.Deserialize(pub.Serialize())
		if err != nil || !pub.IsEqual(pub2) {
			t.Log("pub != pub2")
		}
	}
	t.Log("ss", len(pub.Serialize()))
	fmt.Printf("\nend test pub key.\n")
}

func testComparison(t *testing.T) {
	fmt.Printf("\nbegin test Comparison...\n")
	t.Log("begin testComparison")
	var b = new(big.Int)
	b.SetString("16798108731015832284940804142231733909759579603404752749028378864165570215948", 10)
	sec := NewSeckeyFromBigInt(b)
	t.Log("sec.Hex: ", sec.GetHexString())

	// Add Seckeys
	sum := AggregateSeckeys([]Seckey{*sec, *sec})
	if sum == nil {
		t.Error("AggregateSeckeys failed.")
	}

	// Pubkey
	pub := NewPubkeyFromSeckey(*sec)
	if pub == nil {
		t.Error("NewPubkeyFromSeckey failed.")
	} else {
		fmt.Printf("size of pub key = %v.\n", len(pub.Serialize()))
	}

	// Sig
	sig := Sign(*sec, []byte("hi"))
	fmt.Printf("size of sign = %v\n.", len(sig.Serialize()))
	asig := AggregateSigs([]Signature{sig, sig})
	if !VerifyAggregateSig([]Pubkey{*pub, *pub}, []byte("hi"), asig) {
		t.Error("Aggregated signature does not verify")
	}
	{
		var sig2 Signature
		err := sig2.SetHexString(sig.GetHexString())
		if err != nil || !sig.IsEqual(sig2) {
			t.Error("sig2.SetHexString")
		}
	}
	{
		var sig2 Signature
		err := sig2.Deserialize(sig.Serialize())
		if err != nil || !sig.IsEqual(sig2) {
			t.Error("sig2.Deserialize")
		}
	}
	t.Log("end testComparison")
	fmt.Printf("\nend test Comparison.\n")
}

func testSeckey(t *testing.T) {
	fmt.Printf("\nbegin test sec key...\n")
	t.Log("testSeckey")
	s := "401035055535747319451436327113007154621327258807739504261475863403006987855"
	var b = new(big.Int)
	b.SetString(s, 10)
	sec := NewSeckeyFromBigInt(b)
	str := sec.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)
	{
		var sec2 Seckey
		err := sec2.SetHexString(str)
		if err != nil || !sec.IsEqual(sec2) {
			t.Error("bad SetHexString")
		}
		str = sec2.GetHexString()
		fmt.Printf("sec key import and export again, len=%v, data=%v.\n", len(str), str)
	}
	{
		var sec2 Seckey
		err := sec2.Deserialize(sec.Serialize())
		if err != nil || !sec.IsEqual(sec2) {
			t.Error("bad Serialize")
		}
	}
	fmt.Printf("end test sec key.\n")
}

func testAggregation(t *testing.T) {
	fmt.Printf("\nbegin test Aggregation...\n")
	t.Log("testAggregation")
	//    m := 5
	n := 3
	//    groupPubkeys := make([]Pubkey, m)
	r := base.NewRand()
	seckeyContributions := make([]Seckey, n)
	for i := 0; i < n; i++ {
		seckeyContributions[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	groupSeckey := AggregateSeckeys(seckeyContributions)
	groupPubkey := NewPubkeyFromSeckey(*groupSeckey)
	t.Log("Group pubkey:", groupPubkey.GetHexString())
	fmt.Printf("end test Aggregation.\n")
}

func AggregateSeckeysByBigInt(secs []Seckey) *Seckey {
	secret := big.NewInt(0)
	for _, s := range secs {
		secret.Add(secret, s.GetBigInt())
	}
	secret.Mod(secret, curveOrder)
	return NewSeckeyFromBigInt(secret)
}

func testAggregateSeckeys(t *testing.T) {
	fmt.Printf("\nbegin testAggregateSeckeys...\n")
	t.Log("begin testAggregateSeckeys")
	n := 100
	r := base.NewRand()
	secs := make([]Seckey, n)
	fmt.Printf("begin init 100 sec key...\n")
	for i := 0; i < n; i++ {
		secs[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	fmt.Printf("begin aggr sec key with bigint...\n")
	s1 := AggregateSeckeysByBigInt(secs)
	fmt.Printf("begin aggr sec key...\n")
	s2 := AggregateSeckeys(secs)
	fmt.Printf("sec aggred with int, data=%v.\n", s1.GetHexString())
	fmt.Printf("sec aggred , data=%v.\n", s2.GetHexString())
	if !s1.value.IsEqual(&s2.value) {
		t.Errorf("not same int(%v) VS (%v).\n", s1.GetHexString(), s2.GetHexString())
	}
	t.Log("end testAggregateSeckeys")
	fmt.Printf("end testAggregateSeckeys.\n")
}

func RecoverSeckeyByBigInt(secs []Seckey, ids []ID) *Seckey {
	secret := big.NewInt(0)
	k := len(secs)
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		xs[i] = ids[i].GetBigInt()
	}
	// need len(ids) = k > 0
	for i := 0; i < k; i++ {
		// compute delta_i depending on ids only
		var delta, num, den, diff = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		for j := 0; j < k; j++ {
			if j != i {
				num.Mul(num, xs[j])
				num.Mod(num, curveOrder)
				diff.Sub(xs[j], xs[i])
				den.Mul(den, diff)
				den.Mod(den, curveOrder)
			}
		}
		// delta = num / den
		den.ModInverse(den, curveOrder) //模逆
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)
		// apply delta to secs[i]
		delta.Mul(delta, secs[i].GetBigInt())
		// skip reducing delta modulo curveOrder here
		secret.Add(secret, delta)
		secret.Mod(secret, curveOrder)
	}
	return NewSeckeyFromBigInt(secret)
}

func testRecoverSeckey(t *testing.T) {
	fmt.Printf("\nbegin testRecoverSeckey...\n")
	t.Log("testRecoverSeckey")
	n := 50
	r := base.NewRand()

	secs := make([]Seckey, n)
	ids := make([]ID, n)
	for i := 0; i < n; i++ {
		ids[i] = *NewIDFromInt64(int64(i + 3))
		secs[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	s1 := RecoverSeckey(secs, ids)
	s2 := RecoverSeckeyByBigInt(secs, ids)
	if !s1.value.IsEqual(&s2.value) {
		t.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", s1.GetHexString(), s2.GetHexString())
	}
	fmt.Printf("end testRecoverSeckey.\n")
}

func ShareSeckeyByBigInt(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	// degree of polynomial, need k >= 1, i.e. len(msec) >= 2
	k := len(msec) - 1
	// msec = c_0, c_1, ..., c_k
	// evaluate polynomial f(x) with coefficients c0, ..., ck
	secret.Set(msec[k].GetBigInt())
	x := id.GetBigInt()
	for j := k - 1; j >= 0; j-- {
		secret.Mul(secret, x)
		//sec.secret.Mod(&sec.secret, curveOrder)
		secret.Add(secret, msec[j].GetBigInt())
		secret.Mod(secret, curveOrder)
	}
	return NewSeckeyFromBigInt(secret)
}

func testShareSeckey(t *testing.T) {
	fmt.Printf("\nbegin testShareSeckey...\n")
	t.Log("testShareSeckey")
	n := 100
	msec := make([]Seckey, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		msec[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	id := *NewIDFromInt64(123)
	s1 := ShareSeckeyByBigInt(msec, id)
	s2 := ShareSeckey(msec, id)
	if !s1.value.IsEqual(&s2.value) {
		t.Errorf("bad sec\n%s\n%s", s1.GetHexString(), s2.GetHexString())
	} else {
		buf := s2.Serialize()
		fmt.Printf("size of seckey = %v.\n", len(buf))
	}
	fmt.Printf("end testShareSeckey.\n")
}

func testID(t *testing.T) {
	t.Log("testString")
	fmt.Printf("\nbegin test ID...\n")
	b := new(big.Int)
	b.SetString("001234567890abcdef", 16)
	c := new(big.Int)
	c.SetString("1234567890abcdef", 16)
	idc := NewIDFromBigInt(c)
	id1 := NewIDFromBigInt(b)
	if id1.IsEqual(*idc) {
		fmt.Println("id1 is equal to idc")
	}
	if id1 == nil {
		t.Error("NewIDFromBigInt")
	} else {
		buf := id1.Serialize()
		fmt.Printf("id Serialize, len=%v, data=%v.\n", len(buf), buf)
	}

	str := id1.GetAddrString()
	fmt.Printf("ID export, len=%v, data=%v.\n", len(str), str)
	//test
	str0 := id1.value.GetHexString()
	fmt.Printf("str0 =%v\n", str0)

	///test
	{
		var id2 ID
		err := id2.SetAddrString(id1.GetAddrString())
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetAddrString(), id2.GetAddrString())
		}
	}
	{
		var id2 ID
		err := id2.Deserialize(id1.Serialize())
		fmt.Printf("id2:%v", id2.GetAddrString())
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetAddrString(), id2.GetAddrString())
		}
	}
	fmt.Printf("end test ID.\n")
}

func test(t *testing.T) {
	var tmp []byte
	fmt.Printf("len of empty []byte=%v.\n", len(tmp))
	var ti time.Time
	fmt.Printf("time zero=%v.\n", ti.IsZero())
	var tmp_i = 456
	fmt.Printf("sizeof(int) =%v.\n", unsafe.Sizeof(tmp_i))
	testID(t)
	testSeckey(t)
	TestPubkey(t)
	testAggregation(t)

	testComparison(t)
	testAggregateSeckeys(t)
	testRecoverSeckey(t)
	testShareSeckey(t)
}

func Test_GroupsigIDStringConvert(t *testing.T) {
	str := "0xedb67046af822fd6a778f3a1ec01ad2253e5921d3c1014db958a952fdc1b98e2"
	id := NewIDFromString(str)
	s := id.GetAddrString()
	fmt.Printf("id str:%s\n", s)
	fmt.Printf("id str compare result:%t\n", str == s)
}

func Test_Groupsig_Main1(t *testing.T) {
	fmt.Printf("begin TestMain...\n")
	//fmt.Printf("\ncall test with curve(CurveFp254BNb)=%v...\n", bn_curve.CurveFp254BNb)
	test(t)
	//if bn_curve.GetMaxOpUnitSize() == 6 {
	//	t.Log("CurveFp382_1")
	//	fmt.Printf("\ncall test with curve(CurveFp382_1)=%v...\n", bn_curve.CurveFp382_1)
	//	test(t, bn_curve.CurveFp382_1)
	//	t.Log("CurveFp382_2")
	//	fmt.Printf("\ncall test with curve(CurveFp382_2)=%v...\n", bn_curve.CurveFp382_2)
	//	test(t, bn_curve.CurveFp382_2)
	//}
	return
}

func Test_Groupsig_ID_Deserialize(t *testing.T) {
	//Init(1)
	s := "abc"
	id1 := DeserializeID([]byte(s))
	id2 := NewIDFromString(s)
	t.Log(id1.GetAddrString(), id2.GetAddrString(), id1.IsEqual(*id2))

	t.Log([]byte(s))
	t.Log(id1.Serialize(), id2.Serialize())
	t.Log(id1.GetAddrString(), id2.GetAddrString())

	b := id2.Serialize()
	id3 := DeserializeID(b)
	t.Log(id3.GetAddrString())
}

//Added by FlyingSquirrel-Xu. 2018-08-24.
func testRecover(n int, k int, b *testing.B) {
	//n := 50
	//k := 10

	//Define F(x): <a[0], a[1], ..., a[k-1]>. F(0)=a[0].
	a := make([]Seckey, k)
	r := base.NewRand()
	for i := 0; i < k; i++ {
		a[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	//fmt.Println("a[0]:", a[0].Serialize())

	//Generate n IDs: {IDi}, i=1,..,n.
	ids := make([]ID, n)
	for i := 0; i < n; i++ {
		ids[i] = *NewIDFromInt64(int64(i + 3))
	}

	secs := make([]Seckey, n)
	for j := 0; j < n; j++ {
		bs := ShareSeckey(a, ids[j])
		secs[j].value.SetBigInt(bs.value.GetBigInt())
	}

	// By Lagrange insertion, calculating {<IDi, Si>|i=1..n} s=F(0).
	new_secs := secs[:k]
	s := RecoverSeckey(new_secs, ids)
	//fmt.Println("s:", s.Serialize())

	//检查 F(0) = a[0]?
	if !bytes.Equal(a[0].Serialize(), s.Serialize()) {
		fmt.Errorf("secreky Recover failed.")
	}

	pub := NewPubkeyFromSeckey(*s)

	sig := make([]Signature, n)
	for i := 0; i < n; i++ {
		sig[i] = Sign(secs[i], []byte("hi"))
	}

	//Recover group sig: H = ∑ ∆i(0)·Hi.
	new_sig := sig[:k]
	H := RecoverSignature(new_sig, ids)

	//Verify group sig：Pair(H,Q)==Pair(Hm,Pub)?
	result := VerifySig(*pub, []byte("hi"), *H)
	if result != true {
		fmt.Errorf("VerifySig failed.")
	}
}

func benchmark_GroupsigRecover(n int, k int, b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testRecover(n, k, b)
	}
}

func Benchmark_GroupsigRecover100(b *testing.B)  { benchmark_GroupsigRecover(100, 100, b) }
func Benchmark_GroupsigRecover200(b *testing.B)  { benchmark_GroupsigRecover(200, 200, b) }
func Benchmark_GroupsigRecover500(b *testing.B)  { benchmark_GroupsigRecover(500, 500, b) }
func Benchmark_GroupsigRecover1000(b *testing.B) { benchmark_GroupsigRecover(1000, 1000, b) }

func BenchmarkPubkeyFromSeckey(b *testing.B) {
	b.StopTimer()

	r := base.NewRand()

	//var sec Seckey
	for n := 0; n < b.N; n++ {
		//sec.SetByCSPRNG()
		sec := NewSeckeyFromRand(r.Deri(1))
		b.StartTimer()
		NewPubkeyFromSeckey(*sec)
		b.StopTimer()
	}
}

func BenchmarkSigning(b *testing.B) {
	b.StopTimer()

	r := base.NewRand()

	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		b.StartTimer()
		Sign(*sec, []byte(strconv.Itoa(n)))
		b.StopTimer()
	}
}

func BenchmarkValidation(b *testing.B) {
	b.StopTimer()

	r := base.NewRand()

	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		pub := NewPubkeyFromSeckey(*sec)
		m := strconv.Itoa(n)
		sig := Sign(*sec, []byte(m))
		b.StartTimer()
		result := VerifySig(*pub, []byte(m), sig)
		if result != true {
			fmt.Println("VerifySig failed.")
		}
		b.StopTimer()
	}
}

func BenchmarkValidation2(b *testing.B) {
	b.StopTimer()
	r := base.NewRand()
	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		pub := NewPubkeyFromSeckey(*sec)
		m := strconv.Itoa(n) + "abc"

		sig := Sign(*sec, []byte(m))
		buf := sig.Serialize()
		sig2 := DeserializeSign(buf)

		b.StartTimer()
		result := VerifySig(*pub, []byte(m), *sig2)
		if result != true {
			fmt.Println("VerifySig failed.")
		}
		b.StopTimer()
	}
}

func benchmarkDeriveSeckeyShare(k int, b *testing.B) {
	b.StopTimer()

	r := base.NewRand()
	sec := NewSeckeyFromRand(r.Deri(1))

	msk := sec.GetMasterSecretKey(k)
	var id ID
	for n := 0; n < b.N; n++ {
		err := id.SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(n)})
		if err != nil {
			b.Error(err)
		}
		b.StartTimer()
		err = sec.Set(msk, &id)
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDeriveSeckeyShare500(b *testing.B) { benchmarkDeriveSeckeyShare(500, b) }

func benchmarkRecoverSeckey(k int, b *testing.B) {
	b.StopTimer()

	r := base.NewRand()
	sec := NewSeckeyFromRand(r.Deri(1))

	msk := sec.GetMasterSecretKey(k)

	// derive n shares
	n := k
	secVec := make([]Seckey, n)
	idVec := make([]ID, n)
	for i := 0; i < n; i++ {
		err := idVec[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			b.Error(err)
		}
		err = secVec[i].Set(msk, &idVec[i])
		if err != nil {
			b.Error(err)
		}
	}

	// recover from secVec and idVec
	var sec2 Seckey
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		err := sec2.Recover(secVec, idVec)
		if err != nil {
			b.Errorf("%s\n", err)
		}
	}
}

func BenchmarkRecoverSeckey100(b *testing.B)  { benchmarkRecoverSeckey(100, b) }
func BenchmarkRecoverSeckey200(b *testing.B)  { benchmarkRecoverSeckey(200, b) }
func BenchmarkRecoverSeckey500(b *testing.B)  { benchmarkRecoverSeckey(500, b) }
func BenchmarkRecoverSeckey1000(b *testing.B) { benchmarkRecoverSeckey(1000, b) }

func benchmarkRecoverSignature(k int, b *testing.B) {
	b.StopTimer()

	r := base.NewRand()
	sec := NewSeckeyFromRand(r.Deri(1))

	msk := sec.GetMasterSecretKey(k)

	// derive n shares
	n := k
	idVec := make([]ID, n)
	secVec := make([]Seckey, n)
	signVec := make([]Signature, n)
	for i := 0; i < n; i++ {
		err := idVec[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			b.Error(err)
		}
		err = secVec[i].Set(msk, &idVec[i])
		if err != nil {
			b.Error(err)
		}
		signVec[i] = Sign(secVec[i], []byte("test message"))
	}

	// recover signature
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		RecoverSignature(signVec, idVec)
	}
}

func BenchmarkRecoverSignature100(b *testing.B)  { benchmarkRecoverSignature(100, b) }
func BenchmarkRecoverSignature200(b *testing.B)  { benchmarkRecoverSignature(200, b) }
func BenchmarkRecoverSignature500(b *testing.B)  { benchmarkRecoverSignature(500, b) }
func BenchmarkRecoverSignature1000(b *testing.B) { benchmarkRecoverSignature(1000, b) }

func TestSignature_MarshalJSON(t *testing.T) {
	var sig Signature
	sig.SetHexString("0x0724b751e096becd93127a5be441989a9fd8fe328828f6ce5e1817c70bf10f2f00")
	bs := sig.GetHexString()

	t.Log("len of bs:", len(bs))
	t.Log(string(bs))
}

func TestAddress(t *testing.T) {
	addr := common.StringToAddress("zv0bf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7")
	id := DeserializeID(addr.Bytes())
	t.Log(id.GetAddrString(), len(id.GetAddrString()), addr.AddrPrefixString() == id.GetAddrString())
	t.Log(id.Serialize(), addr.Bytes(), bytes.Equal(id.Serialize(), addr.Bytes()))

	id2 := ID{}
	id2.SetAddrString("0x0bf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7")
	t.Log(id2.GetAddrString())

	json, _ := id2.MarshalJSON()
	t.Log(string(json))

	id3 := &ID{}
	id3.UnmarshalJSON(json)
	t.Log(id3.GetAddrString())
}

func TestDoubleAggregate(t *testing.T) {
	fmt.Printf(" TestDoubleAggregate begin \n")
	msg := []byte("this is a test")
	sks := make([]Seckey, 2)
	pks := make([]Pubkey, 2)
	sigs := make([]Signature, 2)

	sks[0] = *NewSeckeyFromRand(base.NewRand())
	pks[0] = *NewPubkeyFromSeckey(sks[0])
	sigs[0] = Sign(sks[0], msg)
	if !VerifySig(pks[0], msg, sigs[0]) {
		fmt.Printf(" verify failure for 0-th user \n")
	}
	sks[1] = *NewSeckeyFromRand(base.NewRand())
	pks[1] = *NewPubkeyFromSeckey(sks[1])
	sigs[1] = Sign(sks[1], msg)
	if !VerifySig(pks[1], msg, sigs[1]) {
		fmt.Printf(" verify failure for 1-th user \n")
	}
	apk := *AggregatePubkeys(pks)
	aSig := AggregateSigs(sigs)
	if !VerifySig(apk, msg, aSig) {
		fmt.Printf(" verify failure for aggregation \n")
	}
	if !VerifySig(pks[0], msg, aSig) {
		fmt.Printf(" verify failure for badcase 0 \n")
	}
	if !VerifySig(pks[1], msg, aSig) {
		fmt.Printf(" verify failure for badcase 1 \n")
	}
	fmt.Printf(" TestDoubleAggregate end \n")
}

func TestDH(t *testing.T) {
	fmt.Printf(" TestDH begin \n")
	r := base.NewRand()
	sec1 := *NewSeckeyFromRand(r.Deri(1))
	pub1 := NewPubkeyFromSeckey(sec1)

	sec2 := *NewSeckeyFromRand(r.Deri(2))
	pub2 := NewPubkeyFromSeckey(sec2)

	result1 := new(Pubkey)
	result2 := new(Pubkey)

	result1.value.ScalarMult(&pub2.value, &sec1.value.v)
	result2.value.ScalarMult(&pub1.value, &sec2.value.v)

	if !result1.IsEqual(*result2) {
		fmt.Printf(" DH doesn't match \n")
	}
	fmt.Printf(" TestDH end \n")
}

func BenchmarkDH(b *testing.B) {
	r := base.NewRand()
	sec1 := *NewSeckeyFromRand(r.Deri(1))
	//	pub1 := NewPubkeyFromSeckey(sec1)

	sec2 := *NewSeckeyFromRand(r.Deri(2))
	pub2 := NewPubkeyFromSeckey(sec2)

	result := new(Pubkey)

	for n := 0; n < b.N; n++ {
		result.value.ScalarMult(&pub2.value, &sec1.value.v)
	}
}

func TestGroupSignature(t *testing.T) {
	fmt.Printf("TestGroupSignature begin \n")
	msg := []byte("this is test message")
	n := 9
	k := 5

	r := base.NewRand()
	sk := make([]Seckey, n)
	pk := make([]Pubkey, n)
	ids := make([]ID, n)

	for i := 0; i < n; i++ {
		sk[i] = *NewSeckeyFromRand(r.Deri(i))
		pk[i] = *NewPubkeyFromSeckey(sk[i])
		err := ids[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			t.Error(err)
		}
	}

	shares := make([][]Seckey, n)
	for i := 0; i < n; i++ {
		shares[i] = make([]Seckey, n)
		msec := sk[i].GetMasterSecretKey(k)

		for j := 0; j < n; j++ {
			err := shares[i][j].Set(msec, &ids[j])
			if err != nil {
				t.Error(err)
			}
		}
	}

	msk := make([]Seckey, n)
	shareVec := make([]Seckey, n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			shareVec[i] = shares[i][j]
		}
		msk[j] = *AggregateSeckeys(shareVec)
	}

	sigs := make([]Signature, n)
	for i := 0; i < n; i++ {
		sigs[i] = Sign(msk[i], msg)
	}

	gpk := AggregatePubkeys(pk)
	for m := k; m <= n; m++ {
		sigVec := make([]Signature, m)
		idVec := make([]ID, m)

		for i := 0; i < m; i++ {
			sigVec[i] = sigs[i]
			idVec[i] = ids[i]
		}
		gsig := RecoverSignature(sigVec, idVec)

		fmt.Printf("m = %v, sig = %v\n", m, gsig.Serialize())

		if !VerifySig(*gpk, msg, *gsig) {
			fmt.Printf("fail to VerifySig when m= %v \n", m)
		}
	}
	fmt.Printf("TestGroupSignature end \n")
}

func BenchmarkBatchVerify(b *testing.B) {
	b.ResetTimer()
	msg := []byte("this is test message")
	n := 100
	k := 51

	r := base.NewRand()
	sk := make([]Seckey, n)
	pk := make([]Pubkey, n)
	ids := make([]ID, n)

	for i := 0; i < n; i++ {
		sk[i] = *NewSeckeyFromRand(r.Deri(i))
		pk[i] = *NewPubkeyFromSeckey(sk[i])
		err := ids[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			b.Error(err)
		}
	}

	shares := make([][]Seckey, n)
	for i := 0; i < n; i++ {
		shares[i] = make([]Seckey, n)
		msec := sk[i].GetMasterSecretKey(k)

		for j := 0; j < n; j++ {
			err := shares[i][j].Set(msec, &ids[j])
			if err != nil {
				b.Error(err)
			}
		}
	}

	msk := make([]Seckey, n)
	mpk := make([]Pubkey, n)
	shareVec := make([]Seckey, n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			shareVec[i] = shares[i][j]
		}
		msk[j] = *AggregateSeckeys(shareVec)
		mpk[j] = *NewPubkeyFromSeckey(msk[j])
	}

	sigs := make([]Signature, n)
	for i := 0; i < n; i++ {
		sigs[i] = Sign(msk[i], msg)
	}

	gpk := AggregatePubkeys(pk)

	sigVec := make([]Signature, k)
	mpkVec := make([]Pubkey, k)
	idVec := make([]ID, k)

	for i := 0; i < k; i++ {
		sigVec[i] = sigs[i]
		idVec[i] = ids[i]
		mpkVec[i] = mpk[i]
	}
	var gsig *Signature
	gsig = RecoverSignature(sigVec, idVec)
	for n := 0; n < b.N; n++ {
		b.StartTimer()
		ok := BatchVerify(mpkVec, msg, sigVec)
		b.StopTimer()
		if !ok {
			fmt.Printf("batchVerify fail \n")
		}
	}

	if !VerifySig(*gpk, msg, *gsig) {
		fmt.Printf("fail to VerifySig \n")
	}
}
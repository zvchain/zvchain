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

type Expect struct {
	bitLen int
	ok     []byte
}

func testIDconvert(t *testing.T) {
	id := ID{}
	id.SetHexString("0x0000123abcdef")
	fmt.Printf("id result =%v", id.GetHexString())
}

//测试用衍生随机数生成私钥，从私钥萃取公钥，以及公钥的序列化
func testPubkey(t *testing.T) {
	fmt.Printf("\nbegin test pub key...\n")
	t.Log("testPubkey")
	r := base.NewRand() //生成随机数

	fmt.Printf("size of rand = %v\n.", len(r))
	sec := NewSeckeyFromRand(r.Deri(1)) //以r的衍生随机数生成私钥
	if sec == nil {
		t.Fatal("NewSeckeyFromRand")
	}
	pub := NewPubkeyFromSeckey(*sec) //从私钥萃取出公钥
	if pub == nil {
		t.Log("NewPubkeyFromSeckey")
	}
	{
		var pub2 Pubkey
		err := pub2.SetHexString(pub.GetHexString()) //测试公钥的字符串导出
		if err != nil || !pub.IsEqual(pub2) {        //检查字符串导入生成的公钥是否跟之前的公钥相同
			t.Log("pub != pub2")
		}
	}
	{
		var pub2 Pubkey
		err := pub2.Deserialize(pub.Serialize()) //测试公钥的序列化
		if err != nil || !pub.IsEqual(pub2) {    //检查反序列化生成的公钥是否跟之前的公钥相同
			t.Log("pub != pub2")
		}
	}
	fmt.Printf("\nend test pub key.\n")
}

//用big.Int生成私钥，取得公钥和签名。然后对私钥、公钥和签名各复制一份后测试加法后的验证是否正确。
//同时测试签名的序列化。
func testComparison(t *testing.T) {
	fmt.Printf("\nbegin test Comparison...\n")
	t.Log("begin testComparison")
	var b = new(big.Int)
	b.SetString("16798108731015832284940804142231733909759579603404752749028378864165570215948", 10)
	sec := NewSeckeyFromBigInt(b) //从big.Int（固定的常量）生成原始私钥
	t.Log("sec.Hex: ", sec.GetHexString())

	// Add Seckeys
	sum := AggregateSeckeys([]Seckey{*sec, *sec}) //同一个原始私钥相加，生成聚合私钥
	if sum == nil {
		t.Error("AggregateSeckeys failed.")
	}

	// Pubkey
	pub := NewPubkeyFromSeckey(*sec) //从原始私钥萃取出公钥
	if pub == nil {
		t.Error("NewPubkeyFromSeckey failed.")
	} else {
		fmt.Printf("size of pub key = %v.\n", len(pub.Serialize()))
	}

	// Sig
	sig := Sign(*sec, []byte("hi")) //以原始私钥对明文签名，生成原始签名
	fmt.Printf("size of sign = %v\n.", len(sig.Serialize()))
	asig := AggregateSigs([]Signature{sig, sig})                       //同一个原始签名相加，生成聚合签名
	if !VerifyAggregateSig([]Pubkey{*pub, *pub}, []byte("hi"), asig) { //对同一个原始公钥进行聚合后（生成聚合公钥），去验证聚合签名
		t.Error("Aggregated signature does not verify")
	}
	{
		var sig2 Signature
		err := sig2.SetHexString(sig.GetHexString()) //测试原始签名的字符串导出
		if err != nil || !sig.IsEqual(sig2) {        //检查字符串导入生成的签名是否和之前的签名相同
			t.Error("sig2.SetHexString")
		}
	}
	{
		var sig2 Signature
		err := sig2.Deserialize(sig.Serialize()) //测试原始签名的序列化
		if err != nil || !sig.IsEqual(sig2) {    //检查反序列化生成的签名是否跟之前的签名相同
			t.Error("sig2.Deserialize")
		}
	}
	t.Log("end testComparison")
	fmt.Printf("\nend test Comparison.\n")
}

//测试从big.Int生成私钥，以及私钥的序列化
func testSeckey(t *testing.T) {
	fmt.Printf("\nbegin test sec key...\n")
	t.Log("testSeckey")
	s := "401035055535747319451436327113007154621327258807739504261475863403006987855"
	var b = new(big.Int)
	b.SetString(s, 10)
	sec := NewSeckeyFromBigInt(b) //以固定的字符串常量构建私钥
	str := sec.GetHexString()
	fmt.Printf("sec key export, len=%v, data=%v.\n", len(str), str)
	{
		var sec2 Seckey
		err := sec2.SetHexString(str)         //测试私钥的十六进制字符串导出
		if err != nil || !sec.IsEqual(sec2) { //检查字符串导入生成的私钥是否和之前的私钥相同
			t.Error("bad SetHexString")
		}
		str = sec2.GetHexString()
		fmt.Printf("sec key import and export again, len=%v, data=%v.\n", len(str), str)
	}
	{
		var sec2 Seckey
		err := sec2.Deserialize(sec.Serialize()) //测试私钥的序列化
		if err != nil || !sec.IsEqual(sec2) {    //检查反序列化生成的私钥是否和之前的私钥相同
			t.Error("bad Serialize")
		}
	}
	fmt.Printf("end test sec key.\n")
}

//生成n个衍生随机数私钥，对这n个衍生私钥进行聚合生成组私钥，然后萃取出组公钥
func testAggregation(t *testing.T) {
	fmt.Printf("\nbegin test Aggregation...\n")
	t.Log("testAggregation")
	//    m := 5
	n := 3
	//    groupPubkeys := make([]Pubkey, m)
	r := base.NewRand()                      //生成随机数基
	seckeyContributions := make([]Seckey, n) //私钥切片
	for i := 0; i < n; i++ {
		seckeyContributions[i] = *NewSeckeyFromRand(r.Deri(i)) //以r为基，i为递增量生成n个相关性私钥
	}
	groupSeckey := AggregateSeckeys(seckeyContributions) //对n个私钥聚合，生成组私钥
	groupPubkey := NewPubkeyFromSeckey(*groupSeckey)     //从组私钥萃取出组公钥
	t.Log("Group pubkey:", groupPubkey.GetHexString())
	fmt.Printf("end test Aggregation.\n")
}

//把secs切片的每个私钥都转化成big.Int，累加后对曲线域求模，以求模后的big.Int作为参数构建一个新的私钥
func AggregateSeckeysByBigInt(secs []Seckey) *Seckey {
	secret := big.NewInt(0)
	for _, s := range secs {
		secret.Add(secret, s.GetBigInt())
	}
	secret.Mod(secret, curveOrder) //为什么不是每一步都求模，而是全部累加后求模？
	return NewSeckeyFromBigInt(secret)
}

//生成n个衍生随机数私钥，对这n个衍生私钥用聚合法和big.Int聚合法生成聚合私钥，比较2个聚合私钥是否一致。
func testAggregateSeckeys(t *testing.T) {
	fmt.Printf("\nbegin testAggregateSeckeys...\n")
	t.Log("begin testAggregateSeckeys")
	n := 100
	r := base.NewRand() //创建随机数基r
	secs := make([]Seckey, n)
	fmt.Printf("begin init 100 sec key...\n")
	for i := 0; i < n; i++ {
		secs[i] = *NewSeckeyFromRand(r.Deri(i)) //以基r和递增变量i生成随机数，创建私钥切片
	}
	fmt.Printf("begin aggr sec key with bigint...\n")
	s1 := AggregateSeckeysByBigInt(secs) //通过int加法和求模生成聚合私钥
	fmt.Printf("begin aggr sec key...\n")
	s2 := AggregateSeckeys(secs) //生成聚合私钥
	fmt.Printf("sec aggred with int, data=%v.\n", s1.GetHexString())
	fmt.Printf("sec aggred , data=%v.\n", s2.GetHexString())
	if !s1.value.IsEqual(&s2.value) { //比较用简单加法求模生成的聚合私钥和底层库生成的聚合私钥是否不同
		t.Errorf("not same int(%v) VS (%v).\n", s1.GetHexString(), s2.GetHexString())
	}
	t.Log("end testAggregateSeckeys")
	fmt.Printf("end testAggregateSeckeys.\n")
}

//big.Int处理法：以私钥切片和ID切片恢复出组私钥(私钥切片和ID切片的大小都为门限值k)
func RecoverSeckeyByBigInt(secs []Seckey, ids []ID) *Seckey {
	secret := big.NewInt(0) //组私钥
	k := len(secs)          //取得输出切片的大小，即门限值k
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		xs[i] = ids[i].GetBigInt() //把所有的id转化为big.Int，放到xs切片
	}
	// need len(ids) = k > 0
	for i := 0; i < k; i++ { //输入元素遍历
		// compute delta_i depending on ids only
		//为什么前面delta/num/den初始值是1，最后一个diff初始值是0？
		var delta, num, den, diff = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		for j := 0; j < k; j++ { //ID遍历
			if j != i { //不是自己
				num.Mul(num, xs[j])      //num值先乘上当前ID
				num.Mod(num, curveOrder) //然后对曲线域求模
				diff.Sub(xs[j], xs[i])   //diff=当前节点（内循环）-基节点（外循环）
				den.Mul(den, diff)       //den=den*diff
				den.Mod(den, curveOrder) //den对曲线域求模
			}
		}
		// delta = num / den
		den.ModInverse(den, curveOrder) //模逆
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)
		//最终需要的值是delta
		// apply delta to secs[i]
		delta.Mul(delta, secs[i].GetBigInt()) //delta=delta*当前节点私钥的big.Int
		// skip reducing delta modulo curveOrder here
		secret.Add(secret, delta)      //把delta加到组私钥（big.Int形式）
		secret.Mod(secret, curveOrder) //组私钥对曲线域求模（big.Int形式）
	}
	return NewSeckeyFromBigInt(secret) //用big.Int数生成真正的私钥
}

//生成n个ID和n个衍生随机数私钥, 比较2个恢复的私钥是否一致。
func testRecoverSeckey(t *testing.T) {
	fmt.Printf("\nbegin testRecoverSeckey...\n")
	t.Log("testRecoverSeckey")
	n := 50
	r := base.NewRand() //生成随机数基

	secs := make([]Seckey, n) //私钥切片
	ids := make([]ID, n)      //ID切片
	for i := 0; i < n; i++ {
		ids[i] = *NewIDFromInt64(int64(i + 3))  //生成50个ID
		secs[i] = *NewSeckeyFromRand(r.Deri(i)) //以基r和累加值i，生成50个私钥
	}
	s1 := RecoverSeckey(secs, ids)         //调用私钥恢复函数（门限值取100%）
	s2 := RecoverSeckeyByBigInt(secs, ids) //调用big.Int加法求模的私钥恢复函数
	if !s1.value.IsEqual(&s2.value) {      //检查两种方法恢复的私钥是否相同
		t.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", s1.GetHexString(), s2.GetHexString())
	}
	fmt.Printf("end testRecoverSeckey.\n")
}

//big.Int处理法：以master key切片和ID生成属于该ID的（签名）私钥
func ShareSeckeyByBigInt(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	// degree of polynomial, need k >= 1, i.e. len(msec) >= 2
	k := len(msec) - 1
	// msec = c_0, c_1, ..., c_k
	// evaluate polynomial f(x) with coefficients c0, ..., ck
	secret.Set(msec[k].GetBigInt()) //最后一个master key的big.Int值放到secret
	x := id.GetBigInt()             //取得id的big.Int值
	for j := k - 1; j >= 0; j-- {   //从master key切片的尾部-1往前遍历
		secret.Mul(secret, x) //乘上id的big.Int值，每一遍都需要乘，所以是指数？
		//sec.secret.Mod(&sec.secret, curveOrder)
		secret.Add(secret, msec[j].GetBigInt()) //加法
		secret.Mod(secret, curveOrder)          //曲线域求模
	}
	return NewSeckeyFromBigInt(secret) //生成签名私钥
}

//生成n个衍生随机数私钥，然后针对一个特定的ID生成分享片段和big.Int分享片段，比较2个分享片段是否一致。
func testShareSeckey(t *testing.T) {
	fmt.Printf("\nbegin testShareSeckey...\n")
	t.Log("testShareSeckey")
	n := 100
	msec := make([]Seckey, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		msec[i] = *NewSeckeyFromRand(r.Deri(i)) //生成100个随机私钥
	}
	id := *NewIDFromInt64(123)          //随机生成一个ID
	s1 := ShareSeckeyByBigInt(msec, id) //简单加法分享函数
	s2 := ShareSeckey(msec, id)         //分享函数
	if !s1.value.IsEqual(&s2.value) {   //比较2者是否相同
		t.Errorf("bad sec\n%s\n%s", s1.GetHexString(), s2.GetHexString())
	} else {
		buf := s2.Serialize()
		fmt.Printf("size of seckey = %v.\n", len(buf))
	}
	fmt.Printf("end testShareSeckey.\n")
}

//测试从big.Int生成ID，以及ID的序列化
func testID(t *testing.T) {
	t.Log("testString")
	fmt.Printf("\nbegin test ID...\n")
	b := new(big.Int)
	b.SetString("001234567890abcdef", 16)
	c := new(big.Int)
	c.SetString("1234567890abcdef", 16)
	idc := NewIDFromBigInt(c)
	id1 := NewIDFromBigInt(b) //从big.Int生成ID
	if id1.IsEqual(*idc) {
		fmt.Println("id1 is equal to idc")
	}
	if id1 == nil {
		t.Error("NewIDFromBigInt")
	} else {
		buf := id1.Serialize()
		fmt.Printf("id Serialize, len=%v, data=%v.\n", len(buf), buf)
	}

	str := id1.GetHexString()
	fmt.Printf("ID export, len=%v, data=%v.\n", len(str), str)
	//test
	str0 := id1.value.GetHexString()
	fmt.Printf("str0 =%v\n", str0)

	///test
	{
		var id2 ID
		err := id2.SetHexString(id1.GetHexString()) //测试ID的十六进制导出和导入功能
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
		}
	}
	{
		var id2 ID
		err := id2.Deserialize(id1.Serialize()) //测试ID的序列化和反序列化
		fmt.Printf("id2:%v", id2.GetHexString())
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
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
	testID(t)          //测试从big.Int生成ID，以及ID的序列化
	testSeckey(t)      //测试从big.Int生成私钥，以及私钥的序列化
	testPubkey(t)      //测试用衍生随机数生成私钥，从私钥萃取公钥，以及公钥的序列化
	testAggregation(t) //生成n个衍生随机数私钥，对这n个衍生私钥进行聚合生成组私钥，然后萃取出组公钥
	//用big.Int生成私钥，取得公钥和签名。然后对私钥、公钥和签名各复制一份后测试加法后的验证是否正确。
	//同时测试签名的序列化。
	testComparison(t)
	//生成n个衍生随机数私钥，对这n个衍生私钥用聚合法和big.Int聚合法生成聚合私钥，比较2个聚合私钥是否一致。
	//全量聚合，用于生成组成员签名私钥（对收到的秘密分片聚合）和组公钥（由组成员签名私钥萃取出公钥，然后在组内广播，任何一个成员收到全量公钥后聚合即生成组公钥）
	testAggregateSeckeys(t)
	//生成n个ID和n个衍生随机数私钥，比较2个恢复的私钥是否一致。
	//秘密分享恢复函数，门限值取了100%。
	testRecoverSeckey(t)
	//生成n个衍生随机数私钥，然后针对一个特定的ID生成分享片段和big.Int分享片段，比较2个分享片段是否一致。
	//秘密分享，把自己的秘密分片发送给组内不同的成员（对不同成员生成不同的秘密分片）
	testShareSeckey(t)
}

func Test_GroupsigIDStringConvert(t *testing.T) {
	str := "0xedb67046af822fd6a778f3a1ec01ad2253e5921d3c1014db958a952fdc1b98e2"
	id := NewIDFromString(str)
	s := id.GetHexString()
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
	t.Log(id1.GetHexString(), id2.GetHexString(), id1.IsEqual(*id2))

	t.Log([]byte(s))
	t.Log(id1.Serialize(), id2.Serialize())
	t.Log(id1.GetHexString(), id2.GetHexString())

	b := id2.Serialize()
	id3 := DeserializeID(b)
	t.Log(id3.GetHexString())
}

//测试BLS门限签名.
//Added by FlyingSquirrel-Xu. 2018-08-24.
func testRecover(n int, k int, b *testing.B) {
	//n := 50
	//k := 10

	//定义k-1次多项式 F(x): <a[0], a[1], ..., a[k-1]>. F(0)=a[0].
	a := make([]Seckey, k)
	r := base.NewRand()
	for i := 0; i < k; i++ {
		a[i] = *NewSeckeyFromRand(r.Deri(i))
	}
	//fmt.Println("a[0]:", a[0].Serialize())

	//生成n个成员ID: {IDi}, i=1,..,n.
	ids := make([]ID, n)
	for i := 0; i < n; i++ {
		ids[i] = *NewIDFromInt64(int64(i + 3)) //生成50个ID
	}

	//计算得到多项式F(x)上的n个点 <IDi, Si>. 满足F(IDi)=Si.
	secs := make([]Seckey, n) //私钥切片
	for j := 0; j < n; j++ {
		bs := ShareSeckey(a, ids[j])
		secs[j].value.SetBigInt(bs.value.GetBigInt())
	}

	//通过Lagrange插值公式, 由{<IDi, Si>|i=1..n}计算得到组签名私钥s=F(0).
	new_secs := secs[:k]
	s := RecoverSeckey(new_secs, ids)
	//fmt.Println("s:", s.Serialize())

	//检查 F(0) = a[0]?
	if !bytes.Equal(a[0].Serialize(), s.Serialize()) {
		fmt.Errorf("secreky Recover failed.")
	}

	//通过组签名私钥s得到组签名公钥 pub.
	pub := NewPubkeyFromSeckey(*s)

	//成员签名: H[i] = si·H(m)
	sig := make([]Signature, n)
	for i := 0; i < n; i++ {
		sig[i] = Sign(secs[i], []byte("hi")) //以原始私钥对明文签名，生成原始签名
	}

	//Recover组签名: H = ∑ ∆i(0)·Hi.
	new_sig := sig[:k]
	H := RecoverSignature(new_sig, ids) //调用big.Int加法求模的私钥恢复函数

	//组签名验证：Pair(H,Q)==Pair(Hm,Pub)?
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

	r := base.NewRand() //生成随机数

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

	r := base.NewRand() //生成随机数

	for n := 0; n < b.N; n++ {
		sec := NewSeckeyFromRand(r.Deri(1))
		b.StartTimer()
		Sign(*sec, []byte(strconv.Itoa(n)))
		b.StopTimer()
	}
}

func BenchmarkValidation(b *testing.B) {
	b.StopTimer()

	r := base.NewRand() //生成随机数

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
	r := base.NewRand() //生成随机数
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

	r := base.NewRand() //生成随机数
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

	r := base.NewRand() //生成随机数
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

	r := base.NewRand() //生成随机数
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
	addr := common.HexToAddress("0x0bf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7")
	id := DeserializeID(addr.Bytes())
	t.Log(id.GetHexString(), len(id.GetHexString()), addr.Hex() == id.GetHexString())
	t.Log(id.Serialize(), addr.Bytes(), bytes.Equal(id.Serialize(), addr.Bytes()))

	id2 := ID{}
	id2.SetHexString("0x0bf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7")
	t.Log(id2.GetHexString())

	json, _ := id2.MarshalJSON()
	t.Log(string(json))

	id3 := &ID{}
	id3.UnmarshalJSON(json)
	t.Log(id3.GetHexString())
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

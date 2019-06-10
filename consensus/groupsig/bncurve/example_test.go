// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bncurve

//import (
//	"crypto/rand"
//	"testing"
//
//	"fmt"
//	"github.com/ethereum/go-ethereum/common/bitutil"
//	"github.com/stretchr/testify/require"
//)
//
//func TestExamplePair(t *testing.T) {
//	// This implements the tripartite Diffie-Hellman algorithm from "A One
//	// Round Protocol for Tripartite Diffie-Hellman", A. Joux.
//	// http://www.springerlink.com/content/cddc57yyva0hburb/fulltext.pdf
//
//	// Each of three parties, a, b and c, generate a private value.
//	a, _ := rand.Int(rand.Reader, Order)
//	b, _ := rand.Int(rand.Reader, Order)
//	c, _ := rand.Int(rand.Reader, Order)
//
//	// Then each party calculates g₁ and g₂ times their private value.
//	pa := new(G1).ScalarBaseMult(a)
//	qa := new(G2).ScalarBaseMult(a)
//
//	pb := new(G1).ScalarBaseMult(b)
//	qb := new(G2).ScalarBaseMult(b)
//
//	pc := new(G1).ScalarBaseMult(c)
//	qc := new(G2).ScalarBaseMult(c)
//
//	// Now each party exchanges its public values with the other two and
//	// all parties can calculate the shared key.
//	k1 := Pair(pb, qc)
//	k1.ScalarMult(k1, a)
//
//	k2 := Pair(pc, qa)
//	k2.ScalarMult(k2, b)
//
//	k3 := Pair(pa, qb)
//	k3.ScalarMult(k3, c)
//
//	// k1, k2 and k3 will all be equal.
//
//	require.Equal(t, k1, k2)
//	require.Equal(t, k1, k3)
//
//	require.Equal(t, len(np), 4) //Avoid gometalinter varcheck err on np
//}
//
////聚合加密测试
//func TestAggEncrypt(t *testing.T) {
//	//------------1.初始化-----------
//	//生成A B的公私钥对<s1, P1> <s2, P2>
//	s1, _ := randomK(rand.Reader)
//	s2, _ := randomK(rand.Reader)
//	P1, P2 := &G1{}, &G1{}
//	P1.ScalarBaseMult(s1)
//	P2.ScalarBaseMult(s2)
//
//	//------------2.加密-----------
//	//得到随机M
//	M := make([]byte, 32)
//	rand.Read(M)
//	t.Log("M:", M)
//
//	//加密M， 得到<C1, R1>
//	r1, _ := randomK(rand.Reader)
//
//	R1 := &G1{}
//	R1.ScalarBaseMult(r1)
//
//	S1 := &G1{}
//	S1.ScalarMult(P1, r1)
//	C1 := make([]byte, 32)
//	bitutil.XORBytes(C1, M, S1.Marshal())
//
//	//再次加密<C1, R1>， 得到<C2, R1, R2>
//	r2, _ := randomK(rand.Reader)
//	R2 := &G1{}
//	R2.ScalarBaseMult(r2)
//
//	S2 := &G1{}
//	S2.ScalarMult(P2, r2)
//	C2 := make([]byte, 32)
//	bitutil.XORBytes(C2, C1, S2.Marshal())
//
//	//------------3.解密-----------
//	SS1, SS2 := &G1{}, &G1{}
//	SS1 = R1.ScalarMult(R1, s1) //A计算出
//	SS2 = R2.ScalarMult(R2, s2) //B计算出
//
//	//A 先解密，B后解密
//	D1, D2 := make([]byte, 32), make([]byte, 32)
//	bitutil.XORBytes(D1, C2, SS1.Marshal()) //A计算出D1
//	bitutil.XORBytes(D2, D1, SS2.Marshal()) //B计算出D2
//	t.Log("D2:", D2)
//
//	//B先解密，A后解密
//	DD1, DD2 := make([]byte, 32), make([]byte, 32)
//	bitutil.XORBytes(DD1, C2, SS2.Marshal())
//	bitutil.XORBytes(DD2, DD1, SS1.Marshal())
//	t.Log("DD2:", DD2)
//}
//
////盲化加密测试
//func TestAggBlindEncrypt(t *testing.T) {
//	//------------1.初始化-----------
//	//生成A B的公私钥对<s1, P1> <s2, P2>
//	s1, _ := randomK(rand.Reader)
//	s2, _ := randomK(rand.Reader)
//	P1, P2 := &G1{}, &G1{}
//	P1.ScalarBaseMult(s1)
//	P2.ScalarBaseMult(s2)
//
//	//------------2.加密-----------
//	//得到随机M
//	M := make([]byte, 32)
//	rand.Read(M)
//	t.Log("M:", M)
//
//	//A 加密M， 得到<C1, R1>
//	r1, _ := randomK(rand.Reader)
//
//	R1 := &G1{}
//	R1.ScalarBaseMult(r1)
//
//	S1 := &G1{}
//	S1.ScalarMult(P1, r1)
//	C1 := make([]byte, 32)
//	bitutil.XORBytes(C1, M, S1.Marshal())
//
//	//B 盲化R1得到K1=s2*R1
//	K1 := &G1{}
//	K1 = R1.ScalarMult(R1, s2) //A计算出
//
//	//------------3.解密-----------
//	//A 计算得到 L1 = s1·K1
//	L1 := &G1{}
//	L1 = K1.ScalarMult(K1, s1)
//
//	//B脱盲运算
//	t2 := s2.ModInverse(s2, Order) //t2 = 1/s2.
//	T1 := &G1{}
//	T1 = L1.ScalarMult(L1, t2)
//
//	//异或运算:
//	D1 := make([]byte, 32)
//	bitutil.XORBytes(D1, C1, T1.Marshal()) //A计算出D1
//	t.Log("D1:", D1)
//}
//
//func TestCpu(t *testing.T) {
//	t.Log("hasBMI2:", hasBMI2)
//}
//
//func Bytes2Bits(data []byte) []int {
//	dst := make([]int, 0)
//	for _, v := range data {
//		for i := 0; i < 8; i++ {
//			move := uint(7 - i)
//			dst = append(dst, int((v>>move)&1))
//		}
//	}
//	fmt.Println(len(dst))
//	return dst
//}
//
////通过big.Int方式，短签名恢复出签名点(x,y).[BUG修复中]
//func BenchmarkShortSig(b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		_, g1, _ := RandomG1(rand.Reader)
//		px, _, isOdd := g1.GetXY()
//
//		g2 := &G1{}
//		g2.SetX(px, isOdd)
//		ppx, ppy, _ := g2.GetXY()
//
//		buf_xx := make([]byte, 32)
//		buf_yy := make([]byte, 32)
//		ppx.Marshal(buf_xx)
//		ppy.Marshal(buf_yy)
//
//		gfpNeg(ppy, ppy)
//		ppy.Marshal(buf_yy)
//
//		_, h1, _ := RandomG2(rand.Reader)
//		gt1 := Pair(g1, h1)
//
//		gt2 := &GT{}
//		gt2.Set(gt1)
//
//		if PairIsEuqal(gt1, gt2) != true {
//			gt2.p.Invert(gt1.p)
//			if PairIsEuqal(gt1, gt2) != true {
//				b.Error("check failed.")
//			}
//		}
//	}
//}

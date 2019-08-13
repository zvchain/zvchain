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

// Package implements a particular bilinear group at the 128-bit security
// level.
//
// Bilinear groups are the basis of many of the new cryptographic protocols that
// have been proposed over the past decade. They consist of a triplet of groups
// (G₁, G₂ and GT) such that there exists a function e(g₁ˣ,g₂ʸ)=gTˣʸ (where gₓ
// is a generator of the respective group). That function is called a pairing
// function.
//
// This package specifically implements the Optimal Ate pairing over a 256-bit
// Barreto-Naehrig curve as described in
// http://cryptojedi.org/papers/dclxvi-20100714.pdf. Its output is compatible
// with the implementation described in that paper.

package bncurve

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"math/big"

	"github.com/minio/sha256-simd"
)

func randomK(r io.Reader) (k *big.Int, err error) {
	for {
		k, err = rand.Int(r, Order)
		if k.Sign() > 0 || err != nil {
			return
		}
	}
}

// G1 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G1 struct {
	p *curvePoint
}

// RandomG1 returns x and g₁ˣ where x is a random, non-zero number read from r.
func RandomG1(r io.Reader) (*big.Int, *G1, error) {
	k, err := randomK(r)
	if err != nil {
		return nil, nil, err
	}

	return k, new(G1).ScalarBaseMult(k), nil
}

func (g *G1) getXY() (*gfP, *gfP, bool) {
	p := &curvePoint{}
	p.Set(g.p)
	p.MakeAffine()

	return &p.x, &p.y, p.y.IsOdd()
}

func (g *G1) setX(px *gfP, isOdd bool) error {
	//compute t=x³+b in gfP.
	pt := &gfP{}
	gfpMul(pt, px, px)
	gfpMul(pt, pt, px)
	gfpAdd(pt, pt, curveB)
	montDecode(pt, pt)

	//compute y=sqrt(t).
	y := &big.Int{}
	buf := make([]byte, 32)
	pt.Marshal(buf)
	y.SetBytes(buf)
	y.ModSqrt(y, P)

	py := &gfP{}
	yBytes := y.Bytes()
	if len(yBytes) == 32 {
		py.Unmarshal(yBytes)
	} else {
		buf1 := make([]byte, 32)
		copy(buf1[32-len(yBytes):32], yBytes)
		py.Unmarshal(buf1)
	}
	montEncode(py, py)

	if py.IsOdd() != isOdd {
		gfpNeg(py, py)
	}
	g.p = &curvePoint{*px, *py, *newGFp(1), *newGFp(1)}

	return nil
}

// Hash m to a point in Curve.
// Using try-and-increment method
// 	in https://www.normalesup.org/~tibouchi/papers/bnhash-scis.pdf
func hashToCurvePoint(m []byte) (*big.Int, *big.Int) {
	biCurveB := new(big.Int).SetInt64(3)
	one := big.NewInt(1)

	h := sha256.Sum256(m)
	x := new(big.Int).SetBytes(h[:])
	x.Mod(x, P)

	for {
		xxx := new(big.Int).Mul(x, x)
		xxx.Mul(xxx, x)
		t := new(big.Int).Add(xxx, biCurveB)

		y := new(big.Int).ModSqrt(t, P)
		if y != nil {
			return x, y
		}

		x.Add(x, one)
	}
}

func (g *G1) HashToPoint(m []byte) error {
	x, y := hashToCurvePoint(m)
	Px, Py := &gfP{}, &gfP{}

	xStr := x.Bytes()
	if len(xStr) == 32 {
		Px.Unmarshal(xStr)
	} else {
		bufX := make([]byte, 32)
		copy(bufX[32-len(xStr):32], xStr)
		Px.Unmarshal(bufX)
	}
	montEncode(Px, Px)

	yStr := y.Bytes()
	if len(yStr) == 32 {
		Py.Unmarshal(yStr)
	} else {
		bufY := make([]byte, 32)
		copy(bufY[32-len(yStr):32], yStr)
		Py.Unmarshal(bufY)
	}
	montEncode(Py, Py)

	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.x.Set(Px)
	g.p.y.Set(Py)
	g.p.z.Set(newGFp(1))
	g.p.t.Set(newGFp(1))

	if g.IsValid() {
		return nil
	}
	return errors.New("hash to point failed")
}

func (g *G1) String() string {
	return "bncurve.G1" + g.p.String()
}

func (g *G1) IsValid() bool {
	return g.p.IsOnCurve()
}

func (g *G1) IsNil() bool {
	return g.p == nil
}

// ScalarBaseMult sets e to g*k where g is the generator of the group and then
// returns e.
func (g *G1) ScalarBaseMult(k *big.Int) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Mul(curveGen, k)
	return g
}

// ScalarMult sets e to a*k and then returns e.
func (g *G1) ScalarMult(a *G1, k *big.Int) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Mul(a.p, k)
	return g
}

// Add sets e to a+b and then returns e.
func (g *G1) Add(a, b *G1) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Add(a.p, b.p)
	return g
}

// Neg sets e to -a and then returns e.
func (g *G1) Neg(a *G1) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Neg(a.p)
	return g
}

// Set sets e to a and then returns e.
func (g *G1) Set(a *G1) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Set(a.p)
	return g
}

// Marshal converts e to a byte slice.
func (g *G1) Marshal() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	g.p.MakeAffine()
	ret := make([]byte, numBytes+1)
	if g.p.IsInfinity() {
		return ret
	}

	temp := &gfP{}
	montDecode(temp, &g.p.x)
	temp.Marshal(ret)

	if g.p.y.IsOdd() == true {
		ret[numBytes] = 0x1
	} else {
		ret[numBytes] = 0x0
	}

	return ret
}

// Unmarshal sets e to the result of converting the output of Marshal back into
// a group element and then returns e.
func (g *G1) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < numBytes+1 {
		return nil, errors.New("bncurve: not enough data")
	}
	// Unmarshal the points and check their caps
	if g.p == nil {
		g.p = &curvePoint{}
	} else {
		g.p.x, g.p.y = gfP{0}, gfP{0}
	}
	var err error
	if err = g.p.x.Unmarshal(m); err != nil {
		return nil, err
	}
	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&g.p.x, &g.p.x)

	zero := gfP{0}
	if g.p.x == zero && g.p.y == zero {
		// This is the point at infinity.
		g.p.y = *newGFp(1)
		g.p.z = gfP{0}
		g.p.t = gfP{0}
	} else {
		g.p.z = *newGFp(1)
		g.p.t = *newGFp(1)

		isOdd := true
		if m[numBytes] == 0x1 {
			isOdd = true
		} else {
			isOdd = false
		}
		g.setX(&g.p.x, isOdd)

		if !g.p.IsOnCurve() {
			return nil, errors.New("bncurve: malformed point")
		}
	}
	return m[numBytes+1:], nil
}

// G2 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G2 struct {
	p *twistPoint
}

func (e *G2) IsEmpty() bool {
	return e.p == nil
}

// RandomG2 returns x and g₂ˣ where x is a random, non-zero number read from r.
func RandomG2(r io.Reader) (*big.Int, *G2, error) {
	k, err := randomK(r)
	if err != nil {
		return nil, nil, err
	}

	return k, new(G2).ScalarBaseMult(k), nil
}

func (e *G2) String() string {
	return "bncurve.G2" + e.p.String()
}

// ScalarBaseMult sets e to g*k where g is the generator of the group and then
// returns out.
func (e *G2) ScalarBaseMult(k *big.Int) *G2 {
	if e.p == nil {
		e.p = &twistPoint{}
	}
	e.p.Mul(twistGen, k)
	return e
}

// ScalarMult sets e to a*k and then returns e.
func (e *G2) ScalarMult(a *G2, k *big.Int) *G2 {
	if e.p == nil {
		e.p = &twistPoint{}
	}
	e.p.Mul(a.p, k)
	return e
}

// Add sets e to a+b and then returns e.
func (e *G2) Add(a, b *G2) *G2 {
	if e.p == nil {
		e.p = &twistPoint{}
	}
	e.p.Add(a.p, b.p)
	return e
}

// Neg sets e to -a and then returns e.
func (e *G2) Neg(a *G2) *G2 {
	if e.p == nil {
		e.p = &twistPoint{}
	}
	e.p.Neg(a.p)
	return e
}

// Set sets e to a and then returns e.
func (e *G2) Set(a *G2) *G2 {
	if e.p == nil {
		e.p = &twistPoint{}
	}
	e.p.Set(a.p)
	return e
}

// Marshal converts e into a byte slice.
func (e *G2) Marshal() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	if e.p == nil {
		e.p = &twistPoint{}
	}

	e.p.MakeAffine()
	ret := make([]byte, numBytes*4)
	if e.p.IsInfinity() {
		return ret
	}
	temp := &gfP{}

	montDecode(temp, &e.p.x.x)
	temp.Marshal(ret)
	montDecode(temp, &e.p.x.y)
	temp.Marshal(ret[numBytes:])
	montDecode(temp, &e.p.y.x)
	temp.Marshal(ret[2*numBytes:])
	montDecode(temp, &e.p.y.y)
	temp.Marshal(ret[3*numBytes:])

	return ret
}

// Marshal converts e into a byte slice . (compressed mode)
func (e *G2) MarshalCompressed() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	if e.p == nil {
		e.p = &twistPoint{}
	}

	e.p.MakeAffine()
	ret := make([]byte, numBytes*2+1)
	if e.p.IsInfinity() {
		return ret
	}
	temp := &gfP{}

	montDecode(temp, &e.p.x.x)
	temp.Marshal(ret)
	montDecode(temp, &e.p.x.y)
	temp.Marshal(ret[numBytes:])
	if e.p.y.x.IsOdd() {
		ret[numBytes*2] = 0x1
	} else {
		ret[numBytes*2] = 0x2
	}
	return ret
}

// Unmarshal sets e to the result of converting the output of Marshal back into
// a group element and then returns e.
func (e *G2) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < 4*numBytes {
		return nil, errors.New("bncurve: not enough data")
	}
	// Unmarshal the points and check their caps
	if e.p == nil {
		e.p = &twistPoint{}
	}
	var err error
	if err = e.p.x.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = e.p.x.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	if err = e.p.y.x.Unmarshal(m[2*numBytes:]); err != nil {
		return nil, err
	}
	if err = e.p.y.y.Unmarshal(m[3*numBytes:]); err != nil {
		return nil, err
	}
	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&e.p.x.x, &e.p.x.x)
	montEncode(&e.p.x.y, &e.p.x.y)
	montEncode(&e.p.y.x, &e.p.y.x)
	montEncode(&e.p.y.y, &e.p.y.y)

	if e.p.x.IsZero() && e.p.y.IsZero() {
		// This is the point at infinity.
		e.p.y.SetOne()
		e.p.z.SetZero()
		e.p.t.SetZero()
	} else {
		e.p.z.SetOne()
		e.p.t.SetOne()

		if !e.p.IsOnCurve() {
			return nil, errors.New("bncurve: malformed point")
		}
	}
	return m[4*numBytes:], nil
}

// UnmarshalCompressed sets e to the result of converting the output of MarshalCompressed back into
// a group element and then returns e.
func (e *G2) UnmarshalCompressed(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < 2*numBytes+1 {
		return nil, errors.New("bncurve: not enough data")
	}
	// Unmarshal the points and check their caps
	if e.p == nil {
		e.p = &twistPoint{}
	}
	var err error
	if err = e.p.x.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = e.p.x.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}

	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&e.p.x.x, &e.p.x.x)
	montEncode(&e.p.x.y, &e.p.x.y)

	if m[2*numBytes] == 0x0 {
		// This is the point at infinity.
		e.p.y.SetOne()
		e.p.z.SetZero()
		e.p.t.SetZero()
	} else {
		x3 := &gfP2{}
		x3.Square(&e.p.x).Mul(x3, &e.p.x).Add(x3, twistB)

		e.p.z.SetOne()
		e.p.t.SetOne()

		if !e.p.IsOnCurve() {
			return nil, errors.New("bncurve: malformed point")
		}
	}
	return m[2*numBytes:], nil
}

// GT is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type GT struct {
	p *gfP12
}

// Pair calculates an Optimal Ate pairing.
func Pair(g1 *G1, g2 *G2) *GT {
	return &GT{optimalAte(g2.p, g1.p)}
}

// PairingCheck calculates the Optimal Ate pairing for a set of points.
func PairingCheck(a []*G1, b []*G2) bool {
	acc := new(gfP12)
	acc.SetOne()

	for i := 0; i < len(a); i++ {
		if a[i].p.IsInfinity() || b[i].p.IsInfinity() {
			continue
		}
		acc.Mul(acc, miller(b[i].p, a[i].p))
	}
	return finalExponentiation(acc).IsOne()
}

// Miller applies Miller's algorithm, which is a bilinear function from the
// source groups to F_p^12. Miller(g1, g2).Finalize() is equivalent to Pair(g1,
// g2).
func Miller(g1 *G1, g2 *G2) *GT {
	return &GT{miller(g2.p, g1.p)}
}

func (gt *GT) String() string {
	return "bncurve.GT" + gt.p.String()
}

// ScalarMult sets e to a*k and then returns e.
func (gt *GT) ScalarMult(a *GT, k *big.Int) *GT {
	if gt.p == nil {
		gt.p = &gfP12{}
	}
	gt.p.Exp(a.p, k)
	return gt
}

// Add sets e to a+b and then returns e.
func (gt *GT) Add(a, b *GT) *GT {
	if gt.p == nil {
		gt.p = &gfP12{}
	}
	gt.p.Mul(a.p, b.p)
	return gt
}

// Neg sets e to -a and then returns e.
func (gt *GT) Neg(a *GT) *GT {
	if gt.p == nil {
		gt.p = &gfP12{}
	}
	gt.p.Conjugate(a.p)
	return gt
}

// Set sets e to a and then returns e.
func (gt *GT) Set(a *GT) *GT {
	if gt.p == nil {
		gt.p = &gfP12{}
	}
	gt.p.Set(a.p)
	return gt
}

// Finalize is a linear function from F_p^12 to GT.
func (gt *GT) Finalize() *GT {
	ret := finalExponentiation(gt.p)
	gt.p.Set(ret)
	return gt
}

// Marshal converts e into a byte slice.
func (gt *GT) Marshal() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	ret := make([]byte, numBytes*12)
	temp := &gfP{}

	montDecode(temp, &gt.p.x.x.x)
	temp.Marshal(ret)
	montDecode(temp, &gt.p.x.x.y)
	temp.Marshal(ret[numBytes:])
	montDecode(temp, &gt.p.x.y.x)
	temp.Marshal(ret[2*numBytes:])
	montDecode(temp, &gt.p.x.y.y)
	temp.Marshal(ret[3*numBytes:])
	montDecode(temp, &gt.p.x.z.x)
	temp.Marshal(ret[4*numBytes:])
	montDecode(temp, &gt.p.x.z.y)
	temp.Marshal(ret[5*numBytes:])
	montDecode(temp, &gt.p.y.x.x)
	temp.Marshal(ret[6*numBytes:])
	montDecode(temp, &gt.p.y.x.y)
	temp.Marshal(ret[7*numBytes:])
	montDecode(temp, &gt.p.y.y.x)
	temp.Marshal(ret[8*numBytes:])
	montDecode(temp, &gt.p.y.y.y)
	temp.Marshal(ret[9*numBytes:])
	montDecode(temp, &gt.p.y.z.x)
	temp.Marshal(ret[10*numBytes:])
	montDecode(temp, &gt.p.y.z.y)
	temp.Marshal(ret[11*numBytes:])

	return ret
}

// Unmarshal sets e to the result of converting the output of Marshal back into
// a group element and then returns e.
func (gt *GT) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	if len(m) < 12*numBytes {
		return nil, errors.New("bncurve: not enough data")
	}

	if gt.p == nil {
		gt.p = &gfP12{}
	}

	var err error
	if err = gt.p.x.x.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = gt.p.x.x.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.x.y.x.Unmarshal(m[2*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.x.y.y.Unmarshal(m[3*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.x.z.x.Unmarshal(m[4*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.x.z.y.Unmarshal(m[5*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.y.x.x.Unmarshal(m[6*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.y.x.y.Unmarshal(m[7*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.y.y.x.Unmarshal(m[8*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.y.y.y.Unmarshal(m[9*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.y.z.x.Unmarshal(m[10*numBytes:]); err != nil {
		return nil, err
	}
	if err = gt.p.y.z.y.Unmarshal(m[11*numBytes:]); err != nil {
		return nil, err
	}
	montEncode(&gt.p.x.x.x, &gt.p.x.x.x)
	montEncode(&gt.p.x.x.y, &gt.p.x.x.y)
	montEncode(&gt.p.x.y.x, &gt.p.x.y.x)
	montEncode(&gt.p.x.y.y, &gt.p.x.y.y)
	montEncode(&gt.p.x.z.x, &gt.p.x.z.x)
	montEncode(&gt.p.x.z.y, &gt.p.x.z.y)
	montEncode(&gt.p.y.x.x, &gt.p.y.x.x)
	montEncode(&gt.p.y.x.y, &gt.p.y.x.y)
	montEncode(&gt.p.y.y.x, &gt.p.y.y.x)
	montEncode(&gt.p.y.y.y, &gt.p.y.y.y)
	montEncode(&gt.p.y.z.x, &gt.p.y.z.x)
	montEncode(&gt.p.y.z.y, &gt.p.y.z.y)

	return m[12*numBytes:], nil
}

func GetG2Base() *G2 {
	return &G2{twistGen}
}

func PairIsEuqal(g1 *GT, g2 *GT) bool {
	return bytes.Equal(g1.Marshal(), g2.Marshal())
}

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

	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig/bncurve"
)

const PREFIX = "0x"

func revertString(b string) string {
	len := len(b)
	buf := make([]byte, len)
	for i := 0; i < len; i++ {
		buf[i] = b[len-1-i]
	}
	return string(buf)
}

func HashToG1(m string) *bncurve.G1 {
	g := &bncurve.G1{}
	g.HashToPoint([]byte(m))
	return g
}

type BnInt struct {
	v big.Int
}

func (bi *BnInt) IsEqual(b *BnInt) bool {
	return 0 == bi.v.Cmp(&b.v)
}

func (bi *BnInt) SetDecString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

func (bi *BnInt) Add(b *BnInt) error {
	bi.v.Add(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Sub(b *BnInt) error {
	bi.v.Sub(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Mul(b *BnInt) error {
	bi.v.Mul(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Mod() error {
	bi.v.Mod(&bi.v, bncurve.Order)
	return nil
}

func (bi *BnInt) SetBigInt(b *big.Int) error {
	bi.v.Set(b)
	return nil
}

func (bi *BnInt) SetString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

func (bi *BnInt) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]
	bi.v.SetString(buf[:], 16)
	return nil
}

// GetBigInt export BlsInt as big.Int
func (bi *BnInt) GetBigInt() *big.Int {
	return new(big.Int).Set(&bi.v)
}

func (bi *BnInt) GetString() string {
	b := bi.GetBigInt().Bytes()
	return string(b)
}

func (bi *BnInt) GetHexString() string {
	buf := bi.v.Text(16)
	return PREFIX + buf
}

func (bi *BnInt) Serialize() []byte {
	return bi.v.Bytes()
}

func (bi *BnInt) Deserialize(b []byte) error {
	bi.v.SetBytes(b)
	return nil
}

type bnG2 struct {
	v bncurve.G2
}

func (bg *bnG2) Deserialize(b []byte) error {
	bg.v.Unmarshal(b)
	return nil
}

func (bg *bnG2) Serialize() []byte {
	return bg.v.Marshal()
}

func (bg *bnG2) Add(bh *bnG2) error {
	bg.v.Add(&bg.v, &bh.v)
	return nil
}

func (sec *Seckey) GetMasterSecretKey(k int) (msk []Seckey) {
	msk = make([]Seckey, k)
	msk[0] = *sec

	// Generating random number
	r := base.NewRand()
	for i := 1; i < k; i++ {
		msk[i] = *NewSeckeyFromRand(r.Deri(1))
	}
	return msk
}

func GetMasterPublicKey(msk []Seckey) (mpk []Pubkey) {
	n := len(msk)
	mpk = make([]Pubkey, n)
	for i := 0; i < n; i++ {
		mpk[i] = *NewPubkeyFromSeckey(msk[i])
	}
	return mpk
}

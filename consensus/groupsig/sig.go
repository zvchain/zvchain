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
	"bytes"
	"fmt"
	"log"
	"math/big"
	"sort"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig/bncurve"
)

const SignatureLength = 33

type Signature struct {
	value bncurve.G1
}

func (sig *Signature) IsNil() bool {
	return sig.value.IsNil()
}

func (sig *Signature) Add(sig1 *Signature) error {
	newSig := &Signature{}
	newSig.value.Set(&sig.value)
	sig.value.Add(&newSig.value, &sig1.value)

	return nil
}

func (sig *Signature) Mul(bi *big.Int) error {
	g1 := new(bncurve.G1)
	g1.Set(&sig.value)
	sig.value.ScalarMult(g1, bi)
	return nil
}

// IsEqual compare two signatures for the same
func (sig Signature) IsEqual(rhs Signature) bool {

	return bytes.Equal(sig.value.Marshal(), rhs.value.Marshal())
}

// SignatureAMap is an address to signature mapping
type SignatureAMap map[common.Address]Signature
type SignatureIMap map[string]Signature

func (sig Signature) GetHash() common.Hash {
	buf := sig.Serialize()
	return base.Data2CommonHash(buf)
}

// GetRand generate a random number from the signature
func (sig Signature) GetRand() base.Rand {
	// Get the signed byte slice (serialization) first, then generate a
	// random number based on the byte slice
	return base.RandFromBytes(sig.Serialize())
}

func DeserializeSign(b []byte) *Signature {
	sig := &Signature{}
	sig.Deserialize(b)
	return sig
}

// Deserialize initialize signature by byte slice
func (sig *Signature) Deserialize(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("signature Deserialized failed")
	}
	sig.value.Unmarshal(b)
	return nil
}

// Serialize convert the signature to a byte slice
func (sig Signature) Serialize() []byte {
	if sig.IsNil() {
		return []byte{}
	}
	return sig.value.Marshal()
}

func (sig Signature) IsValid() bool {
	s := sig.Serialize()
	if len(s) == 0 {
		return false
	}

	return sig.value.IsValid()
}

// GetHexString convert the signature to a hex string
func (sig Signature) GetHexString() string {
	return PREFIX + common.Bytes2Hex(sig.value.Marshal())
}

// SetAddrString initialize signature by hex string
func (sig *Signature) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]

	if sig.value.IsNil() {
		sig.value = bncurve.G1{}
	}

	sig.value.Unmarshal(common.Hex2Bytes(buf))
	return nil
}

// Sign is a signature function. Sign the plaintext (hash) with
// the private key and return the signature object
func Sign(sec Seckey, msg []byte) (sig Signature) {
	bg := HashToG1(string(msg))
	sig.value.ScalarMult(bg, sec.GetBigInt())
	return sig
}

// VerifySig is a verify the function. Verify that a signature is
// from the private key corresponding to the public key
func VerifySig(pub Pubkey, msg []byte, sig Signature) bool {
	if sig.IsNil() || !sig.IsValid() {
		return false
	}
	if !pub.IsValid() {
		return false
	}
	if sig.value.IsNil() {
		return false
	}
	bQ := bncurve.GetG2Base()
	p1 := bncurve.Pair(&sig.value, bQ)

	Hm := HashToG1(string(msg))
	p2 := bncurve.Pair(Hm, &pub.value)

	return bncurve.PairIsEuqal(p1, p2)
}

// VerifyAggregateSig is a fragmentation merge verification function.
// First merge the public key slices and verify that the signature is
// from the private key corresponding to the public key.
func VerifyAggregateSig(pubs []Pubkey, msg []byte, asig Signature) bool {
	// Combine the public key with the public key slice (all public key slices instead of just k)
	pub := AggregatePubkeys(pubs)
	if pub == nil {
		return false
	}
	// Call validation function
	return VerifySig(*pub, msg, asig)
}

// BatchVerify is a bulk verification function。
func BatchVerify(pubs []Pubkey, msg []byte, sigs []Signature) bool {
	// Combine the signature slices into one, merge the public key
	// signatures into one, and then call the signature verification function.
	return VerifyAggregateSig(pubs, msg, AggregateSigs(sigs))
}

// AggregateSigs is a signature aggregate function
//
// Attention:The AggregateXXX family of functions adds all
// the slices, not the k additions.
func AggregateSigs(sigs []Signature) (sig Signature) {
	n := len(sigs)
	sig = Signature{}
	if n >= 1 {
		sig.value.Set(&sigs[0].value)
		for i := 1; i < n; i++ {
			newsig := &Signature{}
			newsig.value.Set(&sig.value)
			sig.value.Add(&newsig.value, &sigs[i].value)
		}
	}
	return sig
}

// RecoverSignature restore the master signature with signature slice
// and id slice (via Lagrangian interpolation)
//
// Attention : The number of slices of the RecoverXXX family function is fixed at k (threshold value).
func RecoverSignature(sigs []Signature, ids []ID) *Signature {
	// Get the size of the output slice, ie the threshold k
	k := len(sigs)
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		// Convert all ids to big.Int and put them in xs slices
		xs[i] = ids[i].GetBigInt()
	}

	// need len(ids) = k > 0
	sig := &Signature{}
	newSig := &Signature{}
	// Range input element
	for i := 0; i < k; i++ {
		// compute delta_i depending on ids only
		// (Why is the initial delta/num/den initial value of 1, and the last diff initial value is 0?)
		var delta, num, den, diff = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)
		// Range ID
		for j := 0; j < k; j++ {
			if j != i {

				// Num value is multiplied by the current ID
				num.Mul(num, xs[j])
				// Then model the curve domain
				num.Mod(num, curveOrder)
				// Diff=current node (internal loop)-base node (outer loop)
				diff.Sub(xs[j], xs[i])
				// Den=den*diff
				den.Mul(den, diff)
				// Den modeling the curve domain
				den.Mod(den, curveOrder)
			}
		}
		// Delta = num / den
		// Modular inverse
		den.ModInverse(den, curveOrder)
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)

		// The final value needed is delta
		newSig.value.Set(&sigs[i].value)
		newSig.Mul(delta)

		if i == 0 {
			sig.value.Set(&newSig.value)
		} else {
			sig.Add(newSig)
		}
	}
	return sig
}

func randK(m SignatureIMap, k int) SignatureIMap {
	indexs := base.NewRand().RandomPerm(len(m), k)
	sort.Ints(indexs)
	ret := make(SignatureIMap)

	i := 0
	j := 0
	for key, sign := range m {
		if i == indexs[j] {
			ret[key] = sign
			j++
			if j >= k {
				break
			}
		}
		i++
	}
	return ret
}

// RecoverSignatureByMapI is signature recovery function, m is map (
// ID-> signature), k is the threshold
func RecoverSignatureByMapI(m SignatureIMap, k int) *Signature {
	if k < len(m) {
		m = randK(m, k)
	}
	ids := make([]ID, k)
	sigs := make([]Signature, k)
	i := 0

	// Range map
	for sID, si := range m {
		var id ID
		id.SetAddrString(sID)
		// Group member ID value
		ids[i] = id
		// Group member signature
		sigs[i] = si
		i++
		if i >= k {
			break
		}
	}

	// Call signature recovery function
	return RecoverSignature(sigs, ids)
}

// RecoverSignatureByMapA is signature recovery function,
// m is map (address -> signature), k is the threshold
func RecoverSignatureByMapA(m SignatureAMap, k int) *Signature {
	ids := make([]ID, k)
	sigs := make([]Signature, k)
	i := 0
	// Range map
	for a, s := range m {
		// Get the ID corresponding to the address
		id := NewIDFromAddress(a)
		if id == nil {
			log.Printf("RecoverSignatureByMap bad address %s\n", a)
			return nil
		}
		// Group member ID value
		ids[i] = *id
		// Group member signature
		sigs[i] = s
		i++
		if i >= k {
			break
		}
	}
	// Call signature recovery function
	return RecoverSignature(sigs, ids)
}

func (sig *Signature) Recover(signVec []Signature, idVec []ID) error {

	return nil
}

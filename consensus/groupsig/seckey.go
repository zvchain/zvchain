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
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig/bncurve"
)

// Curve and Field order
var curveOrder = bncurve.Order // Curve integer field
var fieldOrder = bncurve.P
var bitLength = curveOrder.BitLen()

const SkLength = 32

// Seckey -- represented by a big.Int modulo curveOrder
type Seckey struct {
	value BnInt
}

// IsEqual compares whether two private keys are equal
func (sec Seckey) IsEqual(rhs Seckey) bool {
	return sec.value.IsEqual(&rhs.value)
}

// SeckeyMap is a mapping of an address to a private key
type SeckeyMap map[common.Address]Seckey

// SeckeyMapInt is a map from addresses to Seckey
type SeckeyMapInt map[int]Seckey

type SeckeyMapID map[string]Seckey

// Serialize converts the private key into byte slices (Little-Endian)
func (sec Seckey) Serialize() []byte {
	bytes := sec.value.Serialize()
	if len(bytes) == SkLength {
		return bytes
	}
	if len(bytes) > SkLength {
		// hold it for now
		panic("Seckey Serialize error: should be 32 bytes")
	}
	buff := make([]byte, SkLength)
	copy(buff[SkLength-len(bytes):], bytes)
	return buff
}

// GetBigInt convert the private key to big.int
func (sec Seckey) GetBigInt() (s *big.Int) {
	s = new(big.Int)
	s.Set(sec.value.GetBigInt())
	return s
}

func (sec Seckey) IsValid() bool {
	bi := sec.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0
}

// getHex returns a hexadecimal string representation without a prefix
func (sec Seckey) getHex() string {
	return sec.value.GetHexString()
}

// GetHexString returns a hexadecimal string representation with a prefix
func (sec Seckey) GetHexString() string {
	return sec.getHex()
}

// Deserialize initializes the private key by byte slice
func (sec *Seckey) Deserialize(b []byte) error {
	length := len(b)
	bytes := b
	if length < SkLength {
		bytes = make([]byte, SkLength)
		copy(bytes[SkLength-length:], b)
	} else if length > SkLength {
		bytes = b[length-SkLength:]
	}
	return sec.value.Deserialize(bytes)
}

func DeserializeSeckey(bs []byte) *Seckey {
	var sk Seckey
	sk.Deserialize(bs)
	return &sk
}

func (sec Seckey) MarshalJSON() ([]byte, error) {
	bs, err := json.Marshal(sec.GetHexString())
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (sec *Seckey) UnmarshalJSON(data []byte) error {
	var hex string
	err := json.Unmarshal(data, &hex)
	if err != nil {
		return err
	}
	return sec.SetHexString(hex)
}

// SetLittleEndian initializes the private key by byte slice (Little-Endian)
func (sec *Seckey) SetLittleEndian(b []byte) error {
	return sec.value.Deserialize(b[:32])
}

// setHex converts from an unprefixed hexadecimal string
func (sec *Seckey) setHex(s string) error {
	return sec.value.SetHexString(s)
}

// SetHexString converts from a prefixed hexadecimal string
func (sec *Seckey) SetHexString(s string) error {
	return sec.setHex(s)
}

// NewSeckeyFromHexString construct private keys from the input hex string
func NewSeckeyFromHexString(s string) *Seckey {
	sec := new(Seckey)
	err := sec.setHex(s)
	if err != nil {
		return nil
	}
	return sec
}

// NewSeckeyFromLittleEndian build private key by a byte slice(Little-Endian)
func NewSeckeyFromLittleEndian(b []byte) *Seckey {
	sec := new(Seckey)
	err := sec.SetLittleEndian(b)
	if err != nil {
		log.Printf("NewSeckeyFromLittleEndian %s\n", err)
		return nil
	}

	sec.value.Mod()
	return sec
}

// NewSeckeyFromRand construct private keys from random Numbers
func NewSeckeyFromRand(seed base.Rand) *Seckey {
	// After converting random Numbers into byte slices (Little-Endian),
	// the private key is constructed
	return NewSeckeyFromLittleEndian(seed.Bytes())
}

// NewSeckeyFromBigInt builds the private key from a large integer
func NewSeckeyFromBigInt(b *big.Int) *Seckey {
	nb := &big.Int{}
	nb.Set(b)

	// Large integers are modulating in the curve domain
	b.Mod(nb, curveOrder)

	sec := new(Seckey)
	sec.value.SetBigInt(b)

	return sec
}

// NewSeckeyFromInt64 build the private key from int64
func NewSeckeyFromInt64(i int64) *Seckey {
	return NewSeckeyFromBigInt(big.NewInt(i))
}

// NewSeckeyFromInt build the private key from int32
func NewSeckeyFromInt(i int) *Seckey {
	return NewSeckeyFromBigInt(big.NewInt(int64(i)))
}

// TrivialSeckey build a private key with low security requirements
func TrivialSeckey() *Seckey {
	// Take 1 as frequency hopping
	return NewSeckeyFromInt64(1)
}

// AggregateSeckeys is a private key aggregation function
func AggregateSeckeys(secs []Seckey) *Seckey {
	// No private keys to aggregate
	if len(secs) == 0 {
		log.Printf("AggregateSeckeys no secs")
		return nil
	}
	// create a new private key
	sec := new(Seckey)
	sec.value.SetBigInt(secs[0].value.GetBigInt())
	for i := 1; i < len(secs); i++ {
		sec.value.Add(&secs[i].value)
	}

	x := new(big.Int)
	x.Set(sec.value.GetBigInt())
	sec.value.SetBigInt(x.Mod(x, curveOrder))
	return sec
}

// ShareSeckey let polynomial replacement to generate a signature
// private key fragment specific to an ID
//
// msec : master private key slice
// id : get the id of the shard
func ShareSeckey(msec []Seckey, id ID) *Seckey {
	secret := big.NewInt(0)
	k := len(msec) - 1

	// Evaluate polynomial f(x) with coefficients c0, ..., ck
	//
	// The big.Int value of the last master key is placed in the secret
	secret.Set(msec[k].GetBigInt())
	// Get the big.Int value of the id
	x := id.GetBigInt()
	newB := &big.Int{}

	// Range from the tail -1 of the master key slice
	for j := k - 1; j >= 0; j-- {
		newB.Set(secret)
		// Multiply the big.Int value of the id, and each time you need to multiply
		secret.Mul(newB, x)

		newB.Set(secret)

		// Addition
		secret.Add(newB, msec[j].GetBigInt())

		newB.Set(secret)

		// Curve domain modeling
		secret.Mod(newB, curveOrder)
	}

	// Generate signature private key
	return NewSeckeyFromBigInt(secret)
}

// ShareSeckeyByAddr generating a private key fragment for the address
// by the master private key slice and the ZV address
func ShareSeckeyByAddr(msec []Seckey, addr common.Address) *Seckey {
	id := NewIDFromAddress(addr)
	if id == nil {
		log.Printf("ShareSeckeyByAddr bad addr=%s\n", addr)
		return nil
	}
	return ShareSeckey(msec, *id)
}

// ShareSeckeyByInt generate signature private key fragmentation by master
// private key slice and integer i
func ShareSeckeyByInt(msec []Seckey, i int) *Seckey {
	return ShareSeckey(msec, *NewIDFromInt64(int64(i)))
}

// ShareSeckeyByMembershipNumber generate the private key fragment of the
// id+1 by the master private key slice and the integer id
func ShareSeckeyByMembershipNumber(msec []Seckey, id int) *Seckey {
	return ShareSeckey(msec, *NewIDFromInt64(int64(id + 1)))
}

// RecoverSeckey restore the master private key with the (signature) private
// key slice slice and id slice (via Lagrangian interpolation)
//
// The number of private key slices and ID slices is fixed to the threshold k
func RecoverSeckey(secs []Seckey, ids []ID) *Seckey {
	// Group private key
	secret := big.NewInt(0)

	// Get the size of the output slice, ie the threshold k
	k := len(secs)
	xs := make([]*big.Int, len(ids))
	for i := 0; i < len(xs); i++ {
		// Convert all ids to big.Int and put them in xs slices
		xs[i] = ids[i].GetBigInt()
	}
	// Need len(ids) = k > 0
	// Input element traversal
	for i := 0; i < k; i++ {
		// Compute delta_i depending on ids only
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
		// apply delta to secs[i]
		// delta=(delta* big.Int of the private key of the current node)
		delta.Mul(delta, secs[i].GetBigInt())
		// skip reducing delta modulo curveOrder here
		// Add delta to the group private key (big.Int form)
		secret.Add(secret, delta)
		// Group private key modulo curve domain (big.Int form)
		secret.Mod(secret, curveOrder)
	}

	return NewSeckeyFromBigInt(secret)
}

// RecoverSeckeyByMap is a private key recovery function, m is map (
// address -> private key), k is the threshold
func RecoverSeckeyByMap(m SeckeyMap, k int) *Seckey {
	ids := make([]ID, k)
	secs := make([]Seckey, k)
	i := 0
	// Range mao
	for a, s := range m {
		// Extract the id corresponding to the address
		id := NewIDFromAddress(a)
		if id == nil {
			log.Printf("RecoverSeckeyByMap bad Address %s\n", a)
			return nil
		}
		// Group member ID
		ids[i] = *id
		// Group member private key
		secs[i] = s
		i++
		// Take the threshold
		if i >= k {
			break
		}
	}
	// Call private key recovery function
	return RecoverSeckey(secs, ids)
}

// RecoverSeckeyByMapInt retrieve the group private key by taking k (
// threshold value) from the signature private key fragment map
func RecoverSeckeyByMapInt(m SeckeyMapInt, k int) *Seckey {
	// k ID
	ids := make([]ID, k)
	// k signature private key fragmentation
	secs := make([]Seckey, k)
	i := 0
	// Take the first k signature private keys in the map to generate the recovery base
	for a, s := range m {
		ids[i] = *NewIDFromInt64(int64(a))
		secs[i] = s
		i++
		if i >= k {
			break
		}
	}
	// Restore the group private key
	return RecoverSeckey(secs, ids)
}

func (sec *Seckey) Set(msk []Seckey, id *ID) error {
	s := ShareSeckey(msk, *id)
	sec.Deserialize(s.Serialize())
	return nil
}

func (sec *Seckey) Recover(secVec []Seckey, idVec []ID) error {
	s := RecoverSeckey(secVec, idVec)
	sec.Deserialize(s.Serialize())

	return nil
}

// CheckSharePiecesValid returns true if all share pieces are valid, otherwise return false
// Parameter:   k   threshold of BLS
func CheckSharePiecesValid(shares []Seckey, ids []ID, k int, pk0 Pubkey) (bool, error) {
	if shares == nil || ids == nil {
		return false, fmt.Errorf("invalid input parameters in CheckSharePiecesValid")
	}
	if len(shares) != len(ids) {
		return false, fmt.Errorf("invalid parameters, shares and ids are not same size")
	}
	n := len(shares)
	if k > n || k <= 0 {
		return false, fmt.Errorf("invalid threshold k in CheckSharePiecesValid")
	}

	xs := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		// Convert all ids to big.Int and put them in xs slices
		xs[i] = ids[i].GetBigInt()
	}
	for m := k + 1; m < n; m++ {
		result := big.NewInt(0)

		for i := 0; i < k; i++ {
			var delta, num, den, diff = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)

			// Range ID
			for j := 0; j < k; j++ {
				if j != i {
					diff.Sub(xs[j], xs[m])
					num.Mul(num, diff)
					num.Mod(num, curveOrder)

					diff.Sub(xs[j], xs[i])
					den.Mul(den, diff)
					den.Mod(den, curveOrder)
				}
			}
			den.ModInverse(den, curveOrder)
			delta.Mul(num, den)
			delta.Mod(delta, curveOrder)

			delta.Mul(delta, shares[i].GetBigInt())
			result.Add(result, delta)
			result.Mod(result, curveOrder)
		}
		if result.Cmp(shares[m].GetBigInt()) != 0 {
			return false, nil
		}
	}

	//check pk0 valid
	a0 := big.NewInt(0)
	for i := 0; i < k; i++ {
		var delta, num, den, diff = big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(0)

		// Range ID
		for j := 0; j < k; j++ {
			if j != i {
				num.Mul(num, xs[j])
				num.Mod(num, curveOrder)

				diff.Sub(xs[j], xs[i])
				den.Mul(den, diff)
				den.Mod(den, curveOrder)
			}
		}
		den.ModInverse(den, curveOrder)
		delta.Mul(num, den)
		delta.Mod(delta, curveOrder)

		delta.Mul(delta, shares[i].GetBigInt())
		a0.Add(a0, delta)
		a0.Mod(a0, curveOrder)
	}
	sk0 := NewSeckeyFromBigInt(a0)
	pk := NewPubkeyFromSeckey(*sk0)
	if !pk.IsEqual(pk0) {
		return false, nil
	}
	return true, nil
}

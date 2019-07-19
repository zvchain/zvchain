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

	"github.com/zvchain/zvchain/common"

	//"fmt"
	"bytes"
	"log"

	"github.com/zvchain/zvchain/consensus/groupsig/bncurve"
	"golang.org/x/crypto/sha3"
)

// Pubkey is the user's public key, based BN Curve
type Pubkey struct {
	value bncurve.G2
}

type PubkeyMap map[string]Pubkey

//Check the public key is empty
func (pub Pubkey) IsEmpty() bool {
	return pub.value.IsEmpty()
}

// IsEqual judge two public key are equal
func (pub Pubkey) IsEqual(rhs Pubkey) bool {
	return bytes.Equal(pub.value.Marshal(), rhs.value.Marshal())
}

// Deserialize initializes the private key by byte slice
func (pub *Pubkey) Deserialize(b []byte) error {
	_, error := pub.value.Unmarshal(b)
	return error
}

// Serialize convert the public key into byte slices
func (pub Pubkey) Serialize() []byte {
	return pub.value.Marshal()
}

// MarshalJSON marshal the public key
func (pub Pubkey) MarshalJSON() ([]byte, error) {
	bs, err := json.Marshal(pub.GetHexString())
	if err != nil {
		return nil, err
	}
	return bs, nil
}

// UnmarshalJSON unmarshal the public key
func (pub *Pubkey) UnmarshalJSON(data []byte) error {
	var hex string
	err := json.Unmarshal(data, &hex)
	if err != nil {
		return err
	}
	return pub.SetHexString(hex)
}

//Check the public key is valid
func (pub Pubkey) IsValid() bool {
	return !pub.IsEmpty()
}

// GetAddress generate address from the public key
func (pub Pubkey) GetAddress() common.Address {
	// Get the SHA3 256-bit hash of the public key
	h := sha3.Sum256(pub.Serialize())
	// Tas160-bit addresses are generated from a 256-bit hash
	return common.BytesToAddress(h[:])
}

// GetHexString converts the public key to a hexadecimal string without the 0x prefix
func (pub Pubkey) GetHexString() string {
	return PREFIX + common.Bytes2Hex(pub.value.Marshal())
}

func (pub Pubkey) String() string {
	return pub.GetHexString()
}

// SetHexString initializes the public key from the hexadecimal string
func (pub *Pubkey) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	_, err := pub.value.Unmarshal(common.FromHex(s))
	return err
}

// NewPubkeyFromHexString generate the public key from the input hex string
func NewPubkeyFromHexString(s string) *Pubkey {
	pub := new(Pubkey)
	err := pub.SetHexString(s)
	if err != nil {
		return nil
	}
	return pub
}

// NewPubkeyFromSeckey generate the public key from the private key
func NewPubkeyFromSeckey(sec Seckey) *Pubkey {
	pub := new(Pubkey)
	pub.value.ScalarBaseMult(sec.value.GetBigInt())
	return pub
}

// TrivialPubkey build a public key that is not very secure
func TrivialPubkey() *Pubkey {
	return NewPubkeyFromSeckey(*TrivialSeckey())
}

func (pub *Pubkey) Add(rhs *Pubkey) error {
	pa := &bncurve.G2{}
	pb := &bncurve.G2{}

	pa.Set(&pub.value)
	pb.Set(&rhs.value)

	pub.value.Add(pa, pb)
	return nil
}

// AggregatePubkeys is a public key aggregation function
func AggregatePubkeys(pubs []Pubkey) *Pubkey {
	if len(pubs) == 0 {
		log.Printf("AggregatePubkeys no pubs")
		return nil
	}

	pub := new(Pubkey)
	pub.value.Set(&pubs[0].value)

	for i := 1; i < len(pubs); i++ {
		pub.Add(&pubs[i])
	}

	return pub
}

// SharePubkey is a public key shard generation function, using polynomial
// substitution to generate public key shards specific to an ID
//
// mpub : master public key slice
// id : get the id of the shard
func SharePubkey(mpub []Pubkey, id ID) *Pubkey {
	pub := &Pubkey{}
	// degree of polynomial, need k >= 1, i.e. len(msec) >= 2
	k := len(mpub) - 1
	// msec = c_0, c_1, ..., c_k
	// evaluate polynomial f(x) with coefficients c0, ..., ck
	pub.Deserialize(mpub[k].Serialize())

	// Get the big.Int value of the id
	x := id.GetBigInt()

	// Range from the tail -1 of the master key slice
	for j := k - 1; j >= 0; j-- {
		pub.value.ScalarMult(&pub.value, x)
		pub.value.Add(&pub.value, &mpub[j].value)
	}

	return pub
}

// SharePubkeyByInt call the public key shard generation function with i as ID
func SharePubkeyByInt(mpub []Pubkey, i int) *Pubkey {
	return SharePubkey(mpub, *NewIDFromInt(i))
}

// SharePubkeyByInt call the public key shard generation function with i+1 as ID
func SharePubkeyByMembershipNumber(mpub []Pubkey, id int) *Pubkey {
	return SharePubkey(mpub, *NewIDFromInt(id + 1))
}

func DeserializePubkeyBytes(bytes []byte) Pubkey {
	var pk Pubkey
	if err := pk.Deserialize(bytes); err != nil {
		return Pubkey{}
	}
	return pk
}

func DH(sk *Seckey, pk *Pubkey) *Pubkey {
	dh := new(Pubkey)
	dh.value.ScalarMult(&pk.value, &sk.value.v)
	return dh
}

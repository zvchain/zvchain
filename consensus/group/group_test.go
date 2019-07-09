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

package group

import (
	"fmt"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"testing"
)

func TestSharePiecesCryptogram(t *testing.T) {
	fmt.Printf("TestSharePiecesCryptogram begin \n")

	n := 9
	k := 5

	sks := make([]groupsig.Seckey, n)
	pks := make([]groupsig.Pubkey, n)
	ids := make([]groupsig.ID, n)
	r := base.NewRand()
	for i := 0; i < n; i++ {
		sks[i] = *groupsig.NewSeckeyFromRand(r.Deri(i))
		pks[i] = *groupsig.NewPubkeyFromSeckey(sks[i])
		err := ids[i].SetLittleEndian([]byte{1, 2, 3, 4, 5, byte(i)})
		if err != nil {
			t.Error(err)
		}
	}

	shares := make([][]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		shares[j] = make([]groupsig.Seckey, n)
		msec := sks[j].GetMasterSecretKey(k)
		for i := 0; i < n; i++ {
			err := shares[j][i].Set(msec, &ids[i])
			if err != nil {
				t.Error(err)
			}
		}
	}

	cs := make([][]byte, n)
	for j := 0; j < n; j++ {
		ct, err := encryptSharePieces(shares[j][:], sks[j], pks[:])
		if err != nil {
			fmt.Printf("fail to encryptSharePieces \n")
			return
		}
		cs[j] = make([]byte, len(ct))
		copy(cs[j], ct)
	}

	ps := make([][]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		ps[j] = make([]groupsig.Seckey, n)
	}

	for i := 0; i < n; i++ {
		pts, err := decryptSharePiecesWithMySK(cs, sks[i], i)
		if err != nil {
			fmt.Printf("fail to decryptSharePieces \n")
			return
		}
		for j := 0; j < n; j++ {
			ps[j][i] = pts[j]
		}
	}

	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			if !ps[j][i].IsEqual(shares[j][i]) {
				fmt.Printf("share piece doesn't match!!!\n")
			}
		}
	}

	for j := 0; j < n; j++ {
		b, err := groupsig.CheckSharePiecesValid(shares[j], ids, k)
		if err != nil {
			t.Error(err)
		}
		if !b {
			fmt.Printf("fail to check share pieces valid. i= %v \n", j)
		}
		b, err = checkEvil(cs[j], ids, shares[j], sks[j], pks)
		if err != nil {
			t.Error(err)
		}
		if b {
			fmt.Printf("i= %v is evil \n", j)
		}
	}

	msk := make([]groupsig.Seckey, n)
	shareVec := make([]groupsig.Seckey, n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			shareVec[i] = shares[i][j]
		}
		msk[j] = *groupsig.AggregateSeckeys(shareVec)
	}

	msg := []byte("this is test message")
	sigs := make([]groupsig.Signature, n)
	for i := 0; i < n; i++ {
		sigs[i] = groupsig.Sign(msk[i], msg)
	}

	gpk := groupsig.AggregatePubkeys(pks)
	for m := k; m <= n; m++ {
		sigVec := make([]groupsig.Signature, m)
		idVec := make([]groupsig.ID, m)

		for i := 0; i < m; i++ {
			sigVec[i] = sigs[i]
			idVec[i] = ids[i]
		}
		gsig := groupsig.RecoverSignature(sigVec, idVec)

		fmt.Printf("m = %v, sig = %v\n", m, gsig.Serialize())

		if !groupsig.VerifySig(*gpk, msg, *gsig) {
			fmt.Printf("fail to VerifySig when m= %v \n", m)
		}
	}
	fmt.Printf("TestSharePiecesCryptogram end \n")
}

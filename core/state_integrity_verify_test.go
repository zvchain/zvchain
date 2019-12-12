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

package core

import (
	"fmt"
	"github.com/zvchain/zvchain/storage/account"
	"os"
	"testing"
)

func TestFullBlockChain_IntegrityVerify(t *testing.T) {
	wch := make(chan string)
	defer close(wch)

	f, err := os.OpenFile("account_data_1", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	go func() {
		for {
			select {
			case s := <-wch:
				_, err := f.WriteString(s + "\n")
				if err != nil {
					fmt.Println("write---------------- ", err)
				}
			}
		}
	}()
	cb := func(stat *account.VerifyStat) {
		wch <- stat.String()
	}

	chain, _ := newBlockChainByDB("/Volumes/NORELSYS/d_b_175w")
	if chain == nil {
		return
	}
	top := chain.Height()
	fmt.Println(top)
	ok, err := chain.IntegrityVerify(top, cb)
	if !ok {
		t.Errorf("verify fail %v", err)
	}

}

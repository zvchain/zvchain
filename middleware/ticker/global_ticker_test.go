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

package ticker

import (
	"fmt"
	"sync"
	"testing"
)

func TestGlobalTicker_RegisterRoutine(t *testing.T) {

	ticker := NewGlobalTicker("test")

	wg := sync.WaitGroup{}
	wg.Add(1)
	var exeNum = 0
	ticker.RegisterPeriodicRoutine("name1", func() bool {
		if exeNum >= 3 {
			go func() {
				defer wg.Done()
			}()
		}
		fmt.Println("execute task...")
		exeNum++
		return true
	}, uint32(2))
	ticker.StartAndTriggerRoutine("name1")
	//ticker.StopTickerRoutine("name3")
	wg.Wait()
}

func TestGlobalTicker_RegisterOneTimeRoutine(t *testing.T) {
	ticker := NewGlobalTicker("test")

	wg := sync.WaitGroup{}
	wg.Add(1)
	ticker.RegisterOneTimeRoutine("onetime_1", func() bool {
		fmt.Println("onetime...")
		go func() {
			defer wg.Done()
		}()
		return true
	}, 2)

	wg.Wait()
}

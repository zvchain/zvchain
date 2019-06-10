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

package net

import (
	"log"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/taslog"
)

type testMachineGenerator struct {
}

func (t *testMachineGenerator) Generate(id string, cnt int) *stateMachine {
	machine := newStateMachine(id)
	machine.appendNode(newStateNode(1, 1, 1, func(msg interface{}) {
		log.Println(1, msg)
	}))
	machine.appendNode(newStateNode(2, 4, 4, func(msg interface{}) {
		log.Println(2, msg)
	}))
	machine.appendNode(newStateNode(3, 1, 4, func(msg interface{}) {
		log.Println(3, msg)
	}))
	machine.appendNode(newStateNode(4, 1, 1, func(msg interface{}) {
		log.Println(4, msg)
	}))
	return machine
}

func TestStateMachines_GetMachine(t *testing.T) {
	logger = taslog.GetLoggerByName("test_machine.log")
	cache := common.MustNewLRUCache(2)
	testMachines := stateMachines{
		name:      "GroupOutsideMachines",
		generator: &testMachineGenerator{},
		machines:  cache,
	}

	machine := testMachines.GetMachine("abc", 4)
	if machine == nil {
		t.Fatal("create machine fail")
	}
	if machine.ID != "abc" {
		t.Errorf("machine id error")
	}

	taslog.Close()
}

func TestStateMachine_Transform(t *testing.T) {
	logger = taslog.GetLoggerByName("test_machine.log")
	cache := common.MustNewLRUCache(2)
	testMachines := stateMachines{
		name:      "GroupOutsideMachines",
		generator: &testMachineGenerator{},
		machines:  cache,
	}

	machine := testMachines.GetMachine("abc", 4)
	if machine == nil {
		t.Fatal("create machine fail")
	}
	if machine.ID != "abc" {
		t.Errorf("machine id error")
	}
	machine.transform(newStateMsg(2, "sharepiece 1", "u1"))
	machine.transform(newStateMsg(2, "sharepiece 2", "u2"))
	machine.transform(newStateMsg(2, "sharepiece 4", "u4"))
	machine.transform(newStateMsg(4, "done 1", "u1"))
	machine.transform(newStateMsg(2, "sharepiece 5", "u5"))
	machine.transform(newStateMsg(2, "sharepiece 3", "u3"))
	machine.transform(newStateMsg(3, "pub 4", "u4"))
	machine.transform(newStateMsg(3, "pub 3", "u3"))
	machine.transform(newStateMsg(1, "init 2", "u4"))
	machine.transform(newStateMsg(3, "pub 2", "u2"))
	machine.transform(newStateMsg(1, "init 1", "u2"))
	//machine.transform(newStateMsg(3, "pub 1", "u1"))

	if !machine.finish() {
		t.Errorf("machine should be finished")
	}
	if machine.allFinished() {
		t.Errorf("machine not all finished yet")
	}

	taslog.Close()
}

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

package common

import (
	"fmt"
	"testing"
)

var (
	PATH = "tas_test.ini"
	cm   = NewConfINIManager(PATH)
)

func TestConfFileManager_Get_SetString(t *testing.T) {
	cm.SetString("test", "hello", "world")
	s := cm.GetString("test", "hello", "")
	if s != "world" {
		t.Error("get value error")
	}
}

func TestConfFileManager_Get_SetBool(t *testing.T) {
	cm.SetBool("test", "hello", true)
	s := cm.GetBool("test", "hello", false)
	if s != true {
		t.Error("get value error")
	}
}

func TestConfFileManager_Get_SetDouble(t *testing.T) {
	cm.SetDouble("test", "hello", 1.1)
	s := cm.GetDouble("test", "hello", 0)
	if s != 1.1 {
		t.Error("get value error")
	}
}

func TestConfFileManager_Get_SetInt(t *testing.T) {
	cm.SetInt("test", "hello", 1)
	s := cm.GetInt("test", "hello", 0)
	if s != 1 {
		t.Error("get value error")
	}
}

func TestConfFileManager_Del(t *testing.T) {
	cm.SetString("test", "hello", "world")

	cm.Del("test", "hello")
	s := cm.GetString("test", "hello", "")
	if s != "" {
		t.Error("del key fail")
	}
}

func TestSectionConfFileManager(t *testing.T) {
	sm := cm.GetSectionManager("test")
	sm.SetInt("hello_int", 1)
	v := sm.GetInt("hello_int", 0)
	if v != 1 {
		t.Error("set int fail")
	}

	sm.SetDouble("hello_double", 2.2)
	d := sm.GetDouble("hello_double", 0)
	if d != 2.2 {
		t.Error("set double error")
	}
	sm.SetBool("hello_bool", true)
	b := sm.GetBool("hello_bool", false)
	if b != true {
		t.Error("set bool error")
	}

	sm.SetString("hello_string", "abc")
	s := sm.GetString("hello_string", "")
	if s != "abc" {
		t.Error("set string error")
	}

	sm.SetString("hello_del", "del value")
	sm.Del("hello_del")
	s = sm.GetString("hello_del", "")
	if s != "" {
		t.Error("del value error")
	}
}

func testfunc() {
	if true {
		fmt.Println("normal execute...")
		return
	}
	defer func() {
		fmt.Println("defer executed")
	}()
}

func TestDefer(t *testing.T) {
	testfunc()
}

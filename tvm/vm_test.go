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

package tvm

import (
	"testing"
)

func TestVmTest(t *testing.T) {
	//db, _ := tasdb.NewMemDatabase()
	//statedb, _ := core.NewAccountDB(common.Hash{}, core.NewDatabase(db))

	contract := &Contract{ContractName: "test"}
	vm := NewTVM(nil, contract, "")
	vm.SetGas(9999999999999999)
	vm.ContractName = "test"
	script := `
a = 1.2
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
	script = `
eval("a = 10")
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
	script = `
exec("a = 10")
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
	script = `
with open("a.txt", "w") as f:
	f.write("a")
`
	if result := vm.executeScriptKindEval(script); result.ResultType != 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		t.Error("wanted false, got true")
	}
}

func BenchmarkAdd(b *testing.B) {
	vm := NewTVM(nil, nil, "")
	vm.SetGas(9999999999999999)
	script := `
a = 1
`
	vm.ExecuteScriptVMSucceed(script)
	script = `
a += 1
`
	for i := 0; i < b.N; i++ { //use b.N for looping
		vm.ExecuteScriptVMSucceed(script)
	}
}

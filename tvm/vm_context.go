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

var controller *Controller // vm controller

// MaxDepth max depth of running stack
const MaxDepth int = 8

// StoreVMContext Store VM Context
func (con *Controller) StoreVMContext(newTvm *TVM) bool {
	if len(con.VMStack) >= MaxDepth {
		return false
	}
	con.VM.createContext()
	currentVM := con.VM
	con.VMStack = append(con.VMStack, currentVM)
	con.VM = newTvm
	return true
}

// RecoverVMContext Recover VM Context
func (con *Controller) RecoverVMContext() {
	logs := con.VM.Logs
	con.VM = con.VMStack[len(con.VMStack)-1]
	con.VM.Logs = logs
	con.VMStack = con.VMStack[:len(con.VMStack)-1]
	con.VM.removeContext()
}

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

package network

import (
	"C"
	"unsafe"
)

//export OnP2PRecved
func OnP2PRecved(id uint64, session uint32, data []byte) {
	netCore.onRecved(id, session, data)
}

//export OnP2PChecked
func OnP2PChecked(p2pType uint32, privateIP string, publicIP string) {
	netCore.onChecked(p2pType, privateIP, publicIP)
}

//export OnP2PListened
func OnP2PListened(ip string, port uint16, latency uint64) {
}

//export OnP2PAccepted
func OnP2PAccepted(id uint64, session uint32, p2pType uint32, ip string, port uint16) {
	netCore.onAccepted(id, session, p2pType, ip, port)
}

//export OnP2PConnected
func OnP2PConnected(id uint64, session uint32, p2pType uint32) {
	netCore.onConnected(id, session, p2pType)
}

//export OnP2PDisconnected
func OnP2PDisconnected(id uint64, session uint32, p2pCode uint32) {
	netCore.onDisconnected(id, session, p2pCode)
}

//export OnP2PSendWaited
func OnP2PSendWaited(session uint32, peerID uint64) {
	netCore.onSendWaited(peerID, session)
}

//export OnP2PLoginSign
func OnP2PLoginSign() unsafe.Pointer {
	return P2PLoginSign()
}

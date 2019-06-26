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

/*
#cgo LDFLAGS: -L ./ -lp2pcore -lstdc++
#cgo windows LDFLAGS: -L -lwsock32 -lws2_32

#include "p2p_api.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

void OnP2PRecved();

void OnP2PListened();

void OnP2PChecked();

void OnP2PAccepted();

void OnP2PConnected();

void OnP2PDisconnected();

void OnP2PSendWaited();

void* OnP2PLoginSign();

void on_p2p_recved(uint64_t id, uint32_t session, char* data, uint32_t size)
{
	_GoBytes_ _data = {data, size, size};
	OnP2PRecved(id, session, _data);
}

void on_p2p_checked(uint32_t type, const char* private_ip, const char* public_ip)
{
    _GoString_ _private_ip = {private_ip, strlen(private_ip)};
    _GoString_ _public_ip = {public_ip, strlen(public_ip)};
    OnP2PChecked(type, _private_ip, _public_ip);
}

void on_p2p_listened(const char* ip, uint16_t port, uint64_t latency)
{
	_GoString_ _ip = {ip, strlen(ip)};
	OnP2PListened(_ip, port, latency);
}

void on_p2p_accepted(uint64_t id, uint32_t session, uint32_t type)
{
	OnP2PAccepted(id, session, type);
}

void on_p2p_connected(uint64_t id, uint32_t session, uint32_t type)
{
	OnP2PConnected(id, session, type);
}

void on_p2p_disconnected(uint64_t id, uint32_t session, uint32_t code)
{
	OnP2PDisconnected(id, session, code);
}

void on_p2p_send_waited(uint32_t session, uint64_t peer_id)
{
	OnP2PSendWaited(session, peer_id);
}

struct p2p_login* on_p2p_login_sign()
{
	return (struct p2p_login*)OnP2PLoginSign();
}

void wrap_p2p_config(uint64_t id)
{
	struct p2p_callback callback = { 0 };
	callback.recved = on_p2p_recved;
	callback.checked = on_p2p_checked;
	callback.listened = on_p2p_listened;
	callback.accepted = on_p2p_accepted;
	callback.connected = on_p2p_connected;
	callback.disconnected = on_p2p_disconnected;
	callback.sign = on_p2p_login_sign;
	p2p_config(id, callback);

}

void wrap_p2p_send_callback()
{
	p2p_send_callback(on_p2p_send_waited);
}

struct p2p_login * wrap_new_p2p_login(uint64_t id, uint64_t cur_time, char* pk, char* sign)
{
	struct p2p_login *p_login = (struct p2p_login *)malloc(sizeof(struct p2p_login));
	p_login->id = id;
	p_login->cur_time = cur_time;
	memcpy(p_login->pk, pk, PK_SIZE);
	memcpy(p_login->sign, sign, SIGN_SIZE);
	return p_login;
}

*/
import "C"
import (
	"unsafe"
)


func P2PConfig(id uint64) {
	C.wrap_p2p_config(C.uint64_t(id))
	C.wrap_p2p_send_callback()
}

func P2PProxy(ip string, port uint16) {
	C.p2p_proxy(C.CString(ip), C.ushort(port))
}

func P2PListen(ip string, port uint16) {
	C.p2p_listen(C.CString(ip), C.ushort(port))
}

func P2PClose() {
	C.p2p_close()
}

func P2PConnect(id uint64, ip string, port uint16) {
	C.p2p_connect(C.uint64_t(id), C.CString(ip), C.ushort(port))
}

func P2PShutdown(session uint32) {
	C.p2p_shutdown(C.uint(session))
}

func P2PSessionRtt(session uint32) uint32 {
	r := C.p2p_kcp_rxrtt(C.uint32_t(session))
	return uint32(r)
}

func P2PSessionSendBufferCount(session uint32) uint32 {
	r := C.p2p_kcp_nsndbuf(C.uint(session))
	return uint32(r)
}

func P2PCacheSize() uint64 {
	r := C.p2p_cache_size()
	return uint64(r)
}

func P2PSend(session uint32, data []byte) {

	pendingSendBuffer := P2PSessionSendBufferCount(session)
	const maxSendBuffer = 10240

	if pendingSendBuffer > maxSendBuffer {
		Logger.Debugf("session kcp send queue over 10240 drop this message,session id:%v pendingSendBuffer:%v ", session, pendingSendBuffer)
		return
	}

	const maxSize = 64 * 1024
	totalLen := len(data)

	curPos := 0
	for curPos < totalLen {
		sendSize := totalLen - curPos
		if sendSize > maxSize {
			sendSize = maxSize
		}
		C.p2p_send(C.uint(session), unsafe.Pointer(&data[curPos]), C.uint(sendSize))
		curPos += sendSize
	}

}

func P2PLoginSign() unsafe.Pointer {
	
	pa := genPeerAuthContext(netServerInstance.config.PK,netServerInstance.config.SK,nil)
	
	return (unsafe.Pointer)(C.wrap_new_p2p_login(C.uint64_t(netServerInstance.netCore.netID), C.uint64_t(pa.CurTime), (*C.char)(unsafe.Pointer(&pa.PK[0])), (*C.char)(unsafe.Pointer(&pa.Sign[0]))))
}
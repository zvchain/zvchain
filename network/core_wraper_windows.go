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

package network

/*
#include <Windows.h>
#include <stdio.h>
void* p2p_api(const char* api)
{
    static HMODULE p2p_core = 0;
    if (p2p_core == 0)
    {
        p2p_core = LoadLibrary("p2p_core.dll");
        if (p2p_core == 0){
        	printf("p2p_core load lib failed !\n");
        }
    }
    return (void*)GetProcAddress(p2p_core, api);
}

#include <stdint.h>
#include <string.h>

extern void OnP2PRecved(uint64_t id, uint32_t session, _GoBytes_ data);
extern void OnP2PChecked(uint32_t type, _GoString_ private_ip, _GoString_ public_ip);
extern void OnP2PListened(_GoString_ ip, uint16_t port, uint64_t latency);
extern void OnP2PAccepted(uint64_t id, uint32_t session, uint32_t type, _GoString_ ip, uint16_t port);
extern void OnP2PConnected(uint64_t id, uint32_t session, uint32_t type);
extern void OnP2PDisconnected(uint64_t id, uint32_t session, uint32_t code);
extern void OnP2PSendWaited(uint32_t session, uint64_t peer_id);
extern void *OnP2PLoginSign();

typedef void(*p2p_recved)(uint64_t id, uint32_t session, char* data, uint32_t size);
typedef void(*p2p_checked)(uint32_t type, const char* private_ip, const char* public_ip);
typedef void(*p2p_listened)(const char* ip, uint16_t port, uint64_t latency);
typedef void(*p2p_accepted)(uint64_t id, uint32_t session, uint32_t type, const char* ip, uint16_t port);
typedef void(*p2p_connected)(uint64_t id, uint32_t session, uint32_t type);
typedef void(*p2p_disconnected)(uint64_t id, uint32_t session, uint32_t code);
typedef void(*p2p_send_waited)(uint32_t session, uint64_t peer_id);
typedef struct p2p_login*(*p2p_login_sign)();

#define PK_SIZE  65
#define SIGN_SIZE  65

#pragma pack(push, 1)

struct p2p_login
{
    uint64_t    id;
    uint64_t    cur_time;
    char        pk[PK_SIZE];
    char        sign[SIGN_SIZE];
};

#pragma pack(pop)
struct p2p_callback
{
    p2p_recved       recved;
    p2p_checked      checked;
    p2p_listened     listened;
    p2p_accepted     accepted;
    p2p_connected    connected;
	p2p_disconnected disconnected;
	p2p_login_sign   sign;
};

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

void on_p2p_accepted(uint64_t id, uint32_t session, uint32_t type, const char* ip, uint16_t port)
{
  	_GoString_ _ip = {ip, strlen(ip)};

	OnP2PAccepted(id, session, type, _ip, port);
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

void p2p_config(uint64_t id)
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
        struct p2p_callback callback = { 0 };
   		callback.recved = on_p2p_recved;
        callback.checked = on_p2p_checked;
        callback.listened = on_p2p_listened;
        callback.accepted = on_p2p_accepted;
        callback.connected = on_p2p_connected;
        callback.disconnected = on_p2p_disconnected;
      	callback.sign = on_p2p_login_sign;
	   ((void(*)(uint64_t id, struct p2p_callback callback))api)(id, callback);
    }
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


void p2p_send_callback()
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
        ((void(*)(p2p_send_waited callback))api)(on_p2p_send_waited);
    }
}

void p2p_proxy(const char* ip, uint16_t port)
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
    	((void(*)(const char* ip, uint16_t port))api)(ip, port);
    }
}

void p2p_listen(const char* ip, uint16_t port)
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
    	((void(*)(const char* ip, uint16_t port))api)(ip, port);
    }
}

void p2p_close()
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
    	((void(*)())api)();
	}
}

void p2p_connect(uint64_t id, const char* ip, uint16_t port)
{
	void* api = p2p_api(__FUNCTION__);
	if (api)
    {
    	((void(*)(uint64_t id, const char* ip, uint16_t port))api)(id, ip, port);
	}
}

void p2p_shutdown(uint32_t session)
{
	void* api = p2p_api(__FUNCTION__);
	if (api)
    {
    	((void(*)(uint32_t session))api)(session);
	}
}

void p2p_send(uint32_t session, const void* data, uint32_t size)
{
	void* api = p2p_api(__FUNCTION__);
	if (api)
    {
    	((void(*)(uint32_t session, const void* data, uint32_t size))api)(session, data, size);
	}
}

*/
import "C"
import (
	"unsafe"
)

func P2PConfig(id uint64) {

	C.p2p_config(C.ulonglong(id))

	C.p2p_send_callback()
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
	C.p2p_connect(C.ulonglong(id), C.CString(ip), C.ushort(port))
}

func P2PShutdown(session uint32) {
	C.p2p_shutdown(C.uint(session))
}

func P2PSend(session uint32, data []byte) {

	maxSize := 64 * 1024
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

	pa := genPeerAuthContext(netServerInstance.config.PK, netServerInstance.config.SK, nil)

	return (unsafe.Pointer)(C.wrap_new_p2p_login(C.uint64_t(netServerInstance.netCore.netID), C.uint64_t(pa.CurTime), (*C.char)(unsafe.Pointer(&pa.PK[0])), (*C.char)(unsafe.Pointer(&pa.Sign[0]))))
}

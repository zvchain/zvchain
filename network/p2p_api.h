//   Copyright (C) 2018 TASChain
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

#pragma once
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

const uint32_t p2p_type_unknown = 0;

const uint32_t p2p_type_full = 1;

const uint32_t p2p_type_host = 2;

const uint32_t p2p_type_fixed = 3;

const uint32_t p2p_type_symmetric = 4;

const uint32_t p2p_type_multiip = 5;


const uint32_t p2p_code_connect_error = 0;

const uint32_t p2p_code_connect_timeout = 1;

const uint32_t p2p_code_disconnect_active = 2;

const uint32_t p2p_code_disconnect_passive = 3;

const uint32_t p2p_code_disconnect_timeout = 4;

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

typedef void(*p2p_recved)(uint64_t id, uint32_t session, char* data, uint32_t size);

typedef void(*p2p_checked)(uint32_t type, const char* host_ip, const char* napt_ip);

typedef void(*p2p_listened)(const char* ip, uint16_t port, uint64_t latency);

typedef void(*p2p_accepted)(uint64_t id, uint32_t session, uint32_t type);

typedef void(*p2p_connected)(uint64_t id, uint32_t session, uint32_t type);

typedef void(*p2p_disconnected)(uint64_t id, uint32_t session, uint32_t code);

typedef void(*p2p_send_waited)(uint32_t session, uint64_t peer_id);

typedef struct p2p_login*(*p2p_login_sign)();

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

extern  void p2p_config(uint64_t id, struct p2p_callback callback);

extern  void p2p_proxy(const char* ip, uint16_t port);

extern  void p2p_listen(const char* ip, uint16_t port);

extern  void p2p_close();

extern  void p2p_connect(uint64_t id, const char* ip, uint16_t port);

extern  void p2p_shutdown(uint32_t session);

extern  void p2p_send(uint32_t session, const void* data, uint32_t size);

extern  void p2p_send_callback(p2p_send_waited callback);

extern  uint32_t p2p_kcp_snd_nxt(uint32_t session);

extern  uint32_t p2p_kcp_rcv_nxt(uint32_t session);

extern  uint32_t p2p_kcp_rxrtt(uint32_t session);

extern  uint32_t p2p_kcp_nsndbuf(uint32_t session);

extern  uint32_t p2p_kcp_nrcvbuf(uint32_t session);

extern  uint64_t p2p_cache_size();

#ifdef __cplusplus
}
#endif
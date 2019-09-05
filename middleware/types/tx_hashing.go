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

package types

import (
	"bytes"
	"github.com/zvchain/zvchain/common"
)

type bufferWriter struct {
	buf bytes.Buffer
}

func (bw *bufferWriter) writeByte(b byte) {
	bw.buf.WriteByte(b)
}

func (bw *bufferWriter) writeBytes(b []byte) {
	// Write len with big-endian
	bw.buf.Write(common.Int32ToByte(int32(len(b))))
	if len(b) > 0 {
		bw.buf.Write(b)
	}
}

func (bw *bufferWriter) Bytes() []byte {
	return bw.buf.Bytes()
}

type txHashing struct {
	src      []byte
	target   []byte
	value    []byte // bytes with big-endian
	gasLimit []byte // bytes with big-endian
	gasPrice []byte // bytes with big-endian
	nonce    []byte // bytes with big-endian
	typ      byte
	data     []byte
	extra    []byte
}

func (th *txHashing) genHash() common.Hash {
	buf := &bufferWriter{}
	buf.writeBytes(th.src)
	buf.writeBytes(th.target)
	buf.writeBytes(th.value)
	buf.writeBytes(th.gasLimit)
	buf.writeBytes(th.gasPrice)
	buf.writeBytes(th.nonce)
	buf.writeByte(th.typ)
	buf.writeBytes(th.data)
	buf.writeBytes(th.extra)
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

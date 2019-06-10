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

/*
	Package serialize is used gob to serialize object
*/
package serialize

import (
	"bytes"
	"github.com/vmihailenco/msgpack"
	"io"
)

func Decode(r io.Reader, val interface{}) error {
	decoder := msgpack.NewDecoder(r)
	if err := decoder.Decode(val); err != nil {
		return err
	}
	return nil
}

func DecodeBytes(b []byte, val interface{}) error {
	return Decode(bytes.NewBuffer(b), val)
}

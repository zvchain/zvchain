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

package tasdb

import "github.com/syndtr/goleveldb/leveldb/opt"

type TasDataSource struct {
	db *LDBDatabase
}

// NewDataSource create levedb instance by file
func NewDataSource(file string, options *opt.Options) (*TasDataSource, error) {
	db, err := getInstance(file, options)
	if err != nil {
		return nil, err
	}
	return &TasDataSource{db: db}, nil
}

// NewPrefixDatabase create logical database by prefix
func (ds *TasDataSource) NewPrefixDatabase(prefix string) (*PrefixedDatabase, error) {
	return &PrefixedDatabase{
		db:     ds.db,
		prefix: prefix,
	}, nil
}

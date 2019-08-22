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

package main

import (
	"flag"
	"fmt"
	"github.com/zvchain/zvchain/browser"
	"runtime/debug"

	"github.com/zvchain/zvchain/cmd/gzv/cli"
)

func main() {
	debug.SetTraceback("all")

	var dbAddr, rpcAddr string
	var dbPort, rpcPort int
	var dbUser, dbPassword string
	var help bool
	var reset bool

	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&reset, "reset", false, "reset database")
	flag.StringVar(&dbAddr, "dbaddr", "10.0.0.13", "database address")
	flag.IntVar(&dbPort, "dbport", 3306, "database port")
	flag.StringVar(&dbUser, "dbuser", "root", "database user")
	flag.StringVar(&dbPassword, "dbpw", "root123", "database password")
	flag.Parse()

	if help {
		flag.Usage()
	}
	fmt.Println("flags:", dbAddr, dbPort, dbUser, dbPassword, rpcAddr, rpcPort, reset)
	browser.NewDBMmanagement(dbAddr, dbPort, dbUser, dbPassword, reset)

	gtas := cli.NewGtas()
	gtas.Run()
}

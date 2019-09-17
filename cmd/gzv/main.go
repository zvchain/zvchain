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
	"github.com/zvchain/zvchain/browser/crontab"
	"runtime/debug"

	"github.com/zvchain/zvchain/cmd/gzv/cli"
)

func main() {
	debug.SetTraceback("all")
	gzv := cli.NewGzv()
	go func() {
		init := <-gzv.InitCha
		if init {
			NewBrowserDBInit()
		}
	}()
	gzv.Run()
}

func NewBrowserDBInit() {
	var browerdbaddr, rpcAddr string
	var dbPort, rpcPort int
	var dbUser, dbPassword string
	var help bool
	var reset bool
	var resetcrontab bool

	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&reset, "reset", false, "reset database")
	flag.BoolVar(&resetcrontab, "resetcrontab", false, "resetcrontab database")
	flag.StringVar(&browerdbaddr, "browerdbaddr", "localhost", "database address")
	flag.StringVar(&rpcAddr, "rpcaddr", "localhost", "RPC address")
	flag.IntVar(&dbPort, "dbport", 3306, "database port")
	flag.IntVar(&rpcPort, "rpcport", 8101, "RPC port")
	flag.StringVar(&dbUser, "dbuser", "user", "database user")
	flag.StringVar(&dbPassword, "browerdbpw", "password", "database password")
	flag.Parse()

	if help {
		flag.Usage()
	}
	fmt.Println("browserdbmmanagement flags:", browerdbaddr, dbPort, dbUser, dbPassword, reset)
	browser.NewDBMmanagement(browerdbaddr, dbPort, dbUser, dbPassword, reset, resetcrontab)
	crontab.NewServer(browerdbaddr, dbPort, dbUser, dbPassword, reset)
}

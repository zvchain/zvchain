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
	"github.com/zvchain/zvchain/browser/crontab"
	"github.com/zvchain/zvchain/browser/ldb"
	browserlog "github.com/zvchain/zvchain/browser/log"
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
	flag.StringVar(&browerdbaddr, "browerdbaddr", "114.55.66.243", "database address")
	flag.StringVar(&rpcAddr, "rpcaddr", "localhost", "RPC address")
	flag.IntVar(&dbPort, "dbport", 3306, "database port")
	flag.IntVar(&rpcPort, "rpcport", 8101, "RPC port")
	flag.StringVar(&dbUser, "dbuser", "root", "database user")
	flag.StringVar(&dbPassword, "browerdbpw", "TASchain@1003", "database password")

	/*
		// for local test
		flag.StringVar(&browerdbaddr, "browerdbaddr", "10.0.0.13", "database address")
		flag.StringVar(&rpcAddr, "rpcaddr", "localhost", "RPC address")
		flag.StringVar(&dbUser, "dbuser", "root", "database user")
		flag.StringVar(&dbPassword, "browerdbpw", "root123", "database password")
	*/

	flag.Parse()

	if help {
		flag.Usage()
	}
	browserlog.InitLog()
	ldb.InitBrowserdb()
	fmt.Println("browserdbmmanagement flags:", browerdbaddr, dbPort, dbUser, dbPassword, reset)
	//browser.NewDBMmanagement(browerdbaddr, dbPort, dbUser, dbPassword, reset, resetcrontab)
	crontab.NewServer(browerdbaddr, dbPort, dbUser, dbPassword, reset)
}

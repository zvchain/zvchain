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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("chat", "A command-line chat application.")

	deployContract = app.Command("deploy", "deploy contract.")
	contractName   = deployContract.Arg("name", "").Required().String()
	contractPath   = deployContract.Arg("path", "").Required().String()

	callContract    = app.Command("call", "call contract.")
	contractAddress = callContract.Arg("contractAddress", "contract address.").Required().String()
	contractAbi     = callContract.Arg("abiPath", "").Required().String()

	exportAbi             = app.Command("export", "export abi.")
	exportAbiContractName = exportAbi.Arg("name", "").Required().String()
	exportAbiContractPath = exportAbi.Arg("path", "").Required().String()

	queryData    = app.Command("query", "query account data.")
	queryAddress = queryData.Arg("account address", "account address.").Required().String()
	queryKey     = queryData.Arg("query key", "").Required().String()
	queryCount   = queryData.Arg("query count", "if count > 0, key is prefix of query db.").Required().Int()
)

func main() {

	tvmCli := NewTvmCli()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	// deploy Token ./cli/erc20.py
	case deployContract.FullCommand():
		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *contractPath) //读取文件
		if err != nil {
			fmt.Println("read the ", *contractPath, " file failed ", err)
			return
		}
		tvmCli.Deploy(*contractName, string(f))

	// call ./cli/call_Token_abi.json
	case callContract.FullCommand():
		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *contractAbi) //读取文件
		if err != nil {
			fmt.Println("read the ", *contractAbi, " file failed ", err)
			return
		}
		tvmCli.Call(*contractAddress, string(f))

	// export ./cli/erc20.py
	case exportAbi.FullCommand():
		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *exportAbiContractPath) //读取文件
		if err != nil {
			fmt.Println("read the ", *exportAbiContractPath, " file failed ", err)
			return
		}
		tvmCli.ExportAbi(*exportAbiContractName, string(f))

	case queryData.FullCommand():
		tvmCli.QueryData(*queryAddress, *queryKey, *queryCount)

	}
}

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
	"bufio"
	"fmt"
	"io"
	"os"
	"testing"
)

func _deployContract(contractName string, filePath string) string {
	tvmCli := NewTvmCli()
	f, err := os.Open(filePath) //读取文件
	if err != nil {
		panic("")
	}
	defer f.Close()
	codeStr := ""
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		codeStr = fmt.Sprintf("%s%s \n", codeStr, line)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic("")
			}
		}
	}
	contractAddress := tvmCli.Deploy(contractName, codeStr)
	tvmCli.DeleteTvmCli()
	return contractAddress
}

func TestTvmCallContract(t *testing.T) {
	contractAddress := _deployContract("Token", "erc20.py")

	tvmCli := NewTvmCli()
	abiJson := `{
	"FuncName": "balance_of",
		"Args": ["0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`
	tvmCli.Call(contractAddress, abiJson)
	tvmCli.DeleteTvmCli()
}

func TestTvmContractCallContract(t *testing.T) {
	erc20Contract := _deployContract("Token", "erc20.py")
	routerContract := _deployContract("Router", "router.py")

	tvmCli := NewTvmCli()
	abiJson := fmt.Sprintf(`{
  "FuncName": "call_contract",
  "Args": ["%s","balance_of","0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`, erc20Contract)
	tvmCli.Call(routerContract, abiJson)
	tvmCli.DeleteTvmCli()
}

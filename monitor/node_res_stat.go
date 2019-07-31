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

package monitor

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/codeskyblue/go-sh"
)

var spaceRe, _ = regexp.Compile("\\s+")

const (
	NtypeVerifier = 1
	NtypeProposal = 2
)

type NodeInfo struct {
	Type         int
	Instance     int
	VrfThreshold float64
	PStake       uint64
	BlockHeight  uint64
	GroupHeight  uint64
	TxPoolCount  int
}

type NodeResStat struct {
	CPU    float64
	Mem    float64
	RcvBps float64
	TxBps  float64

	cmTicker   *time.Ticker
	flowTicker *time.Ticker
}

func initNodeResStat() *NodeResStat {
	ns := &NodeResStat{
		cmTicker:   time.NewTicker(time.Second * 3),
		flowTicker: time.NewTicker(time.Second * 6),
	}
	go ns.startStatLoop()
	return ns
}

func (ns *NodeResStat) startStatLoop() {
	for {
		select {
		case <-ns.cmTicker.C:
			ns.statCPUAndMEM()
		case <-ns.flowTicker.C:
			ns.statFlow()
		}
	}
}

func (ns *NodeResStat) statCPUAndMEM() {
	sess := sh.NewSession()
	bs, err := sess.Command("top", "-b", "-n 1", fmt.Sprintf("-p %v", os.Getpid())).Command("grep", "gzv").Output()

	if err == nil {
		line := spaceRe.ReplaceAllString(strings.TrimSpace(string(bs)), ",")
		arrs := strings.Split(line, ",")
		if len(arrs) < 10 {
			return
		}
		var cpu, mem float64
		cpu, _ = strconv.ParseFloat(arrs[8], 64)
		mems := arrs[5]
		if mems[len(mems)-1:] == "g" {
			f, _ := strconv.ParseFloat(mems[:len(mems)-1], 64)
			mem = f * 1000
		} else if mems[len(mems)-1:] == "m" {
			f, _ := strconv.ParseFloat(mems[:len(mems)-1], 64)
			mem = f
		} else {
			f, _ := strconv.ParseFloat(mems, 64)
			mem = f / 1000
		}
		ns.CPU = cpu
		ns.Mem = mem
	} else {

	}
	return
}

func (ns *NodeResStat) statFlow() {
	sess := sh.NewSession()
	bs, err := sess.Command("sar", "-n", "DEV", "1", "2").Command("grep", "eth").CombinedOutput()

	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(bs)), "\n")
		if len(lines) < 1 {
			return
		}
		line := spaceRe.ReplaceAllString(lines[len(lines)-1], ",")
		arrs := strings.Split(line, ",")
		if len(arrs) < 8 {
			return
		}
		ns.RcvBps, _ = strconv.ParseFloat(arrs[4], 64)
		ns.TxBps, _ = strconv.ParseFloat(arrs[5], 64)
	} else {
	}
	return
}

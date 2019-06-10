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

package statistics

import (
	"sync"
	"time"

	"bytes"
	"encoding/json"
	"fmt"

	"github.com/zvchain/zvchain/common"
)

const (
	KingCasting   = 1
	MessageCast   = 2
	MessageVerify = 3
	NewBlock      = 4
)

const (
	RcvNewBlock  = "RcvNewBlock"
	SendCast     = "SendCast"
	RcvCast      = "RcvCast"
	SendVerified = "SendVerified"
	RcvVerified  = "RcvVerified"
	BroadBlock   = "BroadBlock"
)

var BatchSize = 1000
var Duration time.Duration = 5
var Lock sync.RWMutex
var Lock2 sync.RWMutex
var LogChannel = make(chan *LogObj)
var BlockLogChannel = make(chan *BlockLogObject)
var TimeChannel = make(chan int)
var IsInit = false
var WriteData = make([]*LogObj, 0)
var WriteData2 = make([]*BlockLogObject, 0)
var batch int
var enable = false

type LogObj struct {
	Hash   string
	Status int
	Time   int64
	Batch  int
	Castor string
	Node   string
}

type BlockLogObject struct {
	Code          string
	CodeNum       uint8
	BlockHeight   uint64
	Qn            uint64
	TxCount       int
	Size          int
	TimeStamp     int64
	Castor        string
	GroupID       string
	InstanceIndex int
	CastTime      int64
	BootID        int
}

func NewLogObj(id string) *LogObj {
	lg := new(LogObj)
	lg.Node = id
	lg.Hash = "cf545b9496a1665285aa385d9ee5542154f2fb4dcefc820b4ccb00741b88c9ed"
	lg.Castor = "cf545b9496a1665285aa385d"
	lg.Status = 2
	lg.Time = time.Now().Unix()
	lg.Batch = 1
	return lg
}

func AddLog(Hash string, Status int, Time int64, Castor string, Node string) {
	if enable {
		log := &LogObj{Hash: Hash, Status: Status, Time: Time, Batch: batch, Castor: Castor, Node: Node}
		PutLog(log)
	}
}

func AddBlockLog(bootID int, code string, blockHeight uint64, qn uint64, txCount int, size int, timeStamp int64, castor string, groupID string, instanceIndex int, castTime int64) {
	if enable {
		var cn uint8
		switch code {
		case RcvNewBlock:
			cn = 1
		case SendCast:
			cn = 2
		case RcvCast:
			cn = 3
		case SendVerified:
			cn = 4
		case RcvVerified:
			cn = 5
		case BroadBlock:
			cn = 6
		}
		log := &BlockLogObject{Code: code, CodeNum: cn, BlockHeight: blockHeight, Qn: qn, TxCount: txCount, Size: size, TimeStamp: timeStamp, Castor: castor, GroupID: groupID, InstanceIndex: instanceIndex, CastTime: castTime, BootID: bootID}
		PutBlockLog(log)
	}
}

func PutLog(data *LogObj) {
	LogChannel <- data
}

func PutBlockLog(data *BlockLogObject) {
	BlockLogChannel <- data
}

func InitStatistics(config common.ConfManager) {
	url = config.GetString("statistics", "url", "http://118.31.60.210:9090/send")
	timeout = time.Duration(config.GetInt("statistics", "timeout", 1)) * time.Second
	batch = config.GetInt("statistics", "batch", 0)
	enable = config.GetBool("statistics", "enable", false)
	go ProcessLog()
	go func() {
		t := time.Tick(Duration * time.Second)

		for {
			select {
			case <-t:
				TimeChannel <- 1
			}
		}
	}()
	initCount(config)
}

func HasInit() bool {
	if IsInit {
		return true
	}
	Lock.Lock()
	if IsInit {
		return true
	}
	IsInit = true
	defer Lock.Unlock()
	return false
}

func ProcessLog() {
	if enable {
		for {
			select {
			case log := <-LogChannel:
				Lock.Lock()
				WriteData = append(WriteData, log)
				Lock.Unlock()
				if len(WriteData) >= BatchSize {
					Send(1)
				}
			case log := <-BlockLogChannel:
				Lock2.Lock()
				WriteData2 = append(WriteData2, log)
				Lock2.Unlock()
				if len(WriteData2) >= BatchSize {
					Send(2)
				}
			case <-TimeChannel:
				Send(1)
				Send(2)
			}
		}
	}
}

func Send(code int) {
	if l := len(WriteData); code == 1 && l > 0 {
		Lock.Lock()
		tmp := WriteData
		WriteData = WriteData[0:0]
		Lock.Unlock()
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(tmp)
		fmt.Printf("send log batch len:%d\n", l)
		SendPost(b, "log")
	}
	if l := len(WriteData2); code == 2 && l > 0 {
		Lock2.Lock()
		tmp := WriteData2
		WriteData2 = WriteData2[0:0]
		Lock2.Unlock()
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(tmp)
		fmt.Printf("send block batch len:%d\n", l)
		SendPost(b, "block")
	}
}

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
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/taslog"
)

type countItem struct {
	*sync.Map
}

type innerItem struct {
	count uint32
	size  uint64
}

var countMap = new(sync.Map)
var logger taslog.Logger
var VrfLogger taslog.Logger

func newCountItem() *countItem {
	return &countItem{new(sync.Map)}
}

func newInnerItem(size uint64) *innerItem {
	return &innerItem{count: 1, size: size}
}

func (item *countItem) get(code uint32) *innerItem {
	if v, ok2 := item.Load(code); ok2 {
		return v.(*innerItem)
	}
	return nil
}

func (item *innerItem) increase(size uint64) {
	item.count++
	item.size += size
}

func (item *countItem) print() string {
	var buffer bytes.Buffer
	item.Range(func(code, value interface{}) bool {
		buffer.WriteString(fmt.Sprintf(" %d:%d", code, value))
		item.Delete(code)
		return true
	})
	return buffer.String()
}

func printAndRefresh() {
	countMap.Range(func(name, item interface{}) bool {
		citem := item.(*countItem)
		content := citem.print()
		if logger != nil {
			logger.Infof("%s%s\n", name, content)
		} else {
			fmt.Printf("%s%s\n", name, content)
		}
		return true
	})
}

func AddCount(name string, code uint32, size uint64) {
	if item, ok := countMap.Load(name); ok {
		citem := item.(*countItem)
		if item2, ok := countMap.Load(code); ok {
			citem2 := item2.(*innerItem)
			citem2.increase(size)
		} else {
			citem.Store(code, newInnerItem(size))
		}
	} else {
		citem := newCountItem()
		citem.Store(code, newInnerItem(size))
		countMap.Store(name, citem)
	}
	//logger.Infof("%s %d",name,code)
}

func initCount(config common.ConfManager) {
	logger = taslog.GetLoggerByIndex(taslog.StatisticsLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	VrfLogger = taslog.GetLoggerByIndex(taslog.VRFDebugLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	t1 := time.NewTimer(time.Second * 1)
	go func() {
		for {
			select {
			case <-t1.C:
				printAndRefresh()
				t1.Reset(time.Second * 1)
			}
		}
	}()
}

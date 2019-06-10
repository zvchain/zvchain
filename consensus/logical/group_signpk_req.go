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

package logical

import (
	"sync"
	"time"

	"github.com/zvchain/zvchain/consensus/groupsig"
)

type signPKReqRecord struct {
	reqTime time.Time
	reqUID  groupsig.ID
}

func (r *signPKReqRecord) reqTimeout() bool {
	return time.Now().After(r.reqTime.Add(60 * time.Second))
}

//recordMap mapping idHex to signPKReqRecord
var recordMap sync.Map

func addSignPkReq(id groupsig.ID) bool {
	r := &signPKReqRecord{
		reqTime: time.Now(),
		reqUID:  id,
	}
	_, load := recordMap.LoadOrStore(id.GetHexString(), r)
	return !load
}

func removeSignPkRecord(id groupsig.ID) {
	recordMap.Delete(id.GetHexString())
}

func cleanSignPkReqRecord() {
	recordMap.Range(func(key, value interface{}) bool {
		r := value.(*signPKReqRecord)
		if r.reqTimeout() {
			recordMap.Delete(key)
		}
		return true
	})
}

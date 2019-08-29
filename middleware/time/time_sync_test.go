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

package time

import "testing"
import (
	"fmt"
	"time"

	"github.com/beevik/ntp"
	"github.com/zvchain/zvchain/common"
)

/*
**  Creator: pxf
**  Date: 2019/4/10 上午11:16
**  Description:
 */

func TestNTPQuery(t *testing.T) {
	tt, _ := ntp.Time("time.asia.apple.com")
	t.Log(tt, time.Now())

	rsp, err := ntp.Query("cn.pool.ntp.org")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rsp.Time, rsp.ClockOffset.String(), time.Now(), time.Now().Add(rsp.ClockOffset))

	for i := 1; i < 8; i++ {
		t.Log(ntp.Time(fmt.Sprintf("ntp%v.aliyun.com", i)))
	}
}

func TestSync(t *testing.T) {
	InitTimeSync()
	time.Sleep(time.Second * 3)
}

func TestUTCAndLocal(t *testing.T) {
	InitTimeSync()
	now := time.Now()

	utc := time.Now().UTC()
	t.Log(now, utc)

	t.Log(utc.After(now), utc.Local().After(now), utc.Before(now))
}

func TestTimeMarshal(t *testing.T) {
	now := time.Now().UTC()
	bs, _ := now.MarshalBinary()
	t.Log(bs, len(bs))
}

func TestTimeUnixSec(t *testing.T) {
	now := time.Now()
	t.Log(now.Unix(), now.UTC().Unix())

	t.Log(time.Unix(now.Unix(), 0))
	t.Log(time.Unix(int64(common.MaxUint32), 0))
}

func TestTimeStampString(t *testing.T) {
	t.Logf("ts:%v", TimeToTimeStamp(time.Now()))
}

func TestLocal(t *testing.T){
	d := time.Date(2017, 7, 7, 9, 0, 0, 0, time.Local)
	dl := TimeToTimeStamp(d)
	t.Log(d,dl)
	if !dl.Local().Equal(d) {
		t.Error("Local() test fail")
	}

}

//   Copyright (C) 2019 ZVChain
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

package group

type createStat struct {
	eraCnt          int // Era count
	shouldCreateCnt int // Should create group era count
	successCnt      int // Create success count
	failCnt         int
	outCh           chan struct{}
}

func newCreateStat() *createStat {
	return &createStat{
		outCh: make(chan struct{}, 5),
	}
}

func (st *createStat) loop() {
	for {
		select {
		case <-st.outCh:
			st.outLog()
		}
	}
}

func (st *createStat) increaseEra() {
	st.eraCnt++
}
func (st *createStat) increaseFail() {
	st.failCnt++
}

func (st *createStat) increaseShouldCreate() {
	st.shouldCreateCnt++
}

func (st *createStat) increaseSuccess() {
	st.successCnt++
}

func (st *createStat) outLog() {
	if st.shouldCreateCnt == 0 {
		st.shouldCreateCnt = 1
	}
	logger.Debugf("create group stat: eraCnt=%v, startCreate=%v, successCnt=%v, failCnt=%v, successRate=%v", st.eraCnt, st.shouldCreateCnt, st.successCnt, st.failCnt, float64(st.successCnt)/float64(st.shouldCreateCnt))
}

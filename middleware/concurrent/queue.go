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

package concurrent

import (
	"sync"

	"container/list"
)

type Queue struct {
	data *list.List
	lock sync.Mutex
	max  int
}

func NewQueue(max int) *Queue {
	return &Queue{
		data: list.New(),
		lock: sync.Mutex{},
		max:  max,
	}

}

func (q *Queue) Push(data interface{}) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.data.Len() == q.max {
		return false
	}

	q.data.PushBack(data)
	return true
}

func (q *Queue) Pop() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()

	data := q.data.Front()
	q.data.Remove(data)

	return data
}

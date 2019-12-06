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
package prque

import (
	"container/heap"
)

// Priority queue data structure.
type Prque struct {
	cont *sstack
}

// New creates a new priority queue.
func NewPrque() *Prque {
	return &Prque{newSstack()}
}

// Pushes a value with a given priority into the queue, expanding if necessary.
func (p *Prque) Push(data interface{}, priority int64) {
	heap.Push(p.cont, &Item{data, priority})
}

// Peek returns the value with the greates priority but does not pop it off.
func (p *Prque) Peek() (interface{}, int64) {
	item := p.cont.blocks[0][0]
	return item.Value, item.Priority
}

// Pops the value with the greates priority off the stack and returns it.
// Currently no shrinking is done.
func (p *Prque) Pop() (interface{}, int64) {
	item := heap.Pop(p.cont).(*Item)
	return item.Value, item.Priority
}

// Pops only the item from the queue, dropping the associated priority value.
func (p *Prque) PopItem() interface{} {
	return heap.Pop(p.cont).(*Item).Value
}

// Remove removes the element with the given index.
func (p *Prque) Remove(i int) interface{} {
	if i < 0 {
		return nil
	}
	return heap.Remove(p.cont, i)
}

// Checks whether the priority queue is empty.
func (p *Prque) Empty() bool {
	return p.cont.Len() == 0
}

//// GetCropHeights returns first to cp height slice
func (p *Prque) GetCropHeights(cpHeight, minSize uint64) []*Item {
	if cpHeight <= minSize {
		return nil
	}
	root, h := p.Pop()
	p.Push(root, h)
	if uint64(-h) < cpHeight {
		return nil
	}
	backList := []*Item{}
	cropList := []*Item{}
	var count uint64 = 0
	temp := make(map[uint64]struct{})

	for !p.Empty() {
		root, h = p.Pop()
		if uint64(-h) >= cpHeight {
			backList = append(backList, &Item{root, h})
		} else {
			if _, ok := temp[uint64(-h)]; !ok {
				temp[uint64(-h)] = struct{}{}
				count++
			}
			if count <= minSize {
				backList = append(backList, &Item{root, h})
			} else {
				cropList = append(cropList, &Item{root, h})
			}
		}
	}
	if len(backList) > 0 {
		for _, v := range backList {
			p.Push(v.Value, v.Priority)
		}
	}

	return cropList
}

// Returns the number of element in the priority queue.
func (p *Prque) Size() int {
	return p.cont.Len()
}

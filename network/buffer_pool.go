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

package network

import (
	"bytes"
	"container/list"
	"sync"
)

// BufferPoolItem  maintain the buffers of this size
type BufferPoolItem struct {
	buffers *list.List
	size    int //buffer size
	max     int //max count of cache buffer
	inuse   int //buffer count in use
}

func newBufferPoolItem(size int, max int) *BufferPoolItem {

	item := &BufferPoolItem{
		buffers: list.New(), size: size, max: max,
	}

	return item
}

func (poolItem *BufferPoolItem) getBuffer() *bytes.Buffer {

	if poolItem.buffers.Len() > 0 {
		e := poolItem.buffers.Front()
		buf := e.Value.(*bytes.Buffer)
		poolItem.buffers.Remove(poolItem.buffers.Front())
		buf.Reset()
		return buf
	}
	buf := bytes.NewBuffer(make([]byte, poolItem.size))
	buf.Reset()
	poolItem.inuse++
	return buf
}

func (poolItem *BufferPoolItem) freeBuffer(buf *bytes.Buffer) {

	if buf.Cap() == poolItem.size && poolItem.buffers.Len() < poolItem.max {
		poolItem.buffers.PushBack(buf)
	}
	poolItem.inuse--
}

// BufferPool Cache the buffers used to send and recv data,
// reduce alloc and free times of memory to improve performance
type BufferPool struct {
	items [5]*BufferPoolItem // The key is the network ID
	mutex sync.RWMutex
}

func newBufferPool() *BufferPool {

	pool := &BufferPool{}

	pool.init()

	return pool
}

func (pool *BufferPool) init() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.items[0] = newBufferPoolItem(1024, 1024)
	pool.items[1] = newBufferPoolItem(1024*4, 512)
	pool.items[2] = newBufferPoolItem(1024*32, 256)
	pool.items[3] = newBufferPoolItem(1024*512, 64)
	pool.items[4] = newBufferPoolItem(1024*1024*1.5, 32)
}

// getPoolItem find pool item of this size,if can't found return nil
func (pool *BufferPool) getPoolItem(size int) *BufferPoolItem {

	for i := 0; i < len(pool.items); i++ {
		if pool.items[i].size >= size {
			return pool.items[i]
		}
	}
	return nil
}

func (pool *BufferPool) Print() {

	for i := 0; i < len(pool.items); i++ {
		Logger.Debugf("[ BufferPool Print ] size :%v  count :%v inuse: %v", pool.items[i].size, pool.items[i].buffers.Len(), pool.items[i].inuse)
	}
}

// getBuffer get a buffer of this size from pool, if buffers is run out make a new one
func (pool *BufferPool) getBuffer(size int) *bytes.Buffer {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	poolItem := pool.getPoolItem(size)
	if poolItem != nil {
		buf := poolItem.getBuffer()
		return buf
	}

	return new(bytes.Buffer)
}

//freeBuffer return buffer to pool,if pool if full then free this buffer,if not add to cache
func (pool *BufferPool) freeBuffer(buf *bytes.Buffer) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	poolItem := pool.getPoolItem(buf.Cap())
	if poolItem != nil {
		poolItem.freeBuffer(buf)
	}

}

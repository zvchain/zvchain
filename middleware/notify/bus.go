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

// Package notify implements an event bus framework for the system
package notify

import (
	"sync"
)

// BUS is the unique global instance of the event bus which can accessed from all modules
var BUS *Bus

// Bus is internal message subscription service, or called event bus
type Bus struct {
	topics map[string]*Topic
	lock   sync.RWMutex
}

func NewBus() *Bus {
	return &Bus{
		lock:   sync.RWMutex{},
		topics: make(map[string]*Topic, 10),
	}
}

// Subscribe subscribes a specified event identified by id.
// The handler will be triggered when the event happens
func (bus *Bus) Subscribe(id string, handler Handler) {
	bus.lock.Lock()
	defer bus.lock.Unlock()

	topic, ok := bus.topics[id]
	if !ok {
		topic = &Topic{
			ID: id,
		}
		bus.topics[id] = topic
	}

	topic.Subscribe(handler)
}

// UnSubscribe cancel the subscription for the given event identified by id
func (bus *Bus) UnSubscribe(id string, handler Handler) {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	topic, ok := bus.topics[id]
	if !ok {
		return
	}

	topic.UnSubscribe(handler)
}

// Publish publishes a event identified by id, and all those who care about the event will be notified
func (bus *Bus) Publish(id string, message Message) {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	topic, ok := bus.topics[id]
	if !ok {
		return
	}

	topic.Handle(message, false)
}

func (bus *Bus) PublishWithRecover(id string, message Message) {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	topic, ok := bus.topics[id]
	if !ok {
		return
	}

	topic.Handle(message, true)
}

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

package notify

import (
	"fmt"
	"testing"
)

//hello world2
//hello world
func TestTopic_Subscribe(t *testing.T) {
	topic := &Topic{
		ID: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Handle(&DummyMessage{}, false)
}

//hello world2
func TestTopic_UnSubscribe0(t *testing.T) {
	topic := &Topic{
		ID: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)

	topic.UnSubscribe(handler1)
	topic.Handle(&DummyMessage{}, false)
}

//hello world3
//hello world
func TestTopic_UnSubscribe1(t *testing.T) {
	topic := &Topic{
		ID: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Subscribe(handler3)

	topic.UnSubscribe(handler2)
	topic.Handle(&DummyMessage{}, false)
}

// hello world
// hello world2
func TestTopic_UnSubscribe2(t *testing.T) {
	topic := &Topic{
		ID: "test",
	}

	topic.Subscribe(handler1)
	topic.Subscribe(handler2)
	topic.Subscribe(handler3)

	topic.UnSubscribe(handler3)
	topic.Handle(&DummyMessage{}, false)
}

var reStatus = 0

func TestTopic_With_Recover(t *testing.T) {
	//types.MiddleWareLogger = new(MockLogger)
	//reStatus = 1
	//topic := &Topic{
	//	ID: "test",
	//}
	//
	//topic.Subscribe(handlerPanic)
	//topic.Handle(&DummyMessage{}, true)
	//time.Sleep(time.Second)
	//
	//if reStatus != 2 {
	//	t.Error("should panic")
	//}
}

func TestTopic_Without_Recover(t *testing.T) {
	//types.MiddleWareLogger = new(MockLogger)
	reStatus = 1
	topic := &Topic{
		ID: "test",
	}

	topic.Subscribe(handlerPanic)
	//topic.Handle(&DummyMessage{}, false)//this will panic
}

func handler1(message Message) error {
	fmt.Println("hello world")
	return nil
}

func handler2(message Message) error {
	fmt.Println("hello world2")
	return nil

}

func handler3(message Message) error {
	fmt.Println("hello world3")
	return nil

}

func handlerPanic(message Message) error {
	panic("handler panic")
}

type MockLogger struct {
}

func (mock *MockLogger) Debugf(format string, params ...interface{}) {
	fmt.Println("Debugf", params)
}

func (mock *MockLogger) Infof(format string, params ...interface{}) {
	fmt.Println("Infof", params)
}

func (mock *MockLogger) Warnf(format string, params ...interface{}) error {
	fmt.Println("Warnf", params)
	return nil
}

func (mock *MockLogger) Errorf(format string, params ...interface{}) error {
	fmt.Println("Errorf", format, params)
	reStatus = 2
	return nil
}

func (mock *MockLogger) Debug(v ...interface{}) {
	fmt.Println("Debug", v)
}

func (mock *MockLogger) Info(v ...interface{}) {
	fmt.Println("Info", v)
}

func (mock *MockLogger) Warn(v ...interface{}) error {
	fmt.Println("Warn", v)
	return nil

}

func (mock *MockLogger) Error(v ...interface{}) error {
	fmt.Println("Debugf", v)
	return nil
}

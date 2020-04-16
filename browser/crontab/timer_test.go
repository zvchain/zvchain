package crontab

import (
	"fmt"
	"testing"
	"time"
)

func Test_timer(t *testing.T) {
	time1 := time.Now().Add(-time.Second * 5)
	time2 := time.Now().Add(time.Second * 5)
	fmt.Println(time1.Format("2006 01 02-15:04:05"))
	fmt.Println(time2.Format("2006 01 02-15:04:05"))
	d1 := time1.Sub(time.Now())
	d2 := time2.Sub(time.Now())
	fmt.Println(d1.String())
	fmt.Println(d2.String())

	time.AfterFunc(d1, func() {
		fmt.Println("this is d1")
	})

	time.AfterFunc(d2, func() {
		fmt.Println("this is d2")
	})
	select {}
}

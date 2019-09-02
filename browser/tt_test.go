package browser

import (
	"fmt"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/util"
	"testing"
)

func TestStringTo(test *testing.T) {

	fmt.Println("[crontab]  fetchBlockRewards rewards nil:", nil, nil)

	set := &util.Set{}
	for _, aa := range []string{"1", "2"} {
		set.Add(aa)
	}
	for k, _ := range set.M {
		ss := k.(string)
		fmt.Println(ss)

	}
	poolstakefrom := make([]*models.PoolStake, 0, 0)
	pol := &models.PoolStake{
		Address: "11",
		Stake:   0,
		From:    "1111",
	}
	poolstakefrom = append(poolstakefrom, pol)
	poll := &models.PoolStake{
		Address: "112",
		Stake:   0,
		From:    "11112",
	}
	poolstakefrom = append(poolstakefrom, poll)
	stakelist := make(map[string]map[string]int64)
	tar := "b"
	var arr = []string{"a", "b", "c", "b", "a", "a"}
	for _, source := range arr {
		if _, exists := stakelist[source][tar]; exists {
			stakelist[source][tar] -= 5

		} else {
			stakelist[source] = map[string]int64{}
			stakelist[source][tar] = 8
		}
	}

	fmt.Println(stakelist)

}

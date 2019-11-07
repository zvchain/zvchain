package crontab

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"testing"
)

func TestDB2(t *testing.T) {

	a := common.HexToHash("0x27f576cafbb263ed44be8bd094f66114da26877706f96c4c31d5a97ffebf2e29")
	b := common.BytesToHash(common.Sha256([]byte("transfer")))
	fmt.Println(a, ",", b)

	//server := NewServer("10.0.0.13", 3306, "root", "root123", false)
	//for i := 0; i < 100; i++ {
	//	begin := uint64(i * 1000000)
	//	end := uint64((i + 1) * 1000000)
	//	fmt.Printf("index : begin:%v end:%v\n", begin, end)
	//	acc := server.storage.GetAccountById("0x07eafa7c040e9537837e1c3c3580d87633b019c8ae4f9a5b954c5806340e0886")
	//	bool := server.storage.UpdateAccountByColumn(acc[0], map[string]interface{}{"proposal_stake": 11,
	//		"other_stake":  22,
	//		"verify_stake": 33,
	//		"stake_from":   "CARRIE"})
	//	sys := &models.Sys{
	//		Variable: "block_reward.top_block_height",
	//		SetBy:    "carrie.cxl",
	//	}
	//	//server.storage.add(sys)
	//	fmt.Println("", bool, sys)
	//}

}

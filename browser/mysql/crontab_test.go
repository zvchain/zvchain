package mysql

import (
	"testing"
)

func TestDB(t *testing.T) {
	//storage := NewStorage("10.0.0.13", 3306, "root", "root123", false)
	//for i := 0; i < 100; i++ {
	//	begin := uint64(i * 1000000)
	//	end := uint64((i + 1) * 1000000)
	//	fmt.Printf("index : begin:%v end:%v\n", begin, end)
	//	acc := storage.GetAccountById("0x07eafa7c040e9537837e1c3c3580d87633b019c8ae4f9a5b954c5806340e0886")
	//	bool := storage.UpdateAccountByColumn(acc[0], map[string]interface{}{"proposal_stake": 1,
	//		"other_stake":  2,
	//		"verify_stake": 3,
	//		"stake_from":   "jack"})
	//	sys := &models.Sys{
	//		Variable: "block_reward.top_block_height",
	//		SetBy:    "carrie.cxl",
	//	}
	//	storage.AddBlockRewardSystemconfig(sys)
	//	fmt.Println("", bool)
	//}

}

//func get() string {
//	return nil
//}

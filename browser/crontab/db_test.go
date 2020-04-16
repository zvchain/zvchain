package crontab

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/mysql"
	"github.com/zvchain/zvchain/common"
	"os"
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

func GetTodayStartTs(tm time.Time) time.Time {
	tm1 := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())
	return tm1
}

func initDatabase() *gorm.DB {
	args := fmt.Sprintf("%s:%s@tcp(%s:%d)/gzv?charset=utf8&parseTime=True&loc=Local",
		"root",
		"dan",
		"127.0.0.1",
		3306)
	fmt.Println("[Storage] db args:", args)
	db, err := gorm.Open("mysql", args)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	return db
}

//test
func SetDB(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool, resetcrontab bool) {
	server := &Crontab{
		initdata:       make(chan *models.ForkNotify, 10000),
		initRewarddata: make(chan *models.ForkNotify, 10000),
	}
	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, resetcrontab)
	GlobalCrontab = server

}

func Test_UpdateVoteStatus(t *testing.T) {
	//TestSetVoteInfos(t)

	SetDB("127.0.0.1", 3306, "root", "dan", false, false)
	time.Sleep(time.Second * 3)
	//crontab.UpdateVoteStatus(123456,time.Second*20)
	list := make([]VoteTimer, 0)
	voter1 := VoteTimer{
		VoteStage: 0,
		StartTime: time.Now().Add(time.Second * 10),
		EndTime:   time.Now().Add(time.Second * 20),
	}

	voter2 := VoteTimer{
		VoteStage: 0,
		StartTime: time.Now().Add(time.Second * 15),
		EndTime:   time.Now().Add(time.Second * 25),
	}
	voter3 := VoteTimer{
		VoteStage: 1,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(time.Second * 30),
	}
	voter4 := VoteTimer{
		VoteStage: 1,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(time.Second * 35),
	}
	voter5 := VoteTimer{
		VoteStage: 2,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(-time.Second * 35),
	}
	voter6 := VoteTimer{
		VoteStage: 2,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(-time.Second * 35),
	}
	list = append(list, voter1, voter2, voter3, voter4, voter5, voter6)
	for k, v := range list {
		HandleVoteTimer(uint64(k+1), &v)
	}

	select {}

}

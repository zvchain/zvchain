package browser

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/zvchain/zvchain/browser/crontab"
	"github.com/zvchain/zvchain/browser/models"
	"os"
	"testing"
	"time"

	_ "github.com/zvchain/zvchain/browser/mysql"
	//"testing"
)

type rpcLevel int

const (
	rpcLevelNone     rpcLevel = iota // Won't start rpc service which is the default value if not set
	rpcLevelGtas                     // Only enable the core rpc service functions used by miners or dapp developers
	rpcLevelExplorer                 // Enable both above and explorer related functions
	rpcLevelDev                      // Enable all functions including functions for debug or developer use
)

type minerConfig struct {
	rpcLevel      rpcLevel
	rpcAddr       string
	rpcPort       uint16
	super         bool
	testMode      bool
	natIP         string
	natPort       uint16
	seedIP        string
	applyRole     string
	keystore      string
	enableMonitor bool
	chainID       uint16
	password      string
}

var cfg = &minerConfig{
	rpcLevel:      rpcLevelDev,
	rpcAddr:       "127.0.0.1",
	rpcPort:       8101,
	super:         false,
	testMode:      true,
	natIP:         "",
	natPort:       0,
	applyRole:     "",
	keystore:      "keystore",
	enableMonitor: false,
	chainID:       1,
	password:      "123",
}

//func TestAPI(t *testing.T)  {
//	gtas := cli.NewGtas()
//
//
//}

//func TestDB(t *testing.T) {
//	storage := mysql.NewStorage("10.0.0.13", 3306, "root", "root123", "119.23.68.106", 8101, false)
//	accounts := make(map[string]float64)
//	for i := 0; i < 100; i++ {
//		begin := uint64(i * 1000000)
//		end := uint64((i + 1) * 1000000)
//		fmt.Printf("index : begin:%v end:%v\n", begin, end)
//		sqlStr := fmt.Sprintf("select distinct target from transactions where target <> '' and id > %d and id < %d and value > 1;", begin, end)
//		var db *gorm.DB = storage.GetDB()
//		rows, _ := db.Raw(sqlStr).Rows() // Note: Ignoring errors for brevity
//		for rows.Next() {
//			var target string
//			// This WON'T WORK
//			if err := rows.Scan(&target); err != nil {
//				// ERROR: sql: expected X destination arguments in Scan, not 1
//			}
//			_, ok := accounts[target]
//			if !ok {
//				//client, err := rpc.Dial(storage.rpcAddrStr)
//				//if err != nil {
//				//	//	fmt.Println("[fetcher] Error in dialing. err:",err)
//				//	continue
//				//}
//				//defer client.Close()
//				//
//				//var result map[string]interface{}
//				////call remote procedure with args
//				//err = client.Call(&result, "GZV_explorerAccount", target)
//				//if err != nil {
//				//	fmt.Println("[fetcher]  GZV_explorerAccount error :", err)
//				//}
//				//
//				//if result["data"] != nil {
//				//	data := result["data"].(map[string]interface{})
//				//
//				//	accounts[target] = data["balance"].(float64)
//				//}
//				//chain := core.BlockChainImpl
//				//block := chain.QueryBlockCeil(tm.blockHeight)
//
//				if accounts[target]/1000000000 > 1 {
//					fmt.Printf("%v  : %.6f \n", target, accounts[target]/1000000000)
//				}
//
//			}
//		}
//	}
//
//}

func TestGetGroups(t *testing.T) {
	var dbAddr, rpcAddr string
	var dbPort, rpcPort int
	var dbUser, dbPassword string
	var help bool
	var reset bool

	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&reset, "reset", false, "reset database")
	flag.StringVar(&dbAddr, "dbaddr", "10.0.0.13", "database address")
	flag.StringVar(&rpcAddr, "rpcaddr", "localhost", "RPC address")
	flag.IntVar(&dbPort, "dbport", 3306, "database port")
	flag.IntVar(&rpcPort, "rpcport", 8101, "RPC port")
	flag.StringVar(&dbUser, "dbuser", "root", "database user")
	flag.StringVar(&dbPassword, "dbpw", "root123", "database password")
	flag.Parse()

	if help {
		flag.Usage()
	}
	fmt.Println("browserdbmmanagement flags:", dbAddr, dbPort, dbUser, dbPassword, reset)

	tm := NewDBMmanagement(dbAddr, dbPort, dbUser, dbPassword, reset, false)

	if !tm.GetGroups() {
		t.Fatal("获取失败")
	}

}

func TestSetVoteInfos(t *testing.T) {
	db := initDatabase()

	options := []string{
		"dowork",
		"listen mUsic",
		"go hiking",
	}
	optionsStr, _ := json.Marshal(options)

	voteInfo1 := models.VoteDetails{
		0: &models.VoteStat{
			Count: 2,
			Voter: []string{"0xfasfgsaf", "oxalksgha"},
		},
		1: &models.VoteStat{
			Count: 5,
			Voter: []string{"0xfasfgsaf", "oxalksgha", "0xgalkhnga", "gals;ihgfls;ak", "gouajh"},
		},
	}
	voteInfo1Str, _ := json.Marshal(voteInfo1)

	err := db.Create(&models.Vote{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		VoteId:         1,
		Title:          "A test Title",
		Options:        string(optionsStr),
		Intro:          "A test Intro",
		Promoter:       "zv123456",
		ContractAddr:   "zvabcdef",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour * 24),
		OptionsCount:   uint8(len(options)),
		Status:         0,
		OptionsDetails: string(voteInfo1Str),
	}).Error

	err = db.Create(&models.Vote{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		VoteId:         2,
		Title:          "A test Title",
		Options:        string(optionsStr),
		Intro:          "A test Intro",
		Promoter:       "zv123456",
		ContractAddr:   "zvabcdef",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour * 24),
		OptionsCount:   uint8(len(options)),
		Status:         0,
		OptionsDetails: string(voteInfo1Str),
	}).Error

	err = db.Create(&models.Vote{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		VoteId:         3,
		Title:          "A test Title",
		Options:        string(optionsStr),
		Intro:          "A test Intro",
		Promoter:       "zv123456",
		ContractAddr:   "zvabcdef",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour * 24),
		OptionsCount:   uint8(len(options)),
		Status:         0,
		OptionsDetails: string(voteInfo1Str),
	}).Error

	err = db.Create(&models.Vote{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		VoteId:         4,
		Title:          "A test Title",
		Options:        string(optionsStr),
		Intro:          "A test Intro",
		Promoter:       "zv123456",
		ContractAddr:   "zvabcdef",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour * 24),
		OptionsCount:   uint8(len(options)),
		Status:         0,
		OptionsDetails: string(voteInfo1Str),
	}).Error

	err = db.Create(&models.Vote{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		VoteId:         5,
		Title:          "A test Title",
		Options:        string(optionsStr),
		Intro:          "A test Intro",
		Promoter:       "zv123456",
		ContractAddr:   "zvabcdef",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour * 24),
		OptionsCount:   uint8(len(options)),
		Status:         0,
		OptionsDetails: string(voteInfo1Str),
	}).Error

	err = db.Create(&models.Vote{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		VoteId:         6,
		Title:          "A test Title",
		Options:        string(optionsStr),
		Intro:          "A test Intro",
		Promoter:       "zv123456",
		ContractAddr:   "zvabcdef",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour * 24),
		OptionsCount:   uint8(len(options)),
		Status:         0,
		OptionsDetails: string(voteInfo1Str),
	}).Error

	fmt.Println(err)

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

////test
//func SetDB(dbAddr string, dbPort int, dbUser string, dbPassword string, reset bool, resetcrontab bool) {
//	server := &Crontab{
//		initdata:       make(chan *models.ForkNotify, 10000),
//		initRewarddata: make(chan *models.ForkNotify, 10000),
//	}
//	server.storage = mysql.NewStorage(dbAddr, dbPort, dbUser, dbPassword, reset, resetcrontab)
//	GlobalCrontab = server
//
//}

func Test_UpdateVoteStatus(t *testing.T) {
	//crontab.SetDB("127.0.0.1" ,3306,"root","dan",false,false)
	//TestSetVoteInfos(t)
	db := initDatabase()
	time.Sleep(time.Second * 3)
	//crontab.UpdateVoteStatus(123456,time.Second*20)
	list := make([]crontab.VoteTimer, 0)
	voter1 := crontab.VoteTimer{
		VoteStage: 0,
		StartTime: time.Now().Add(time.Second * 10),
		EndTime:   time.Now().Add(time.Second * 20),
	}

	voter2 := crontab.VoteTimer{
		VoteStage: 0,
		StartTime: time.Now().Add(time.Second * 15),
		EndTime:   time.Now().Add(time.Second * 25),
	}
	voter3 := crontab.VoteTimer{
		VoteStage: 1,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(time.Second * 30),
	}
	voter4 := crontab.VoteTimer{
		VoteStage: 1,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(time.Second * 35),
	}
	voter5 := crontab.VoteTimer{
		VoteStage: 2,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(-time.Second * 35),
	}
	voter6 := crontab.VoteTimer{
		VoteStage: 2,
		StartTime: time.Now().Add(-time.Second * 100),
		EndTime:   time.Now().Add(-time.Second * 35),
	}
	list = append(list, voter1, voter2, voter3, voter4, voter5, voter6)
	for k, v := range list {
		crontab.HandleVoteTimer(uint64(k+1), &v)
	}

	select {}

}

//func (tm *DBMmanagement) GetGroups( ) bool {
//	client, err := rpc.Dial("0.0.0.0:8101")
//	if err != nil {
//		fmt.Println("[fetcher] Error in dialing. err:", err)
//		return false
//	}
//	defer client.Close()
//
//	var result map[string]interface{}
//	//call remote procedure with args
//
//	groups := make([]*models.Group, 0)
//	for i:=uint64(0);i<1000;i++ {
//		fmt.Println("=======================GROUP HIGH",i,tm.groupHeight)
//		//tm.groupHeight=i
//		err = client.Call(&result, "Explorer_explorerGroupsAfter", i)
//		if err != nil {
//			fmt.Println("[fetcher] GetGroups  client.Call error :", err)
//			return false
//		}
//		fmt.Println("[fetcher] GetGroups  result :", result)
//		if result["data"] == nil {
//			return false
//		}
//		groupsData := result["data"].([]interface{})
//		for _, g := range groupsData {
//			group := dataToGroup(g.(map[string]interface{}))
//			if group != nil {
//				groups = append(groups, group)
//			}
//		}
//
//		if groups != nil {
//			for i := 0; i < len(groups); i++ {
//				tm.storage.AddGroup(groups[i])
//				if groups[i].Height >= tm.groupHeight {
//					tm.groupHeight = groups[i].Height + 1
//				}
//			}
//		}else {
//			return false
//		}
//
//	}
//	return true
//}

package browser

import (
	//"github.com/jinzhu/gorm"
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

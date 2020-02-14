package core

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"os"
	"testing"
)
func init(){
	os.RemoveAll("d_small_test")
}

func initDB() (error,*smallStateStore){
	smallStateDs, err := tasdb.NewDataSource("d_small_test" ,nil)
	if err != nil {
		return err,nil
	}
	smallStateDb, err := smallStateDs.NewPrefixDatabase("")
	if err != nil {
		return err,nil
	}
	return nil,initSmallStore(smallStateDb)
}



func initTestData(db *smallStateStore){
	var i uint64
	for i = 0;i<10000;i++{
		db.db.Put(db.generateDataKey(common.Uint64ToByte(i)),common.Uint64ToByte(i))
	}
}

func getDataByHeight(height uint64,db *smallStateStore)[]byte{
	key :=db.generateDataKey(common.Uint64ToByte(height))
	value,_ := db.db.Get(key)
	return value
}

func TestDelete(t *testing.T){
	err,db := initDB()
	if err != nil{
		t.Fatalf("init error,error is %v",err)
	}
	defer func(){
		db.db.Close()
		os.RemoveAll("d_small_test")
	}()
	initTestData(db)

	db.DeletePreviousOf(10000)
	var i uint64
	for i = 0;i<10000;i++{
		vl := getDataByHeight(i,db)
		if len(vl) > 0{
			t.Fatalf("expect nil,but got value,height is %v",i)
		}
	}
	db.db.Close()
	os.RemoveAll("d_small_test")

	err,db = initDB()
	if err != nil{
		t.Fatalf("init error,error is %v",err)
	}
	initTestData(db)
	db.DeletePreviousOf(9998)
	vl := getDataByHeight(9999,db)
	if len(vl) == 0{
		t.Fatalf("expect not nil,but got nil")
	}

	for i = 0;i<9999;i++{
		vl := getDataByHeight(i,db)
		if len(vl) > 0{
			t.Fatalf("expect nil,but got value,height is %v",i)
		}
	}
}

func TestIterator(t *testing.T){
	err,db := initDB()
	if err != nil{
		t.Fatalf("init error,error is %v",err)
	}
	defer func(){
		db.db.Close()
		os.RemoveAll("d_small_test")
	}()
	initTestData(db)
	count := 0
	db.iterateData(func(key, value []byte) (b bool, e error) {
		count++
		return true,nil
	})
	if count != 10000{
		t.Fatalf("expect 10000,but got %v",count)
	}
	count = 0
	db.iterateData(func(key, value []byte) (b bool, e error) {
		count++
		return false,nil
	})
	if count != 1{
		t.Fatalf("expect 1,but got %v",count)
	}


}
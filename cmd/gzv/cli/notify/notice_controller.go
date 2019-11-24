package notify

import (
	"context"
	"fmt"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/storage/tasdb"
	"runtime"
	"time"
)

const OldVersion = false
const NewVersion = true
const NoticeDB = "db_notify"
const UpdatePath = "update"
const GzvFile = "gzv"
const System = runtime.GOOS
const CheckVersioGap = time.Second * 10
const Timeout = time.Second * 60
const Download_Filename = "gzv_mac.zip"

var (
	RequestUrl = "http://127.0.0.1:8000/request"
	UrlLinux   = "http://127.0.0.1:8000/linux"
	//UrlDarwin  = "http://127.0.0.1:8000/drawin"
	UrlDarwin  = "https://developer.zvchain.io/zip/gzv_mac.zip"
	UrlWindows = "http://127.0.0.1:8000/windows"
)

type VersionChecker struct {
	version           string
	notify_gap        uint64
	effective_height  uint64
	priority          uint64
	notice_content    string
	filesize          int64
	download_filename string
}

func NewVersionChecker() *VersionChecker {
	versionChecker := &VersionChecker{}
	return versionChecker
}

func InitVersionChecker() {
	vc := NewVersionChecker()
	nm := NewNotifyManager()

	tiker := time.NewTicker(CheckVersioGap)
	for i := 0; i < 20; i++ {
		select {
		case <-tiker.C:
			ctx, cancel := context.WithCancel(context.Background())
			//Check if the local running program is the latest version
			bl, err := vc.checkversion()
			if err != nil {
				log.DefaultLogger.Errorln(err)
				continue
			}

			if !bl {
				go nm.processOutput(ctx, i)
				nm.versionChecker = vc

				//Check if the latest version has been downloaded locally
				if isFileExist(UpdatePath+"/"+vc.version+"/"+vc.download_filename, vc.filesize) {
					fmt.Println("The latest version has been downloaded locally, but not yet run\n")
					time.Sleep(CheckVersioGap)
					cancel()
					continue
				}

				err := vc.download()
				if err != nil {
					log.DefaultLogger.Errorln(err)
					fmt.Println(" DownLoad Err :", err)
					time.Sleep(CheckVersioGap)
					cancel()
					continue
				}

				time.Sleep(CheckVersioGap)
				cancel()
			}
		}
	}
}

type NotifyManager struct {
	versionChecker *VersionChecker
	stateDb        *tasdb.PrefixedDatabase
}

func NewNotifyManager() *NotifyManager {
	ds, err := tasdb.NewDataSource(NoticeDB, nil)
	if err != nil {
		log.DefaultLogger.Errorln(err)
		return nil
	}
	pd, err := ds.NewPrefixDatabase("")
	if err != nil {
		log.DefaultLogger.Errorln(err)
		return nil
	}

	nm := &NotifyManager{
		stateDb: pd,
	}

	return nm
}

//func (nm *NotifyManager) getNotice() {
//
//}
//func (nm *NotifyManager) removeNotice() {
//
//}

func (nm *NotifyManager) processOutput(ctx context.Context, i int) {
	gap := nm.versionChecker.notify_gap

	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * time.Duration(int64(gap)))
			output := fmt.Sprintf("The current Gzv program is not the latest version. It needs to be updated to the latest version %s as soon as possible\n", nm.versionChecker.version)
			log.DefaultLogger.Errorln(fmt.Errorf(output))
			fmt.Printf("processOutput version --->>> [ %v ],%v", i, output)
		}
	}
}

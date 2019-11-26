package notify

import (
	"fmt"
	"github.com/zvchain/zvchain/log"
	"runtime"
	"time"
)

const OldVersion = false
const NewVersion = true
const NoticeDB = "db_notify"
const UpdatePath = "update"
const System = runtime.GOOS
const CheckVersioGap = time.Hour
const Timeout = time.Second * 60

var (
	RequestUrl = "http://127.0.0.1:8000/request"
)

type VersionChecker struct {
	version          string
	notifyGap        uint64
	effectiveHeight  uint64
	priority         uint64
	noticeContent    string
	filesize         int64
	downloadFilename string
	fileUpdateLists  *UpdateInfos
}

func NewVersionChecker() *VersionChecker {
	versionChecker := &VersionChecker{
		fileUpdateLists: &UpdateInfos{},
	}
	return versionChecker
}

func InitVersionChecker() {
	vc := NewVersionChecker()
	nm := NewNotifyManager()

	ctiker := time.NewTicker(CheckVersioGap)

	for {
		select {
		case <-ctiker.C:
			//Check if the local running program is the latest version
			bl, err := vc.checkversion()
			if err != nil {
				log.DefaultLogger.Errorln(err)
				continue
			}

			if !bl {
				timeOut := time.After(CheckVersioGap)
				nm.versionChecker = vc
				go nm.processOutput(timeOut)

				//Check if the latest version has been downloaded locally
				if isFileExist(UpdatePath+"/"+vc.version+"/"+vc.downloadFilename, vc.filesize) {
					fmt.Println("The latest version has been downloaded locally, but not yet run\n")
					continue
				}

				err := vc.download()
				if err != nil {
					log.DefaultLogger.Errorln(err)
					fmt.Println(" DownLoad Err :", err)
					continue
				}
			}
		}
	}
}

type NotifyManager struct {
	versionChecker *VersionChecker
}

func NewNotifyManager() *NotifyManager {

	nm := &NotifyManager{
		&VersionChecker{},
	}

	return nm
}

func (nm *NotifyManager) processOutput(timeout <-chan time.Time) {
	gap := nm.versionChecker.notifyGap
	for {
		select {
		case <-timeout:
			return
		default:
			time.Sleep(time.Second * time.Duration(int64(gap)))
			output := fmt.Sprintf("The current Gzv program is not the latest version. It needs to be updated to the latest version %s as soon as possible\n", nm.versionChecker.version)
			log.DefaultLogger.Errorln(fmt.Errorf(output))
			fmt.Printf("processOutput version --->>> %v", output)
		}
	}
}

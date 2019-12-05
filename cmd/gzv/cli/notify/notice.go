package notify

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"runtime"
	"time"
)

const OldVersion = false
const NewVersion = true
const UpdatePath = "update"
const System = runtime.GOOS
const CheckVersioGap = time.Hour
const Timeout = time.Second * 60
const DefaultRequestURL = "http://127.0.0.1:8000/request"

var (
	RequestUrl string
)

type VersionChecker struct {
	version          string
	notifyGap        uint64
	effectiveHeight  uint64
	priority         uint64
	noticeContent    string
	filesize         int64
	downloadFilename string
	fileUpdateLists  *UpdateInfo
}

func NewVersionChecker() *VersionChecker {
	versionChecker := &VersionChecker{
		fileUpdateLists: &UpdateInfo{},
	}
	return versionChecker
}

func InitVersionChecker() {
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Errorln("InitVersionChecker err:", err)
			fmt.Println("InitVersionChecker err:", err)
		}
	}()
	RequestUrl = common.GlobalConf.GetString("gzv", "url_for_version_request", DefaultRequestURL)
	vc := NewVersionChecker()
	nm := NewNotifyManager()

	ctiker := time.NewTicker(CheckVersioGap)

	for {
		select {
		case <-ctiker.C:
			//Check if the local running program is the latest version
			log.DefaultLogger.Infoln("start checkversion ...")
			bl, err := vc.checkVersion()
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
					log.DefaultLogger.Errorln("The latest version has been downloaded locally, but not yet run")
					fmt.Println("The latest version has been downloaded locally, but not yet run")
					continue
				}
				log.DefaultLogger.Infoln("start download ...")
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
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Errorln("processOutput err:", err)
		}
	}()

	gap := nm.versionChecker.notifyGap
	for {
		select {
		case <-timeout:
			return
		default:
			time.Sleep(time.Second * time.Duration(int64(gap)))
			//output := fmt.Sprintf("The current Gzv program is not the latest version. It needs to be updated to the latest version %s as soon as possible\n", nm.versionChecker.version)
			output := fmt.Sprintf("[ Version ] : %s \n "+
				"[ EffectiveHeight ] : %d \n "+
				"[ Priority ] : %d \n "+
				"[ Contents ] : %v \n ",
				nm.versionChecker.version,
				nm.versionChecker.effectiveHeight,
				nm.versionChecker.priority,
				nm.versionChecker.noticeContent)
			log.DefaultLogger.Errorln(fmt.Errorf(output))
			fmt.Printf("\n================= New version notification ================= \n %v \n============================================================ \n\n", output)
		}
	}
}

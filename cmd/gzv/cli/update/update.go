package update

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
const CheckVersionGap = time.Hour
const Timeout = time.Second * 60
const DefaultRequestURL = "http://47.110.159.248:8000/request"

var (
	RequestUrl string
)

type VersionChecker struct {
	version          string
	notifyGap        uint64
	effectiveHeight  uint64
	priority         uint64
	noticeContent    string
	fileSize         int64
	downloadFilename string
	localFileName    string
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
			log.DefaultLogger.Errorln("init version checker recover ,err:", err)
			fmt.Println("init version checker recover err:", err)
		}
	}()
	RequestUrl = common.GlobalConf.GetString("gzv", "url_for_version_request", DefaultRequestURL)
	vc := NewVersionChecker()
	nm := NewNotifyManager()

	checkVersion(vc, nm)
	ticker := time.NewTicker(CheckVersionGap)
	for {
		select {
		case <-ticker.C:
			checkVersion(vc, nm)
		}
	}
}

func checkVersion(vc *VersionChecker, nm *NotifyManager) {
	//Check if the local running program is the latest version
	log.DefaultLogger.Infoln("start check version ...")
	bl, err := vc.checkVersion()
	if err != nil {
		log.DefaultLogger.Errorln(err)
		return
	}

	if !bl {
		timeOut := time.After(CheckVersionGap)
		nm.versionChecker = vc
		go nm.processOutput(timeOut)

		//Check if the latest version has been downloaded locally
		if isFileExist(vc.localFileName) {
			log.DefaultLogger.Errorln("The latest version has been downloaded locally, but not yet run")
			fmt.Println("The latest version has been downloaded locally, but not yet run")
			return
		}
		log.DefaultLogger.Infoln("start download ...")
		err := vc.download()
		if err != nil {
			log.DefaultLogger.Errorln(err)
			fmt.Println(" DownLoad Err :", err)
			return
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
			output := fmt.Sprintf("[ Version ] : %s \n "+
				"[ EffectiveHeight ] : %d \n "+
				"[ Priority ] : %d \n "+
				"[ Contents ] : %v \n ",
				nm.versionChecker.version,
				nm.versionChecker.effectiveHeight,
				nm.versionChecker.priority,
				nm.versionChecker.noticeContent)
			fmt.Printf("\n================= New version notification ================= \n %v \n============================================================ \n\n", output)
		}
	}
}

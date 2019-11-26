package notify

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"io/ioutil"
	"net/http"
)

func (vc *VersionChecker) checkversion() (bool, error) {
	notice, err := requestVersion()
	if err != nil {
		return NewVersion, err
	}
	if notice == nil {
		return NewVersion, fmt.Errorf("Request version returned empty\n")
	}

	if notice.Version == common.GzvVersion {
		return NewVersion, nil
	}

	vc.version = notice.Version
	vc.notifyGap = notice.NotifyGap
	vc.effectiveHeight = notice.EffectiveHeight
	vc.priority = notice.Priority
	vc.noticeContent = notice.NoticeContent
	vc.fileUpdateLists = notice.UpdateInfo

	return OldVersion, nil
}

func requestVersion() (*Notice, error) {
	resp, err := http.Get(RequestUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := &Result{}
	err = json.Unmarshal(responseBytes, res)
	if err != nil {
		return nil, err
	}

	notice := &Notice{
		UpdateInfo: &UpdateInfos{},
	}

	if res.Data == nil {
		return nil, fmt.Errorf("version response is empty\n")
	}

	n := res.Data.(map[string]interface{})["data"].(map[string]interface{})

	notice.Version = n["version"].(string)
	notice.NotifyGap = uint64(n["notifyGap"].(float64))
	notice.EffectiveHeight = uint64(n["effectiveHeight"].(float64))
	notice.Priority = uint64(n["priority"].(float64))
	notice.NoticeContent = n["noticeContent"].(string)

	list := make(map[string]interface{}, 0)

	switch System {
	case "darwin":
		list = n["update_for_drawin"].(map[string]interface{})
	case "linux":
		list = n["update_for_linux"].(map[string]interface{})
	case "windows":
		list = n["update_for_windows"].(map[string]interface{})
	}

	notice.UpdateInfo.PackgeUrl = list["packge_url"].(string)
	notice.UpdateInfo.Packgemd5 = list["packge_md5"].(string)
	updateFileList := list["filelist"].([]interface{})
	for _, file := range updateFileList {
		notice.UpdateInfo.Filelist = append(notice.UpdateInfo.Filelist, file.(string))
	}

	fmt.Printf("notice : %v [%v]", notice, notice.UpdateInfo)

	return notice, nil
}

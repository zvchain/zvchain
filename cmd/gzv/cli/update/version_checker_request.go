package update

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"io/ioutil"
	"net/http"
)

func (vc *VersionChecker) checkVersion() (bool, error) {
	notice, err := vc.requestVersion()
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
	vc.fileUpdateLists = notice.UpdateInfos

	return OldVersion, nil
}

func (vc *VersionChecker) requestVersion() (*Notice, error) {
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
		UpdateInfos: &UpdateInfo{},
	}

	if res.Data == nil {
		return nil, fmt.Errorf("version response is empty\n")
	}

	if n, ok := res.Data.(map[string]interface{})["data"].(map[string]interface{}); ok {

		v, ok := n["version"].(string)
		if ok {
			notice.Version = v
		}

		ng, ok := n["notify_gap"].(float64)
		if ok {
			notice.NotifyGap = uint64(ng)
		}

		eh, ok := n["effective_height"].(float64)
		if ok {
			notice.EffectiveHeight = uint64(eh)
		}

		pr, ok := n["priority"].(float64)
		if ok {
			notice.EffectiveHeight = uint64(pr)
		}

		nc, ok := n["notice_content"].(string)
		if ok {
			notice.NoticeContent = nc
		}

		list := make(map[string]interface{}, 0)

		switch System {
		case "darwin":
			list, ok = n["update_for_darwin"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("assertion err")
			}
		case "linux":
			list, ok = n["update_for_linux"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("assertion err")
			}
		case "windows":
			list, ok = n["update_for_windows"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("assertion err")
			}
		}

		url, ok := list["package_url"].(string)
		if ok {
			notice.UpdateInfos.PackageUrl = url
		}

		md5, ok := list["package_md5"].(string)
		if ok {
			notice.UpdateInfos.Packagemd5 = md5
		}

		updateFileList, ok := list["file_list"].([]interface{})
		if ok {
			for _, file := range updateFileList {
				notice.UpdateInfos.Filelist = append(notice.UpdateInfos.Filelist, file.(string))
			}
		}
		fmt.Printf("VersionInfo : %v [%v] \n", notice, notice.UpdateInfos)
	}

	return notice, err
}

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
		return newVersion, err
	}
	if notice == nil {
		return newVersion, fmt.Errorf("Request version returned empty\n")
	}

	if notice.Version == common.GzvVersion {
		return newVersion, nil
	}

	vc.version = notice.Version
	if notice.NotifyGap == 0 {
		notice.NotifyGap = defaultNotifyGap
	}
	vc.notifyGap = notice.NotifyGap
	vc.effectiveHeight = notice.EffectiveHeight
	vc.required = notice.Required
	vc.noticeContent = notice.NoticeContent
	vc.fileUpdateLists = notice.UpdateInfos

	return oldVersion, nil
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

		pr, ok := n["required"].(string)
		if ok {
			notice.Required = pr
		}

		nc, ok := n["notice_content"].(string)
		if ok {
			notice.NoticeContent = nc
		}

		list := make(map[string]interface{}, 0)

		switch system {
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
			notice.UpdateInfos.PackageMd5 = md5
		}

		updateFileList, ok := list["file_list"].([]interface{})
		if ok {
			for _, file := range updateFileList {
				notice.UpdateInfos.FileList = append(notice.UpdateInfos.FileList, file.(string))
			}
		}
	}

	return notice, err
}

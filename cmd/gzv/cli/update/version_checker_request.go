package update

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"io/ioutil"
	"math/big"
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
	if notice.Version == "" {
		return newVersion, fmt.Errorf("the version is empty")
	}

	if notice.Version == common.GzvVersion {
		return newVersion, nil
	}

	vc.version = notice.Version
	if notice.NotifyGap == "0" {
		notice.NotifyGap = defaultNotifyGap
	}
	ng := new(big.Int)
	ng.SetString(notice.NotifyGap, 10)
	vc.notifyGap = ng

	eh := new(big.Int)
	eh.SetString(notice.EffectiveHeight, 10)
	vc.effectiveHeight = eh

	vc.required = notice.Required
	vc.noticeContent = notice.NoticeContent
	vc.fileUpdateLists = notice.UpdateInfos

	if len(notice.WhiteList) < 1 {
		return newVersion, fmt.Errorf("white list is empty")
	}
	for _, list := range notice.WhiteList {
		if notice.Version == list && notice.Version != common.GzvVersion {
			return oldVersion, nil
		}
		if list != common.GzvVersion {
			continue
		}
	}
	return newVersion, nil
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
		return nil, fmt.Errorf("version response is empty")
	}

	if n, ok := res.Data.(map[string]interface{})["data"].(map[string]interface{}); ok {

		notice.Version, ok = n["version"].(string)
		if !ok || n["version"] == "" {
			return nil, fmt.Errorf("version assertion err")
		}

		notice.NotifyGap, ok = n["notify_gap"].(string)
		if !ok || n["notify_gap"] == "" {
			return nil, fmt.Errorf("notify_gap assertion err")
		}

		notice.EffectiveHeight, ok = n["effective_height"].(string)
		if !ok || n["effective_height"] == "" {
			return nil, fmt.Errorf("effective_height assertion err")
		}

		notice.Required, ok = n["required"].(string)
		if !ok || n["required"] == "" {
			return nil, fmt.Errorf("required assertion err")
		}

		notice.NoticeContent, ok = n["notice_content"].(string)
		if !ok || n["notice_content"] == "" {
			return nil, fmt.Errorf("notice_content assertion err")
		}

		wl, ok := n["white_list"].([]interface{})
		if !ok || n["white_list"] == nil {
			return nil, fmt.Errorf("white_list assertion err")
		}
		for _, wlist := range wl {
			if _, ok := wlist.(string); ok {
				notice.WhiteList = append(notice.WhiteList, wlist.(string))
			}
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
		if !ok || list["file_list"] == nil {
			return nil, fmt.Errorf("file_list assertion err")
		}
		for _, file := range updateFileList {
			if _, ok := file.(string); ok {
				notice.UpdateInfos.FileList = append(notice.UpdateInfos.FileList, file.(string))
			}
		}
	}

	return notice, err
}

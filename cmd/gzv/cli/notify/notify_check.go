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
	vc.notify_gap = notice.NotifyGap
	vc.effective_height = notice.EffectiveHeight
	vc.priority = notice.Priority
	vc.notice_content = notice.NoticeContent

	return OldVersion, nil
}

//func copyBuffer(file os.File, resp http.Response) error {
//	var (
//		buf     = make([]byte, 32*1024)
//		written int64
//		err     error
//	)
//
//	for {
//		//读取bytes
//		nr, er := resp.Body.Read(buf)
//		if nr > 0 {
//			//写入bytes
//			nw, ew := file.Write(buf[0:nr])
//			//数据长度大于0
//			if nw > 0 {
//				written += int64(nw)
//			}
//			//写入出错
//			if ew != nil {
//				err = ew
//				break
//			}
//			//读取是数据长度不等于写入的数据长度
//			if nr != nw {
//				err = io.ErrShortWrite
//				break
//			}
//		}
//		if er != nil {
//			if er != io.EOF {
//				err = er
//			}
//			break
//		}
//		//没有错误了快使用 callback
//		//fb(fsize, written)
//	}
//	fmt.Println(err)
//	return err
//}

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

	notice := &Notice{}

	if res.Data == nil {
		return nil, fmt.Errorf("version response is empty\n")
	}

	n := res.Data.(map[string]interface{})["data"].(map[string]interface{})

	notice.Version = n["version"].(string)
	notice.NotifyGap = uint64(n["notifyGap"].(float64))
	notice.EffectiveHeight = uint64(n["effectiveHeight"].(float64))
	notice.Priority = uint64(n["priority"].(float64))
	notice.NoticeContent = n["noticeContent"].(string)

	li := n["update_lists_for_linux"].([]interface{})
	dr := n["update_lists_for_drawin"].([]interface{})
	wi := n["update_lists_for_windows"].([]interface{})

	for _, file := range li {
		notice.UpdateListsForLinux = append(notice.UpdateListsForLinux, file.(string))
	}

	for _, file := range dr {
		notice.UpdateListsForDrawin = append(notice.UpdateListsForDrawin, file.(string))
	}

	for _, file := range wi {
		notice.UpdateListsForWindows = append(notice.UpdateListsForWindows, file.(string))
	}

	fmt.Println("=========  notice ============", notice)

	return notice, nil
}

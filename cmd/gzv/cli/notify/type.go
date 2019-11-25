package notify

type UpdateInfos struct {
	PackgeUrl string   `json:"packge_url"`
	Packgemd5 string   `json:"packge_md5"`
	Filelist  []string `json:"filelist"`
}

type Notice struct {
	Version         string       `json:"version"`
	NotifyGap       uint64       `json:"notifyGap"`
	EffectiveHeight uint64       `json:"effectiveHeight"`
	Priority        uint64       `json:"priority"`
	NoticeContent   string       `json:"noticeContent"`
	UpdateInfo      *UpdateInfos `json:"update_info"`
}

type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

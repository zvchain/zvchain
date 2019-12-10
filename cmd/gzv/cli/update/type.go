package update

type UpdateInfo struct {
	PackageUrl  string   `json:"package_url"`
	PackageMd5  string   `json:"package_md5"`
	PackageSign string   `json:"package_sign"`
	FileList    []string `json:"file_list"`
}

type Notice struct {
	Version         string      `json:"version"`
	NotifyGap       uint64      `json:"notify_gap"`
	EffectiveHeight uint64      `json:"effective_height"`
	Required        string      `json:"required"`
	NoticeContent   string      `json:"notice_content"`
	UpdateInfos     *UpdateInfo `json:"update_info"`
}

type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

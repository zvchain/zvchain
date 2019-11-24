package notify

type Notice struct {
	Version               string   `json:"version"`
	NotifyGap             uint64   `json:"notifyGap"`
	EffectiveHeight       uint64   `json:"effectiveHeight"`
	Priority              uint64   `json:"priority"`
	NoticeContent         string   `json:"noticeContent"`
	UpdateListsForLinux   []string `json:"update_lists_for_linux"`
	UpdateListsForDrawin  []string `json:"update_lists_for_drawin"`
	UpdateListsForWindows []string `json:"update_lists_for_windows"`
}

type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

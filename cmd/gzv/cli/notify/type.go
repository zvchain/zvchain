package notify

type Notice struct {
	Version          string `json:"version"`
	Notify_Gap       uint64 `json:"notify_gap"`
	Effective_Height uint64 `json:"effective_height"`
	Priority         uint64 `json:"priority"`
	Notice_Content   string `json:"notice_content"`
}

type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

package browser

type StakeDetail struct {
	Value        uint64 `json:"value"`
	UpdateHeight uint64 `json:"update_height"`
	MType        string `json:"m_type"`
	Status       string `json:"status"`
}

type MinerStakeDetails struct {
	//Overview []*MortGage               `json:"overview,omitempty"`
	Details map[string][]*StakeDetail `json:"details,omitempty"`
}

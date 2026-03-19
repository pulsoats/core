package websocket

type RawCandle struct {
	Start    int64  `json:"start"`
	End      int64  `json:"end"`
	Interval string `json:"interval"`
	Open     string `json:"open"`
	Close    string `json:"close"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Volume   string `json:"volume"`
	Turnover string `json:"turnover"`
	Confirm  bool   `json:"confirm"`
	Ts       int64  `json:"timestamp"`
}

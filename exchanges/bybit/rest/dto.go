package rest

type candlesResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List   [][]string `json:"list"`
		Cursor string     `json:"nextPageCursor"`
	} `json:"result"`
	RetExtInfo any `json:"retExtInfo"`
	Time       any `json:"time"`
}

type instrumentsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			Symbol string `json:"symbol"`
		} `json:"list"`
		Cursor string `json:"nextPageCursor"`
	} `json:"result"`
}

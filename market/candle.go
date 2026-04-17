package market

type Candle struct {
	Time      int64
	Open      int64
	High      int64
	Low       int64
	Close     int64
	Volume    int64
	Turnover  float64
}

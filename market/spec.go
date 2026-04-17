package market

type Spec struct {
	Exchange string   `json:"exchange"`
	Category Category `json:"category"`
	Symbol   string   `json:"symbol"`
}

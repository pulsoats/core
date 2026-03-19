package exchange

import (
	"context"
	"time"
)

type TradeMode string

const (
	TradeModeDemo TradeMode = "demo"
	TradeModeReal           = "real"
)

type APICredentials struct {
	ID        int64
	Exchange  string
	TradeMode TradeMode
	Label     string
	APIKey    string
	APISecret string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type APICredentialsRepository interface {
	Upsert(ctx context.Context, exchangeCode string, mode TradeMode, label, apiKey, apiSecret string) (*APICredentials, error)
	Find(ctx context.Context, exchangeCode string, mode TradeMode) (*APICredentials, error)
}

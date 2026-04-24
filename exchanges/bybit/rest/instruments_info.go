package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const bybitInstrumentsInfoPath = "/v5/market/instruments-info"

func (r *Client) InstrumentExists(ctx context.Context, category string, symbol string) (bool, error) {
	if r.client == nil {
		r.client = http.DefaultClient
	}

	symbol = strings.ToUpper(symbol)

	r.log.Debug("rest instruments-info request", "category", category, "symbol", symbol)

	u, err := url.Parse(BybitV5URL)
	if err != nil {
		return false, fmt.Errorf("parse base url: %w", err)
	}

	u = u.JoinPath(bybitInstrumentsInfoPath)

	q := url.Values{}
	q.Set("category", category)
	q.Set("symbol", symbol)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}

	res, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		r.log.Warn("instrument exists non-200", "status", res.StatusCode, "symbol", symbol, "category", category)
		return false, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	var resp instrumentsResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 10<<20)).Decode(&resp); err != nil {
		return false, fmt.Errorf("decode: %w", err)
	}

	if resp.RetCode != 0 {
		r.log.Warn("instrument exists retCode", "code", resp.RetCode, "msg", resp.RetMsg)
		return false, fmt.Errorf("bybit error %d: %s", resp.RetCode, resp.RetMsg)
	}

	if len(resp.Result.List) == 0 {
		return false, nil
	}

	return true, nil
}

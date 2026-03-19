package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pulsoats/core/domain/derrors"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/exchanges/bybit/specs"
)

type fees struct {
	Symbol       string `json:"symbol,omitempty"`
	BaseCoin     string `json:"baseCoin,omitempty"`
	TakerFeeRate string `json:"takerFeeRate"`
	MakerFeeRate string `json:"makerFeeRate"`
}

type feeRateResp struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List   []fees `json:"list"`
		Cursor string `json:"nextPageCursor"`
	} `json:"result"`
	RetExtInfo any    `json:"retExtInfo"`
	Time       string `json:"time"`
}

const bybitFeeRatePath = "/v5/account/fee-rate"

func (r *Client) FeeRate(ctx context.Context, category market.Category, symbol, baseCoin string) (market.TakerMakerFees, error) {
	if r.client == nil {
		r.client = http.DefaultClient
	}

	if r.apiKey == "" || r.apiSecret == "" {
		return market.TakerMakerFees{}, fmt.Errorf("bybit: fee-rate: %w", derrors.ErrUnauthorized)
	}

	u, err := url.Parse(BybitV5URL)
	if err != nil {
		return market.TakerMakerFees{}, err
	}
	u = u.JoinPath(bybitFeeRatePath)

	q := url.Values{}
	q.Set("category", string(category))

	switch category {
	case specs.CategoryOption:
		if baseCoin == "" {
			return market.TakerMakerFees{}, fmt.Errorf("%w: option fee-rate requires base coin", derrors.ErrRequired)
		}
		q.Set("baseCoin", strings.ToUpper(baseCoin))
	default:
		if symbol == "" {
			return market.TakerMakerFees{}, fmt.Errorf("%w: fee-rate requires symbol for category=%s", derrors.ErrRequired, category)
		}
		q.Set("symbol", strings.ToUpper(symbol))
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return market.TakerMakerFees{}, err
	}

	// payloadSuffix = query string
	if err := setAuthHeaders(r.apiKey, r.apiSecret, u.RawQuery, time.Now(), req); err != nil {
		return market.TakerMakerFees{}, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return market.TakerMakerFees{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return market.TakerMakerFees{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		r.log.Warn("fee-rate non-200 status", "status", resp.StatusCode, "body", string(body))
		return market.TakerMakerFees{}, fmt.Errorf("bybit fee-rate: status=%d body=%s",
			resp.StatusCode, string(body))
	}

	var dto feeRateResp
	if err := json.Unmarshal(body, &dto); err != nil {
		return market.TakerMakerFees{}, fmt.Errorf("unmarshal fee-rate resp: %w", err)
	}

	if dto.RetCode != 0 {
		r.log.Warn("fee-rate retCode", "code", dto.RetCode, "msg", dto.RetMsg)
		return market.TakerMakerFees{}, fmt.Errorf("bybit fee-rate error: code=%d msg=%s",
			dto.RetCode, dto.RetMsg)
	}

	if len(dto.Result.List) == 0 {
		r.log.Warn("fee-rate empty list", "category", category, "symbol", symbol, "baseCoin", baseCoin)
		return market.TakerMakerFees{}, fmt.Errorf("%w: fee list is empty", derrors.ErrNotFound)
	}

	f := dto.Result.List[0]

	takerPPM, err := FeeStringToPPM(f.TakerFeeRate)
	if err != nil {
		return market.TakerMakerFees{}, fmt.Errorf("parse takerFeeRate: %w", err)
	}

	makerPPM, err := FeeStringToPPM(f.MakerFeeRate)
	if err != nil {
		return market.TakerMakerFees{}, fmt.Errorf("parse makerFeeRate: %w", err)
	}

	return market.TakerMakerFees{
		TakerFeeRate: takerPPM,
		MakerFeeRate: makerPPM,
	}, nil
}

func FeeStringToPPM(s string) (int64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(f * 1_000_000)), nil
}

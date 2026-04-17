package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/market"
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
		return market.TakerMakerFees{}, fmt.Errorf("bybit rest: fee-rate credentials: %w", errorsx.ErrUnauthorized)
	}

	u, err := url.Parse(BybitV5URL)
	if err != nil {
		return market.TakerMakerFees{}, err
	}
	u = u.JoinPath(bybitFeeRatePath)

	r.log.Debug("rest fee-rate request",
		"category", category,
		"symbol", symbol,
		"baseCoin", baseCoin,
	)

	q := url.Values{}
	q.Set("category", string(category))

	switch category {
	case specs.CategoryOption:
		if baseCoin == "" {
			return market.TakerMakerFees{}, fmt.Errorf("bybit rest: fee-rate option requires base coin: %w", errorsx.ErrRequired)
		}
		q.Set("baseCoin", strings.ToUpper(baseCoin))
	default:
		if symbol == "" {
			return market.TakerMakerFees{}, fmt.Errorf("bybit rest: fee-rate requires symbol for category=%s: %w", category, errorsx.ErrRequired)
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
		return market.TakerMakerFees{}, errors.Join(
			fmt.Errorf("bybit rest: fee-rate read response body: %w", errorsx.ErrInternal),
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		r.log.Warn("fee-rate non-200 status", "status", resp.StatusCode, "body", string(body))
		return market.TakerMakerFees{}, fmt.Errorf("bybit rest: fee-rate status=%d body=%s: %w",
			resp.StatusCode, string(body), errorsx.ErrInternal)
	}

	var dto feeRateResp
	if err := json.Unmarshal(body, &dto); err != nil {
		return market.TakerMakerFees{}, errors.Join(
			fmt.Errorf("bybit rest: fee-rate unmarshal response: %w", errorsx.ErrInternal),
			err,
		)
	}

	if dto.RetCode != 0 {
		r.log.Warn("fee-rate retCode", "code", dto.RetCode, "msg", dto.RetMsg)
		return market.TakerMakerFees{}, fmt.Errorf("bybit rest: fee-rate retCode=%d msg=%s: %w",
			dto.RetCode, dto.RetMsg, errorsx.ErrInternal)
	}

	if len(dto.Result.List) == 0 {
		r.log.Warn("fee-rate empty list", "category", category, "symbol", symbol, "baseCoin", baseCoin)
		return market.TakerMakerFees{}, fmt.Errorf("bybit rest: fee-rate list is empty: %w", errorsx.ErrNotFound)
	}

	f := dto.Result.List[0]

	takerPPM, err := FeeStringToPPM(f.TakerFeeRate)
	if err != nil {
		return market.TakerMakerFees{}, errors.Join(
			fmt.Errorf("bybit rest: fee-rate parse takerFeeRate: %w", errorsx.ErrInternal),
			err,
		)
	}

	makerPPM, err := FeeStringToPPM(f.MakerFeeRate)
	if err != nil {
		return market.TakerMakerFees{}, errors.Join(
			fmt.Errorf("bybit rest: fee-rate parse makerFeeRate: %w", errorsx.ErrInternal),
			err,
		)
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

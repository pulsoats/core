package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/lib/parse"
	"github.com/pulsoats/core/lib/units"
)

const bybitMarketPath = "/v5/market"

const lastPricePath = "kline"

// Candles loads candles(klines) from Bybit in ASC order ("to" time excluded)
func (r *Client) Candles(ctx context.Context, spec market.Spec, interval market.Interval, from time.Time, to time.Time) ([]market.Candle, error) {
	if r.client == nil {
		r.client = http.DefaultClient
	}
	if !specs.IsSupportedCategory(spec.Category) {
		return nil, fmt.Errorf("bybit rest: candles category=%v: %w", spec.Category, errorsx.ErrNotFound)
	}

	intervalStr, ok := specs.SupportedIntervals[interval]
	if !ok {
		return nil, fmt.Errorf("bybit rest: candles interval=%v: %w", interval, errorsx.ErrNotFound)
	}

	if !from.Before(to) {
		return nil, fmt.Errorf("bybit rest: candles from must be < to: %w", errorsx.ErrInvalidArgument)
	}

	intervalDur := time.Duration(interval)
	if intervalDur <= 0 {
		return nil, fmt.Errorf("bybit rest: candles invalid interval=%v: %w", interval, errorsx.ErrInvalidArgument)
	}

	u, err := url.Parse(BybitV5URL)
	if err != nil {
		return nil, err
	}
	u = u.JoinPath(bybitMarketPath)

	u = u.JoinPath(lastPricePath)

	const limit = 1000
	step := time.Duration(limit) * intervalDur
	start := from.Add(-intervalDur)
	toMs := to.UnixMilli()
	fromMs := from.UnixMilli()

	candles := make([]market.Candle, 0, 9000)
	var lastAppendedTs int64 = -1

	for start.Before(to) {
		end := start.Add(step)
		if end.After(to) {
			end = to
		}

		v := url.Values{}
		v.Set("category", string(spec.Category))
		v.Set("symbol", spec.Symbol)
		v.Set("interval", intervalStr)
		v.Set("start", strconv.FormatInt(start.UnixMilli(), 10))
		v.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
		v.Set("limit", strconv.Itoa(limit))
		u.RawQuery = v.Encode()

		r.log.Debug("rest candles request",
			"symbol", spec.Symbol,
			"category", spec.Category,
			"interval", interval,
			"start", start,
			"end", end,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}

		res, err := r.client.Do(req)
		if err != nil {
			return nil, err
		}

		// НЕ defer в цикле — закрываем сразу
		var resp candlesResponse
		var rawBody []byte
		func() {
			defer res.Body.Close()

			rawBody, err = io.ReadAll(res.Body)
			if err != nil {
				err = fmt.Errorf("bybit rest: candles read body: %w", err)
				return
			}

			if res.StatusCode < 200 || res.StatusCode >= 300 {
				r.log.Warn("candles non-200 status",
					"status", res.StatusCode,
					"headers", res.Header,
					"body", string(rawBody),
				)
				err = fmt.Errorf("bybit rest: candles http=%d body=%s: %w", res.StatusCode, string(rawBody), errorsx.ErrInternal)
				return
			}

			if derr := json.Unmarshal(rawBody, &resp); derr != nil {
				r.log.Warn("candles decode failed",
					"err", derr,
					"headers", res.Header,
					"body", string(rawBody),
				)
				err = derr
				return
			}
		}()
		if err != nil {
			return nil, err
		}

		if resp.RetCode != 0 {
			if resp.RetCode == 10006 {
				r.log.Debug("bybit rest: candles: rate limit hit", "retCode", resp.RetCode, "msg", resp.RetMsg, "headers", res.Header, "body", string(rawBody))
				const fallbackSleep = time.Second
				sleepTime := fallbackSleep
				rawResetTs := res.Header.Get("X-Bapi-Limit-Reset-Timestamp")
				if rawResetTs != "" {
					resetTs, err := strconv.ParseInt(rawResetTs, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("bybit rest: candles: invalid X-Bapi-Limit-Reset-Timestamp=%q: %w", rawResetTs, errorsx.ErrInternal)
					}
					if d := time.Duration(resetTs-time.Now().UnixMilli()+100) * time.Millisecond; d > 0 {
						sleepTime = d
					}
				}
				if sleepTime > 0 {
					r.log.Debug("bybit rest: candles: rate limit sleep", "duration", sleepTime.String())
					select {
					case <-time.After(sleepTime):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
				continue // retry the same start/end window
			}
			r.log.Warn("candles retCode", "code", resp.RetCode, "msg", resp.RetMsg, "headers", res.Header, "body", string(rawBody))
			return nil, fmt.Errorf("bybit rest: candles retCode=%d retMsg=%s: %w", resp.RetCode, resp.RetMsg, errorsx.ErrInternal)
		}

		if len(resp.Result.List) == 0 {
			start = end
			continue
		}

		page := make([]market.Candle, 0, len(resp.Result.List))
		for _, raw := range resp.Result.List {
			c, derr := decodeCandle(raw)
			if derr != nil {
				return nil, derr
			}
			page = append(page, c)
		}
		slices.Reverse(page)

		startMs := start.UnixMilli()
		var appendedAny bool
		for _, c := range page {
			if c.Time < startMs || c.Time >= toMs {
				continue
			}
			if c.Time == lastAppendedTs {
				continue
			}
			candles = append(candles, c)
			lastAppendedTs = c.Time
			appendedAny = true
		}

		prevStart := start
		if appendedAny {
			last := candles[len(candles)-1]
			start = time.UnixMilli(last.Time)

			if !start.After(prevStart) {
				start = prevStart.Add(intervalDur)
			}
		} else {
			start = end
		}
	}

	out := candles[:0]
	for _, c := range candles {
		if c.Time < fromMs || c.Time >= toMs {
			continue
		}
		out = append(out, c)
	}

	if len(out) == 0 {
		r.log.Warn("candles result empty", "symbol", spec.Symbol, "category", spec.Category)
		return nil, fmt.Errorf("bybit rest: candles not received: %w", errorsx.ErrNotFound)
	}
	return out, nil
}

func decodeCandle(rawCandle []string) (market.Candle, error) {
	if len(rawCandle) < 7 {
		return market.Candle{}, fmt.Errorf("bybit rest: decode candle len=%d want>=7: %w", len(rawCandle), errorsx.ErrInvalidArgument)
	}
	ts, err := strconv.ParseInt(rawCandle[0], 10, 64)
	if err != nil {
		return market.Candle{}, err
	}

	openVal, err := parse.StrToCents(rawCandle[1])
	if err != nil {
		return market.Candle{}, err
	}
	highVal, err := parse.StrToCents(rawCandle[2])
	if err != nil {
		return market.Candle{}, err
	}
	lowVal, err := parse.StrToCents(rawCandle[3])
	if err != nil {
		return market.Candle{}, err
	}
	closeVal, err := parse.StrToCents(rawCandle[4])
	if err != nil {
		return market.Candle{}, err
	}

	volumeFloat, err := strconv.ParseFloat(rawCandle[5], 64)
	if err != nil {
		return market.Candle{}, err
	}
	volume := int64(volumeFloat * float64(units.PPM))

	turnover, err := strconv.ParseFloat(rawCandle[6], 64)
	if err != nil {
		return market.Candle{}, err
	}

	return market.Candle{
		Time:     ts,
		Open:     openVal,
		High:     highVal,
		Low:      lowVal,
		Close:    closeVal,
		Volume:   volume,
		Turnover: turnover,
	}, nil
}

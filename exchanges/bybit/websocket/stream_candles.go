package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	websocket3 "github.com/pulsoats/core/transport/websocket"
	"github.com/pulsoats/core/transport/websocket/router"
)

func (w *Client) StreamCandles(ctx context.Context, spec market.CandleSpec, confirmedOnly bool) (chan market.Candle, <-chan error, error) {
	w.log.Info("stream candles request",
		"category", spec.Category,
		"symbol", spec.Symbol,
		"interval", spec.Interval,
		"confirmed_only", confirmedOnly,
	)
	iv, ok := specs.SupportedIntervals[spec.Interval]
	if !ok {
		return nil, nil, fmt.Errorf("bybit websocket: stream candles interval=%s: %w", spec.Interval, errorsx.ErrInvalidArgument)
	}

	url, err := resolveURL(scopePublic, spec.Category)
	if err != nil {
		w.log.Error("resolve url failed", "err", err, "category", spec.Category)
		return nil, nil, err
	}

	cmds := make(chan websocket3.Command, 8)
	streamID := fmt.Sprintf("kline.%s", spec.Category)

	w.mu.RLock()
	sess, ok := w.conns[streamID]
	if !ok {
		w.mu.RUnlock()
		r, err := router.NewRouter(router.Deps{
			Cmds:       cmds,
			MsgBuilder: bybitMsgBuilder{},
			MsgDecoder: bybitMsgDecoder{},
		}, router.WithLogger(w.log))
		if err != nil {
			w.log.Error("init router failed", "err", err)
			return nil, nil, err
		}

		s, err := websocket3.NewStream(
			url,
			cmds,
			websocket3.WithDispatch(r.Dispatch),
			websocket3.WithOnReconnect(r.OnReconnect),
			websocket3.WithReconnect(time.Second, 30*time.Second),
			websocket3.WithPingEvery(time.Second*20),
			websocket3.WithLogger(w.log),
		)
		if err != nil {
			w.log.Error("init stream failed", "err", err)
			return nil, nil, err
		}

		sess = &conn{stream: s, router: r}
		if _, err = s.Connect(ctx); err != nil {
			w.log.Error("connect stream failed", "err", err)
			return nil, nil, err
		}
		w.mu.Lock()
		w.conns[streamID] = sess
		w.mu.Unlock()
		w.log.Info("stream session created", "stream_id", streamID)
	} else {
		w.mu.RUnlock()
		w.log.Debug("reuse stream session", "stream_id", streamID)
	}

	iv = strings.ToUpper(strings.TrimSpace(iv))
	topic := fmt.Sprintf("kline.%s.%s", iv, spec.Symbol)

	pipes, err := sess.router.Acquire(ctx, []string{topic})
	if err != nil {
		w.log.Error("router acquire failed", "err", err, "topic", topic)
		return nil, nil, err
	}

	pipe, ok := pipes[topic]
	if !ok {
		return nil, nil, fmt.Errorf("bybit websocket: stream candles pipe not found for topic=%s: %w", topic, errorsx.ErrNotFound)
	}

	out := make(chan market.Candle, 256)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		sendErr := func(e error) {
			select {
			case errCh <- e:
			default:
			}
		}

		for {
			select {
			case <-ctx.Done():
				if releaseErr := sess.router.Release(ctx, []string{topic}); releaseErr != nil {
					sendErr(fmt.Errorf("release topic %s: %w", topic, releaseErr))
					w.log.Warn("router release failed", "err", releaseErr, "topic", topic)
				}
				return
			case raw, ok := <-pipe:
				if !ok {
					sendErr(fmt.Errorf("router pipe closed for topic %s", topic))
					w.log.Warn("router pipe closed", "topic", topic)
					return
				}

				var payload struct {
					Data []RawCandle `json:"data"`
				}
				if err := json.Unmarshal(raw, &payload); err != nil {
					sendErr(fmt.Errorf("decode candle batch: %w", err))
					w.log.Warn("decode candle batch failed", "err", err, "topic", topic)
					return
				}

				for _, k := range payload.Data {
					if confirmedOnly && !k.Confirm {
						continue
					}
					c, _, err := DecodeCandle(k)
					if err != nil {
						sendErr(fmt.Errorf("decode candle: %w", err))
						w.log.Warn("decode candle failed", "err", err)
						return
					}
					out <- c
				}
			}
		}
	}()

	return out, errCh, nil
}

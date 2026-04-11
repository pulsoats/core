package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/transport/websocket"
	"github.com/pulsoats/core/transport/websocket/router"
)

// categoryMaxTopics returns the max number of topics per WebSocket connection for a category.
// 0 means no limit.
// Options: 2000 topics per connection (Bybit docs).
func categoryMaxTopics(cat market.Category) int {
	if cat == specs.CategoryOption {
		return 2000
	}
	return 0
}

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

	iv = strings.ToUpper(strings.TrimSpace(iv))
	topic := fmt.Sprintf("kline.%s.%s", iv, spec.Symbol)
	streamID := fmt.Sprintf("kline.%s", spec.Category)
	maxTopics := categoryMaxTopics(spec.Category)

	sess, pipes, err := w.acquireConn(ctx, streamID, url, topic, maxTopics)
	if err != nil {
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
				if releaseErr := sess.router.Unsubscribe(ctx, pipes); releaseErr != nil {
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

// acquireConn finds an existing conn in the pool with capacity for topic,
// or creates a new one. Returns the conn and the result of Subscribe.
func (w *Client) acquireConn(
	ctx context.Context,
	streamID, url, topic string,
	maxTopics int,
) (*conn, map[string]chan json.RawMessage, error) {
	w.mu.RLock()
	pool := w.conns[streamID]
	w.mu.RUnlock()

	// Try existing conns in order.
	for _, c := range pool {
		pipes, err := c.router.Subscribe(ctx, []string{topic})
		if err == nil {
			return c, pipes, nil
		}
		if !errors.Is(err, errorsx.ErrCapacityExceeded) {
			w.log.Error("router subscribe failed", "err", err, "stream_id", streamID)
			return nil, nil, err
		}
		w.log.Debug("conn at capacity, trying next", "stream_id", streamID)
	}

	// All conns full (or no conns yet): open a new one.
	newSess, err := w.openConn(ctx, url, streamID, maxTopics)
	if err != nil {
		return nil, nil, err
	}

	pipes, err := newSess.router.Subscribe(ctx, []string{topic})
	if err != nil {
		return nil, nil, fmt.Errorf("subscribe on new conn: %w", err)
	}

	w.mu.Lock()
	w.conns[streamID] = append(w.conns[streamID], newSess)
	w.mu.Unlock()

	return newSess, pipes, nil
}

// openConn creates, connects, and returns a new conn for streamID.
func (w *Client) openConn(ctx context.Context, url, streamID string, maxTopics int) (*conn, error) {
	cmds := make(chan websocket.Command, 8)

	r, err := router.NewRouter(router.Config{
		Cmds:       cmds,
		MsgBuilder: bybitMsgBuilder{},
		MsgDecoder: bybitMsgDecoder{},
		MaxTopics:  maxTopics,
		Logger:     w.log,
	})
	if err != nil {
		w.log.Error("init router failed", "err", err)
		return nil, err
	}

	s, err := websocket.NewStream(websocket.StreamConfig{
		URL:          url,
		Cmds:         cmds,
		Dispatch:     r.Dispatch,
		OnReconnect:  r.OnReconnect,
		BackoffStart: time.Second,
		BackoffMax:   30 * time.Second,
		PingEvery:    20 * time.Second,
		PingMsg: struct {
			Op string `json:"op"`
		}{"ping"},
		Logger: w.log,
	})
	if err != nil {
		w.log.Error("init stream failed", "err", err)
		return nil, err
	}

	if _, err = s.Connect(ctx); err != nil {
		w.log.Error("connect stream failed", "err", err)
		return nil, err
	}

	w.log.Info("stream session created", "stream_id", streamID, "max_topics", maxTopics)
	return &conn{stream: s, router: r}, nil
}

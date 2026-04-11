package router

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulsoats/core/errorsx"
)

// The Acquire function subscribes (if no such topic is subscribed in WebSocket connection) to topics
// and returns map json.RawMessage channels by topic. Each call gets its own channel (fan-out).
// Returns ErrCapacityExceeded if MaxTopics is set and adding new topics would exceed the limit.
func (r *Router) Subscribe(ctx context.Context, topics []string) (map[string]chan json.RawMessage, error) {
	r.mu.Lock()

	// First pass: count genuinely new topics and validate.
	newCount := 0
	for _, topic := range topics {
		if topic == "" {
			r.mu.Unlock()
			return nil, fmt.Errorf("Subscribe: topic: %w", errorsx.ErrRequired)
		}
		if _, ok := r.pipes[topic]; !ok {
			newCount++
		}
	}

	// Capacity check: only new topics consume a slot.
	if r.maxTopics > 0 && len(r.pipes)+newCount > r.maxTopics {
		r.mu.Unlock()
		return nil, fmt.Errorf("Subscribe: max topics %d reached (%d+%d): %w",
			r.maxTopics, len(r.pipes), newCount, errorsx.ErrCapacityExceeded)
	}

	// Second pass: create channels and update pipes.
	out := make(map[string]chan json.RawMessage, len(topics))
	newTopics := make([]string, 0, newCount)
	buf := r.pipeBuf
	if buf < 1 {
		buf = 1
	}
	for _, topic := range topics {
		ch := make(chan json.RawMessage, buf)
		if p, ok := r.pipes[topic]; ok {
			p.subs[ch] = struct{}{}
		} else {
			r.pipes[topic] = &pipe{
				topic: topic,
				subs:  map[chan json.RawMessage]struct{}{ch: {}},
			}
			newTopics = append(newTopics, topic)
		}
		out[topic] = ch
	}
	r.mu.Unlock()

	if len(newTopics) > 0 {
		if err := r.sendBatched(ctx, OpSubscribe, newTopics); err != nil {
			r.log.Warn("batched subscribe had errors", "err", err)
		}
	}
	return out, nil
}

// The Release function unsubscribes the specific channels returned by Subscribe.
// When the last subscriber for a topic is removed, the WebSocket unsubscribe is sent.
func (r *Router) Unsubscribe(ctx context.Context, subs map[string]chan json.RawMessage) error {
	unsubTopics := make([]string, 0, len(subs))
	var toClose []chan json.RawMessage

	r.mu.Lock()
	for topic, ch := range subs {
		if topic == "" {
			r.mu.Unlock()
			return fmt.Errorf("Unsubscribe: topic: %w", errorsx.ErrRequired)
		}
		p, ok := r.pipes[topic]
		if !ok {
			continue
		}
		if _, exists := p.subs[ch]; exists {
			delete(p.subs, ch)
			toClose = append(toClose, ch)
		}
		if len(p.subs) == 0 {
			delete(r.pipes, topic)
			unsubTopics = append(unsubTopics, topic)
		}
	}
	r.mu.Unlock()

	for _, ch := range toClose {
		close(ch)
	}

	if len(unsubTopics) == 0 {
		return nil
	}

	if err := r.sendBatched(ctx, OpUnsubscribe, unsubTopics); err != nil {
		r.log.Warn("batched unsubscribe had errors", "err", err)
	}
	return nil
}

// OnReconnect subscribes on existing topics from Router
func (r *Router) OnReconnect(ctx context.Context) error {
	r.mu.Lock()
	firstConnect := !r.connected
	if !r.connected {
		r.connected = true
	}
	r.state = ConnStateConnected

	// Snapshot topics already covered by in-flight pending requests (raced Subscribe calls
	// that arrived on the new connection before OnReconnect was called).
	inPending := make(map[string]struct{})
	for _, req := range r.pending {
		for _, t := range req.topics {
			inPending[t] = struct{}{}
		}
	}
	// Stale pending reqs from the old connection are now invalid; reset.
	r.pending = make(map[string]*pendingReq)

	restartTopics := make([]string, 0, len(r.pipes))
	for _, p := range r.pipes {
		if _, skip := inPending[p.topic]; !skip {
			restartTopics = append(restartTopics, p.topic)
		}
	}
	r.mu.Unlock()

	if firstConnect {
		// Первое соединение: ждём обычные subscribe из Acquire, чтобы не дублировать запросы.
		return nil
	}

	if len(restartTopics) == 0 {
		return nil
	}
	if err := r.sendBatched(ctx, OpSubscribe, restartTopics); err != nil {
		r.log.Warn("batched resubscribe had errors", "err", err)
	}
	return nil
}

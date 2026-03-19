package router

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulsoats/core/domain/derrors"
)

// The Acquire function subscribes (if no such topic is subscribed in WebSocket connection) to topics
// and returns map json.RawMessage channels by topic.
func (r *Router) Acquire(ctx context.Context, topics []string) (map[string]chan json.RawMessage, error) {
	out := make(map[string]chan json.RawMessage, len(topics))
	newTopics := make([]string, 0, len(topics))

	r.mu.Lock()
	for _, topic := range topics {
		if topic == "" {
			r.mu.Unlock()
			return nil, fmt.Errorf("%w: topic is required", derrors.ErrRequired)
		}
		if p, ok := r.pipes[topic]; ok {
			p.ref++
			out[topic] = p.ch
			continue
		}
		buf := r.pipeBuf
		if buf < 1 {
			buf = 1
		}
		ch := make(chan json.RawMessage, buf)
		r.pipes[topic] = &pipe{topic: topic, ch: ch, ref: 1}
		out[topic] = ch
		newTopics = append(newTopics, topic)
	}
	r.mu.Unlock()

	if len(newTopics) > 0 {
		if err := r.sendBatched(ctx, OpSubscribe, newTopics); err != nil {
			r.log.Warn("batched subscribe had errors", "err", err)
		}
	}
	return out, nil
}

// The Release function unsubscribes from provided topic strings.
func (r *Router) Release(ctx context.Context, topics []string) error {
	unsubTopics := make([]string, 0, len(topics))
	var toClose []*pipe

	r.mu.Lock()
	for _, topic := range topics {
		if topic == "" {
			r.mu.Unlock()
			return fmt.Errorf("%w: topic is required", derrors.ErrRequired)
		}
		if p, ok := r.pipes[topic]; ok {
			p.ref--
			if p.ref == 0 {
				delete(r.pipes, topic)
				unsubTopics = append(unsubTopics, topic)
				toClose = append(toClose, p)
			}
		}
	}
	r.mu.Unlock()

	for _, p := range toClose {
		p.closeOnce()
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
	r.pending = make(map[string]*pendingReq)
	restartTopics := make([]string, 0, len(r.pipes))
	for _, p := range r.pipes {
		restartTopics = append(restartTopics, p.topic)
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

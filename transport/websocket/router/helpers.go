package router

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pulsoats/core/transport/websocket"
)

func (r *Router) removeTopics(topics []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range topics {
		if p, ok := r.pipes[t]; ok {
			delete(r.pipes, t)
			close(p.ch)
		}
	}
}

func (p *pipe) closeOnce() {
	p.once.Do(func() { close(p.ch) })
}

func chunkStrings(xs []string, size int) [][]string {
	if len(xs) == 0 {
		return nil
	}
	if size <= 0 || size >= len(xs) {
		return [][]string{xs}
	}
	out := make([][]string, 0, (len(xs)+size-1)/size)
	for i := 0; i < len(xs); i += size {
		j := i + size
		if j > len(xs) {
			j = len(xs)
		}
		out = append(out, xs[i:j])
	}
	return out
}

// sendBatched шлёт запросы пакетами topicsPerReq с троттлингом reqPerSec.
// op — OpSubscribe / OpUnsubscribe.
// pending пишется ТОЛЬКО после успешной отправки в r.cmds.
func (r *Router) sendBatched(ctx context.Context, op Op, allTopics []string) error {
	if len(allTopics) == 0 {
		return nil
	}

	batches := chunkStrings(allTopics, r.topicsPerReq)

	var tick *time.Ticker
	if r.reqPerSec > 0 {
		tick = time.NewTicker(time.Second)
		defer tick.Stop()
	}
	sentThisSecond := 0

	waitIfNeeded := func() error {
		if tick == nil {
			return nil
		}
		if sentThisSecond == r.reqPerSec {
			select {
			case <-tick.C:
				sentThisSecond = 0
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	var errs []error

	for _, topics := range batches {
		if err := waitIfNeeded(); err != nil {
			errs = append(errs, err)
			break
		}
		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			break
		default:
		}

		// Bybit ограничивает длину req_id 32 символами, поэтому убираем дефисы UUID.
		reqID := strings.ReplaceAll(uuid.New().String(), "-", "")
		req, err := r.msgBuilder.Build(reqID, op, topics)
		if err != nil {
			r.log.Error("build batched request failed", "op", op, "err", err)
			errs = append(errs, err)
			continue
		}

		// сначала пробуем отправить
		sent := false
		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
		case r.cmds <- websocket.Command{Op: websocket.CmdSendJSON, Payload: req}:
			sent = true
		default:
			// мягкая попытка с ожиданием
			t := time.NewTimer(300 * time.Millisecond)
			select {
			case <-ctx.Done():
				errs = append(errs, ctx.Err())
			case r.cmds <- websocket.Command{Op: websocket.CmdSendJSON, Payload: req}:
				sent = true
			case <-t.C:
				r.log.Warn("cmds channel full, dropping batched request", "op", op, "req_id", reqID)
				errs = append(errs, fmt.Errorf("cmds full for %s %s", op, reqID))
			}
			t.Stop()
		}

		if sent {
			// только теперь — в pending
			r.mu.Lock()
			r.pending[reqID] = &pendingReq{
				reqID:  reqID,
				op:     op,
				topics: topics,
				sentAt: time.Now(),
			}
			r.mu.Unlock()
			sentThisSecond++
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

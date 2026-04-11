package router

import (
	"context"
	"encoding/json"
	"strings"
)

// The Dispatch is routing message to pipe by topic
func (r *Router) Dispatch(ctx context.Context, raw json.RawMessage) error {
	msg, err := r.msgDecoder.Decode(ctx, raw)
	if err != nil {
		return err
	}

	switch msg.Kind {
	case StreamMsgKindData:
		if msg.Topic == "" {
			return nil
		}
		r.mu.RLock()
		p, ok := r.pipes[msg.Topic]
		if !ok {
			r.mu.RUnlock()
			return nil
		}
		for ch := range p.subs {
			select {
			case ch <- raw:
			default:
				r.log.Warn("pipe channel full, dropping message", "topic", msg.Topic)
			}
		}
		r.mu.RUnlock()
		return nil
	case StreamMsgKindAck:
		if msg.ReqID == "" {
			return nil
		}
		rawPayload := string(msg.Raw)
		retMsg := strings.ToLower(msg.RetMsg)
		r.mu.RLock()
		pReq, ok := r.pending[msg.ReqID]
		r.mu.RUnlock()
		if !ok {
			return nil
		}
		if msg.Success {
			if len(msg.FailedTopics) > 0 {
				r.log.Warn("ack partially failed", "req_id", msg.ReqID, "topics", msg.FailedTopics, "raw", rawPayload)
				r.removeTopics(msg.FailedTopics)
			}
			r.mu.Lock()
			delete(r.pending, msg.ReqID)
			r.mu.Unlock()
			return nil
		}
		// Bybit отвечает success=false c ret_msg "error:already subscribed" при повторной подписке.
		if retMsg != "" && strings.Contains(retMsg, "already subscribed") {
			r.log.Debug("ack already subscribed", "req_id", msg.ReqID, "topics", pReq.topics, "raw", rawPayload)
			r.mu.Lock()
			delete(r.pending, msg.ReqID)
			r.mu.Unlock()
			return nil
		}
		r.log.Warn("ack failed", "req_id", msg.ReqID, "op", pReq.op, "topics", pReq.topics, "raw", rawPayload)
		r.removeTopics(pReq.topics)
		r.mu.Lock()
		delete(r.pending, msg.ReqID)
		r.mu.Unlock()
		return nil
	default:
		return nil
	}
}

package router

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/transport/websocket"
)

type Router struct {
	mu sync.RWMutex

	cmds         chan websocket.Command
	pipes        map[string]*pipe
	pending      map[string]*pendingReq
	msgBuilder   MsgBuilder
	msgDecoder   MsgDecoder
	pipeBuf      int
	topicsPerReq int
	reqPerSec    int
	pendingTTL   time.Duration
	maxTopics    int
	log          *slog.Logger
	connected    bool
	state        ConnState
}

// State returns the current connection state.
func (r *Router) State() ConnState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

func NewRouter(cfg Config) (*Router, error) {
	if cfg.Cmds == nil {
		return nil, fmt.Errorf("NewRouter: cmds: %w", errorsx.ErrRequired)
	}
	if cfg.MsgBuilder == nil {
		return nil, fmt.Errorf("NewRouter: msg builder: %w", errorsx.ErrRequired)
	}
	if cfg.MsgDecoder == nil {
		return nil, fmt.Errorf("NewRouter: msg decoder: %w", errorsx.ErrRequired)
	}

	if cfg.PipeBuf <= 0 {
		cfg.PipeBuf = 64
	}
	if cfg.TopicsPerReq <= 0 {
		cfg.TopicsPerReq = 10
	}
	if cfg.ReqPerSec < 0 {
		cfg.ReqPerSec = 1
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.DiscardHandler)
	}

	return &Router{
		cmds:         cfg.Cmds,
		pipes:        make(map[string]*pipe),
		pending:      make(map[string]*pendingReq),
		msgBuilder:   cfg.MsgBuilder,
		msgDecoder:   cfg.MsgDecoder,
		pipeBuf:      cfg.PipeBuf,
		topicsPerReq: cfg.TopicsPerReq,
		reqPerSec:    cfg.ReqPerSec,
		pendingTTL:   cfg.PendingTTL,
		maxTopics:    cfg.MaxTopics,
		log:          cfg.Logger.With("component", "ws.router"),
	}, nil
}

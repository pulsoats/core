package router

import (
	"fmt"
	"log/slog"
	"sync"

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
	log          *slog.Logger
	connected    bool
}

// NewRouter — конструктор с дефолтами, валидацией и нормализацией.
func NewRouter(deps Deps, opts ...Option) (*Router, error) {
	if deps.Cmds == nil {
		return nil, fmt.Errorf("websocket router: cmds channel: %w", errorsx.ErrRequired)
	}
	if deps.MsgBuilder == nil {
		return nil, fmt.Errorf("websocket router: msg builder: %w", errorsx.ErrRequired)
	}
	if deps.MsgDecoder == nil {
		return nil, fmt.Errorf("websocket router: msg decoder: %w", errorsx.ErrRequired)
	}

	c := &cfg{
		pipeBuf:      64,
		topicsPerReq: 10,
		reqPerSec:    10,
		logger:       slog.New(slog.DiscardHandler),
		maxPipeBuf:   1 << 20, // 1MB «мягкая» верхняя граница буфера
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	c.cmds = deps.Cmds
	c.msgBuilder = deps.MsgBuilder
	c.msgDecoder = deps.MsgDecoder

	if c.pipeBuf <= 0 {
		c.pipeBuf = 64
	}
	if c.pipeBuf > c.maxPipeBuf {
		c.pipeBuf = c.maxPipeBuf
	}
	if c.topicsPerReq <= 0 {
		c.topicsPerReq = 10
	}
	if c.reqPerSec < 0 {
		c.reqPerSec = 1
	}

	r := &Router{
		cmds:         c.cmds,
		pipes:        make(map[string]*pipe),
		pending:      make(map[string]*pendingReq),
		msgBuilder:   c.msgBuilder,
		msgDecoder:   c.msgDecoder,
		pipeBuf:      c.pipeBuf,
		topicsPerReq: c.topicsPerReq,
		reqPerSec:    c.reqPerSec,
		log:          c.logger.With("component", "ws.router"),
	}

	return r, nil
}

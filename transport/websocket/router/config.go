package router

import (
	"fmt"

	"github.com/pulsoats/core/domain/derrors"
	"github.com/pulsoats/core/transport/websocket"
)

type Option func(*cfg) error

type cfg struct {
	cmds         chan websocket.Command
	msgBuilder   MsgBuilder
	msgDecoder   MsgDecoder
	pipeBuf      int
	topicsPerReq int
	reqPerSec    int
	logger       Logger
	maxPipeBuf   int
}

func WithMsgDecoder(d MsgDecoder) Option {
	return func(c *cfg) error {
		if d == nil {
			return fmt.Errorf("%w: MsgDecoder cannot be nil", derrors.ErrRequired)
		}
		c.msgDecoder = d
		return nil
	}
}

func WithPipeBuf(n int) Option {
	return func(c *cfg) error {
		c.pipeBuf = n
		return nil
	}
}

func WithLimits(topicsPerReq, reqPerSec int) Option {
	return func(c *cfg) error {
		c.topicsPerReq = topicsPerReq
		c.reqPerSec = reqPerSec
		return nil
	}
}

func WithLogger(l Logger) Option {
	return func(c *cfg) error {
		if l == nil {
			l = nopLogger{}
		}
		c.logger = l
		return nil
	}
}

package router

import (
	"encoding/json"
	"time"
)

type Op string

const (
	OpSubscribe   Op = "subscribe"
	OpUnsubscribe Op = "unsubscribe"
)

// ConnState describes the current connection state of the Router.
type ConnState int

const (
	ConnStateDisconnected ConnState = iota
	ConnStateConnecting
	ConnStateConnected
)

type pipe struct {
	topic string
	subs  map[chan json.RawMessage]struct{}
}

type pendingReq struct {
	reqID  string
	op     Op
	topics []string
	connID string
	sentAt time.Time
}

type StreamMsgKind int

const (
	StreamMsgKindUnknown StreamMsgKind = iota
	StreamMsgKindData
	StreamMsgKindAck
)

type StreamMsg struct {
	Kind         StreamMsgKind
	Topic        string
	Op           string
	Success      bool
	ReqID        string
	Type         string
	RetMsg       string
	FailedTopics []string
	Data         json.RawMessage
	Raw          json.RawMessage
}

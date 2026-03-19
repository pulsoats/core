package router

import (
	"encoding/json"
	"sync"
	"time"
)

type Op string

const (
	OpSubscribe   Op = "subscribe"
	OpUnsubscribe Op = "unsubscribe"
)

type pipe struct {
	topic string
	ch    chan json.RawMessage
	ref   int

	once sync.Once
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

package run

type StatusCode int

const (
	StatusCodeUnspecified StatusCode = iota
	StatusCodePending
	StatusCodeRunning
	StatusCodeDone
	StatusCodeFailed
)

func ParseStatusCode(code int) (StatusCode, bool) {
	sc := StatusCode(code)
	switch sc {
	case StatusCodeUnspecified, StatusCodePending, StatusCodeRunning, StatusCodeDone, StatusCodeFailed:
		return sc, true
	default:
		return StatusCodeUnspecified, false
	}
}

const (
	StatusMessageUnspecified = "unspecified"
	StatusMessagePending     = "pending"
	StatusMessageRunning     = "running"
	StatusMessageDone        = "done"
)

type Status struct {
	Code    StatusCode
	Message string
}

var (
	StatusUnspecified = Status{Code: StatusCodeUnspecified, Message: StatusMessageUnspecified}
	StatusPending     = Status{Code: StatusCodePending, Message: StatusMessagePending}
	StatusRunning     = Status{Code: StatusCodeRunning, Message: StatusMessageRunning}
	StatusDone        = Status{Code: StatusCodeDone, Message: StatusMessageDone}
)

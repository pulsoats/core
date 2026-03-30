package websocket

import "log/slog"

var nopLogger = slog.New(slog.DiscardHandler)

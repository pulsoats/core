package logx

import (
	"log/slog"
	"strings"
)

// Discard returns a *slog.Logger that silently drops all records.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// ParseLevel converts a string to slog.Level. Defaults to slog.LevelInfo.
func ParseLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(s))); err != nil {
		return slog.LevelInfo
	}
	return l
}

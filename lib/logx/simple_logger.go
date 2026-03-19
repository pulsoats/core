package logx

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// SimpleLoggerConfig configures NewSimpleLogger.
type SimpleLoggerConfig struct {
	Writer io.Writer
	Level  Level
	JSON   bool
}

// NewSimpleLogger returns a basic Logger implementation that writes either console or JSON lines.
func NewSimpleLogger(cfg SimpleLoggerConfig) Logger {
	if cfg.Writer == nil {
		cfg.Writer = io.Discard
	}
	return &simpleLogger{cfg: cfg}
}

type simpleLogger struct {
	cfg SimpleLoggerConfig
	mu  sync.Mutex
}

func (l *simpleLogger) Debug(msg string, kv ...any) { l.log(LevelDebug, "debug", msg, kv...) }
func (l *simpleLogger) Info(msg string, kv ...any)  { l.log(LevelInfo, "info", msg, kv...) }
func (l *simpleLogger) Warn(msg string, kv ...any)  { l.log(LevelWarn, "warn", msg, kv...) }
func (l *simpleLogger) Error(msg string, kv ...any) { l.log(LevelError, "error", msg, kv...) }

func (l *simpleLogger) log(level Level, levelLabel, msg string, kv ...any) {
	if !level.enabled(l.cfg.Level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	if l.cfg.JSON {
		line := make(map[string]any, len(kv)/2+3)
		line["ts"] = ts
		line["level"] = levelLabel
		line["msg"] = msg
		for i := 0; i+1 < len(kv); i += 2 {
			key, ok := kv[i].(string)
			if !ok {
				continue
			}
			line[key] = kv[i+1]
		}
		payload, err := json.Marshal(line)
		if err != nil {
			fmt.Fprintf(l.cfg.Writer, "%s [%s] %s marshal_error=%v\n", ts, strings.ToUpper(levelLabel), msg, err)
			return
		}
		_, _ = l.cfg.Writer.Write(append(payload, '\n'))
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s [%s] %s", ts, strings.ToUpper(levelLabel), msg)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(key)
		b.WriteByte('=')
		fmt.Fprintf(&b, "%v", kv[i+1])
	}
	b.WriteByte('\n')
	_, _ = l.cfg.Writer.Write([]byte(b.String()))
}

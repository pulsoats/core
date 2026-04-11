package router

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_removeTopics(t *testing.T) {
	type fixture struct {
		all map[string]chan json.RawMessage
	}

	newRouter := func(topics ...string) (*Router, fixture) {
		pipes := make(map[string]*pipe, len(topics))
		chans := make(map[string]chan json.RawMessage, len(topics))
		for _, tp := range topics {
			ch := make(chan json.RawMessage, 1)
			pipes[tp] = &pipe{topic: tp, subs: map[chan json.RawMessage]struct{}{ch: {}}}
			chans[tp] = ch
		}
		return &Router{pipes: pipes}, fixture{all: chans}
	}

	assertClosed := func(t *testing.T, ch chan json.RawMessage, msg string) {
		t.Helper()
		select {
		case _, ok := <-ch:
			assert.False(t, ok, msg)
		default:
			t.Fatalf("%s (non-blocking read should be ready on closed channel)", msg)
		}
	}

	assertOpen := func(t *testing.T, ch chan json.RawMessage, msg string) {
		t.Helper()
		require.NotPanics(t, func() {
			ch <- json.RawMessage(`"probe"`)
		}, msg)
		select {
		case <-ch:
		case <-time.After(20 * time.Millisecond):
		}
	}

	tests := []struct {
		name          string
		initialTopics []string
		remove        []string
		wantRemain    []string
		wantRemoved   []string
	}{
		{
			name:          "remove single existing",
			initialTopics: []string{"kline.1.BTCUSDT", "kline.3.BTCUSDT"},
			remove:        []string{"kline.3.BTCUSDT"},
			wantRemain:    []string{"kline.1.BTCUSDT"},
			wantRemoved:   []string{"kline.3.BTCUSDT"},
		},
		{
			name:          "remove multiple existing",
			initialTopics: []string{"a", "b", "c"},
			remove:        []string{"a", "c"},
			wantRemain:    []string{"b"},
			wantRemoved:   []string{"a", "c"},
		},
		{
			name:          "remove non-existing",
			initialTopics: []string{"only"},
			remove:        []string{"ghost"},
			wantRemain:    []string{"only"},
			wantRemoved:   []string{},
		},
		{
			name:          "empty remove list",
			initialTopics: []string{"x", "y"},
			remove:        []string{},
			wantRemain:    []string{"x", "y"},
			wantRemoved:   []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, fx := newRouter(tt.initialTopics...)

			r.removeTopics(tt.remove)

			for _, tp := range tt.wantRemoved {
				_, ok := r.pipes[tp]
				assert.Falsef(t, ok, "topic %q must be removed from r.pipes", tp)
				assertClosed(t, fx.all[tp], "removed channel must be closed")
			}

			for _, tp := range tt.wantRemain {
				_, ok := r.pipes[tp]
				assert.Truef(t, ok, "topic %q must remain in r.pipes", tp)
				assertOpen(t, fx.all[tp], "kept channel must be open")
			}

			assert.Len(t, r.pipes, len(tt.wantRemain))
		})
	}
}

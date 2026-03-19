package csv

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

const defaultBufferSize = 1 << 20

type Encoder[T any] func(T) []string

type Option func(*config)

type config struct {
	header            []string
	bufferSize        int
	autoFlushInterval time.Duration
	forceWriteHeader  bool
	syncOnClose       bool
}

func WithHeader(header []string) Option {
	return func(c *config) {
		if header == nil {
			c.header = nil
			return
		}
		c.header = append([]string(nil), header...)
	}
}

func WithBufferSize(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.bufferSize = size
		}
	}
}

func WithAutoFlush(interval time.Duration) Option {
	return func(c *config) {
		if interval < 0 {
			return
		}
		c.autoFlushInterval = interval
	}
}

func WithForceWriteHeader(force bool) Option {
	return func(c *config) {
		c.forceWriteHeader = force
	}
}

func WithSyncOnClose(sync bool) Option {
	return func(c *config) {
		c.syncOnClose = sync
	}
}

type Writer[T any] struct {
	mu sync.Mutex

	bw     *bufio.Writer
	cw     *csv.Writer
	encode Encoder[T]

	header        []string
	headerPending bool

	autoFlushInterval time.Duration
	autoFlushStop     chan struct{}
	autoFlushOnce     sync.Once
	autoFlushWG       sync.WaitGroup

	syncOnClose bool
	closer      io.Closer
	syncer      interface{ Sync() error }

	err    error
	closed bool
}

func NewWriter[T any](dst io.Writer, encode Encoder[T], opts ...Option) (*Writer[T], error) {
	cfg := config{
		bufferSize: defaultBufferSize,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if encode == nil {
		return nil, ErrNilEncoder
	}

	bw := bufio.NewWriterSize(dst, cfg.bufferSize)
	cw := csv.NewWriter(bw)

	w := &Writer[T]{
		bw:                bw,
		cw:                cw,
		encode:            encode,
		header:            cfg.header,
		autoFlushInterval: cfg.autoFlushInterval,
		syncOnClose:       cfg.syncOnClose,
	}

	if c, ok := dst.(io.Closer); ok {
		w.closer = c
	}
	if syncer, ok := dst.(interface{ Sync() error }); ok {
		w.syncer = syncer
	}

	w.headerPending = len(w.header) > 0
	if !cfg.forceWriteHeader && len(w.header) > 0 {
		if statter, ok := dst.(interface{ Stat() (os.FileInfo, error) }); ok {
			fi, err := statter.Stat()
			if err != nil {
				return nil, err
			}
			if fi.Size() > 0 {
				w.headerPending = false
			}
		}
	}

	if cfg.autoFlushInterval > 0 {
		w.autoFlushStop = make(chan struct{})
		w.autoFlushWG.Add(1)
		go w.autoFlush()
	}

	return w, nil
}

func (w *Writer[T]) autoFlush() {
	defer w.autoFlushWG.Done()

	ticker := time.NewTicker(w.autoFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = w.Flush()
		case <-w.autoFlushStop:
			return
		}
	}
}

func (w *Writer[T]) Write(ctx context.Context, v T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	record := w.encode(v)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer closed")
	}
	if w.err != nil {
		return w.err
	}
	if err := w.ensureHeaderLocked(); err != nil {
		return err
	}
	if err := w.cw.Write(record); err != nil {
		w.recordErrorLocked(err)
		return w.err
	}
	return nil
}

func (w *Writer[T]) WriteAll(ctx context.Context, vs []T) (int, error) {
	if err := w.Err(); err != nil {
		return 0, err
	}

	if err := ctx.Err(); err != nil {
		return 0, err
	}

	records := make([][]string, 0, len(vs))
	for i, v := range vs {
		if err := ctx.Err(); err != nil {
			return i, err // НЕ recordErrorLocked
		}
		records = append(records, w.encode(v))
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return 0, w.err
	}

	if err := w.ensureHeaderLocked(); err != nil {
		return 0, err
	}

	for i, rec := range records {
		if err := ctx.Err(); err != nil {
			w.recordErrorLocked(err)
			return i, w.err
		}

		if err := w.cw.Write(rec); err != nil {
			w.recordErrorLocked(err)
			return i, w.err
		}
	}

	return len(records), w.err
}

func (w *Writer[T]) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errors.New("writer closed")
	}
	return w.flushLocked()
}

func (w *Writer[T]) flushLocked() error {
	if w.err != nil {
		return w.err
	}

	w.cw.Flush()
	if err := w.cw.Error(); err != nil {
		w.recordErrorLocked(err)
		return w.err
	}

	if err := w.bw.Flush(); err != nil {
		w.recordErrorLocked(err)
	}

	return w.err
}

func (w *Writer[T]) ensureHeaderLocked() error {
	if !w.headerPending {
		return nil
	}

	if len(w.header) == 0 {
		w.headerPending = false
		return nil
	}

	if err := w.cw.Write(w.header); err != nil {
		w.recordErrorLocked(err)
		return w.err
	}

	w.headerPending = false
	return w.err
}

func (w *Writer[T]) Err() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

func (w *Writer[T]) Close() error {
	w.stopAutoFlush()

	w.mu.Lock()
	if w.closed {
		err := w.err
		w.mu.Unlock()
		return err
	}
	defer w.mu.Unlock()

	w.closed = true

	_ = w.flushLocked()

	if w.syncOnClose && w.syncer != nil {
		if err := w.syncer.Sync(); err != nil {
			w.recordErrorLocked(err)
		}
	}

	if w.closer != nil {
		if err := w.closer.Close(); err != nil {
			w.recordErrorLocked(err)
		}
	}

	return w.err
}

func (w *Writer[T]) stopAutoFlush() {
	if w.autoFlushStop == nil {
		return
	}

	w.autoFlushOnce.Do(func() {
		close(w.autoFlushStop)
	})
	w.autoFlushWG.Wait()
}

func (w *Writer[T]) recordErrorLocked(err error) {
	if err != nil && w.err == nil {
		w.err = err
	}
}

var ErrNilEncoder = errors.New("csv: encoder is nil")

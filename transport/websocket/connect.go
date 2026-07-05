package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pulsoats/core/errorsx"
)

var errStopStream = errors.New("stream stopped")

func (s *Stream) Connect(ctx context.Context) (chan json.RawMessage, error) {
	var out chan json.RawMessage
	if s.dispatch == nil {
		if s.outBuf <= 0 {
			s.outBuf = 256
		}
		out = make(chan json.RawMessage, s.outBuf)
	}

	go func() {
		if out != nil {
			defer close(out)
		}

		backoff := s.backoffStart

		for {
			connCtx, cancelConn := context.WithCancel(ctx)

			s.log.Info("dial websocket", "url", s.url)
			conn, _, dialErr := websocket.Dial(connCtx, s.url, s.dialOptions)
			if dialErr != nil {
				s.log.Warn("dial failed", "err", dialErr)
				cancelConn()
				if !sleepBackoff(ctx, &backoff, s.backoffMax) {
					return
				}
				continue
			}
			s.log.Info("stream connected", "url", s.url)

			if s.auth != nil {
				payload, err := s.auth(connCtx)
				if err != nil {
					s.log.Error("auth payload build failed", "err", err)
					_ = conn.Close(websocket.StatusNormalClosure, "auth failed")
					cancelConn()
					if !sleepBackoff(ctx, &backoff, s.backoffMax) {
						return
					}
					continue
				}
				if payload != nil {
					wctx, cancel := context.WithTimeout(connCtx, 5*time.Second)
					writeErr := wsjson.Write(wctx, conn, payload)
					cancel()
					if writeErr != nil {
						s.log.Error("auth write failed", "err", writeErr)
						_ = conn.Close(websocket.StatusAbnormalClosure, "auth write failed")
						cancelConn()
						if !sleepBackoff(ctx, &backoff, s.backoffMax) {
							return
						}
						continue
					}
				}
			}

			backoff = s.backoffStart

			if s.onReconnect != nil {
				if err := s.onReconnect(ctx); err != nil {
					s.log.Error("onReconnect failed", "err", err)
					_ = conn.Close(websocket.StatusAbnormalClosure, "onReconnect failed")
					cancelConn()
					if !sleepBackoff(ctx, &backoff, s.backoffMax) {
						return
					}
					continue
				}
			}

			readCh := make(chan json.RawMessage, 256)
			errCh := make(chan error, 1)

			// reader pump
			go func(c *websocket.Conn, rc chan<- json.RawMessage, ec chan<- error, cctx context.Context) {
				defer close(rc)
				for {
					var msg json.RawMessage
					if err := wsjson.Read(cctx, c, &msg); err != nil {
						if closeErr, ok := errors.AsType[*websocket.CloseError](err); ok {
							s.log.Info("websocket closed by server",
								"code", closeErr.Code,
								"reason", closeErr.Reason,
							)
						} else if normalCloseErr(err) {
							s.log.Info("websocket read ended", "err", err)
						} else {
							s.log.Warn("websocket read error", "err", err)
						}

						if normalCloseErr(err) {
							select {
							case ec <- nil:
							default:
							}
						} else {
							select {
							case ec <- err:
							default:
							}
						}
						return
					}

					// опционально: чтобы не зависнуть, если main-loop умер
					select {
					case rc <- msg:
					case <-cctx.Done():
						return
					}
				}
			}(conn, readCh, errCh, connCtx)

			var pingStop chan struct{}
			if s.pingEvery > 0 {
				pingStop = make(chan struct{})
				go func(c *websocket.Conn, cctx context.Context, stop <-chan struct{}, every time.Duration) {
					t := time.NewTicker(every)
					defer t.Stop()
					for {
						select {
						case <-t.C:
							pingCtx, cancel := context.WithTimeout(cctx, 5*time.Second)
							var pingErr error
							if s.pingMsg != nil {
								pingErr = wsjson.Write(pingCtx, c, s.pingMsg)
							} else {
								pingErr = c.Ping(pingCtx)
							}
							cancel()
							if pingErr != nil {
								// закрываем соединение — main-loop подберёт ошибку
								_ = c.Close(websocket.StatusAbnormalClosure, "ping failed")
								return
							}
						case <-cctx.Done():
							return
						case <-stop:
							return
						}
					}
				}(conn, connCtx, pingStop, s.pingEvery)
			}

			runErr := func() error {
				for {
					select {
					case raw, ok := <-readCh:
						if !ok {
							select {
							case e := <-errCh:
								if normalCloseErr(e) {
									return nil
								}
								return e
							default:
								return fmt.Errorf("connect: reader: %w", errors.Join(errorsx.ErrInternal, errorsx.ErrClosed))
							}
						}

						if s.dispatch != nil {
							_ = s.dispatch(ctx, raw)
							continue
						}

						select {
						case out <- raw:
						default:
							s.log.Warn("out channel full, dropping raw message")
						}

					case cmd := <-s.cmds:
						switch cmd.Op {
						case CmdClose:
							_ = conn.Close(websocket.StatusNormalClosure, "bye")
							return errStopStream
						case CmdSendJSON:
							wctx, cancel := context.WithTimeout(connCtx, 5*time.Second)
							err := wsjson.Write(wctx, conn, cmd.Payload)
							cancel()
							if err != nil {
								s.log.Warn("write command failed", "err", err)
								_ = conn.Close(websocket.StatusAbnormalClosure, "write failed")
								return err
							}
						}

					case <-ctx.Done():
						_ = conn.Close(websocket.StatusNormalClosure, "ctx done")
						return ctx.Err()

					case <-connCtx.Done():
						_ = conn.Close(websocket.StatusNormalClosure, "conn ctx done")
						return connCtx.Err()
					}
				}
			}()

			if pingStop != nil {
				close(pingStop)
			}
			cancelConn()
			_ = conn.Close(websocket.StatusNormalClosure, "end session")

			// 1) Внешний ctx отменён или явная остановка через CmdClose — завершаем стрим.
			if ctx.Err() != nil || errors.Is(runErr, errStopStream) {
				s.log.Info("stream loop ended", "err", runErr)
				return
			}

			// 2) Нормальное завершение сессии (сервер закрыл) — reconnect.
			if runErr == nil {
				s.log.Info("session ended normally; reconnecting")
				if !sleepBackoff(ctx, &backoff, s.backoffMax) {
					return
				}
				continue
			}

			// 3) Любая другая ошибка — лог + reconnect с backoff.
			s.log.Warn("session error; reconnecting", "err", runErr)
			if !sleepBackoff(ctx, &backoff, s.backoffMax) {
				return
			}
			continue
		}
	}()

	return out, nil
}

func normalCloseErr(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	if _, ok := errors.AsType[*websocket.CloseError](err); ok {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

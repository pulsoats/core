## pulsoats-core

A shared toolkit for exchange integrations, detectors, and WebSocket transport.

### Layer Overview

1. **domain/** — pure contracts and business entities.
2. **exchanges/** — concrete exchange adapters (REST + WS).
3. **transport/websocket/** — reusable stream/connection/router logic.
4. **lib/** — cross-cutting helpers (csv, format, parse, error roots).

### Domain Layer

| Package | Contents |
| --- | --- |
| `domain/market` | `Category`, `Interval`, `Spec`, `CandleSpec`. Intervals are `time.Duration` constants with JSON helpers. |
| `domain/exchange` | Interface `API`: `Candles`, `StreamCandles(ctx, spec, confirmedOnly) (chan market.Candle, <-chan error, error)`, `FeeRate`, `InstrumentExists`. |
| `domain/detect` | Registry for detectors (`detectors.Registry`). `Candle()` view enforces `DetectorKindCandle`. |
| `domain/derrors` | Domain-only roots: `ErrNotFound`, `ErrInvalidArgument`, `ErrRequired`, `ErrAlreadyExists`, `ErrUnauthorized`. Use `lib/errorsx` for transport/internal failures. |

### Bybit Exchange

| Area | Highlights                                                                                                                                                                                                                                                                           |
| --- |--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| REST (`exchanges/bybit/rest`) | `Candles` handles pagination / interval validation. `FeeRate`, `InstrumentExists` translate HTTP errors into domain errors.                                                                                                                                                          |
| WebSocket (`exchanges/bybit/websocket`) | `StreamCandles` shares connections per category, builds topics `kline.<interval>.<symbol>`, returns data/error channels. `response.go` decodes frames into `router.StreamMsg`. `resolve.go` maps scope → endpoint. Connetions should be divided by scope (private/public/trade etc.) |
| Client (`exchanges/bybit/client.go`) | Combines REST + WS to implement `domain/exchange.API`.                                                                                                                                                                                                                               |

### Transport / WebSocket

| Component | Description |
| --- | --- |
| `stream.go` | Low-level WebSocket manager (`github.com/coder/websocket`). Options: `WithAuth(func(ctx) (any, error))`, `WithDispatch`, `WithReconnect`, `WithOutBuf`, `WithPingEvery`. Auth function returns payload; `Stream` writes it via `wsjson`. |
| `connect.go` | Dial → auth payload → `onReconnect` → reader pump → command loop (`CmdSendJSON`, `CmdClose`). |
| `router/` | Tracks topics per connection: `Acquire`/`Release`, `OnReconnect`, `Dispatch`. `sendBatched` batches subscribe/unsubscribe with rate limits. `ports.go` defines `MsgBuilder`/`MsgDecoder` contracts and a light `Logger`. `StreamMsgKind` differentiates data vs ack frames. |

### Lib Helpers

| Package | Purpose |
| --- | --- |
| `lib/errorsx` | Technical roots: `ErrInternal`, `ErrClosed`, `ErrNotImplemented`. |
| `lib/csv` | CSV encoders for candles/signals plus buffered writers (headers, buffer size, auto flush). |
| `lib/logx` | Shared `Logger` interface plus a simple `NewSimpleLogger` implementation. External apps can adapt any engine (zerolog, slog, zap) to this interface and pass it into core components. |
| `lib/format`, `lib/parse`, `lib/units` | Money/delta helpers (`Cents`, `PPM`, `ToCents`). |

### Extending the Core

1. Add exchange-specific `MsgBuilder`, `MsgDecoder`, and topic helpers.
2. Implement REST client following `domain/exchange.API`, wrapping domain/tech errors consistently.
3. Build a WebSocket facade (e.g., `StreamOrderbook`) that reuses `stream.Stream` + `router`.
4. Use `WithAuth` if the exchange requires a login handshake—return the JSON payload, let `Stream` send it.

### Testing

Run `go test ./...` to cover csv utilities, router helpers, bybit decoders, etc., before pushing changes.

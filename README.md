## pulsoats-core

pulsoats-core — ядро с общими компонентами для биржевых интеграций, детекторов и WebSocket-транспорта.

### Слои

1. **domain/** — чистые контракты и бизнес-сущности.
2. **exchanges/** — конкретные адаптеры бирж (REST и WebSocket).
3. **transport/websocket/** — переиспользуемая логика потоков, подключений и роутера.
4. **lib/** — вспомогательные утилиты (CSV, форматирование, парсинг, корни ошибок).

### Domain

| Пакет | Содержимое |
| --- | --- |
| `domain/market` | `Category`, `Interval`, `Spec`, `CandleSpec`. Интервалы — константы `time.Duration` с JSON-хелперами. |
| `domain/exchange` | Интерфейс `API`: `Candles`, `StreamCandles(ctx, spec, confirmedOnly) (chan market.Candle, <-chan error, error)`, `FeeRate`, `InstrumentExists`. |
| `domain/detect` | Реестр детекторов (`detectors.Registry`). Представление `Candle()` гарантирует `DetectorKindCandle`. |
| `domain/derrors` | Доменные корни ошибок: `ErrNotFound`, `ErrInvalidArgument`, `ErrRequired`, `ErrAlreadyExists`, `ErrUnauthorized`. Для транспортных и внутренних сбоев используйте `lib/errorsx`. |

### Биржа Bybit

| Область | Ключевые моменты |
| --- | --- |
| REST (`exchanges/bybit/rest`) | `Candles` обрабатывает пагинацию и валидирует интервалы. `FeeRate`, `InstrumentExists` приводят HTTP-ошибки к доменным. |
| WebSocket (`exchanges/bybit/websocket`) | `StreamCandles` шарит подключения по категориям, формирует темы `kline.<interval>.<symbol>` и возвращает каналы данных/ошибок. `response.go` декодирует фреймы в `router.StreamMsg`, `resolve.go` сопоставляет scope → endpoint. Делите подключения по scope (private/public/trade и т. д.). |
| Клиент (`exchanges/bybit/client.go`) | Объединяет REST и WebSocket, чтобы реализовать `domain/exchange.API`. |

### Transport / WebSocket

| Компонент | Описание |
| --- | --- |
| `stream.go` | Низкоуровневый менеджер WebSocket-подключений (`github.com/coder/websocket`). Опции: `WithAuth(func(ctx) (any, error))`, `WithDispatch`, `WithReconnect`, `WithOutBuf`, `WithPingEvery`. Auth-функция возвращает payload, `Stream` отправляет его через `wsjson`. |
| `connect.go` | Dial → auth payload → `onReconnect` → reader loop → цикл команд (`CmdSendJSON`, `CmdClose`). |
| `router/` | Отслеживает темы на соединение: `Acquire`/`Release`, `OnReconnect`, `Dispatch`. `sendBatched` группирует subscribe/unsubscribe с лимитами. В `ports.go` описаны контракты `MsgBuilder`/`MsgDecoder` и лёгкий `Logger`. `StreamMsgKind` различает данные и ack-фреймы. |

### Lib

| Пакет | Назначение                                                                                                                                 |
| --- |--------------------------------------------------------------------------------------------------------------------------------------------|
| `lib/errorsx` | Технические ошибки: `ErrInternal`, `ErrClosed`, `ErrNotImplemented`.                                                                       |
| `lib/csv` | CSV-энкодеры для свечей и сигналов плюс буферизированные писатели (заголовки, размер буфера, автофлаш).                                    |
| `lib/logx` | Общий интерфейс `Logger` и простая реализация `NewSimpleLogger`. Внешние приложения могут адаптировать zerolog/slog/zap и передать в core. |
| `lib/format`, `lib/parse`, `lib/units` | Хелперы для денег и дельт (`Cents`, `PPM`, `ToCents`).                                                                                     |

### Расширение ядра

1. Добавьте биржевые `MsgBuilder`, `MsgDecoder` и утилиты для формирования тем.
2. Реализуйте REST-клиент по контракту `domain/exchange.API`, единообразно мапя доменные и технические ошибки.
3. Соберите WebSocket-фасад (например, `StreamOrderbook`), переиспользуя `stream.Stream` и `router`.
4. Подключайте `WithAuth`, если биржа требует рукопожатие: верните JSON-payload, `Stream` отправит его сам.

### Тесты

Перед пушем запускайте `go test ./...`, чтобы покрыть CSV-утилиты, роутер, декодеры Bybit и другие пакеты.

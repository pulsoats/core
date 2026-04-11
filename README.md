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
| `domain/exchange` | Интерфейс `API`: `Candles`, `StreamCandles`, `FeeRate`, `DefaultFees`, `InstrumentExists`. |
| `domain/detect` | Реестр детекторов (`detectors.Registry`). Методы `NewCandle`, `NewCandleOptsPtr`, `ListCandleMetas` гарантируют `DetectorKindCandle`. |
| `errorsx` | Общие корни ошибок: `ErrNotFound`, `ErrInvalidArgument`, `ErrRequired`, `ErrAlreadyExists`, `ErrUnauthorized`, `ErrInternal` и др. |

### Биржа Bybit

| Область | Ключевые моменты |
| --- | --- |
| REST (`exchanges/bybit/rest`) | `Candles` обрабатывает пагинацию и валидирует интервалы. `FeeRate` мапит HTTP-ошибки к `errorsx`, `InstrumentExists` ходит в `/v5/market/instruments-info` и возвращает `bool`. |
| WebSocket (`exchanges/bybit/websocket`) | `StreamCandles` шарит подключения по категориям, формирует темы `kline.<interval>.<symbol>` и возвращает каналы данных/ошибок. `response.go` декодирует фреймы в `router.StreamMsg`, `resolve.go` сопоставляет scope → endpoint. Делите подключения по scope (private/public/trade и т. д.). |
| Клиент (`exchanges/bybit/client.go`) | Объединяет REST и WebSocket, чтобы реализовать `domain/exchange.API`. |

### Transport / WebSocket

| Компонент | Описание |
| --- | --- |
| `stream.go` | Низкоуровневый менеджер WebSocket-подключений (`github.com/coder/websocket`). Настраивается через `StreamConfig` (`Auth`, `Dispatch`, `OnReconnect`, `OutBuf`, `PingEvery`, `PingMsg` и т. д.). Auth-функция возвращает payload, `Stream` отправляет его через `wsjson`. |
| `connect.go` | Dial → auth payload → `onReconnect` → reader loop → цикл команд (`CmdSendJSON`, `CmdClose`). |
| `router/` | Отслеживает темы на соединение: `Subscribe`/`Unsubscribe`, `OnReconnect`, `Dispatch`. `sendBatched` группирует subscribe/unsubscribe с лимитами. В `ports.go` описаны контракты `MsgBuilder`/`MsgDecoder`. `StreamMsgKind` различает данные и ack-фреймы. |

### Lib

| Пакет | Назначение                                                                                                                                 |
| --- |--------------------------------------------------------------------------------------------------------------------------------------------|
| `errorsx/` | Корни ошибок (`ErrInternal`, `ErrClosed`, `ErrNotImplemented`, `ErrCapacityExceeded` и т. д.).                                                  |
| `lib/csv` | CSV-энкодеры для свечей и сигналов плюс буферизированные писатели (заголовки, размер буфера, автофлаш).                                    |
| `lib/logx` | Вспомогательные функции для `slog`: `Discard`, `ParseLevel`. Внешние приложения могут адаптировать zerolog/slog/zap и передать логгер в core. |
| `lib/format`, `lib/parse`, `lib/units` | Хелперы для денег и дельт (`Cents`, `PPM`, `StrToCents`).                                                                                  |

### Расширение ядра

1. Добавьте биржевые `MsgBuilder`, `MsgDecoder` и утилиты для формирования тем.
2. Реализуйте REST-клиент по контракту `domain/exchange.API`, единообразно мапя доменные и технические ошибки.
3. Соберите WebSocket-фасад (например, `StreamOrderbook`), переиспользуя `stream.Stream` и `router`.
4. Заполните `StreamConfig.Auth`, если биржа требует рукопожатие: верните JSON-payload, `Stream` отправит его сам.

### Тесты

Перед пушем запускайте `go test ./...`, чтобы покрыть CSV-утилиты, роутер, декодеры Bybit и другие пакеты.

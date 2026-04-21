## pulsoats-core

pulsoats-core — ядро с общими компонентами для биржевых интеграций, детекторов и WebSocket-транспорта.

### Слои

1. **exchange/** — контракты бирж: интерфейсы, типы, реестр.
2. **exchanges/** — конкретные адаптеры бирж (REST и WebSocket).
3. **market/** — базовые рыночные типы.
4. **detect/** — контракты и реестр детекторов сигналов.
5. **transport/websocket/** — переиспользуемая логика потоков, подключений и роутера.
6. **lib/** — вспомогательные утилиты (CSV, форматирование, парсинг).
7. **errorsx/** — корни ошибок.

### Exchange

| Пакет | Содержимое |
| --- | --- |
| `exchange` | `PublicClient` — публичные методы без авторизации. `Client` — полный набор методов (расширяет `PublicClient`). `Meta` — статические возможности биржи (`Code`, `Intervals`, `Categories`). `Factory` — фабричная функция `func(*slog.Logger, bool) (Client, error)`. `Registry` — реестр фабрик с методами `NewFromEnv` и `NewPublic`. |
| `exchanges` | `Registry` — реестр бирж. `NewRegistry` создаёт реестр с предрегистрированными биржами. |
| `exchanges/bybit` | Реализация `exchange.Client` для Bybit. `NewClient(logger, auth)` — основной конструктор. `NewBybitClient(key, secret, logger)` — прямое создание. `NewFromEnv(logger)` — создание с ключами из env: `BYBIT_API_KEY`, `BYBIT_API_SECRET`. |
| `market` | `Category`, `Interval`, `Spec`, `Candle`. Интервалы — константы `time.Duration` с JSON-хелперами. |

#### Реестр бирж

```go
// создание реестра — bybit предрегистрирован
reg := exchanges.NewRegistry(logger)

// лайв-сервис — авторизованный клиент, ключи из BYBIT_API_KEY / BYBIT_API_SECRET
client, err := reg.NewFromEnv("bybit")

// анализ — публичный клиент без ключей
client, err := reg.NewPublic("bybit")
```

#### Добавление новой биржи

```go
reg.Register("okx", func(l *slog.Logger, auth bool) (exchange.Client, error) {
    return okx.NewClient(l, auth)
})
```

### Detect

| Пакет | Содержимое |
| --- | --- |
| `detect/detectors` | Реестр детекторов (`Registry`). Методы `NewCandle`, `NewCandleOptsPtr`, `ListCandleMetas` гарантируют `DetectorKindCandle`. |

### Transport / WebSocket

| Компонент | Описание |
| --- | --- |
| `stream.go` | Низкоуровневый менеджер WebSocket-подключений (`github.com/coder/websocket`). Настраивается через `StreamConfig` (`Auth`, `Dispatch`, `OnReconnect`, `OutBuf`, `PingEvery`, `PingMsg` и т. д.). Auth-функция возвращает payload, `Stream` отправляет его через `wsjson`. |
| `connect.go` | Dial → auth payload → `onReconnect` → reader loop → цикл команд (`CmdSendJSON`, `CmdClose`). |
| `router/` | Отслеживает темы на соединение: `Subscribe`/`Unsubscribe`, `OnReconnect`, `Dispatch`. `sendBatched` группирует subscribe/unsubscribe с лимитами. В `ports.go` описаны контракты `MsgBuilder`/`MsgDecoder`. `StreamMsgKind` различает данные и ack-фреймы. |

### Lib

| Пакет | Назначение |
| --- | --- |
| `errorsx/` | Корни ошибок (`ErrNotFound`, `ErrInvalidArgument`, `ErrRequired`, `ErrAlreadyExists`, `ErrUnauthorized`, `ErrInternal` и др.). |
| `lib/csv` | CSV-энкодеры для свечей и сигналов плюс буферизированные писатели (заголовки, размер буфера, автофлаш). |
| `lib/logx` | Вспомогательные функции для `slog`: `ParseLevel`. Внешние приложения передают `*slog.Logger` в конструкторы. |
| `lib/format`, `lib/parse`, `lib/units` | Хелперы для денег и дельт (`Cents`, `PPM`, `StrToCents`). |

### Расширение ядра

1. Реализуйте `exchange.Client` для новой биржи в `exchanges/<name>/`.
2. Добавьте `NewClient(logger, auth)` и `NewFromEnv(logger)` по образцу bybit.
3. Зарегистрируйте биржу через `reg.Register` в точке входа сервиса.
4. Для WebSocket — реализуйте `MsgBuilder`/`MsgDecoder` и переиспользуйте `stream.Stream` и `router`.

### Тесты

Перед пушем запускайте `go test ./...`.

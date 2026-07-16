## pulsoats-core

pulsoats-core — ядро с общими компонентами для биржевых интеграций, детекторов и WebSocket-транспорта.

### Пакеты

1. **exchange/** — контракты бирж: интерфейсы, типы, реестр.
2. **exchanges/** — конкретные адаптеры бирж (REST и WebSocket).
3. **market/** — базовые рыночные типы.
4. **detect/** — контракты, реестр детекторов сигналов и фильтры.
5. **transport/websocket/** — переиспользуемая логика потоков, подключений и роутера.
6. **lib/** — вспомогательные утилиты (CSV, форматирование, парсинг).
7. **errorsx/** — корни ошибок.

### Exchange

| Пакет | Содержимое |
| --- | --- |
| `exchange` | `PublicClient` — публичные методы без авторизации. `Client` — полный набор методов (расширяет `PublicClient`). `Credentials` — ключи авторизации (`APIKey`, `APISecret`, `Passphrase`). `Meta` — статические возможности биржи (`Code`, `Intervals`, `Categories`). `Factory` — фабричная функция `func(*slog.Logger, *Credentials) (Client, error)`. |
| `exchanges` | `Registry` — реестр бирж. `NewRegistry` создаёт реестр с предрегистрированным Bybit. Методы: `New(code, creds)`, `NewPublic(code)`, `CreateAllPublic()`. |
| `exchanges/bybit` | Реализация `exchange.Client` для Bybit. `NewClient(logger, creds)` — основной конструктор (если `creds == nil`, клиент публичный). `NewBybitClient(key, secret, logger)` — прямое создание. |
| `market` | `Category`, `Interval`, `Spec`, `Candle`. Интервалы — константы `time.Duration` с JSON-хелперами. |

#### Реестр бирж

```go
// создание реестра — bybit предрегистрирован
reg := exchanges.NewRegistry(logger)

// авторизованный клиент
client, err := reg.New("bybit", &exchange.Credentials{
    APIKey:    os.Getenv("BYBIT_API_KEY"),
    APISecret: os.Getenv("BYBIT_API_SECRET"),
})

// публичный клиент без авторизации
client, err := reg.NewPublic("bybit")

// все зарегистрированные биржи как публичные клиенты
clients, err := reg.CreateAllPublic(logger)
```

### Detect

| Пакет | Содержимое                                                                                                                                                                                                                                                     |
| --- |----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `detect` | `Signal` — результат работы детектора (информация, необходимая для дальнейшего выставления ордера - цены, время свечи, метаданные).                                                                                                                            |
| `detect/detector` | `Detector` — интерфейс детектора на свечах (`Detect`, `WindowSize`, `BarsForBuy`, `BarsForSell`). `Meta` — метаданные (`Code`, `Version`, `Description`, `OptsSchema`). `Registry` — реестр детекторов. `Register[Opts]` — регистрация типизированной фабрики. `Wrap` — оборачивает `Detector` фильтрами. |
| `detect/filter` | `Func` — логический фильтр `func(detectorWindow, lookBackWindow []market.Candle) (bool, error)`. `Filter` — исполняемый фильтр (`Func` + `Period`). `Config` — конфигурация (`Code`, `Period`). `Registry` — реестр фильтров. `Register` — регистрация. `FilterFromConfig` — создание фильтра из конфига. |

#### Реестр детекторов

```go
// регистрация (обычно вызывается из пакета с реализацией)
detector.Register(registry, detector.Meta{
    Code:    "w",
    Version: "v1",
}, NewWDetector)

// создание экземпляра
det, err := registry.New("w", "v1", "label", WOpts{...})

// десериализация опций из JSON
opts, err := registry.UnmarshalOpts("w", "v1", rawJSON)
```

#### Фильтры

Фильтры отсеивают ложные сигналы детектора, используя lookback-данные — свечи, предшествующие окну детектора. `Period` задаёт глубину этого lookback.

```go
// регистрация фильтра
filter.Register(filterRegistry, filter.Meta{Code: "trend"}, func(window, lookBack []market.Candle) (bool, error) {
    // window — свечи детектора, lookBack — Period свечей до окна
    return true, nil
})

// создание фильтра из конфига
f, err := filter.FilterFromConfig(filterRegistry, filter.Config{Code: "trend", Period: 50})

// детектор, обёрнутый фильтрами — WindowSize увеличивается на max(Period)
det = detector.Wrap(det, []filter.Filter{f})
```

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

### Тесты

`go test ./...`.

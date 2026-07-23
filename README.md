## pulsoats-core

pulsoats-core — ядро с общими компонентами для биржевых интеграций, детекторов и WebSocket-транспорта.

### Пакеты

1. **exchange/** — контракты бирж: интерфейсы, типы, реестр.
2. **exchanges/** — конкретные адаптеры бирж (REST и WebSocket).
3. **market/** — базовые рыночные типы.
4. **detect/** — контракты, реестр детекторов сигналов и фильтры.
5. **run/** — базовые типы результата запуска детектора (`Base`, `Status`).
6. **transport/websocket/** — переиспользуемая логика потоков, подключений и роутера.
7. **lib/** — вспомогательные утилиты (CSV, форматирование, парсинг).
8. **xgrpc/** - мапперы из proto-сообщений в core-типы (`MarketSpec`, `Detector/FilterConfig`, `Fees`) и конвертеры времени/UUID.
9. **errorsx/** — корни ошибок.
10. **envvars/** — строковые константы имён переменных окружения.
11. **tlsconfig/** — mTLS-провайдер с автоматической ротацией Vault Agent сертификатов.

### Exchange

| Пакет | Содержимое |
| --- | --- |
| `exchange` | `PublicClient` — публичные методы без авторизации. `Client` — полный набор методов (расширяет `PublicClient`). `Credentials` — ключи авторизации (`APIKey`, `APISecret`, `Passphrase`). `Meta` — статические возможности биржи (`Code`, `Intervals`, `Categories`). `Factory` — фабричная функция `func(*slog.Logger, *Credentials) (Client, error)`. |
| `exchanges` | `Registry` — реестр бирж. `NewRegistry` создаёт реестр с предрегистрированным Bybit (единственная поддерживаемая биржа). Логгер передаётся один раз при создании реестра и используется во всех методах. Методы: `New(code, creds) Client`, `NewPublic(code) PublicClient`, `CreateAllPublic(logger) map[string]PublicClient`. |
| `exchanges/bybit` | Реализация `exchange.Client` для Bybit. `NewClient(logger, creds)` — основной конструктор (если `creds == nil`, клиент публичный). `NewBybitClient(key, secret, logger)` — прямое создание. |
| `market` | `Category`, `Interval`, `Spec`, `Candle`, `TakerMakerFees`. Интервалы — константы `time.Duration` с JSON-хелперами (`1m`…`1M`). |

#### Реестр бирж

```go
// создание реестра — bybit предрегистрирован; logger хранится внутри и используется во всех методах
reg := exchanges.NewRegistry(logger)

// авторизованный клиент (exchange.Client)
client, err := reg.New(code, &exchange.Credentials{
    APIKey:     apiKey,
    APISecret:  apiSecret,
    Passphrase: passphrase, // опционально
})

// публичный клиент без авторизации (exchange.PublicClient)
pub, err := reg.NewPublic(code)

// все зарегистрированные биржи как публичные клиенты; logger для этого вызова передаётся отдельно
clients, err := reg.CreateAllPublic(logger)
```

### Detect

| Пакет | Содержимое |
| --- |---|
| `detect` | `Signal` — результат работы детектора (цены, время свечи, метаданные). `Metadata map[string]string`. |
| `detect/detector` | `Detector` — интерфейс детектора на свечах (`Detect`, `WindowSize`, `BarsForBuy`, `BarsForSell`). `Config` — сериализуемая конфигурация (`Code`, `Version`, `OptsLabel`, `Opts json.RawMessage`). `Meta` — метаданные (`Code`, `Version`, `Description`, `OptsSchema`). `Registry` — реестр детекторов. `Register[Opts]` — регистрация типизированной фабрики. `Wrap` — оборачивает `Detector` фильтрами. |
| `detect/filter` | `Func` — логический фильтр `func(detectorWindow, lookBackWindow []market.Candle) (bool, error)`. `Filter` — исполняемый фильтр (`Func` + `Period`). `Config` — конфигурация (`Code`, `Period`). `Registry` — реестр фильтров. `Register` — регистрация. `FilterFromConfig` — создание фильтра из конфига. |

#### Реестр детекторов

```go
// регистрация (обычно вызывается из пакета с реализацией)
detector.Register(registry, detector.Meta{
    Code:    "w",
    Version: "v1",
}, NewWDetector)

// создание экземпляра из конфига (opts десериализуются из JSON)
det, err := registry.NewFromConfig(detector.Config{
    Code:      "w",
    Version:   "v1",
    OptsLabel: "label",
    Opts:      json.RawMessage(`{...}`),
})

// список всех зарегистрированных детекторов
metas := registry.ListMetas()

// список версий конкретного детектора
versions := registry.ListVersions("w")
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

### Run

Пакет описывает базовый результат работы одного детектора на одном рынке. Сторонние приложения могут встраивать `Base` в собственные структуры.

| Тип | Описание |
| --- | --- |
| `Base` | UUID, рынок, интервал, конфиг детектора, фильтры, счётчик сигналов, временной диапазон свечей. |
| `Status` | `Code` (`Pending`, `Running`, `Done`, `Failed`) + `Message`. |
| `ParseStatusCode(int)` | Разбирает int → `StatusCode`, `bool`. |

### xgrpc

Пакет содержит две группы утилит для работы с protobuf.

**Мапперы** (`mappers.go`) — конвертация core-типов ↔ protobuf:

| Функция | Описание |
| --- | --- |
| `MarketSpecToProto` / `MarketSpecFromProto` | `market.Spec` ↔ `corepb.MarketSpec` |
| `DetectorConfigToProto` / `DetectorConfigFromProto` | `detector.Config` ↔ `corepb.DetectorConfig` |
| `FilterConfigToProto` / `FilterConfigFromProto` | `filter.Config` ↔ `corepb.FilterConfig` |
| `FeesToProto` / `FeesFromProto` | `*market.TakerMakerFees` ↔ `*corepb.Fees` |

**Конвертеры** (`convert.go`) — время и UUID:

| Функция | Описание |
| --- | --- |
| `TimeFromProto(ts)` | `*timestamppb.Timestamp` → `time.Time` (nil → zero) |
| `TimePtrFromProto(ts)` | `*timestamppb.Timestamp` → `*time.Time` (nil → nil) |
| `TimePtrToProto(t)` | `*time.Time` → `*timestamppb.Timestamp` (nil → nil) |
| `UUIDPtrToProto(id)` | `*uuid.UUID` → `*string` (nil → nil) |
| `UUIDPtrFromProto(op, field, raw)` | `*string` → `*uuid.UUID` (nil → nil, ошибка → `ErrInternal`) |

### envvars

Строковые константы имён переменных окружения, используемых в лайв-сервисах:

```go
envvars.LivePostgresDSN        // "POSTGRES_DSN"
envvars.LiveExchangeCode       // "EXCHANGE_CODE"
envvars.LiveExchangeAPIKey     // "EXCHANGE_API_KEY"
envvars.LiveExchangeAPISecret  // "EXCHANGE_API_SECRET"
envvars.LiveExchangePassphrase // "EXCHANGE_PASSPHRASE"
envvars.LiveGRPCTLSCert        // "GRPC_TLS_CERT"
envvars.LiveGRPCTLSKey         // "GRPC_TLS_KEY"
envvars.LiveGRPCCACert         // "GRPC_CA_CERT"
```

### tlsconfig

`Provider` загружает TLS-сертификаты из файлов. CA загружается один раз при старте; cert+key перечитываются с диска при каждом TLS handshake (кеш 1 минута). При ошибке перечитывания отдаётся устаревший сертификат, чтобы не обрывать активные соединения.

```go
p, err := tlsconfig.New(certFile, keyFile, caFile)

// для gRPC-сервера (mTLS, RequireAndVerifyClientCert, TLS 1.3)
grpc.NewServer(grpc.Creds(credentials.NewTLS(p.ServerConfig())))

// для исходящих gRPC-соединений (mTLS, TLS 1.3)
grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(p.ClientConfig())))
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

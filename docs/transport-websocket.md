# transport/websocket — подробная схема работы

## Архитектура: два независимых слоя

```
┌────────────────────────────────────────────────────────┐
│                  Caller / Exchange                     │
│  StreamCandles(ctx, spec) → chan Candle                │
└───────────────┬───────────────────────────────────────┘
                │ Subscribe(topics) / Unsubscribe(topics)
                ▼
┌────────────────────────────────────────────────────────┐
│               router.Router                            │
│                                                        │
│  pipes   map[topic → *pipe{ch, ref}]                   │
│  pending map[reqID → *pendingReq]                      │
│  MsgBuilder  — строит subscribe/unsubscribe запросы    │
│  MsgDecoder  — разбирает входящие фреймы               │
└───────────────┬───────────────────────────────────────┘
                │  chan Command  (единственная точка связи)
                │  ◄── Router пишет CmdSendJSON
                │  ──► Stream читает и отправляет в сокет
                │
                │  Dispatch(raw) — коллбэк: Stream вызывает Router
                │  OnReconnect() — коллбэк: Stream вызывает Router
                ▼
┌────────────────────────────────────────────────────────┐
│               websocket.Stream                         │
│                                                        │
│  dial · auth · reconnect · backoff · ping              │
│  reader-pump горутина                                  │
│  main-loop горутина                                    │
└───────────────┬───────────────────────────────────────┘
                │ TCP/TLS
                ▼
          WebSocket server
```

**Принцип разделения:**
- `Stream` не знает про топики, subscribe, биржевые протоколы.
- `Router` не знает про сетевые соединения, реконнекты, ping.
- Связь строго через `chan Command` (Router → Stream) и два коллбэка (Stream → Router).

---

## Часть 1: websocket.Stream

**Пакет:** `transport/websocket`
**Файлы:** `stream.go`, `connect.go`, `config.go`, `command.go`, `backoff.go`

### Конфигурация: StreamConfig

```go
type StreamConfig struct {
    URL         string
    Cmds        chan Command           // shared с Router; обязателен
    DialOptions *websocket.DialOptions

    Auth        func(ctx) (any, error)              // авторизация при подключении
    Dispatch    func(ctx, raw json.RawMessage) error // роутинг входящих сообщений
    OnReconnect func(ctx) error                      // вызывается после каждого подключения

    OutBuf       int           // размер out-канала если Dispatch=nil (def: 256)
    BackoffStart time.Duration // начальная задержка реконнекта (def: 1s)
    BackoffMax   time.Duration // максимальная задержка реконнекта (def: 30s)
    PingEvery    time.Duration // интервал ping; 0 — отключён
    PingMsg      any           // если не nil — JSON ping вместо WS control frame
    Logger       *slog.Logger
}
```

Если `Dispatch` задан — `Stream` вызывает его для каждого сообщения и не создаёт выходной канал (`Connect` возвращает `nil`). Именно этот режим используется в связке с `Router`.

Если `Dispatch` не задан — `Stream` кладёт сырые `json.RawMessage` в буферизированный канал, который возвращает `Connect`.

### NewStream и Connect

```
NewStream(cfg) → *Stream    // только валидация и нормализация defaults
s.Connect(ctx) → (chan json.RawMessage, error)  // немедленно запускает горутину
```

`Connect` не блокируется. Вся работа идёт в одной фоновой горутине — **session-loop**.

### Session-loop (connect.go)

Горутина крутится в бесконечном цикле, поднимая сессию за сессией:

```
backoff = BackoffStart

loop:
  1. connCtx = context.WithCancel(ctx)    // ctx сессии, живёт одно подключение

  2. websocket.Dial(connCtx, url, dialOptions)
     ошибка → cancelConn() → sleepBackoff() → continue
              (если ctx отменён во время sleep — return)

  3. Auth (если задан):
     payload, err = cfg.Auth(connCtx)
     err  → Close(NormalClosure) → cancelConn() → sleepBackoff() → continue
     payload != nil → wsjson.Write(connCtx, conn, payload, timeout=5s)
     write err → Close(AbnormalClosure) → cancelConn() → sleepBackoff() → continue

  4. backoff = BackoffStart   // сброс после успешного dial+auth

  5. OnReconnect (если задан):
     err = cfg.OnReconnect(ctx)       // ← здесь Router реподписывается
     err → Close(AbnormalClosure) → cancelConn() → sleepBackoff() → continue

  6. Запустить reader-pump (отдельная горутина, см. ниже)

  7. Запустить ping-ticker (отдельная горутина, если PingEvery > 0, см. ниже)

  8. main-loop (блокирующий select, см. ниже)
     → возвращает runErr

  9. Завершение сессии:
     close(pingStop)    // останавливает ping-горутину
     cancelConn()       // останавливает reader-pump
     conn.Close(NormalClosure)

  10. Принятие решения о реконнекте:
      ctx.Err() != nil → return          // внешняя отмена — стоп навсегда
      runErr == errStopStream → return   // CmdClose — стоп навсегда
      runErr == nil → sleepBackoff() → continue    // сервер закрыл — реконнект
      runErr != nil → sleepBackoff() → continue    // ошибка — реконнект с backoff
```

### Reader-pump (горутина внутри сессии)

```go
go func(conn, readCh chan<- json.RawMessage, errCh chan<- error, connCtx) {
    defer close(readCh)
    for {
        wsjson.Read(connCtx, conn, &msg)
        // ошибка → классифицировать:
        //   CloseError (сервер закрыл) → errCh <- nil  (нормальное завершение)
        //   normalCloseErr             → errCh <- nil
        //   иное                       → errCh <- err
        // → return
        //
        // успех → readCh <- msg
        //         (с fallback на connCtx.Done, чтобы не зависнуть)
    }
}
```

Горутина не имеет прямого доступа к основному loop — только через каналы `readCh` / `errCh`.

`normalCloseErr` считает нормальными: `context.Canceled`, `context.DeadlineExceeded`, `net.ErrClosed`, любой `*websocket.CloseError`, `"use of closed network connection"`.

### Ping-горутина (внутри сессии)

```go
go func(conn, connCtx, pingStop chan struct{}, every) {
    ticker := time.NewTicker(every)
    for {
        select {
        case <-ticker.C:
            pingCtx, cancel = context.WithTimeout(connCtx, 5s)
            if PingMsg != nil:
                err = wsjson.Write(pingCtx, conn, PingMsg)  // JSON heartbeat
            else:
                err = conn.Ping(pingCtx)                    // WS control frame
            cancel()
            if err → conn.Close(AbnormalClosure) → return
                    // main-loop увидит закрытый readCh → reconnect
        case <-connCtx.Done(): return
        case <-pingStop:       return
        }
    }
}
```

`PingMsg` нужен для бирж, которые требуют application-level heartbeat (Bybit: `{"op":"ping"}`), а не WS protocol-level PING frame.

### Main-loop (внутри сессии)

```go
for {
    select {
    case raw, ok := <-readCh:
        if !ok:
            // reader закрыл канал → читаем errCh
            e := <-errCh
            if normalCloseErr(e) → return nil     // нормальное закрытие
            return e                               // ошибка → reconnect

        if Dispatch != nil:
            Dispatch(ctx, raw)    // Router.Dispatch — роутинг по топикам
        else:
            out <- raw            // drop если out заполнен (с логом)

    case cmd := <-cmds:
        switch cmd.Op:
        case CmdClose:
            conn.Close(NormalClosure)
            return errStopStream   // → session-loop: стоп навсегда

        case CmdSendJSON:
            wsjson.Write(connCtx, conn, cmd.Payload, timeout=5s)
            write err → conn.Close(AbnormalClosure) → return err  // → reconnect

    case <-ctx.Done():
        conn.Close(NormalClosure)
        return ctx.Err()          // → session-loop: стоп навсегда

    case <-connCtx.Done():
        conn.Close(NormalClosure)
        return connCtx.Err()
    }
}
```

### Backoff (backoff.go)

```
delay = cur + rand(0 .. cur/5)    // +0..20% джиттер
next  = cur × 1.7
next  = min(next, BackoffMax)
cur   = next
```

При `BackoffStart=1s`, `BackoffMax=30s` последовательность задержек (без джиттера):
`1s → 1.7s → 2.9s → 4.9s → 8.3s → 14.1s → 24s → 30s → 30s → ...`

Если `ctx` отменён во время ожидания — `sleepBackoff` возвращает `false`, session-loop завершается.

### Command channel

```go
type Command struct {
    Op      CmdOp   // CmdSendJSON | CmdClose
    Payload any     // для CmdSendJSON — тело, сериализуется wsjson.Write
}
```

Канал буфером `8` (задаётся при создании в адаптере). Многие писатели (`Router.sendBatched`), один читатель (main-loop `Stream`). `Router` пишет в него subscribe/unsubscribe запросы — `Stream` просто перекладывает байты в сокет, не разбирая содержимое.

---

## Часть 2: router.Router

**Пакет:** `transport/websocket/router`
**Файлы:** `router.go`, `config.go`, `livecycle.go`, `dispatch.go`, `helpers.go`, `types.go`, `ports.go`

### Конфигурация: router.Config

```go
type Config struct {
    Cmds       chan websocket.Command   // shared со Stream; обязателен
    MsgBuilder MsgBuilder              // строит subscribe/unsubscribe запросы
    MsgDecoder MsgDecoder              // разбирает входящие фреймы

    PipeBuf      int           // буфер каждого топик-канала (def: 64)
    TopicsPerReq int           // макс. топиков в одном запросе (def: 10)
    ReqPerSec    int           // rate limit в секунду (def: 10)
    PendingTTL   time.Duration // TTL для pending; 0 — cleaner отключён
    Logger       *slog.Logger
}
```

### Внутренние структуры данных

```
Router.pipes   map[string]*pipe
  pipe {
      topic string
      ch    chan json.RawMessage   // куда Dispatch кладёт данные
      ref   int                   // количество активных подписчиков
      once  sync.Once             // защита от двойного close(ch)
  }

Router.pending  map[string]*pendingReq
  pendingReq {
      reqID  string       // UUID без дефисов (≤32 символа)
      op     Op           // OpSubscribe | OpUnsubscribe
      topics []string     // топики этого запроса
      sentAt time.CandleTime    // для TTL cleaner
  }
```

`pipes` — живая таблица активных подписок. `pending` — запросы, отправленные в сокет, ожидающие ACK от сервера.

### Subscribe(ctx, topics) → map[topic → chan json.RawMessage]

```
mu.Lock()
для каждого топика:
  если уже есть в pipes:
    pipe.ref++
    out[topic] = pipe.ch       // тот же канал, новый подписчик
  иначе:
    ch = make(chan json.RawMessage, PipeBuf)
    pipes[topic] = pipe{ch, ref:1}
    out[topic] = ch
    newTopics = append(newTopics, topic)
mu.Unlock()

если len(newTopics) > 0:
    sendBatched(ctx, OpSubscribe, newTopics)   // асинхронно шлёт в cmds
```

Важно: несколько вызовов `Subscribe` с одним топиком возвращают **один и тот же** канал и увеличивают `ref`. Данные получат все читатели из одного канала — нет дублирования запросов к серверу.

### Unsubscribe(ctx, topics) error

```
mu.Lock()
для каждого топика:
  pipe.ref--
  если ref == 0:
    delete(pipes, topic)
    toClose = append(toClose, pipe)
    unsubTopics = append(unsubTopics, topic)
  если ref < 0: warn, skip   // защита от underflow
mu.Unlock()

для каждого pipe в toClose:
    pipe.closeOnce()     // close(ch) — читатели получат ok=false

если len(unsubTopics) > 0:
    sendBatched(ctx, OpUnsubscribe, unsubTopics)
```

Канал закрывается через `sync.Once` — безопасно даже если Dispatch и Unsubscribe гонятся.

### sendBatched(ctx, op, allTopics) — внутренний метод

Центральный механизм отправки subscribe/unsubscribe запросов:

```
batches = chunkStrings(allTopics, TopicsPerReq)
// например 25 топиков при TopicsPerReq=10 → [[t1..t10], [t11..t20], [t21..t25]]

если len(batches) > 1 && ReqPerSec > 0:
    tick = time.NewTicker(1s)   // для rate limiting

для каждого батча:
  1. Rate limit:
     если sentThisSecond == ReqPerSec:
         <-tick.C   // ждём начала следующей секунды
         sentThisSecond = 0

  2. reqID = UUID без дефисов (32 символа)
     req = MsgBuilder.Build(ctx, reqID, op, topics)

  3. Отправить в cmds (неблокирующая попытка):
     select {
     case cmds <- Command{CmdSendJSON, req}: sent=true
     default:
         // канал занят — ждать 300ms
         select {
         case cmds <- ...: sent=true
         case <-time.After(300ms): warn, drop
         }
     }

  4. ТОЛЬКО если sent:
     mu.Lock()
     pending[reqID] = pendingReq{reqID, op, topics, time.Now()}
     mu.Unlock()
     sentThisSecond++
```

Запись в `pending` происходит строго после успешной отправки в `cmds` — гарантия того, что нет "потерянных" pending, которые никогда не были отправлены.

### Dispatch(ctx, raw json.RawMessage) — коллбэк из Stream

Вызывается из main-loop `Stream` на каждое входящее сообщение:

```
msg = MsgDecoder.Decode(ctx, raw)

switch msg.Kind:

case StreamMsgKindData:
    // обычный push-фрейм с данными
    mu.RLock()
    pipe = pipes[msg.Topic]
    mu.RUnlock()
    если pipe не найден → ignore (топик уже Unsubscribe'd или не наш)

    select {
    case pipe.ch <- raw:       // передаём RAW (не msg.Data!) подписчику
    default: warn "pipe full"  // дроп без блокировки
    case <-ctx.Done(): return ctx.Err()
    }

case StreamMsgKindAck:
    // ответ сервера на subscribe/unsubscribe
    если msg.ReqID == "" → ignore

    pReq = pending[msg.ReqID]
    если не найден → ignore (уже evicted cleaner'ом или дубликат)

    если msg.Success == true && len(msg.FailedTopics) == 0:
        // полный успех
        delete(pending, reqID)

    если msg.Success == true && len(msg.FailedTopics) > 0:
        // частичный успех — некоторые топики отклонены
        warn "ack partially failed"
        removeTopics(msg.FailedTopics)   // закрыть их каналы
        delete(pending, reqID)

    если msg.Success == false && ret_msg содержит "already subscribed":
        // идемпотентная ошибка — считаем успехом
        debug "already subscribed"
        delete(pending, reqID)

    если msg.Success == false (иное):
        warn "ack failed"
        removeTopics(pReq.topics)        // закрыть каналы топиков
        delete(pending, reqID)

case StreamMsgKindUnknown:
    ignore
```

`Dispatch` получает **raw** `json.RawMessage` и кладёт её целиком в `pipe.ch` — без лишнего копирования. Подписчик сам десериализует нужные поля.

### OnReconnect(ctx) — коллбэк из Stream

Вызывается после каждого успешного `dial` + `auth`, перед тем как начать читать сообщения:

```
mu.Lock()

firstConnect = !registry.connected
если firstConnect: registry.connected = true

registry.state = ConnStateConnected

// Фиксируем топики из in-flight pending (Subscribe вызван параллельно
// до OnReconnect — они уже улетели в сокет этого соединения)
inPending = {topic for req in pending for topic in req.topics}

// Сбрасываем pending: ACK'и со старого соединения уже не придут
registry.pending = make(map[string]*pendingReq)

// Собираем топики для реподписки (исключая те, что уже в новых pending)
restartTopics = [p.topic for p in pipes if p.topic not in inPending]

mu.Unlock()

если firstConnect:
    return nil   // первое подключение: Subscribe сам придёт из caller'а

если len(restartTopics) > 0:
    sendBatched(ctx, OpSubscribe, restartTopics)
```

Логика `inPending` нужна для race condition: `Subscribe` может прийти уже после `dial`, но до `OnReconnect` — такие топики уже записаны в новые `pending` нового соединения, их не нужно повторно подписывать.

### StartCleaner(ctx)

```
если PendingTTL == 0 → no-op

каждые PendingTTL/2:
    mu.Lock()
    для каждого req в pending:
        если now - req.sentAt >= PendingTTL:
            warn "pending request expired"
            delete(pending, reqID)
    mu.Unlock()
```

Защита от утечки памяти: если сервер не прислал ACK (упал, потерял сообщение), `pending` разрастается бесконечно. Cleaner периодически evict'ит старые записи. Вызывать `registry.StartCleaner(ctx)` нужно после создания роутера — адаптер биржи сам решает, нужен ли он.

### Интерфейсы (ports.go)

```go
type MsgBuilder interface {
    Build(ctx context.Context, reqID string, op Op, topics []string) (any, error)
}
// Вызывается из sendBatched. Возвращает любой сериализуемый объект —
// Stream сделает wsjson.Write(conn, payload).

type MsgDecoder interface {
    Decode(ctx context.Context, raw json.RawMessage) (*StreamMsg, error)
}
// Вызывается из Dispatch. Разбирает сырой фрейм, определяет Kind.
```

`StreamMsg`:
```go
type StreamMsg struct {
    Kind         StreamMsgKindUnknown | StreamMsgKindData | StreamMsgKindAck
    Topic        string          // для Data
    Op           string          // для Ack (например "subscribe")
    Success      bool            // для Ack
    ReqID        string          // для Ack
    Type         string          // "snapshot" / "delta" / "COMMAND_RESP"
    RetMsg       string          // текст ошибки от сервера
    FailedTopics []string        // топики, которые сервер отклонил
    Data         json.RawMessage
    Raw          json.RawMessage // полный исходный фрейм
}
```

---

## Часть 3: Bybit-адаптер

**Пакет:** `exchanges/bybit/websocket`
**Файлы:** `request.go`, `response.go`, `auth.go`, `stream_candles.go`, `client.go`

### Реализация MsgBuilder — bybitMsgBuilder

```go
func (bybitMsgBuilder) Build(_ context.Context, reqID string, op Op, topics []string) (any, error) {
    return request{ReqID: reqID, Op: string(op), Args: topics}, nil
}
```

Сериализуется в: `{"req_id":"<uuid32>","op":"subscribe","args":["kline.1.BTCUSDT"]}`

### Реализация MsgDecoder — bybitMsgDecoder

Разбирает входящие фреймы Bybit по наличию полей:

```
есть topic && нет success  →  KindData
  {"topic":"kline.1.BTCUSDT","type":"snapshot","data":{...}}

есть success || op || req_id  →  KindAck
  {"success":true,"op":"subscribe","req_id":"abc","type":"COMMAND_RESP",
   "data":{"successTopics":["kline.1.BTCUSDT"],"failTopics":[]}}

иное  →  ErrUnknownFrame (Kind=Unknown, игнорируется Router'ом)
```

Особый случай ACK: если `success=false` и `ret_msg` содержит `"already subscribed"` — это не ошибка, Router обработает как успех.

### Авторизация — Client.Auth

Используется только для приватных endpoints (позиции, ордера). Для публичных (kline) `Auth` не передаётся.

```
expires = now + 1000ms
signature = HMAC-SHA256(apiSecret, "GET/realtime" + expires)
payload = {"op":"auth","args":[apiKey, expires, signature]}
```

`Stream` отправляет этот payload сразу после dial, до `OnReconnect`.

### Управление сессиями — Client

```go
type Client struct {
    mu    sync.RWMutex
    conns map[string]*conn  // streamID → {stream, router}
    ...
}
type conn struct {
    stream *websocket.Stream
    router *router.Router
}
```

`streamID` — ключ вида `"kline.{category}"`. Одна сессия WebSocket на категорию (`linear`, `spot`, `inverse`). Все символы одной категории мультиплексируются в одном соединении через Router.

### Полный путь StreamCandles(ctx, spec, confirmedOnly)

```
1. Валидация:
   specs.SupportedIntervals[spec.Interval] → iv (например "1")
   resolveURL(scopePublic, category) → url

2. cmds = make(chan Command, 8)
   streamID = "kline.linear"

3. Поиск существующей сессии:
   mu.RLock() → conns[streamID]

   НЕТ → создать новую:
     mu.RUnlock()
     router.NewRouter(Config{Cmds:cmds, MsgBuilder, MsgDecoder, Logger})
     websocket.NewStream(StreamConfig{
         URL, Cmds: cmds,
         Dispatch:    registry.Dispatch,      // Router обрабатывает входящее
         OnReconnect: registry.OnReconnect,   // Router реподписывается при реконнекте
         BackoffStart: 1s, BackoffMax: 30s,
         PingEvery: 20s,
         PingMsg: {"op":"ping"},       // Bybit требует JSON heartbeat
         Logger,
     })
     s.Connect(ctx)   // запускает session-loop горутину, не блокирует
     mu.Lock() → conns[streamID] = {s, registry}

   ЕСТЬ → mu.RUnlock(), reuse

4. topic = "kline.1.BTCUSDT"
   pipes = sess.router.Subscribe(ctx, [topic])
   pipe  = pipes[topic]   // chan json.RawMessage

5. out  = make(chan market.Candle, 256)
   errCh = make(chan error, 1)

6. Запустить горутину-читатель:
   for {
       select {
       case <-ctx.Done():
           sess.router.Unsubscribe(ctx, [topic])
           return

       case raw, ok := <-pipe:
           if !ok → sendErr("pipe closed") → return

           json.Unmarshal(raw, &payload{Data []RawCandle})
           для каждого k в payload.Data:
               если confirmedOnly && !k.Confirm → skip
               candle = DecodeCandle(k)
               out <- candle
       }
   }

7. return out, errCh, nil
```

---

## Горутины и их время жизни

```
s.Connect(ctx) запускает:
└── session-loop горутина [живёт пока ctx не отменён или CmdClose]
    └── при каждой сессии:
        ├── reader-pump горутина [живёт одну сессию, до закрытия conn]
        └── ping-ticker горутина [живёт одну сессию, до pingStop или connCtx.Done]

registry.StartCleaner(ctx) запускает (опционально):
└── cleaner горутина [живёт пока ctx не отменён]

StreamCandles запускает:
└── reader горутина на каждый (spec, confirmedOnly) вызов [живёт пока ctx не отменён]
```

При отмене `ctx`:
1. Reader горутина получает `<-ctx.Done()` → вызывает `Unsubscribe` → выходит
2. Session-loop получает `<-ctx.Done()` в main-loop → закрывает соединение → выходит
3. Reader-pump получает `connCtx.Done()` → выходит
4. Ping-ticker получает `connCtx.Done()` → выходит
5. Cleaner получает `<-ctx.Done()` → выходит

---

## Потокобезопасность

| Компонент | Механизм | Комментарий |
|-----------|----------|-------------|
| `Router.pipes` / `Router.pending` | `sync.RWMutex` | Subscribe, Unsubscribe, Dispatch, OnReconnect — все под одним мьютексом |
| `pipe.ch` close | `sync.Once` | Unsubscribe и removeTopics не могут закрыть канал дважды |
| `chan Command` | Go channel | Многие писатели (sendBatched), один читатель (Stream main-loop) |
| `Stream` внутренние поля | immutable после `NewStream` | Только чтение из горутин |
| `Client.conns` | `sync.RWMutex` | Чтение при reuse, запись при создании новой сессии |
| `out chan Candle` | Go channel | Один писатель (reader горутина), читатель — caller |

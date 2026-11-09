# rockredis

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockredis/README_RU.md — правьте README в gorock-kit. -->

::: tip Место в GOROCK · Toolkit · Services
Готовый сервис Redis для `pkg/services` (кэш, блокировки, pub/sub). Реализует `rockengine.Resource`. → [Services](/architecture/services)
:::

```sh
go get github.com/arzab/gorock-kit/rockredis@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockredis)

Обёртка над [go-redis/v9](https://github.com/redis/go-redis) с интеграцией в `rockconfig` и чистым типизированным интерфейсом `Service` для наиболее частых операций Redis.

## Быстрый старт

```go
svc := rockredis.NewService(rockredis.Configs{
    Addr:     "localhost:6379",
    Password: "secret",
    DB:       0,
})

// С rockengine: инициализируется до apps, закрывается после них.
rockengine.RegisterResource("redis", svc)

// Без движка:
if err := svc.Init(ctx); err != nil { // создаёт клиент и делает ping
    log.Fatal(err)
}
defer svc.Stop()
```

## Загрузка конфига из файла

`rockredis.Configs` совместим с `rockconfig.InitFromFile`. Все поля опциональны.

```yaml
# config.yaml
addr: "localhost:6379"
password: "secret"
db: 0
dial_timeout: "5s"
read_timeout: "3s"
write_timeout: "3s"
pool_size: 10
min_idle_conns: 2
tls_enabled: false
```

```go
cfg, err := rockconfig.InitFromFile[rockredis.Configs]("config.yaml")
if err != nil {
    log.Fatal(err)
}
svc := rockredis.NewService(*cfg)
```

## Конфигурация

```go
type Configs struct {
    Addr     string        // host:port; по умолчанию "localhost:6379"
    Password string
    DB       int           // индекс базы данных Redis (0–15)

    DialTimeout  time.Duration // по умолчанию 5s
    ReadTimeout  time.Duration // по умолчанию 3s
    WriteTimeout time.Duration // по умолчанию 3s

    PoolSize     int // размер пула соединений; по умолчанию 10
    MinIdleConns int // минимальное число idle-соединений

    TLSEnabled bool // включить TLS (системный пул сертификатов)
}
```

## ErrNil

Когда ключ или поле не существует, операции возвращают `rockredis.ErrNil` (преобразование из `redis.Nil`):

```go
val, err := svc.Get(ctx, "missing-key")
if errors.Is(err, rockredis.ErrNil) {
    // ключ не существует
}
```

## Операции

### Строки

```go
// Установить с TTL (0 = без срока истечения)
svc.Set(ctx, "key", "value", 10*time.Minute)

// Получить
val, err := svc.Get(ctx, "key")

// Установить только если ключ не существует
ok, err := svc.SetNX(ctx, "lock", "1", 30*time.Second)

// Удалить
svc.Del(ctx, "key1", "key2")

// Проверить существование (возвращает число найденных ключей)
n, err := svc.Exists(ctx, "key1", "key2")

// Управление TTL
svc.Expire(ctx, "key", 5*time.Minute)
ttl, err := svc.TTL(ctx, "key") // -1 = нет TTL, -2 = ключ не существует

// Счётчики
svc.Incr(ctx, "counter")
svc.IncrBy(ctx, "counter", 10)
```

### Hash

```go
// Установить поля (чередующиеся пары поле/значение)
svc.HSet(ctx, "user:1", "name", "Alice", "age", 30)

// Получить одно поле
val, err := svc.HGet(ctx, "user:1", "name")

// Получить все поля
m, err := svc.HGetAll(ctx, "user:1")

// Удалить поля
svc.HDel(ctx, "user:1", "age")

// Проверить существование поля
ok, err := svc.HExists(ctx, "user:1", "name")
```

### List

```go
// Добавить
svc.LPush(ctx, "queue", "first", "second") // "second" становится головой
svc.RPush(ctx, "queue", "last")

// Извлечь (возвращает ErrNil если список пуст)
val, err := svc.LPop(ctx, "queue")
val, err  = svc.RPop(ctx, "queue")

// Длина
n, err := svc.LLen(ctx, "queue")

// Диапазон (включительно, 0-based; -1 = последний элемент)
vals, err := svc.LRange(ctx, "queue", 0, -1)
```

### Set

```go
svc.SAdd(ctx, "tags", "go", "redis")
svc.SRem(ctx, "tags", "redis")
members, err := svc.SMembers(ctx, "tags")
ok, err      := svc.SIsMember(ctx, "tags", "go")
n, err       := svc.SCard(ctx, "tags")
```

### Sorted Set

```go
svc.ZAdd(ctx, "leaderboard",
    rockredis.Z{Score: 100, Member: "alice"},
    rockredis.Z{Score: 200, Member: "bob"},
)

score, err  := svc.ZScore(ctx, "leaderboard", "alice")
rank, err   := svc.ZRank(ctx, "leaderboard", "alice")  // 0-based, по возрастанию
vals, err   := svc.ZRange(ctx, "leaderboard", 0, -1)    // по возрастанию
n, err      := svc.ZCard(ctx, "leaderboard")

svc.ZRem(ctx, "leaderboard", "alice")
```

### Pub/Sub

```go
// Публикация
n, err := svc.Publish(ctx, "events", "hello")

// Подписка
sub := svc.Subscribe(ctx, "events", "other-channel")
defer sub.Close()

for msg := range sub.Channel() {
    fmt.Println(msg.Channel, msg.Payload)
}
```

`Subscription.Close()` отписывается и закрывает соединение. Канал из `Channel()` закрывается автоматически при вызове `Close()`.

## TLS

```go
svc := rockredis.NewService(rockredis.Configs{
    Addr:       "redis.example.com:6380",
    TLSEnabled: true,
})
```

Используется системный пул сертификатов. Для взаимного TLS настройте подключение вне `rockredis`.

## Ограничения

- `Init` закрывает клиент, если `Ping` не прошёл; повторный `Init` заменяет и закрывает предыдущий клиент. После `Stop` вызовы возвращают `redis: client is closed`.
- `Subscribe.Channel()` — буферизированный горутинный мост; медленный консьюмер может задержать доставку сообщений.
- `TLSEnabled: true` использует системный пул сертификатов; взаимный TLS требует кастомной настройки.
- `ZRank` работает в порядке возрастания. Для убывающего порядка, как и для любой команды, которой нет в интерфейсе `Service`, используйте клиент go-redis: `rockredis.Client(svc).ZRevRank(ctx, key, member)`. До `Init` `Client` возвращает `nil`.

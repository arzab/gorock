# rockcron

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockcron/README_RU.md — правьте README в gorock-kit. -->

::: tip Место в GOROCK · Engine · Apps
Планировщик — App в `cmd/apps`. Задачи описываются в `internal/delivery/jobs`. → [Apps](/architecture/apps)
:::

```sh
go get github.com/arzab/gorock-kit/rockcron@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockcron)

Планировщик периодических задач для Go-приложений. Интегрируется с [rockengine](./rockengine) и предоставляет чистый интерфейс для запуска задач по расписанию.

Каждый джоб определяет своё расписание — не нужно разносить расписание и логику по разным местам. Джобы выполняются в изолированных горутинах: если предыдущий запуск ещё не завершён — следующий тик пропускается. При остановке джобы, которые прямо сейчас выполняются, досчитываются до конца — джобы в ожидании следующего тика просто не запускаются.

Реализует интерфейс rockengine `App` (`Init / Exec / Stop`).

## Быстрый старт

```go
app := rockcron.NewApp(cfg,
    func(ctx context.Context, job rockcron.Job, err error) {
        log.Error("cron error", rocklog.Str("job", rockcron.JobName(job)), rocklog.Err(err))
    },
    rockcron.Every("sync-cache",  5*time.Minute, syncCache),
    rockcron.Cron("daily-report", "0 3 * * *",  prepareReport, sendReport),
)

engine.MustRegister("cron", app, rockengine.RestartPolicy{})
engine.Run()
```

## Конфигурация

```go
type Config struct {
    // IANA-имя временной зоны для вычисления расписания cron.
    // Пример: "Europe/Moscow". По умолчанию: UTC.
    Location string `config:",omitempty"`
}
```

## Жизненный цикл

```
NewApp(cfg, onError, jobs...) → Init → Exec (блокирует) → Stop
```

- **Init** проверяет `Location` и все расписания и регистрирует джобы. Ошибки конфигурации возвращаются здесь — старт падает сразу, а не крутит рестарты. Безопасно вызывать повторно после `Stop` для сценариев рестарта.
- **Exec** запускает планировщик и блокирует до отмены `ctx` или вызова `Stop`.
- **Stop** идемпотентен и безопасен для вызова до `Init`. Он только подаёт сигнал; `Exec` возвращается, когда текущие джобы завершатся.

## Определение джобов

### Inline-хелперы (простые случаи)

```go
// Every — запускает с фиксированным интервалом
rockcron.Every("sync-cache", 5*time.Minute, syncCache)

// Cron — запускает по 5-field cron-выражению
rockcron.Cron("daily-report", "0 3 * * *", prepareReport, sendReport)
```

Справка по cron-выражениям:

```
┌───── минута (0-59)
│ ┌─── час (0-23)
│ │ ┌─ день месяца (1-31)
│ │ │ ┌ месяц (1-12)
│ │ │ │ ┌ день недели (0-6, воскресенье=0)
* * * * *
```

### Свой struct (сложные случаи с зависимостями)

Когда джобу нужны инжектированные зависимости (db, repo, mailer), реализуй интерфейс `Job` напрямую. Расписание живёт рядом с бизнес-логикой.

```go
type SyncCacheJob struct {
    repo Repository
}

func (j *SyncCacheJob) Schedule() string                { return "@every 5m" }
func (j *SyncCacheJob) JobName()  string                { return "sync-cache" }
func (j *SyncCacheJob) Run(ctx context.Context) error   { return j.repo.SyncCache(ctx) }
```

```go
app := rockcron.NewApp(cfg, onError, &SyncCacheJob{repo: repo})
```

`JobName()` опционален — реализуй `rockcron.Namer` чтобы задать имя для логов.

## Цепочка хендлеров

`Every` и `Cron` принимают несколько хендлеров, которые выполняются последовательно. Если один хендлер вернул ошибку — остальные пропускаются и вызывается `onError`.

```go
rockcron.Every("pipeline", 10*time.Minute,
    fetchData,    // шаг 1: упал → шаг 2 и 3 не выполняются
    processData,  // шаг 2
    saveResults,  // шаг 3
)
```

## Обработка ошибок

`onError` вызывается когда джоб вернул ошибку или запаниковал:

```go
func(ctx context.Context, job rockcron.Job, err error) {
    switch {
    case errors.Is(err, rockcron.ErrPanic):
        log.Error("job panicked", rocklog.Str("job", rockcron.JobName(job)), rocklog.Err(err))
    default:
        log.Warn("job failed", rocklog.Str("job", rockcron.JobName(job)), rocklog.Err(err))
    }
}
```

Паника внутри самого `onError` перехватывается и пишется в `stderr`.

### Sentinel-ошибки

```go
errors.Is(err, rockcron.ErrPanic) // хендлер джоба запаниковал, паника перехвачена
```

## Поведение

- **SkipIfStillRunning** — если предыдущий запуск джоба ещё не завершён при наступлении следующего тика — тик молча пропускается. Конкурентных дублей не будет.
- **Graceful shutdown** — при остановке планировщик ждёт завершения всех текущих джобов прежде чем `Exec` вернётся.
- **Контекст** — `ctx`, передаваемый в хендлеры, производный от контекста `Exec` и отменяется при остановке App — и при отмене `ctx`, и при `Stop`. Долгие джобы должны это учитывать.
- **Восстановление после паники** — паники в любом хендлере перехватываются и конвертируются в ошибку, обёрнутую в `ErrPanic`, которая передаётся в `onError`. Горутина не падает.

## Ограничения

- **Нет динамической регистрации** — все джобы передаются в `NewApp`. Добавление джобов после старта `Exec` не поддерживается.
- **Один экземпляр на джоб** — в каждый момент времени выполняется не более одного запуска джоба (SkipIfStillRunning).
- **Минутная точность для Cron** — стандартное 5-field выражение имеет гранулярность 1 минута. Для суб-минутных интервалов используй `Every`.
- **Every — не меньше 1 секунды** — robfig/cron округляет более короткие интервалы `@every` до 1s.

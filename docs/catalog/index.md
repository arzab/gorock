# Каталог

Компоненты, привязанные к конкретной технологии: HTTP-сервер, PostgreSQL, Redis, логгер, шина событий, планировщик, Telegram-бот. Они построены по правилам GOROCK поверх [ядра](/gorock-kit/) и лежат в отдельном репозитории [gorock-catalog](https://github.com/arzab/gorock-catalog), чтобы ядро оставалось маленьким и стабильным.

| Модуль | Что это | Место в GOROCK | Статус |
|---|---|---|---|
| [rockhttp](./rockhttp) | HTTP App на `net/http` для маршрутов [rockhandlers](/gorock-kit/rockhandlers) | `cmd/apps` | экспериментальный, рекомендуемый HTTP App |
| [rockfiber](./rockfiber) | HTTP App на Fiber v2, своя модель endpoints | `cmd/apps` | перенесён из kit v0.2.0 |
| [rockbus](./rockbus) | шина событий внутри процесса | `cmd/apps` | перенесён из kit v0.2.0 |
| [rockcron](./rockcron) | планировщик (robfig/cron) | `cmd/apps` | перенесён из kit v0.2.0 |
| [rocktelebot](./rocktelebot) | Telegram-бот (telebot.v3) | `cmd/apps` | перенесён из kit v0.2.0 |
| [rockbun](./rockbun) | PostgreSQL (bun) | `pkg/services` | перенесён из kit v0.2.0 |
| [rockredis](./rockredis) | Redis (go-redis) | `pkg/services` | перенесён из kit v0.2.0 |
| [rocklog](./rocklog) | логгер на `log/slog` | `pkg/infra` | перенесён из kit v0.2.0 |

```sh
go get github.com/arzab/gorock-catalog/rockhttp@latest
```

## «Перенесён из kit v0.2.0»

Эти модули — код gorock-kit `v0.2.0` под новым путём `github.com/arzab/gorock-catalog/<module>`. Они работают и покрыты тестами, но ещё не переведены на новое ядро:

- **Apps** (`rockfiber`, `rockbus`, `rockcron`, `rocktelebot`) реализуют `rockengine.App`, но их обработчики пока не маршруты rockhandlers.
- **Сервисы** (`rockbun`, `rockredis`) не реализуют `Name()` из `rockservices.Service`. Используйте их как дочерние сервисы внутри своего: например, `pkg/services/repository` владеет `rockbun.Service`, сам поднимает его в `Init` и закрывает в `Stop` — см. [Services](/architecture/services).

Их перевод на новое ядро идёт постепенно; страница каждого модуля говорит, что именно ещё не переведено.

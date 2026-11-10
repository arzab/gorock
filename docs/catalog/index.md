# Каталог

Готовые App и сервисы под конкретные технологии. Они построены по правилам GOROCK поверх [ядра](/gorock-kit/) и лежат в отдельном репозитории [gorock-catalog](https://github.com/arzab/gorock-catalog), чтобы ядро оставалось маленьким и стабильным. Берёте только нужные.

| Модуль | Что это | Слой | Статус |
|---|---|---|---|
| [rockhttp](./rockhttp) | HTTP App на `net/http` для обработчиков rockhandlers | App | экспериментальный, рекомендуемый HTTP App · v0.5.0 |
| [rockbun](./rockbun) | клиент PostgreSQL (bun) | Services | перенесён из kit v0.2.0 · v0.5.0 |
| [rockredis](./rockredis) | клиент Redis (go-redis) | Services | перенесён из kit v0.2.0 · v0.5.0 |
| [rocklog](./rocklog) | логгер на `log/slog` | Services · логгер | перенесён из kit v0.2.0 · v0.5.0 |
| [rockfiber](./rockfiber) | HTTP App на Fiber v2, своя модель endpoints | App | перенесён из kit v0.2.0 · v0.5.0 |
| [rockbus](./rockbus) | шина событий внутри процесса | App | перенесён из kit v0.2.0 · v0.5.0 |
| [rockcron](./rockcron) | планировщик (robfig/cron) | App | перенесён из kit v0.2.0 · v0.5.0 |
| [rocktelebot](./rocktelebot) | Telegram-бот (telebot.v3) | App | перенесён из kit v0.2.0 · v0.5.0 |

```sh
go get github.com/arzab/gorock-catalog/rockhttp@v0.5.0
```

Версии каталога совпадают с версиями ядра: каталог `v0.5.0` работает с gorock-kit `v0.5.0`.

## «Перенесён из kit v0.2.0»

Эти модули — код gorock-kit `v0.2.0` под новым путём `github.com/arzab/gorock-catalog/<module>`. Они работают и покрыты тестами, но ещё не переведены на текущее ядро полностью:

- **Сервисы** (`rockbun`, `rockredis`) реализуют `Init` и `Stop`, поэтому подходят каталогу [rockservices](/gorock-kit/rockservices) как есть. Используйте их как дочерние сервисы внутри своего или регистрируйте напрямую — см. [Как писать Service](/guide/services). В их README ещё показана старая регистрация через движок (`RegisterResource`) — такого механизма в ядре нет.
- **Логгер** (`rocklog`) настраивается сервисом-логгером — см. [Логгер — тоже сервис](/guide/services#логгер-тоже-сервис).
- **App** (`rockfiber`, `rockbus`, `rockcron`, `rocktelebot`) реализуют контракт движка, но их обработчики — пока не обработчики rockhandlers.

Страница каждого модуля говорит, что именно ещё не переведено.

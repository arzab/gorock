# Быстрый старт

## Установка CLI

Убедитесь что GOPATH и GOBIN настроены:

```sh
go env -w GOPATH=$HOME/go GOBIN=$HOME/go/bin
export PATH=$GOBIN:$PATH
```

Установите gorock CLI:

```sh
go install github.com/arzab/gorock/cmd/gorock@latest
```

---

## Создание проекта

Создайте модуль и перейдите в корень:

```sh
mkdir myapp && cd myapp
go mod init github.com/yourname/myapp
```

---

## Создание сервиса

Scaffold первого сервиса — например, репозиторий базы данных:

```sh
gorock internal service repository
```

Команда создаст `pkg/services/repository/` с готовым шаблоном: `configs.go`, `interface.go`, `service.go`.

Сервис доступен из любого слоя через геттер:

```go
import "github.com/yourname/myapp/pkg/services"

repo := services.Repository()
```

---

## Создание HTTP endpoint

```sh
gorock delivery endpoint exec
```

Создаёт `internal/delivery/endpoints/exec/` с четырьмя файлами: `endpoint.go`, `handlers.go`, `params.go`, `response.go`.

---

## Структура проекта

После scaffolding проект выглядит так:

```
myapp/
├── cmd/
│   ├── main/               — точка входа: Exec(), Configs
│   └── apps/               — HTTP-сервер, consumer и др.
├── internal/
│   ├── delivery/           — бизнес-логика
│   └── models/             — сущности предметной области
├── pkg/
│   ├── services/           — обёртки над внешними системами
│   ├── infra/              — логирование, трейсинг, eventbus
│   └── libs/               — переиспользуемые библиотеки
└── configs/
    └── configs.json
```

---

## Запуск

```sh
go run ./cmd/main ./configs/configs.json
```

Путь к конфигу — необязательный аргумент. Если не указан — используется `./configs/configs.json`.

---

## Следующий шаг

- [Apps](/architecture/apps) — как устроен слой приложений
- [Delivery](/architecture/delivery) — где живёт бизнес-логика

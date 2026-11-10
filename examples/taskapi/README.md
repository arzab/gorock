# taskapi

A GOROCK project on gorock-kit `v0.5.0` and gorock-catalog `v0.5.0`: create a task and get it over HTTP, tasks are stored in PostgreSQL.

```
pkg/services/        services.go (keys, Configs, Register), logger, repository (PostgreSQL via rockbun)
internal/handlers/   tasks/create.go, tasks/get.go (+ tests without HTTP and PostgreSQL), middleware/log.go
internal/models/     task.go
cmd/apps/            apps.go (HTTP App), http.go (which handler on which method and path)
cmd/main/            main.go
configs/             configs.yaml
migrations/          001_tasks.sql
```

```sh
go test ./...   # handlers, with a fake repository — no PostgreSQL, no HTTP

docker run -d --name tasks-db -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tasks \
  -v "$PWD/migrations:/docker-entrypoint-initdb.d:ro" postgres:16-alpine

DB_PASSWORD=postgres go run ./cmd/main
```

```sh
curl -X POST localhost:8080/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":"write docs"}'
# 201 {"task":{"id":"…","title":"write docs","done":false,"created_at":"…"}}

curl localhost:8080/api/v1/tasks/<id>
# 200 {"id":"…","title":"write docs","done":false,"created_at":"…"}

curl -X POST localhost:8080/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":""}'
# 400 {"code":400,"status":"Bad Request","kind":"invalid","message":"invalid task","fields":{"title":"required"}}

curl localhost:8080/api/v1/tasks/nope
# 404 {"code":404,"status":"Not Found","kind":"not_found","message":"task not found"}
```

Events and scheduled jobs (rockbus, rockcron) will come back once those catalog modules move to the new core.

Docs: https://arzab.github.io/gorock/

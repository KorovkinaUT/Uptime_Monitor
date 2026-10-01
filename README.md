# Uptime Monitor

Uptime Monitor — небольшое клиент-серверное приложение для проверки доступности HTTP/HTTPS-сервисов.
Пользователь добавляет monitor, задаёт интервал и timeout, а приложение автоматически сохраняет результаты проверок в PostgreSQL.

## Архитектура

Проект разделён на три слоя:

- `internal/domain` — модели и правила валидации предметной области
- `internal/application` — use cases, repository contracts, scheduler и worker pool
- `internal/infrastructure` — PostgreSQL, http/client, http/server, конфигурация и static frontend

`cmd/app/main.go` является composition root и собирает все зависимости приложения.

## Стек

- Go 1.26, `net/http`, chi и `log/slog`
- PostgreSQL и `pgx/v5/pgxpool`
- HTML, CSS и Vanilla JavaScript
- Docker Compose и SQL migrations

## Запуск через Docker

Скопируйте пример конфигурации при необходимости:

```bash
cp .env.example .env
docker compose up --build
```

После запуска frontend доступен по адресу `http://localhost:8080`.

Compose запускает сервисы в следующем порядке:

1. PostgreSQL проходит healthcheck
2. `migrate` применяет SQL migrations отдельным process
3. `app` запускает HTTP server

Остановить приложение можно командой:

```bash
docker compose down
```

## Конфигурация

| Переменная | Default | Назначение |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | Порт HTTP listener внутри контейнера |
| `DATABASE_URL` | — | PostgreSQL connection string для локального запуска |
| `WORKER_COUNT` | `5` | Количество worker goroutines |
| `JOB_QUEUE_CAPACITY` | `100` | Максимальный размер очереди jobs |
| `SCHEDULER_INTERVAL` | `1s` | Интервал работы scheduler |
| `DATABASE_PING_TIMEOUT` | `5s` | Timeout проверки подключения к PostgreSQL |
| `SHUTDOWN_TIMEOUT` | `10s` | Timeout graceful shutdown |
| `APP_PORT` | `8080` | Порт хоста в Docker Compose |
| `POSTGRES_DB` | `uptime` | Имя базы данных в Compose |
| `POSTGRES_USER` | `uptime` | Пользователь PostgreSQL в Compose |
| `POSTGRES_PASSWORD` | `uptime` | Пароль PostgreSQL в Compose |

## REST API

| Method | Path | Назначение |
| --- | --- | --- |
| `POST` | `/monitors` | Создать monitor |
| `GET` | `/monitors` | Получить monitors с последним check |
| `GET` | `/monitors/{id}` | Получить один monitor |
| `PUT` | `/monitors/{id}` | Полностью обновить monitor |
| `DELETE` | `/monitors/{id}` | Удалить monitor и checks |
| `GET` | `/monitors/{id}/checks?limit=100` | Получить историю checks |
| `POST` | `/monitors/{id}/check` | Запустить ручную проверку |

`limit` имеет default `100`, допустимый диапазон — от `1` до `1000`.

Пример создания:

```bash
curl -X POST http://localhost:8080/monitors \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Example",
    "url": "https://example.com/health",
    "interval_seconds": 30,
    "timeout_seconds": 5,
    "enabled": true
  }'
```

Проверка считается `up`, если HTTP status находится в диапазоне `200..399`.
Для timeout и transport-level ошибок сохраняется `down`, `status_code` остаётся `null`, а текст ошибки сохраняется в поле `error`.
До первой проверки статус monitor равен `unknown` и в базе данных не хранится.

## Scheduler и worker pool

Scheduler с интервалом `SCHEDULER_INTERVAL` запрашивает только enabled monitors, для которых истёк interval или ещё не было check.
Jobs попадают в bounded channel с capacity `JOB_QUEUE_CAPACITY`.
Фиксированное количество workers выполняет HTTP checks и сохраняет результаты.

In-flight registry не позволяет одновременно проверять один monitor scheduler-ом и ручным endpoint-ом.
Это временное координационное состояние, persistent данные хранятся только в PostgreSQL.

## Migrations

Миграции находятся в `migrations/` и выполняются отдельным Compose service через `golang-migrate/migrate`.
Основной HTTP process не выполняет migrations при старте.

## Тесты и проверки

```bash
go test ./...
go vet ./...
go build ./...
docker compose config
```

Для запуска без Docker требуется доступный PostgreSQL и переменная `DATABASE_URL`:

```bash
go run ./cmd/app
```

# metrics-service

Go-сервис для хранения метрик клиентских приложений в PostgreSQL.
MVP: HTTP API на приём метрик — по одной (`POST /api/v1/metrics`) и батчем
(`POST /api/v1/metrics/batch`, JSON-массив, до `service.MaxBatchSize` = 1000 штук), и healthcheck (`GET /healthz`).
Батч атомарный: пишется через `pgx.CopyFrom`, при любой ошибке (валидация, дубликат) не сохраняется ничего.
Чтения и агрегаций метрик пока нет.

## Стек
- Go 1.26, модуль `metrics-service`
- PostgreSQL 17 (docker-compose, порт **5433** на хосте)
- `pgx/v5` + `pgxpool` — драйвер, без ORM, SQL пишется руками
- `golang-migrate` — миграции в `migrations/` (`NNNNNN_name.up.sql` / `.down.sql`)
- `chi/v5` — HTTP-роутер и middleware (хендлеры — обычные `http.HandlerFunc`)
- `testify/require` — в тестах

## Структура
- `cmd/server/` — точка входа: сборка зависимостей, HTTP-сервер, graceful shutdown по SIGINT/SIGTERM
- `internal/config/` — конфигурация из env (`DATABASE_URL`, `HTTP_ADDR`; есть дефолты для локальной разработки)
- `internal/storage/` — создание пула `pgxpool` с пингом при старте
- `internal/models/` — доменные структуры
- `internal/repository/` — доступ к БД, один репозиторий на сущность; ошибки Postgres маппит в
  доменные (`ErrDuplicate` на unique violation)
- `internal/service/` — бизнес-логика и валидация (`ErrValidation`)
- `internal/transport/http/` — роутер, хендлеры, DTO запросов с json-тегами (в `models` json-тегов нет)
- `migrations/` — SQL-миграции

Поток запроса: handler → service → repository. Хендлер маппит ошибки в HTTP-коды:
`ErrValidation` → 400, `ErrDuplicate` → 409, прочее → 500 (с логированием).
Интерфейсы репозиториев объявлять на стороне потребителя (в `service`), не рядом с реализацией.
Не добавлять лишних слоёв и пакетов (`pkg/`, usecase/adapter и т.п.) без реальной необходимости.

## Команды
```bash
docker compose up -d   # поднять Postgres
make migrate-up        # применить миграции
make migrate-down      # откатить одну миграцию
make run               # запустить сервер
make test              # тесты (нужна запущенная БД с применёнными миграциями)
make lint              # golangci-lint
```
`DATABASE_URL` можно переопределить: `make test DATABASE_URL=...`. Сервер по умолчанию слушает `:8080`.

Тесты хендлеров (`internal/transport/http`) и сервиса (`internal/service`) используют фейки и БД не требуют.

## Соглашения
- Не делай сам коммиты
- Не пиши избыточные комментарии к коду
- Тесты репозиториев — интеграционные, ходят в реальную БД. Это намеренно:
  тесты **не должны** проходить без БД, не мокать БД и не скипать тесты при её отсутствии.
- Код форматируется `gofmt` (табы); перед коммитом — `make lint`, должно быть 0 issues.
- Ошибки оборачивать с контекстом: `fmt.Errorf("что делали: %w", err)`.
- Строки подключения и прочие настройки не хардкодить — только через `internal/config`.
- Схема БД меняется только новой миграцией; существующие миграции не редактировать.
- В таблице `metrics` уникальный индекс `(client_id, app_id, timestamp)` — учитывать при вставках
  (в будущем, вероятно, понадобится upsert).

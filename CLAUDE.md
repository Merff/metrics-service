# metrics-service

Go-сервис для хранения метрик клиентских приложений в PostgreSQL.
MVP: HTTP API на приём метрик — по одной (`POST /api/v1/metrics`) и батчем
(`POST /api/v1/metrics/batch`, JSON-массив, до `service.MaxBatchSize` = 1000 штук), чтение дневных сумм
(`GET /api/v1/metrics/daily?client_id=&app_id=&from=YYYY-MM-DD&to=YYYY-MM-DD`) и healthcheck (`GET /healthz`).
Батч пишется одним запросом `INSERT ... SELECT FROM unnest(...) ON CONFLICT DO NOTHING`: дубликаты (в т.ч. внутри
одного батча) молча пропускаются, ответ `200 {"inserted": N, "skipped": M}`. Ошибка валидации любого элемента — 400, не сохраняется ничего.
Дневные суммы: `app_id` опционален (без него — сумма по всем приложениям клиента), `from`/`to` включительно,
сутки по UTC, период до `service.MaxRangeDays` = 366. Репозиторий отдаёт только дни с данными (`to` у него
исключающий), сервис дозаполняет пропущенные дни нулями.
Чтения и агрегаций метрик пока нет.

## Стек
- Go 1.26, модуль `metrics-service`
- PostgreSQL 17 (docker-compose, порт **5433** на хосте)
- `pgx/v5` + `pgxpool` — драйвер, без ORM, SQL пишется руками
- `golang-migrate/v4` как Go-библиотека (драйвер `pgx/v5`, источник `iofs`) — миграции в `migrations/`
  (`NNNNNN_name.up.sql` / `.down.sql`), встроены в бинарник через `embed` (`migrations/migrations.go`).
  Ставить CLI `migrate` не нужно: `make migrate-*` запускает официальный CLI через `go run`.
- `chi/v5` — HTTP-роутер и middleware (хендлеры — обычные `http.HandlerFunc`)
- `testify/require` — в тестах

## Структура
- `cmd/server/` — точка входа: миграции при старте, сборка зависимостей, HTTP-сервер,
  graceful shutdown по SIGINT/SIGTERM
- `internal/config/` — конфигурация из env (`DATABASE_URL`, `TEST_DATABASE_URL`, `HTTP_ADDR`; есть дефолты для локальной разработки)
- `internal/storage/` — создание пула `pgxpool` с пингом при старте; `MigrateUp`
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
make migrate-up        # применить миграции (сервер и так применяет их при старте)
make migrate-down      # откатить одну миграцию
make run               # запустить сервер
make test              # тесты (нужен запущенный docker compose)
make lint              # golangci-lint
```
`DATABASE_URL` можно переопределить: `make test DATABASE_URL=...`. Сервер по умолчанию слушает `:8080`.

Тесты хендлеров (`internal/transport/http`) и сервиса (`internal/service`) используют фейки и БД не требуют.

## Соглашения
- Не делай сам коммиты
- Не пиши избыточные комментарии к коду
- Тесты репозиториев — интеграционные, ходят в реальную БД. Это намеренно:
  тесты **не должны** проходить без БД, не мокать БД и не скипать тесты при её отсутствии.
- Тесты работают с отдельной БД `metrics_test` (`TEST_DATABASE_URL`, читается через `internal/config`),
  не с рабочей `metrics`. `TestMain` в `internal/repository/helpers_test.go` отказывается работать с БД,
  чьё имя не оканчивается на `_test`, создаёт её при необходимости, накатывает миграции и открывает
  один общий пул `testDB`. Репозиторий в тестах получать через `newTestRepo(t)`: он делает `TRUNCATE`
  до и после теста. Тесты с БД не помечать `t.Parallel()`; новые таблицы добавлять в `TRUNCATE`.
- Код форматируется `gofmt` (табы); перед коммитом — `make lint`, должно быть 0 issues.
- Ошибки оборачивать с контекстом: `fmt.Errorf("что делали: %w", err)`.
- Строки подключения и прочие настройки не хардкодить — только через `internal/config`.
- Схема БД меняется только новой миграцией; существующие миграции не редактировать.
- В таблице `metrics` уникальный индекс `(client_id, app_id, timestamp)` — учитывать при вставках
  (в будущем, вероятно, понадобится upsert).

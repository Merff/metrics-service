# metrics-service

HTTP-сервис на Go для сбора метрик клиентских приложений в PostgreSQL.

Принимает метрики по одной или пачкой и отдаёт суммы по дням за период.
Метрика описывает клиента и его приложение на момент времени: число запросов, заходов на лендинг,
регистраций через API и сколько из них фрод.

## Требования

- Go 1.26+
- Docker (для PostgreSQL)

## Запуск

```bash
docker compose up -d   # PostgreSQL на localhost:5433
make run               # сервер на :8080, миграции применяются при старте
```

Настройки задаются переменными окружения:

| Переменная | По умолчанию |
|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable` |
| `HTTP_ADDR` | `:8080` |
| `TEST_DATABASE_URL` | `postgres://postgres:postgres@localhost:5433/metrics_test?sslmode=disable` |

## API

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/v1/metrics` | Сохранить одну метрику. `201`; дубликат — `409` |
| `POST` | `/api/v1/metrics/batch` | Сохранить массив до 1000 метрик. Дубликаты пропускаются: `200 {"inserted": N, "skipped": M}` |
| `GET` | `/api/v1/metrics/daily` | Суммы по дням (UTC) за период |
| `GET` | `/healthz` | Проверка доступности БД |

Дубликат — метрика с теми же `client_id`, `app_id` и `timestamp`.

```bash
curl -X POST localhost:8080/api/v1/metrics -d '{
  "client_id": 1, "app_id": 1,
  "requests": 200, "landings": 50,
  "api_register_total": 100, "api_register_scam": 10,
  "timestamp": "2026-10-14T12:00:00Z"
}'

curl "localhost:8080/api/v1/metrics/daily?client_id=1&from=2026-10-01&to=2026-10-14"
```

Для `/daily`: `client_id`, `from`, `to` обязательны, `app_id` — опционален (без него суммируются все приложения
клиента).

## Тесты

```bash
docker compose up -d
make test
```

Тесты репозитория работают с отдельной базой `metrics_test`: создают её при необходимости, накатывают миграции
и очищают таблицы до и после каждого теста.

## Миграции

SQL-файлы лежат в `migrations/` и встроены в бинарник; сервер применяет их сам при старте.

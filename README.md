# TripGo — Сервис управления поездками

Бэкенд-сервис для управления поездками такси, разработанный в рамках совместной образовательной программы Авито и НИУ ВШЭ. Проект реализует принципы чистой архитектуры (Clean Architecture), сквозную наблюдаемость (логирование, трейсинг, метрики), транзакционную работу с PostgreSQL и изолированное тестирование.

---

## Требования к окружению

* **Go:** 1.26+
* **Инфраструктура:** Docker, Docker Desktop, утилита `tripgoctl`
* **Инструменты сборки и кодогенерации:** `oapi-codegen`, `goose`, `golangci-lint`, `mockgen`

---

## Быстрый запуск

1. **Поднятие локальной инфраструктуры (PostgreSQL, Jaeger, Prometheus, Grafana):**
   ```bash
   tripgoctl environment start
   tripgoctl connect
   ```
2. **Применение миграций базы данных:**
   ```bash
   make migrate
   ```
3. **Локальный запуск сервиса:**
   ```bash
   make run
   ```

---

## Проверка качества кода и тестирование

Все проверки выполняются с жесткими флагами согласно стандартам разработки:

```bash
make test      # Запуск unit-тестов с флагами race detector (-race) и проверкой независимости (-count=2)
make lint      # Статический анализ кода с помощью golangci-lint
make generate  # Полный цикл кодогенерации (OpenAPI-типы, chi-сервер и mockgen-моки)
make build     # Компиляция бинарника в bin/trip-service
```

---

## Конфигурация (Переменные окружения)

Конфигурирование сервиса выполняется исключительно через переменные окружения. Значения по умолчанию и шаблон для развертывания зафиксированы в файле `.env.example`.

| Переменная | Описание | Значение по умолчанию |
|---|---|---|
| `HTTP_ADDR` | Сетевой адрес и порт HTTP-сервера | `:8080` |
| `HTTP_READ_TIMEOUT` | Таймаут на чтение запроса клиентом | `5s` |
| `HTTP_READ_HEADER_TIMEOUT` | Таймаут на чтение заголовков HTTP | `2s` |
| `HTTP_WRITE_TIMEOUT` | Таймаут на запись тела ответа клиенту | `10s` |
| `HTTP_IDLE_TIMEOUT` | Таймаут удержания keep-alive соединений | `15s` |
| `SHUTDOWN_TIMEOUT` | Лимит времени на Graceful Shutdown | `10s` |
| `LOG_LEVEL` | Уровень детализации логов (`debug`, `info`, `warn`, `error`) | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT`| Эндпоинт коллектора OpenTelemetry (OTLP/HTTP) | `http://localhost:22318` |
| `DATABASE_URL` | URL подключения к базе данных PostgreSQL | — (обязательная) |
| `DATABASE_MAX_CONNS` | Максимальное количество соединений в пуле `pgxpool` | `10` |
| `DATABASE_MIN_CONNS` | Минимальное гарантированное количество соединений | `2` |
| `DATABASE_MAX_CONN_LIFETIME` | Максимальный срок жизни соединения в пуле | `30m` |
| `DATABASE_CONNECT_TIMEOUT` | Таймаут установки соединения с базой данных | `5s` |
| `DATABASE_QUERY_TIMEOUT` | Базовый таймаут выполнения SQL-запроса | `3s` |

---

## Архитектурные решения

### 1. Слоистая архитектура (Clean Architecture)
Кодовая база разделена на четко разграниченные слои с направлением зависимостей снаружи внутрь:
* **Domain (`internal/domain`):** содержит сущности предметной области, правила валидации и чистые ошибки домена (`ErrDriverBusy`, `ErrTripNotFound`, `ErrTripCompleted`, `ValidationError`). Не имеет внешних зависимостей.
* **Use Cases (`internal/usecase`):** сценарии бизнес-логики (`create`, `get`, `finish`). Опираются только на узкие интерфейсы репозитория и менеджера транзакций, объявленные на стороне потребителя (*consumer-side interfaces*).
* **Repository (`internal/repository`):** слой работы с базой данных через пул `pgxpool` и конструктор SQL-запросов `squirrel`.
* **Transport / Delivery (`internal/http`):** маршрутизация Chi, адаптер к контракту `api.ServerInterface`, маппинг ошибок и middleware.

### 2. Централизованная обработка ошибок (RFC 9457)
Ошибки слоя бизнес-логики отделены от HTTP-статусов:
* Доменные ошибки транслируются в едином месте (`internal/http/domainerror`) в стандарт `application/problem+json`.
* Поле `code` возвращает стабильный машиночитаемый идентификатор (`invalid_request`, `driver_busy`, `trip_not_found`, `trip_completed`, `internal_error`).
* Технические подробности (SQL-запросы, ошибки драйвера базы данных) никогда не отдаются клиенту наружу, а изолированно фиксируются в логах.

### 3. Наблюдаемость (Observability)
* **Структурированное логирование (`log/slog`):** логи пишутся в stdout в JSON-формате. Реализован `ContextHandler`, который прозрачно извлекает активный спан OpenTelemetry из контекста и внедряет поля `trace_id` и `span_id` во все логи запроса.
* **Распределенная трассировка (OpenTelemetry + Jaeger):** HTTP-запросы автоматически трассируются middleware `otelhttp`, передавая спаны через OTLP HTTP-экспортер в Jaeger UI.
* **Сбор метрик (OpenTelemetry + Prometheus + Grafana):**
  1. `http.server.request.duration` (histogram, сек) — распределение времени обработки запросов по корзинам.
  2. `http.server.active_requests` (up-down counter, шт) — количество активных HTTP-запросов в обработке.
  3. `db.client.operation.duration` (histogram, сек) — задержка SQL-операций к PostgreSQL.
  4. `tripgo.trips.created` (counter, шт) — суммарный счетчик созданных поездок.
  5. `tripgo.trips.completed` (counter, шт) — суммарный счетчик завершенных поездок.
* **Защита от проблемы High Cardinality:** в атрибут `http.route` передается неизменный шаблон маршрута (`/api/v1/trips/{tripId}`), а не динамический `tripId`, что защищает Prometheus от лавинообразного роста числа временных рядов (time series) в оперативной памяти.

### 4. Работа с транзакциями и защита от состояний гонки (Race Conditions)
* **Менеджер транзакций (`TxManager`):** контекстный менеджер на базе `pgx.Tx` с уровнем изоляции **Read Committed**. Автоматически выполняет `ROLLBACK` при ошибках и паниках внутри функции `Do`, а также переиспользует активную транзакцию при вложенных вызовах.
* **Запрет двух активных поездок у водителя:** реализован через частичный уникальный индекс в PostgreSQL:
  ```sql
  CREATE UNIQUE INDEX idx_trips_driver_active_unique ON trips(driver_id) WHERE status = 'active';
  ```
  Ошибка СУБД `23505` (*unique_violation*) перехватывается репозиторием и транслируется в `domain.ErrDriverBusy`.
* **Безопасное конкурентное завершение:** метод `FinishTrip` защищен от гонки Lost Update с помощью блокировки строки:
  ```sql
  SELECT status FROM trips WHERE id = $1 FOR UPDATE;
  ```
  При одновременных запросах на завершение один запрос успешно обновляет статус, а второй получает ошибку `409 Conflict (trip_completed)`.
```
eof
```
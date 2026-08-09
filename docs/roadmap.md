# Roadmap OrderFlow

Roadmap оформлен как Markdown task list:

- `- [ ]` — задача ещё не выполнена;
- `- [x]` — задача выполнена и проверена.

Галочка ставится только после реализации и проверки результата. Чекбокс всего этапа отмечается после выполнения его Definition of Done. При работе над проектом этот файл нужно обновлять в том же commit, что и соответствующий код.

## Общий прогресс

- [ ] Этап 0. Основа репозитория
- [ ] Этап 1. Catalog и Orders
- [ ] Этап 2. Inventory
- [ ] Этап 3. Payment sandbox
- [ ] Этап 4. Outbox и Kafka
- [ ] Этап 5. Consumers, Inbox и DLQ
- [ ] Этап 6. Checkout Saga
- [ ] Этап 7. Выделение сервисов
- [ ] Этап 8. Observability
- [ ] Этап 9. Kubernetes
- [ ] Этап 10. Performance

## Этап 0. Основа репозитория

### 0.1. Go module и структура

- [x] Инициализировать Go module.
- [x] Создать `cmd/api` и минимальный `main.go`.
- [x] Создать базовые каталоги `internal`, `db`, `api` и `tests`.
- [x] Добавить Makefile с командами `build`, `test` и `lint`.

### 0.2. Конфигурация и логирование

- [x] Описать типизированную конфигурацию приложения.
- [x] Загружать настройки из environment variables.
- [x] Добавить `.env.example` без секретов.
- [x] Валидировать обязательные настройки при старте.
- [x] Настроить структурированные логи через `slog`.

### 0.3. HTTP server и жизненный цикл

- [ ] Создать HTTP server с read, write, idle и shutdown timeout.
- [ ] Добавить middleware для request ID и correlation ID и включить их в HTTP-логи.
- [ ] Реализовать `GET /live`.
- [ ] Реализовать `GET /ready`.
- [ ] Обрабатывать `SIGINT` и `SIGTERM`.
- [ ] Реализовать graceful shutdown через `context.Context`.
- [ ] Добавить unit-тесты health endpoints.

### 0.4. PostgreSQL

- [ ] Добавить PostgreSQL в `compose.yaml`.
- [ ] Настроить healthcheck контейнера.
- [ ] Подключить `pgxpool`.
- [ ] Проверять подключение к БД в readiness probe.
- [ ] Подключить `goose`.
- [ ] Создать первую техническую миграцию.
- [ ] Добавить команды `migrate-up` и `migrate-down`.

### 0.5. Сборка и CI

- [ ] Добавить multi-stage Dockerfile.
- [ ] Добавить `.dockerignore`.
- [ ] Настроить `golangci-lint`.
- [ ] Настроить GitHub Actions для build, lint и tests.
- [ ] Добавить race detector в CI.

### Definition of Done этапа 0

- [ ] `make test` выполняется успешно.
- [ ] `make lint` выполняется успешно.
- [ ] `make compose-up` запускает приложение и PostgreSQL.
- [ ] `curl localhost:8080/live` возвращает успешный ответ.
- [ ] `curl localhost:8080/ready` отражает доступность PostgreSQL.
- [ ] Этап 0 завершён.

## Этап 1. Catalog и Orders

### 1.1. Схема Catalog

- [ ] Создать PostgreSQL-схему `catalog`.
- [ ] Создать миграцию `catalog.products`.
- [ ] Добавить constraints и индексы.
- [ ] Описать SQL-запросы для `sqlc`.
- [ ] Настроить генерацию Go-кода через `sqlc`.

### 1.2. Домен Catalog

- [ ] Реализовать сущность Product.
- [ ] Реализовать создание товара.
- [ ] Реализовать изменение товара.
- [ ] Реализовать получение товара по ID.
- [ ] Реализовать список активных товаров.
- [ ] Добавить table-driven unit tests.

### 1.3. Catalog HTTP API

- [ ] Реализовать `POST /v1/admin/products`.
- [ ] Реализовать `PATCH /v1/admin/products/{productId}`.
- [ ] Реализовать `GET /v1/products`.
- [ ] Реализовать `GET /v1/products/{productId}`.
- [ ] Добавить единый формат Problem Details.
- [ ] Добавить integration-тесты Catalog API.

### 1.4. Домен Orders

- [ ] Реализовать Money без floating point.
- [ ] Реализовать OrderItem со snapshot названия и цены.
- [ ] Реализовать агрегат Order.
- [ ] Реализовать бизнес-статусы заказа.
- [ ] Реализовать допустимые переходы статусов.
- [ ] Реализовать расчёт итоговой стоимости.
- [ ] Добавить table-driven unit tests домена.

### 1.5. Схема и persistence Orders

- [ ] Создать PostgreSQL-схему `orders`.
- [ ] Создать таблицу `orders.orders`.
- [ ] Создать таблицу `orders.order_items`.
- [ ] Создать таблицу `orders.order_status_history`.
- [ ] Добавить SQL-запросы и генерацию `sqlc`.
- [ ] Реализовать транзакционное создание заказа.
- [ ] Реализовать чтение заказа с позициями и историей.

### 1.6. Orders HTTP API

- [ ] Реализовать `POST /v1/orders`.
- [ ] Реализовать `GET /v1/orders/{orderId}`.
- [ ] Реализовать `GET /v1/orders`.
- [ ] Реализовать `POST /v1/orders/{orderId}/cancel`.
- [ ] Добавить валидацию входных данных.
- [ ] Преобразовать доменные ошибки в Problem Details.

### 1.7. HTTP-идемпотентность

- [ ] Создать таблицу `orders.idempotency_keys`.
- [ ] Проверять наличие `Idempotency-Key`.
- [ ] Вычислять и сохранять hash тела запроса.
- [ ] Создавать idempotency record и заказ одной транзакцией.
- [ ] Сохранять код и тело успешного ответа.
- [ ] Возвращать сохранённый ответ при повторе.
- [ ] Возвращать `409 Conflict` для того же ключа с другим body.

### Definition of Done этапа 1

- [ ] Повторный запрос не создаёт второй заказ.
- [ ] Название и цена товара сохранены snapshot-ом.
- [ ] Для денег нигде не используется `float32` или `float64`.
- [ ] Недопустимые переходы статусов отклоняются.
- [ ] Integration-тесты работают с настоящей PostgreSQL через Testcontainers.
- [ ] Этап 1 завершён.

## Этап 2. Inventory

### 2.1. Модель данных

- [ ] Создать PostgreSQL-схему `inventory`.
- [ ] Создать таблицу `inventory.stock_items`.
- [ ] Создать таблицы резервов и их позиций.
- [ ] Добавить constraints для остатков.
- [ ] Добавить SQL-запросы и генерацию `sqlc`.

### 2.2. Операции с остатками

- [ ] Реализовать административное изменение остатка.
- [ ] Реализовать чтение остатка.
- [ ] Реализовать атомарное резервирование.
- [ ] Реализовать идемпотентное освобождение резерва.
- [ ] Реализовать подтверждение списания.
- [ ] Запретить отрицательный доступный остаток на уровне БД.

### 2.3. Expiration

- [ ] Добавить срок действия резерва.
- [ ] Реализовать поиск просроченных резервов через `SKIP LOCKED`.
- [ ] Реализовать bounded worker pool для очистки.
- [ ] Корректно останавливать cleaner через context.

### 2.4. Конкурентные тесты

- [ ] Проверить одновременное резервирование одного товара.
- [ ] Проверить повторное освобождение резерва.
- [ ] Запустить тесты с `-race` и `-shuffle=on`.

### Definition of Done этапа 2

- [ ] При 50 параллельных попытках купить последний товар успешен ровно один заказ.
- [ ] После expiration товар снова доступен.
- [ ] Повторная команда не изменяет остаток второй раз.
- [ ] Этап 2 завершён.

## Этап 3. Payment sandbox

### 3.1. Модель платежей

- [ ] Создать PostgreSQL-схему `payments`.
- [ ] Создать таблицу payment intents.
- [ ] Описать статусы и допустимые переходы платежа.
- [ ] Ввести уникальный operation key.

### 3.2. Платёжные операции

- [ ] Реализовать создание payment intent.
- [ ] Реализовать authorize.
- [ ] Реализовать capture.
- [ ] Реализовать cancel authorization.
- [ ] Реализовать refund.
- [ ] Реализовать получение статуса.

### 3.3. Управляемые сбои

- [ ] Добавить поведение `success`.
- [ ] Добавить поведение `declined`.
- [ ] Добавить поведение `timeout`.
- [ ] Добавить поведение `temporary_error`.
- [ ] Добавить поведение `duplicate_callback`.
- [ ] Добавить поведение `late_success`.

### 3.4. Надёжность платежей

- [ ] Сделать все операции идемпотентными по operation key.
- [ ] Разделить retryable и terminal errors.
- [ ] Реализовать сверку статуса после неопределённого результата.
- [ ] Добавить тесты повторной команды и позднего callback.

### Definition of Done этапа 3

- [ ] Повторная `AuthorizePayment` не создаёт второе списание.
- [ ] Timeout и late success не приводят к повторной оплате.
- [ ] Declined payment не повторяется автоматически.
- [ ] Этап 3 завершён.

## Этап 4. Outbox и Kafka

### 4.1. Контракты сообщений

- [ ] Описать единый event envelope.
- [ ] Добавить `eventId`, correlation ID и causation ID.
- [ ] Зафиксировать правила event type и version.
- [ ] Зафиксировать Kafka topics и message keys.
- [ ] Добавить contract tests для JSON-событий.

### 4.2. Transactional Outbox

- [ ] Создать схему `messaging`.
- [ ] Создать таблицу и индекс Outbox.
- [ ] Записывать доменные события в бизнес-транзакции.
- [ ] Получать batch через `FOR UPDATE SKIP LOCKED`.

### 4.3. Outbox Relay

- [ ] Подключить `franz-go` producer.
- [ ] Реализовать bounded channel и worker pool.
- [ ] Реализовать backpressure.
- [ ] Отмечать успешно опубликованные сообщения.
- [ ] Реализовать graceful shutdown relay.

### 4.4. Retry и наблюдаемость

- [ ] Классифицировать ошибки публикации.
- [ ] Реализовать exponential backoff с jitter.
- [ ] Ограничить число попыток.
- [ ] Добавить метрики pending, attempts и oldest age.
- [ ] Добавить структурированные логи публикации.

### Definition of Done этапа 4

- [ ] При недоступной Kafka заказ сохраняется вместе с Outbox event.
- [ ] После восстановления Kafka событие публикуется автоматически.
- [ ] Повторно создавать заказ для публикации не требуется.
- [ ] Несколько relay безопасно обрабатывают общую таблицу.
- [ ] Этап 4 завершён.

## Этап 5. Consumers, Inbox и DLQ

### 5.1. Базовый consumer

- [ ] Подключить consumer group через `franz-go`.
- [ ] Реализовать ручной commit offset.
- [ ] Ограничить параллелизм обработки.
- [ ] Реализовать graceful stop чтения Kafka.

### 5.2. Inbox

- [ ] Создать таблицу Inbox.
- [ ] Выполнять Inbox, domain changes и Outbox в одной транзакции.
- [ ] Подтверждать offset только после commit DB.
- [ ] Игнорировать уже обработанный `messageId`.

### 5.3. Notification Worker

- [ ] Обработать команды уведомлений.
- [ ] Сохранять отправленные уведомления в БД.
- [ ] Выводить результат в структурированный лог.
- [ ] Добавить integration-тест повторной доставки.

### 5.4. Retry и DLQ

- [ ] Добавить ограниченную retry policy.
- [ ] Создать хранилище dead letters.
- [ ] Сохранять исходные topic, partition, offset, headers и payload.
- [ ] Реализовать просмотр DLQ.
- [ ] Реализовать retry с исходным `eventId`.
- [ ] Реализовать discard с аудитом.

### 5.5. Failure tests

- [ ] Воспроизвести падение после DB commit до commit offset.
- [ ] Проверить poison message.
- [ ] Проверить десятикратную доставку одного сообщения.

### Definition of Done этапа 5

- [ ] Одно Kafka-сообщение, доставленное десять раз, создаёт одно уведомление.
- [ ] После рестарта consumer безопасно продолжает обработку.
- [ ] Необрабатываемое сообщение попадает в DLQ.
- [ ] Этап 5 завершён.

## Этап 6. Checkout Saga

### 6.1. Модель Workflow

- [ ] Создать таблицу checkout workflow.
- [ ] Описать состояния и допустимые переходы.
- [ ] Хранить версию процесса и optimistic concurrency token.
- [ ] Хранить последнюю ошибку и время следующей попытки.

### 6.2. Успешный сценарий

- [ ] Запускать workflow по `OrderCreated`.
- [ ] Отправлять `ReserveStock`.
- [ ] После `StockReserved` отправлять `AuthorizePayment`.
- [ ] После `PaymentAuthorized` отправлять `CreateDelivery`.
- [ ] После `DeliveryCreated` подтверждать заказ.
- [ ] Запрашивать уведомление.

### 6.3. Компенсации

- [ ] Отклонять заказ при недостаточном остатке.
- [ ] Освобождать резерв при ошибке оплаты.
- [ ] Возвращать платёж при ошибке доставки.
- [ ] Освобождать резерв после refund.
- [ ] Обрабатывать ошибку компенсации.
- [ ] Сделать компенсации идемпотентными.

### 6.4. Timeout и восстановление

- [ ] Добавить deadline каждого шага.
- [ ] Планировать ограниченные повторные попытки.
- [ ] Восстанавливать незавершённые workflow после рестарта.
- [ ] Обрабатывать события, пришедшие повторно или поздно.
- [ ] Не применять недопустимые переходы.

### 6.5. Сценарные тесты

- [ ] Success.
- [ ] Stock failure.
- [ ] Payment failure.
- [ ] Delivery failure.
- [ ] Cancel.
- [ ] Timeout.
- [ ] Compensation failure.

### Definition of Done этапа 6

- [ ] Все сценарные тесты проходят.
- [ ] После рестарта Saga продолжает работу с сохранённого состояния.
- [ ] Повторные события не запускают бизнес-операции второй раз.
- [ ] Этап 6 завершён.

## Этап 7. Выделение сервисов

### 7.1. Подготовка границ

- [ ] Зафиксировать ADR о границах сервисов.
- [ ] Удалить межмодульный доступ к чужим таблицам.
- [ ] Описать Protobuf-контракты.
- [ ] Настроить Buf lint и breaking checks.
- [ ] Добавить contract tests.

### 7.2. Inventory Service

- [ ] Выделить отдельный процесс.
- [ ] Выделить отдельную БД или PostgreSQL instance.
- [ ] Перевести синхронные операции на gRPC.
- [ ] Оставить события состояния в Kafka.
- [ ] Проверить совместимость и отказ сервиса.

### 7.3. Payment Service

- [ ] Выделить отдельный процесс и хранилище.
- [ ] Добавить gRPC API.
- [ ] Перенести payment events в Kafka.
- [ ] Проверить idempotency через сетевые retry.

### 7.4. Delivery Service

- [ ] Выделить отдельный процесс и хранилище.
- [ ] Добавить gRPC API.
- [ ] Перенести delivery events в Kafka.
- [ ] Проверить компенсации при недоступности сервиса.

### 7.5. End-to-end проверка

- [ ] Обновить Compose для всех сервисов.
- [ ] Добавить E2E happy path.
- [ ] Добавить E2E failure scenarios.
- [ ] Проверить graceful shutdown каждого процесса.

### Definition of Done этапа 7

- [ ] Order Service не имеет прямого доступа к таблицам Inventory, Payment или Delivery.
- [ ] Контракты проходят Buf breaking check.
- [ ] E2E-сценарии проходят с отдельными сервисами.
- [ ] Этап 7 завершён.

## Этап 8. Observability

### 8.1. Базовая телеметрия

- [ ] Подключить OpenTelemetry SDK.
- [ ] Настроить OTLP export через Collector.
- [ ] Добавить resource attributes и service names.
- [ ] Добавить trace ID и correlation ID в `slog`.

### 8.2. Distributed tracing

- [ ] Инструментировать HTTP server и clients.
- [ ] Инструментировать PostgreSQL.
- [ ] Передавать W3C Trace Context через Kafka headers.
- [ ] Создать spans для Outbox, consumers и Saga steps.

### 8.3. Метрики

- [ ] Добавить HTTP RED metrics.
- [ ] Добавить PostgreSQL pool metrics.
- [ ] Добавить Kafka consumer и handler metrics.
- [ ] Добавить Outbox metrics.
- [ ] Добавить бизнес-метрики Orders и Checkout.

### 8.4. Observability stack

- [ ] Добавить Prometheus.
- [ ] Добавить Grafana.
- [ ] Добавить Tempo.
- [ ] Добавить Loki.
- [ ] Подготовить dashboards и datasource provisioning.

### 8.5. Проверка SLO

- [ ] Зафиксировать SLO и методы измерения.
- [ ] Создать тестовый заказ и сохранить пример trace.
- [ ] Проверить поиск логов по trace ID и correlation ID.
- [ ] Добавить alerts для lag, DLQ и старых Outbox events.

### Definition of Done этапа 8

- [ ] Заказ прослеживается от HTTP-запроса до уведомления.
- [ ] Метрики доступны в Grafana.
- [ ] Логи, метрики и traces связываются идентификаторами.
- [ ] Этап 8 завершён.

## Этап 9. Kubernetes

### 9.1. Контейнеры и конфигурация

- [ ] Проверить non-root Docker images.
- [ ] Добавить ConfigMap и Secret.
- [ ] Добавить Service для каждого сетевого компонента.
- [ ] Добавить Deployment для каждого приложения.

### 9.2. Health и shutdown

- [ ] Настроить startup probe.
- [ ] Настроить readiness probe.
- [ ] Настроить liveness probe.
- [ ] Согласовать `terminationGracePeriodSeconds` с shutdown timeout.
- [ ] Проверить SIGTERM во время HTTP и Kafka обработки.

### 9.3. Helm

- [ ] Создать Helm chart.
- [ ] Вынести image, replicas, resources и endpoints в values.
- [ ] Добавить values для локального и тестового окружения.
- [ ] Добавить `helm lint` и template validation в CI.

### 9.4. Устойчивость и масштабирование

- [ ] Добавить resource requests и limits.
- [ ] Добавить HPA для подходящих компонентов.
- [ ] Добавить PodDisruptionBudget.
- [ ] Настроить rolling update strategy.
- [ ] Провести тест удаления pod во время обработки.

### Definition of Done этапа 9

- [ ] Helm chart устанавливается в чистый namespace.
- [ ] Rolling update не теряет и не применяет повторно бизнес-операции.
- [ ] Неготовые pod не получают трафик.
- [ ] Этап 9 завершён.

## Этап 10. Performance

### 10.1. Baseline

- [ ] Определить нагрузочный профиль.
- [ ] Подготовить k6-сценарии.
- [ ] Замерить latency, throughput и error rate.
- [ ] Зафиксировать CPU, memory и DB/Kafka показатели.

### 10.2. Benchmarks и profiling

- [ ] Добавить Go benchmarks для критических функций.
- [ ] Снять CPU profile.
- [ ] Снять heap и allocation profiles.
- [ ] Снять goroutine и mutex profiles.
- [ ] Проверить блокировки и утечки goroutine.

### 10.3. Оптимизация

- [ ] Выбрать bottleneck на основании измерений.
- [ ] Внести одно измеримое изменение.
- [ ] Повторить benchmarks и k6-тест.
- [ ] Проверить отсутствие регрессий и усложнения без пользы.

### 10.4. Результаты

- [ ] Сохранить параметры окружения и команды запуска.
- [ ] Описать baseline.
- [ ] Описать найденные bottlenecks.
- [ ] Описать сделанные изменения.
- [ ] Добавить сравнение результатов до и после.

### Definition of Done этапа 10

- [ ] Результаты воспроизводимы по документации.
- [ ] Оптимизация подтверждена измерениями.
- [ ] В README есть ссылка на performance report.
- [ ] Этап 10 завершён.

## Критерий готового portfolio-проекта

- [ ] Локальный запуск выполняется одной командой.
- [ ] Есть архитектурная диаграмма, описание Saga и sequence diagrams.
- [ ] Есть OpenAPI и Protobuf-контракты.
- [ ] Есть миграции, integration, contract и end-to-end tests.
- [ ] Есть воспроизводимые failure scenarios.
- [ ] Есть Grafana dashboards и пример trace одного заказа.
- [ ] Есть Kubernetes manifests и Helm chart.
- [ ] CI pipeline проверяет build, lint, tests и контракты.
- [ ] Сохранены результаты k6 и `pprof`.
- [ ] Ключевые решения и компромиссы зафиксированы в ADR.

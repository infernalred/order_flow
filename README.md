# OrderFlow

OrderFlow — учебная event-driven платформа оформления, оплаты, резервирования и доставки заказов для небольшого маркетплейса.

Главная цель проекта — изучить Go на production-like backend-системе, а не переносить привычный CRUD из .NET один в один. Проект начинается как модульный монолит и постепенно развивается в набор взаимодействующих сервисов с Kafka, Saga, наблюдаемостью и Kubernetes.

> Event-driven здесь не означает Event Sourcing. Актуальное состояние хранится в PostgreSQL, а события используются для надёжного взаимодействия компонентов.

## Чему должен научить проект

- идиоматичному Go: композиции, маленьким интерфейсам, явной обработке ошибок и `context.Context`;
- построению HTTP API и, на позднем этапе, gRPC API;
- работе с PostgreSQL, транзакциями, `pgx/v5`, `sqlc` и миграциями;
- конкурентной обработке без бесконтрольного создания goroutine;
- надёжной доставке сообщений через Transactional Outbox и Inbox;
- идемпотентности HTTP-запросов, consumers и внешних операций;
- оркестрации Saga, компенсациям, retry и DLQ;
- graceful shutdown, тестированию с race detector и профилированию;
- OpenTelemetry, Prometheus, Grafana, Tempo и Loki;
- контейнеризации, Kubernetes, Helm и CI/CD.

## Возможности системы

Покупатель сможет:

- просматривать каталог;
- создавать и оплачивать заказ;
- получать рассчитанную итоговую стоимость;
- отслеживать состояние и историю заказа;
- отменять заказ;
- получать уведомления.

Администратор сможет:

- управлять товарами и остатками;
- просматривать заказы и состояние workflow;
- просматривать DLQ и повторно запускать неудачные операции.

Система должна корректно переживать повторные HTTP-запросы и сообщения Kafka, падение consumer, недоступность Kafka, тайм-аут платёжного провайдера, конкуренцию за последний товар, рестарт процесса, SIGTERM во время обработки и нарушение порядка доставки событий.

## Архитектурный подход

Развитие проекта намеренно разбито на этапы:

1. Модульный монолит с модулями `catalog`, `orders`, `inventory`, `payments`, `delivery` и `notifications`.
2. Transactional Outbox, Kafka и первый отдельный consumer — Notification Worker.
3. Checkout Workflow, управляющий Saga.
4. Последовательное выделение Inventory, Payment и Delivery в отдельные сервисы.
5. Развёртывание компонентов в Kubernetes.

На первом этапе модули работают в одном процессе и используют одну PostgreSQL с отдельными схемами. Межмодульное взаимодействие выполняется синхронно через Go-интерфейсы. Это позволяет сначала освоить язык и транзакции, не пряча ошибки проектирования за сетевыми границами.

Подробности: [архитектура и надёжность](docs/architecture.md).

## Основной сценарий

```text
POST /v1/orders
        │
        ▼
Order + OrderCreated сохранены в PostgreSQL одной транзакцией
        │
        ▼
Outbox Relay публикует событие в Kafka
        │
        ▼
Checkout Workflow запускает Saga
        │
        ├── ReserveStock
        ├── AuthorizePayment
        ├── CreateDelivery
        └── OrderConfirmed → NotificationRequested
```

Компенсации:

- недостаточно товара: заказ получает статус `REJECTED`;
- оплата отклонена: резерв освобождается, заказ отклоняется;
- доставка не создана: платёж возвращается, резерв освобождается, заказ отклоняется.

Бизнес-статус заказа отделён от технического статуса Saga. Например, заказ может оставаться `PENDING`, пока workflow проходит шаги `STOCK_PENDING`, `PAYMENT_PENDING` и `DELIVERY_PENDING`.

## Модули

| Модуль | Ответственность |
| --- | --- |
| Catalog | Товары, названия, описание, актуальная цена, активность |
| Orders | Состав и стоимость заказа, статус, история, отмена |
| Inventory | Остатки, резервирование, освобождение и expiration |
| Payments | Sandbox-платежи, authorize, refund и управляемые сбои |
| Delivery | Адрес, доставка, временное окно и статусы |
| Checkout Workflow | Состояние Saga, retry и компенсации |
| Notifications | Уведомления по событиям заказа |

Деньги хранятся только в минимальных единицах валюты (`199900` = `1 999,00 RUB`). `float32` и `float64` для денег не используются. Название и цена товара сохраняются в заказе как snapshot.

## Технологии

| Область | Выбор |
| --- | --- |
| Язык | Go 1.26.5 |
| HTTP | `net/http` + `chi` |
| Внутреннее API | gRPC, после выделения сервисов |
| Контракты | Protobuf + Buf, OpenAPI |
| База данных | PostgreSQL |
| Доступ к данным | `pgx/v5` + `sqlc` |
| Миграции | `goose` |
| Брокер | Apache Kafka |
| Kafka client | `franz-go` |
| Логи | `log/slog` |
| Наблюдаемость | OpenTelemetry, Prometheus, Grafana, Tempo, Loki |
| Интеграционные тесты | Testcontainers for Go |
| Нагрузочные тесты | k6 |
| Доставка | Docker, Kubernetes, Helm, GitHub Actions |

Redis добавляется только тогда, когда появится подтверждённый сценарий его применения.

## Планируемый HTTP API

```text
POST   /v1/admin/products
PATCH  /v1/admin/products/{productId}
GET    /v1/products
GET    /v1/products/{productId}

PUT    /v1/admin/stock/{productId}
GET    /v1/admin/stock/{productId}

POST   /v1/orders
GET    /v1/orders/{orderId}
GET    /v1/orders
POST   /v1/orders/{orderId}/cancel

GET    /v1/admin/workflows/{orderId}
GET    /v1/admin/outbox
GET    /v1/admin/dead-letters
POST   /v1/admin/dead-letters/{id}/retry
POST   /v1/admin/dead-letters/{id}/discard
```

`POST /v1/orders` требует заголовок `Idempotency-Key`. В синхронной версии endpoint возвращает `201 Created`, а после перехода на асинхронную Saga — `202 Accepted` и ссылку на созданный заказ.

## Планируемая структура репозитория

```text
orderflow/
├── cmd/
│   ├── api/
│   ├── outbox-relay/
│   └── notification-worker/
├── internal/
│   ├── catalog/
│   ├── orders/
│   ├── inventory/
│   ├── payments/
│   ├── delivery/
│   ├── notifications/
│   ├── messaging/
│   └── platform/
├── db/
│   ├── migrations/
│   ├── queries/
│   └── sqlc.yaml
├── api/
│   ├── openapi/
│   └── proto/
├── deployments/
│   ├── docker/
│   ├── compose/
│   ├── helm/
│   └── kubernetes/
├── docs/
├── tests/
│   ├── integration/
│   ├── contract/
│   └── load/
├── Makefile
├── Dockerfile
├── compose.yaml
└── go.mod
```

Правило зависимостей: `cmd → internal modules → platform abstractions`. Один доменный модуль не импортирует PostgreSQL-реализацию другого модуля.

## Путь из .NET в Go

Несколько правил для проекта:

- не создавать интерфейс для каждого типа; интерфейс объявляет потребитель и только для реально нужных методов;
- не имитировать исключения: ошибки возвращаются явно, оборачиваются через `%w` и классифицируются через `errors.Is`/`errors.As`;
- не строить DI-контейнер на старте: зависимости явно собираются в `main`;
- не добавлять goroutine без понятного владельца, отмены через context и ожидания завершения;
- не прятать SQL за универсальным repository/framework слоем;
- начинать с стандартной библиотеки и добавлять зависимости для конкретной задачи;
- принимать `context.Context` первым аргументом и не хранить его в структурах;
- предпочитать table-driven tests.

Пример небольшого интерфейса со стороны потребителя:

```go
type ProductReader interface {
	GetProducts(ctx context.Context, ids []uuid.UUID) ([]Product, error)
}
```

## Проверки качества

Минимальный набор команд для каждого этапа:

```bash
go test ./...
go test -race ./...
go test -shuffle=on ./...
```

Позже он будет объединён в команды `make test`, `make lint` и CI pipeline.

## Roadmap

Разработка начинается с HTTP-сервера, конфигурации, PostgreSQL и graceful shutdown. Kafka, микросервисы и Kubernetes добавляются только после появления работающего модульного монолита.

Полный порядок задач и Definition of Done: [roadmap](docs/roadmap.md). Roadmap разбит на небольшие подэтапы и оформлен как Markdown task list, поэтому выполненные задачи можно отмечать галочками прямо по ходу разработки.

## Что намеренно не входит в первую версию

Elasticsearch, GraphQL, Event Sourcing, отдельная CQRS read database, Service Mesh, Kubernetes Operator, Vault, Keycloak, несколько Kafka-кластеров, распределённый кеш «для всего» и универсальная event-bus библиотека.

Эти технологии можно изучать отдельно, но они не должны мешать основной цели — освоить Go и надёжную обработку заказов.

## Текущий статус

Проект находится на этапе проектирования. Следующий шаг — **Этап 0: основа репозитория**.

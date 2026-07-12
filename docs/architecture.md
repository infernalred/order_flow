# Архитектура OrderFlow

Этот документ фиксирует целевую архитектуру и правила надёжности. Реализовывать всё сразу не требуется: порядок внедрения описан в [roadmap](roadmap.md).

## Границы системы

```text
                         ┌───────────────────┐
                         │      Client       │
                         └─────────┬─────────┘
                                   │ HTTP
                         ┌─────────▼─────────┐
                         │     Order API     │
                         └─────────┬─────────┘
                                   │
                         ┌─────────▼─────────┐
                         │ Orders + Outbox DB│
                         └─────────┬─────────┘
                                   │ Outbox Relay
┌──────────────────────────────────▼──────────────────────────────────┐
│                              Kafka                                 │
│ order.events · checkout.commands · stock.events · payment.events   │
│ delivery.events · notification.commands                            │
└──────┬──────────────────┬──────────────────┬───────────────────────┘
       │                  │                  │
┌──────▼───────┐  ┌───────▼───────┐  ┌──────▼───────┐
│   Workflow   │  │   Inventory   │  │    Payment   │
│   Service    │  │    Service    │  │    Service   │
└──────────────┘  └───────────────┘  └──────────────┘
       │
┌──────▼───────┐  ┌──────────────────────┐
│   Delivery   │  │ Notification Worker  │
│   Service    │  │ Email / Webhook      │
└──────────────┘  └──────────────────────┘
```

Это конечная, а не начальная архитектура. Первые этапы работают одним процессом и одной базой данных с отдельными PostgreSQL-схемами.

## Состояния

Бизнес-состояние заказа:

```go
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusConfirmed  OrderStatus = "CONFIRMED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusShipped    OrderStatus = "SHIPPED"
	OrderStatusDelivered  OrderStatus = "DELIVERED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
	OrderStatusRejected   OrderStatus = "REJECTED"
)
```

Техническое состояние checkout-процесса:

```go
type CheckoutStatus string

const (
	CheckoutStatusStarted         CheckoutStatus = "STARTED"
	CheckoutStatusStockPending    CheckoutStatus = "STOCK_PENDING"
	CheckoutStatusPaymentPending  CheckoutStatus = "PAYMENT_PENDING"
	CheckoutStatusDeliveryPending CheckoutStatus = "DELIVERY_PENDING"
	CheckoutStatusCompleted       CheckoutStatus = "COMPLETED"
	CheckoutStatusCompensating    CheckoutStatus = "COMPENSATING"
	CheckoutStatusCompensated     CheckoutStatus = "COMPENSATED"
	CheckoutStatusFailed          CheckoutStatus = "FAILED"
)
```

Workflow хранит текущий шаг, успешные шаги, необходимые компенсации, следующую попытку, последнюю ошибку и версию процесса. Он не владеет данными заказа, склада, оплаты или доставки.

## Saga

Успешная цепочка:

```text
OrderCreated
  → ReserveStock
  → StockReserved
  → AuthorizePayment
  → PaymentAuthorized
  → CreateDelivery
  → DeliveryCreated
  → OrderConfirmed
  → NotificationRequested
```

Если оплата не прошла:

```text
StockReserved
  → AuthorizePayment
  → PaymentFailed
  → ReleaseStock
  → StockReleased
  → OrderRejected
```

Если доставка не создана:

```text
PaymentAuthorized
  → CreateDelivery
  → DeliveryCreationFailed
  → RefundPayment
  → ReleaseStock
  → OrderRejected
```

## События и команды

Событие — уже произошедший факт, поэтому называется в прошедшем времени: `OrderCreated`, `StockReserved`, `PaymentAuthorized`.

Команда — намерение выполнить действие, поэтому называется в повелительной форме: `ReserveStock`, `ReleaseStock`, `AuthorizePayment`, `RefundPayment`.

Основные контракты:

| События | Команды |
| --- | --- |
| `order.created.v1` | `stock.reserve.v1` |
| `order.confirmed.v1` | `stock.release.v1` |
| `order.rejected.v1` | `payment.authorize.v1` |
| `stock.reserved.v1` | `payment.refund.v1` |
| `stock.reservation_failed.v1` | `delivery.create.v1` |
| `payment.authorized.v1` | `delivery.cancel.v1` |
| `payment.failed.v1` | `notification.send.v1` |
| `delivery.created.v1` |  |
| `delivery.creation_failed.v1` |  |

Все сообщения имеют единый envelope:

```json
{
  "eventId": "019bd0af-8b44-7f3c-9c03-e5683e34f11c",
  "eventType": "order.created",
  "eventVersion": 1,
  "producer": "order-service",
  "aggregateType": "order",
  "aggregateId": "3f76dcbc-6c76-418a-9664-d57546a2787f",
  "occurredAt": "2026-07-12T10:25:11.424Z",
  "correlationId": "45599fc7-da47-455d-b2b0-aa335150172a",
  "causationId": "adc4a10e-afcd-489e-91f4-a46362c37639",
  "payload": {}
}
```

Для событий заказа Kafka key равен `orderId`. Так события одного заказа попадают в одну partition и сохраняют порядок внутри неё. W3C Trace Context передаётся в Kafka headers.

## Transactional Outbox

Заказ, позиции, история статуса и событие Outbox сохраняются одной PostgreSQL-транзакцией:

```text
INSERT order
INSERT order_items
INSERT order_status_history
INSERT outbox_events
COMMIT
```

Публикация в Kafka после отдельного `repository.Save` ненадёжна: процесс может завершиться между сохранением состояния и публикацией.

Несколько экземпляров relay могут безопасно получать пачки через `FOR UPDATE SKIP LOCKED`. Неопубликованные записи имеют `available_at`, число попыток и последнюю ошибку. Успешная публикация отмечает `published_at`.

## Inbox и consumer

Consumer обрабатывает сообщение так:

1. Получает сообщение и начинает DB-транзакцию.
2. Проверяет пару `(consumer_name, message_id)` в Inbox.
3. Изменяет бизнес-состояние.
4. Записывает новые события в Outbox.
5. Записывает сообщение в Inbox и делает commit DB.
6. Только после этого подтверждает Kafka offset.

Если процесс завершится между commit DB и commit offset, сообщение придёт повторно, но Inbox предотвратит повторное бизнес-действие.

## Идемпотентность HTTP

`POST /v1/orders` требует `Idempotency-Key`.

- Первый запрос создаёт запись `PROCESSING`; заказ создаётся в той же транзакции.
- Код и тело ответа сохраняются.
- Повторный запрос возвращает сохранённый ответ.
- Тот же ключ с другим request body возвращает `409 Conflict`.

Надёжность обеспечивается PostgreSQL, без Redis.

## Concurrency

Goroutine используется только при наличии параллельной работы, владельца и понятного жизненного цикла.

Outbox Relay применяет ограниченный worker pool, bounded channel, backpressure, `context.Context` и ожидание workers. Cleaner просроченных резервов использует `FOR UPDATE SKIP LOCKED`.

Резервирование остатка выполняется атомарно:

```sql
UPDATE inventory.stock_items
SET reserved_quantity = reserved_quantity + $2,
    version = version + 1
WHERE product_id = $1
  AND on_hand_quantity - reserved_quantity >= $2
RETURNING *;
```

Ноль обновлённых строк означает недостаточный остаток.

## Retry и DLQ

Повторяются временные ошибки: network timeout, недоступность Kafka, временная ошибка провайдера, serialization failure и HTTP `502/503/504`.

Не повторяются ошибки валидации, отклонённая оплата, отсутствующий товар, недостаточный остаток и недопустимый переход состояния.

Задержка ограничена сверху и содержит jitter:

```text
delay = min(base × 2^attempt + jitter, maxDelay)
```

После исчерпания попыток сообщение попадает в DLQ вместе с исходными topic, partition, offset, key, headers, payload, consumer name, ошибкой и числом попыток. При ручном retry сохраняется исходный `eventId`.

## Failure Lab

Проект должен позволять намеренно воспроизводить сбои:

1. **Consumer падает после DB commit.** После рестарта Kafka доставляет сообщение снова, Inbox блокирует повторную операцию.
2. **Kafka недоступна.** Заказ сохраняется, событие остаётся в Outbox и публикуется после восстановления Kafka.
3. **Два заказа покупают последний товар.** Ровно один атомарный резерв проходит успешно.
4. **Платёж отвечает поздно.** Тайм-аут клиента не приводит к повторному списанию; состояние сверяется по идемпотентному operation key.
5. **SIGTERM во время обработки.** Readiness выключается, новые запросы и сообщения не принимаются, текущие handlers завершаются, затем закрываются Kafka clients и PostgreSQL pool.

## Observability и SLO

Один trace проходит через HTTP, PostgreSQL, Outbox, Kafka, Workflow, Inventory, Payment и Delivery. Traces и metrics экспортируются через OpenTelemetry Collector, структурированные логи `slog` пишутся в stdout.

Начальные цели:

- 99% запросов `POST /orders` принимаются менее чем за 300 мс;
- 99% заказов завершают Saga менее чем за 10 секунд;
- 99,9% событий Outbox публикуются менее чем за 30 секунд;
- повторная доставка не приводит к повторному списанию.


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

Это конечная, а не начальная архитектура. Первые этапы работают одним процессом и одной базой данных с отдельными PostgreSQL-схемами. Каждый модуль владеет своими таблицами с самого начала. До Kafka checkout вызывает модули через Go-интерфейсы и сохраняет прогресс шагов в БД; после перехода на Saga те же бизнес-правила выполняются через сообщения. Сетевые операции не входят в долгоживущую DB-транзакцию.

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
	CheckoutStatusStarted            CheckoutStatus = "STARTED"
	CheckoutStatusStockPending       CheckoutStatus = "STOCK_PENDING"
	CheckoutStatusPaymentPending     CheckoutStatus = "PAYMENT_PENDING"
	CheckoutStatusDeliveryPending    CheckoutStatus = "DELIVERY_PENDING"
	CheckoutStatusCapturePending     CheckoutStatus = "CAPTURE_PENDING"
	CheckoutStatusStockCommitPending CheckoutStatus = "STOCK_COMMIT_PENDING"
	CheckoutStatusConfirmPending     CheckoutStatus = "CONFIRM_PENDING"
	CheckoutStatusManualReview       CheckoutStatus = "MANUAL_REVIEW"
	CheckoutStatusCompleted          CheckoutStatus = "COMPLETED"
	CheckoutStatusCompensating       CheckoutStatus = "COMPENSATING"
	CheckoutStatusCompensated        CheckoutStatus = "COMPENSATED"
	CheckoutStatusFailed             CheckoutStatus = "FAILED"
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
  → CapturePayment
  → PaymentCaptured
  → CommitStock
  → StockCommitted
  → ConfirmOrder
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
  → CancelAuthorization
  → PaymentAuthorizationCancelled
  → ReleaseStock
  → OrderRejected
```

Авторизация удерживает средства, capture завершает списание. Refund применяется только после capture; до него используется отмена авторизации. Каждая операция имеет отдельный устойчивый operation key в области платежа и вида операции; повторы сохраняют этот ключ.

Delivery сначала создаёт отменяемую предварительную заявку. Физическая отгрузка до подтверждения заказа не начинается. Если capture окончательно отклонён, workflow отменяет заявку, оставшуюся авторизацию и активный резерв. Timeout не доказывает отказ: сначала выполняется сверка статуса.

`CommitStock` атомарно переводит активный непросроченный резерв в `COMMITTED`, уменьшая `on_hand_quantity` и `reserved_quantity`. Expiration, release и commit конкурируют за один переход состояния. Просроченный или освобождённый резерв нельзя списать. Если после capture списание резерва окончательно отклонено, необходимы refund и отмена доставки; если резерв ещё активен — также release. Повторы этих операций не меняют результат второй раз.

После успешного CommitStock временный сбой подтверждения заказа приводит к повтору ConfirmOrder. Исчерпание автоматических попыток означает `MANUAL_REVIEW` с сохранением всех обязательств, а не отклонение оплаченного и списанного заказа. ReleaseStock не возвращает уже списанный товар.

Пользовательская отмена в первой версии допускается только до запуска capture. Переход к capture и запрос отмены сериализуются по состоянию/версии workflow. После этой границы API возвращает конфликт. Возврат после подтверждения/отгрузки требует отдельного процесса и не входит в первую версию.

Изменение workflow, Inbox и следующая команда Outbox фиксируются одной транзакцией. Поздние результаты после timeout или компенсации требуют сверки и, если нужно, дополнительной компенсации. Их нельзя просто игнорировать как недопустимый переход. В синхронном checkout этапа 3B прогресс и intent следующей операции сохраняются до её вызова, поэтому после рестарта операция повторяется с прежним ключом.

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
| `delivery.creation_failed.v1` | `payment.capture.v1` |
| `payment.captured.v1` | `payment.cancel_authorization.v1` |
| `payment.authorization_cancelled.v1` | `stock.commit.v1` |
| `stock.committed.v1` | `order.confirm.v1` |
| `stock.commit_failed.v1` | `order.reject.v1` |
| `stock.released.v1` | `order.cancel.v1` |
| `payment.refunded.v1` |  |
| `delivery.cancelled.v1` |  |

Все сообщения, включая команды, имеют единый envelope. `eventId` — идентификатор сообщения, а не Kafka offset; название сохраняется и для команд. Для событий `aggregateVersion` — версия состояния источника; для команд — ожидаемая версия адресата, если известна, иначе `null`. Команда также несёт operation key в payload. Таблица выше перечисляет основные контракты; каждый шаг дополнительно описывает окончательный отказ, неопределённый результат и ответ на повтор в contract tests.

```json
{
  "eventId": "019bd0af-8b44-7f3c-9c03-e5683e34f11c",
  "eventType": "order.created",
  "eventVersion": 1,
  "producer": "order-service",
  "aggregateType": "order",
  "aggregateId": "3f76dcbc-6c76-418a-9664-d57546a2787f",
  "aggregateVersion": 1,
  "occurredAt": "2026-07-12T10:25:11.424Z",
  "correlationId": "45599fc7-da47-455d-b2b0-aa335150172a",
  "causationId": "adc4a10e-afcd-489e-91f4-a46362c37639",
  "payload": {}
}
```

Для событий заказа Kafka key равен `orderId`; при фиксированной схеме partitioning это направляет их в одну partition одного topic. Kafka сохраняет порядок записи в partition, но не восстанавливает бизнес-порядок: конкурентные relay могут опубликовать версии в обратном порядке, а между topics общего порядка нет. Изменение количества partitions требует отдельного решения о порядке.

В учебной реализации перестановки допускаются явно: consumer использует версию агрегата источника, состояние workflow и идентификатор операции. Событие с пропущенным предшественником сохраняется в durable pending-хранилище для повторной обработки или вызывает сверку состояния источника. Нельзя отмечать его как успешно применённое в Inbox. Версии разных агрегатов не сравниваются между собой. Если Kafka offset продвигается после сохранения pending, запись должна быть надёжно зафиксирована и иметь собственный механизм retry/recovery. Позднее событие о внешнем успехе проверяется на необходимость компенсации даже при устаревшей версии.

W3C Trace Context передаётся в Kafka headers. Контрактная совместимость проверяется до развёртывания producer и consumer.

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

Первая реализация relay выбирает небольшой batch через `FOR UPDATE SKIP LOCKED` и удерживает DB-транзакцию и блокировки до подтверждения публикации и записи `published_at`. Время ожидания Kafka ограничено; при сбое транзакция откатывается, затем планируется следующая попытка. Каждый worker владеет собственной транзакцией и batch, не делит транзакцию с другими goroutine. Этот подход прост, но удерживает соединение и блокировки во время I/O; batch и concurrency ограничены размером pool. Lease/claim с восстановлением просроченных claims — последующее улучшение, а не неявное освобождение блокировок сразу после SELECT.

Неопубликованные записи имеют `available_at`, число попыток и последнюю ошибку. `published_at` ставится только после broker ack. Падение после ack до DB commit приводит к повторной публикации: это at-least-once, а не exactly-once. Producer idempotence не заменяет Inbox для повторов relay после рестарта.

Недоступность Kafka не удаляет событие и не исчерпывает право на будущую публикацию: попытки перепланируются с backoff, порог вызывает alert. Неисправимые payload помещаются в карантин с диагностикой и ручным replay. Тест восстановления брокера должен включать простой дольше порога оповещения.

## Inbox и consumer

Consumer обрабатывает сообщение так:

1. Получает сообщение и начинает DB-транзакцию.
2. Вставляет `(consumer_name, event_id)` в Inbox с unique constraint. При конфликте уже успешной обработки бизнес-действие не повторяется.
3. Применяет допустимое бизнес-изменение и записывает исходящие события в Outbox.
4. Делает commit DB; ошибка откатывает также вставку Inbox.
5. Только после надёжной обработки фиксирует Kafka offset — позицию следующего сообщения.

SELECT перед INSERT сам по себе не защищает от двух одновременных обработчиков. Уникальность Inbox и бизнес-изменение обязаны находиться в одной транзакции. Если процесс завершится между DB commit и commit offset, Inbox предотвратит повторное изменение БД.

Первая реализация обрабатывает сообщения последовательно внутри partition и параллельно между partitions с общим лимитом workers. Нельзя commit позицию после offset 11, пока offset 10 ещё обрабатывается. При rebalance/revoke прекращается выдача работы, текущая обработка завершается или отменяется; незавершённые сообщения не подтверждаются. Параллелизм внутри partition с учётом непрерывно обработанных offsets — отдельное продвинутое упражнение.

После окончательной ошибки dead letter сохраняется надёжно до commit offset, с уникальностью по consumer/topic/partition/offset. Неуспешная операция не получает успешную запись Inbox. Replay адресуется исходному consumer и сохраняет `eventId`; повторный успешный replay уже блокируется Inbox. Пропуск сообщения через DLQ не снимает зависимости последующих событий: они откладываются или сверяются с источником по правилам версий.

Учебный Notification Worker создаёт одну запись уведомления в БД. Это не обещание ровно одной отправки email/webhook: внешний эффект требует отдельной доставки, reconciliation и поддержки идемпотентности провайдером.

## HTTP и синхронный checkout

Создание заказа и запуск checkout разделены границей commit. На этапе 3B первая транзакция создаёт заказ, запись прогресса и сохраняет ответ `201` со ссылкой на ресурс. Затем handler пытается синхронно пройти checkout в пределах deadline; сохранённый ответ создания не переписывается финальным статусом. Текущее состояние клиент читает через GET. При обрыве запроса незавершённый процесс подхватывает recovery worker. Повтор POST возвращает сохранённый ответ, а не запускает второй процесс; два исполнителя одного workflow исключаются блокировкой/версией и устойчивыми ключами операций.

При переходе на Saga транзакция создания сохраняет заказ, Outbox и ответ `202`; обработка продолжается через сообщения. Контракт и проверки идемпотентности обновляются вместе.

## Идемпотентность HTTP

`POST /v1/orders` требует `Idempotency-Key`.

- Ключ уникален в области `(client_id, operation, key)`. Identity получена после аутентификации.
- Первый запрос создаёт запись `PROCESSING`; заказ и сохранённый ответ создаются в той же транзакции. Незакоммиченный PROCESSING не является отдельно видимым состоянием для другого запроса.
- Код и тело ответа сохраняются.
- Повторный запрос возвращает сохранённый ответ.
- Тот же ключ с другим request body возвращает `409 Conflict`.

- Unique constraint сериализует конкурирующие одинаковые ключи; ожидающий запрос после commit читает ответ, после rollback может выполнить операцию. Ожидание имеет timeout.
- Контракт определяет сравнение исходных байтов или нормализованного JSON, срок хранения ключа и поведение после TTL. После удаления ключа прежний запрос может считаться новым.
- Проверяются параллельные запросы, потерянный ответ, rollback и изоляция клиентов.

Надёжность обеспечивается PostgreSQL, без Redis. Владение заказом и права администратора проверяются независимо от идемпотентности.

## Concurrency

Goroutine используется только при наличии параллельной работы, владельца и понятного жизненного цикла.

Outbox Relay применяет ограниченный worker pool, bounded channel, backpressure, `context.Context` и ожидание workers. Cleaner начинается с одного worker и использует `FOR UPDATE SKIP LOCKED`; увеличение параллелизма обосновывается нагрузкой.

Резервирование остатка выполняется атомарно:

```sql
UPDATE inventory.stock_items
SET reserved_quantity = reserved_quantity + $2,
    version = version + 1
WHERE product_id = $1
  AND $2 > 0
  AND on_hand_quantity - reserved_quantity >= $2
RETURNING *;
```

Ноль обновлённых строк означает отсутствие товара, недопустимое количество или недостаточный остаток; API различает причины согласно контракту. Все позиции резерва изменяются одной транзакцией, товары блокируются в стабильном порядке. Constraints защищают неотрицательные остатки и `reserved_quantity <= on_hand_quantity`, включая административные изменения. Отдельный ключ операции предотвращает повторный reserve.

Money хранит сумму в `int64` и валюту; проверяются диапазон, совместимость валют и переполнение арифметики. JSON PATCH различает отсутствие поля, нулевое значение и `null`. Списки имеют ограниченный размер страницы и устойчивую сортировку.

## Retry и DLQ

Временные ошибки могут повторяться только с учётом семантики операции и её идемпотентности. Network timeout и HTTP `502/503/504` не доказывают, что операция не выполнена: для платежей требуется сверка по operation key. Serialization failure требует повтора всей DB-транзакции, а не одного SQL-запроса.

Не повторяются ошибки валидации, отклонённая оплата, отсутствующий товар, недостаточный остаток и недопустимый переход состояния.

Задержка ограничена сверху и содержит jitter:

```text
delay = min(base × 2^attempt + jitter, maxDelay)
```

После исчерпания попыток обработки consumer сообщение попадает в DLQ вместе с исходными topic, partition, offset, key, headers, payload, consumer name, ошибкой и числом попыток. При ручном retry сохраняется исходный `eventId`. Outbox при недоступном брокере продолжает отложенные попытки; Saga сохраняет обязательства и после лимита автоматических попыток требует ручного разбора.

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


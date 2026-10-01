# Dead Letter Queue (DLQ) support for doq

## Context

Today a "poison" message — one a consumer can never successfully process — gets redelivered forever. `Queue.Nack` (`pkg/queue/queue.go:495-544`) unconditionally re-enqueues on every nack, and the ack-timeout sweep (`Queue.monitorAckQueue`, `pkg/queue/queue.go:131-174`) does the same whenever a consumer never acks in time. Neither path tracks a delivery/attempt count, and neither has any escape hatch. There's no `MaxRetries`-style setting (`entity.QueueSettings`, `pkg/entity/queue.go:8-12`) and no attempt counter on `entity.Message` (`pkg/entity/message.go:8-14`). Operators currently have to smuggle a retry counter into the free-form `metadata` map themselves, and even then nothing in doq acts on it — a bad message just loops, burning consumer throughput, with no visibility.

The goal: bound retries per queue and give exhausted messages somewhere to land (a dead-letter queue) instead of looping or vanishing, matching the SQS `redrivePolicy`/RabbitMQ `x-dead-letter-exchange` pattern, built from doq's existing primitives rather than a new subsystem.

## Approach

A DLQ is modeled as **an ordinary doq queue**, not a new queue type. For a source queue named `orders`, its dead-letter queue is `orders.dlq`, auto-created on demand with the same queue type as the source. Both `Nack` and the ack-timeout redelivery path route into it once a per-queue `MaxRetries` is exceeded. No new RaftCommand, RPC, or HTTP route is needed — dead-lettering is a deterministic side effect of the existing Nack/redelivery flow (same way fair-queue weight release already happens without a separate Raft round), and "redrive" is just dequeuing from the DLQ and enqueuing back into the source using APIs that already exist.

### 1. Data model — attempt counter + retry limit

- `pkg/proto/doq.proto`: add `uint32 attempts = 6;` to `message Message` (line ~64-70), and `uint32 max_retries = 4;` to `message QueueSettings` (line ~45-55). Run `make proto_compile`.
- `pkg/entity/message.go`: add `Attempts uint32` to `Message`; wire through `ToProto`, `ToBytes`, `MessageFromBytes` (same pattern already used for every other field).
- `pkg/entity/queue.go`: add `MaxRetries uint32` to `QueueSettings`; wire through `ToProto`/`QueueConfigFromBytes` (same pattern as `Strategy`/`MaxUnacked`/`AckTimeout`).
- `MaxRetries: 0` means "unlimited" (today's exact behavior) — fully backward compatible, opt-in per queue.

### 2. Storage — persist the attempt count

- `pkg/storage/store.go` / `pkg/storage/badger_store.go:352-391`: extend `UpdateMessage(queueName, id, priority, content, metadata, attempts uint32)` to also set `msg.Attempts` before re-marshaling (mirrors the existing `priority`/`content`/`metadata` conditional-update pattern). Update the single production call site (`pkg/queue/queue.go:530`) and the existing test (`pkg/storage/badger_store_test.go:98`).

### 3. Core routing logic — pkg/queue

- `pkg/queue/manager.go`: add a method on `QueueManager`:
  ```go
  func (qm *QueueManager) GetOrCreateDeadLetterQueue(source *entity.QueueConfig) (*Queue, error)
  ```
  derives `dlqName := source.Name + ".dlq"`, tries `GetQueue(dlqName)`, and on `errors.ErrQueueNotFound` calls `CreateQueue(source.Type, dlqName, entity.QueueSettings{Strategy: source.Settings.Strategy, AckTimeout: source.Settings.AckTimeout, MaxUnacked: 0, MaxRetries: 0})` — `MaxRetries: 0` so a DLQ never dead-letters itself.
- Introduce a narrow interface so `Queue` doesn't need to import `QueueManager` (avoid a cycle):
  ```go
  type DeadLetterSink interface {
      GetOrCreateDeadLetterQueue(source *entity.QueueConfig) (*Queue, error)
  }
  ```
  `QueueManager` satisfies it automatically. Thread it into `NewQueue(store, cfg, promMetrics, sink)` (`pkg/queue/queue.go:49-63`) as a new `deadLetterSink DeadLetterSink` field; update `QueueManager`'s construction of `Queue` instances to pass itself.
- Add a private helper, `pkg/queue/queue.go`:
  ```go
  func (q *Queue) redeliverOrDeadLetter(message *entity.Message) error
  ```
  which: increments `message.Attempts`, persists it via `q.store.UpdateMessage(...)`, and either:
  - **exceeded** (`q.config.Settings.MaxRetries > 0 && message.Attempts > q.config.Settings.MaxRetries`): resolve/create the DLQ via `q.deadLetterSink`, stamp provenance onto `message.Metadata` (`x-doq-original-queue`, `x-doq-attempts`, `x-doq-dead-lettered-at`), enqueue into the DLQ's store + in-memory queue, then `q.store.Delete(q.config.Name, message.ID)` — **not** re-enqueued into `q.queue`.
  - **otherwise**: today's exact logic — `q.queue.Enqueue(...)`, fair-queue `UpdateWeights`, `q.notify()`.
- Replace the duplicated inline re-enqueue logic in both call sites with this helper:
  - `Queue.Nack` (`pkg/queue/queue.go:495-544`)
  - `Queue.monitorAckQueue` (`pkg/queue/queue.go:131-174`)
- Optional, small: add a `IncrementDeadLetter()`-style stat (mirrors `IncrementNack()`) to `metrics.QueueStats`/Prometheus so dead-lettering is observable per queue like enqueue/dequeue/ack/nack already are.

### 4. Expose `MaxRetries` through the API surface (settings only — no new endpoints)

- `pkg/http/schemas.go`: add `MaxRetries uint32` to the `QueueSettings` struct (same tag style as `MaxUnacked`/`AckTimeout`); wire into `pkg/http/queues.go`'s `CreateQueue`/`UpdateQueue` handlers.
- `pkg/grpc/server.go`: wire `req.Settings.MaxRetries` into the same `entity.QueueSettings{}` construction already used for `Strategy`/`MaxUnacked`/`AckTimeout`.
- `pkg/raft/fsm.go`: `applyCreateQueue` (`pkg/raft/fsm.go:425-461`) and `applyUpdateQueue` already copy every `QueueSettings` field 1:1 from proto — add `MaxRetries` there too.

### 5. Admin UI — small, optional addition

Because the DLQ is just another managed queue, it automatically shows up in the existing `QueueList.tsx` table and is clickable into `QueueDetails.tsx` — no new page needed. Add a `Max Retries` numeric field to `admin_ui/src/components/CreateQueueModal.tsx` and `admin_ui/src/types/queues.tsx` so it's configurable without raw API calls.

### Out of scope for v1

- A dedicated "redrive" endpoint — for now, requeuing from `orders.dlq` back to `orders` is just a manual dequeue+enqueue using existing APIs.
- Any new queue `type` enum value for DLQs, or visual badging of DLQs in the UI.

## Verification

- `make proto_compile` after the `.proto` changes, then `make test` (`go test ./... -race -cover`) to confirm no regressions.
- New/extended unit tests in `pkg/queue` covering: Nack under the limit still redelivers to the source queue; Nack past `MaxRetries` moves the message into `<queue>.dlq` with provenance metadata and removes it from the source; the ack-timeout path (`monitorAckQueue`) exercises the same dead-letter branch.
- Extend `pkg/storage/badger_store_test.go` for `UpdateMessage`'s new `attempts` parameter and for `Attempts` round-tripping through `Message.ToBytes`/`MessageFromBytes`.
- Manual end-to-end check: `make run_node_0/1/2`, create a queue with `max_retries: 2`, enqueue a message, dequeue+nack it 3 times via HTTP/gRPC, confirm it disappears from the source queue's stats and shows up under `<queue>.dlq` in the admin UI / `GET /API/v1/queues`.

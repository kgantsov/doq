# Examples

Runnable programs demonstrating how to talk to a DOQ cluster. Each is its own
`main` package — build/run directly, e.g.:

```bash
go run ./examples/consumer -address localhost:10000 -queue my-queue
```

## gRPC clients

| Example | Queue type | Style | Notes |
|---|---|---|---|
| `round_robin_producer` / `consumer` | FAIR/ROUND_ROBIN | producer streams into a single group; consumer streams out of any existing queue | |
| `weighted_producer` / `concurrent_consumer` | FAIR/WEIGHTED | multi-group weighted producer; consumer spawns N concurrent workers (`-consumers`), each on its own stream | |
| `simple_producer` / `simple_consumer` | FAIR/ROUND_ROBIN | unary `Enqueue`/`Dequeue` calls instead of streaming | |
| `simulate_load_producer` | FAIR/ROUND_ROBIN | high-volume streaming load generator (up to 1M messages, skewed group distribution) | |
| `delayed_producer` / `delayed_consumer` | DELAYED | producer enqueues messages with random priorities; consumer streams them back out lowest-priority-number-first, showing the binary-heap ordering | |
| `generate_ids` | — | loops the `GenerateIDs` RPC to exercise Snowflake ID generation | |

## HTTP client

- `simulate_http_load` — concurrent producer/consumer goroutines hitting the REST API directly (`GET`/`POST` on `/API/v1/queues/...`).

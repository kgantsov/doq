# DOQ [![Build Status](https://drone.coroutine.dev/api/badges/kgantsov/doq/status.svg)](https://drone.coroutine.dev/kgantsov/doq) [![Go Report](https://goreportcard.com/badge/github.com/kgantsov/doq)](https://goreportcard.com/report/github.com/kgantsov/doq) [![codecov](https://codecov.io/gh/kgantsov/doq/graph/badge.svg?token=GMSIM3WYVX)](https://codecov.io/gh/kgantsov/doq)

**DOQ** (Distributed Ordered Queue) is a distributed, consensus-based message
queue. Every write is replicated through the [Raft](https://raft.github.io/)
consensus algorithm ([hashicorp/raft](https://github.com/hashicorp/raft)) and
persisted to [BadgerDB](https://dgraph.io/blog/post/badger/). Only the elected
leader accepts writes, and a message is acknowledged to the client only once a
**majority of nodes** have durably stored it — giving you an ordered queue that
survives node failures. See [Architecture](docs/architecture.md) for how
consensus, replication, and failover work.

## Features

- **Work-queue semantics** — each message is delivered to a single consumer
  (competing consumers). Ideal for distributing tasks/jobs; it is not a
  pub/sub system and does not fan out a message to multiple subscribers.
- **Raft-replicated** and durable — majority quorum on every write, automatic
  leader election and failover.
- **Two queue types** — `DELAYED` (priority / scheduled delivery) and `FAIR`
  (round-robin or weighted fairness across groups, including hierarchical groups).
- **HTTP + gRPC** APIs, including bidirectional streaming for high-throughput
  producers and consumers.
- **Tunable delivery** — at-most-once (ack on dequeue) or at-least-once
  (explicit ack / nack / touch with automatic redelivery of timed-out messages).
- **Embedded admin UI** and **Prometheus metrics** out of the box.
- **Full & incremental backups** over a simple HTTP API.
- **Kubernetes-native** — SRV-based peer discovery and a leader-following service.

## Quick start

Build the binary (this also builds and embeds the admin UI):

```bash
git clone git@github.com:kgantsov/doq.git
cd doq
make build_dev
```

Run a single node:

```bash
cd cmd/server
./doq --cluster.node_id node-0 --http.port 8000 \
  --raft.address localhost:9000 --grpc.address localhost:10000
```

Create a queue, enqueue a message, then dequeue and acknowledge it:

```bash
# Create a delayed queue
curl -X POST http://localhost:8000/API/v1/queues \
  -H 'Content-Type: application/json' \
  -d '{"name": "user_indexing_queue", "type": "delayed"}'

# Enqueue a message
curl -X POST http://localhost:8000/API/v1/queues/user_indexing_queue/messages \
  -H 'Content-Type: application/json' \
  -d '{"content": "{\"user_id\": 1}", "priority": 60}'

# Dequeue and acknowledge in one call
curl 'http://localhost:8000/API/v1/queues/user_indexing_queue/messages?ack=true'
```

Then explore:

- **Admin UI & API** — open `http://localhost:8000`
- **Swagger docs** — `http://localhost:8000/docs`

For a local 3-node cluster, use `make run_node_0`, `make run_node_1`, and
`make run_node_2` in separate terminals.

> **Security:** DOQ has no built-in authentication or TLS — the HTTP API, gRPC
> API, and admin UI are unauthenticated. Run it on a trusted network and put
> auth/TLS in front of it. See [Security](docs/deployment.md#security).

## Documentation

| Guide | What's inside |
|-------|---------------|
| [Architecture](docs/architecture.md) | Raft consensus, the FSM, request flow, storage, consistency & fault tolerance. |
| [Configuration & CLI](docs/configuration.md) | Every flag, env var, and config-file option; Makefile targets; the `doq` subcommands. |
| [Queues & Messages](docs/queues.md) | Delayed vs fair queues, message fields, and the ack / nack / touch delivery lifecycle. |
| [API Reference](docs/api.md) | Full HTTP REST and gRPC reference, including streaming producer/consumer examples. |
| [Deployment & Operations](docs/deployment.md) | Local, Docker, and Kubernetes deployment; backups; monitoring; benchmarking; security. |
| [Development & Contributing](docs/development.md) | Building from source, the admin UI, protobuf, and running the Go and Python tests. |

## License

DOQ is released under the [MIT License](LICENSE).

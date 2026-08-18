# Deployment & Operations

> **Security:** DOQ ships with **no authentication and no TLS**. The HTTP API,
> gRPC API, and admin UI are all unauthenticated, and inter-node Raft/gRPC
> traffic is unencrypted. Run DOQ on a trusted, isolated network and put
> authentication and TLS termination in front of it (e.g. an API gateway,
> service mesh, or reverse proxy). Do not expose it directly to the public
> internet. See [Security](#security) below.

## Ports

Each node listens on three ports (all configurable — see
[configuration.md](configuration.md)). The metrics, health, admin UI, and
Swagger endpoints are all served on the HTTP port.

| Port | Default | Flag | Purpose |
|------|---------|------|---------|
| HTTP | `8000` | `--http.port` | REST API, admin UI, `/docs`, `/metrics`, `/livez`, `/readyz`. |
| Raft | `9000` | `--raft.address` | Inter-node consensus / log replication. |
| gRPC | `10000` | `--grpc.address` | gRPC API and leader proxying. |
| pprof | `6060` | `--profiling.port` | Profiling (only when `--profiling.enabled=true`). |

## Local cluster

Build the binary (embeds the admin UI):

```bash
make build_dev
```

Run a single node:

```bash
cd cmd/server
./doq --cluster.node_id node-0 --http.port 8000 \
  --raft.address localhost:9000 --grpc.address localhost:10000
```

Add more nodes by pointing them at the first node's HTTP address with
`--cluster.join_addr`:

```bash
./doq --cluster.node_id node-1 --http.port 8001 \
  --raft.address localhost:9001 --grpc.address localhost:10001 \
  --cluster.join_addr localhost:8000
./doq --cluster.node_id node-2 --http.port 8002 \
  --raft.address localhost:9002 --grpc.address localhost:10002 \
  --cluster.join_addr localhost:8000
```

The Makefile wraps this up as `make run_node_0`, `make run_node_1`,
`make run_node_2`.

## Docker

The repository ships a Docker Compose file that builds a 3-node cluster
(`node-0`, `node-1`, `node-2`) from `Dockerfile-dev`, with `node-1`/`node-2`
joining `node-0`:

```bash
docker compose up --build
```

HTTP is exposed on `8000`/`8001`/`8002` and gRPC on `10000`/`10001`/`10002`.

See [README.Docker.md](../README.Docker.md) for building and pushing production
images.

## Kubernetes

Deployment is managed with [Kustomize](https://kustomize.io/). Manifests live in
`deploy/` — a `base/` plus overlays for `dev`, `staging`, and `prod`.

```bash
kubectl apply -k deploy/overlays/dev   # or staging / prod
```

This creates a **StatefulSet** (3 replicas by default) and two services:

1. **`doq-internal`** — a headless service that lets pods discover each other and
   form the Raft cluster. It publishes not-ready addresses so nodes can find each
   other during startup. Not for client use.
2. **`doq`** — the client-facing service. It targets the current leader; when a
   new leader is elected, the service's selector is updated to point at the new
   leader pod.

Peers are discovered via Kubernetes SRV DNS records against
`doq-internal.<namespace>.svc.cluster.local` (`pkg/cluster/`). The ordinal-0 pod
seeds the cluster and the others join it — controlled by `--cluster.bootstrap`
(see [configuration.md](configuration.md#cluster)). Liveness and readiness are
wired to `/livez` and `/readyz`.

Give the cluster a minute to initialize and elect a leader before sending
traffic.

## Admin UI

DOQ bundles a web admin UI (a React + TypeScript app under `admin_ui/`) that is
built and embedded into the server binary by `make build_dev` / `make build_web`.
Once a node is running, open it at the HTTP port:

```
http://localhost:8000
```

From the UI you can browse queues and their live stats, create and delete
queues, enqueue and dequeue messages, acknowledge/nack in-flight messages, and
view per-queue throughput charts. The interactive Swagger API explorer is
available alongside it at `http://localhost:8000/docs`.

> The admin UI is unauthenticated — see [Security](#security).

For working on the UI itself (dev server, hot reload), see
[development.md](development.md#admin-ui).

## Backups & restore

DOQ supports full and incremental backups of the leader's database over HTTP.

### Create a backup

`POST /db/backup` with a `since` parameter:

- `since: 0` produces a **full** backup.
- For an **incremental** backup, use the previous backup's `X-Last-Version`
  response header value **plus one** as `since`.

```bash
curl -v --raw --request POST \
  --url http://localhost:8000/db/backup \
  --header 'Content-Type: application/json' \
  --data '{"since": 0}' -o backup-0.bak
```

The `X-Last-Version` response header is the version of the last entry dumped;
feed it (incremented by 1) into the next incremental backup.

### Restore

Restore a single full backup over HTTP:

```bash
curl --request POST \
  --url http://localhost:8000/db/restore \
  --header 'Content-Type: multipart/form-data' \
  --form 'file=@backup-0.bak'
```

To restore from a full backup plus incremental backups, apply them in order.
This is easiest with the `restore` subcommand against a stopped node's data
directory:

```bash
./doq restore -f backup-0.bak -i backup-1.bak -i backup-2.bak
```

## Monitoring

Enable Prometheus metrics with `--prometheus.enabled=true`; they are served at
`/metrics`. Per-queue metrics (labelled by `queue_name`) use the `doq_queues_`
prefix:

| Metric | Type | Description |
|--------|------|-------------|
| `doq_queues_enqueue_total` | counter | Messages enqueued. |
| `doq_queues_dequeue_total` | counter | Messages dequeued. |
| `doq_queues_ack_total` | counter | Messages acknowledged. |
| `doq_queues_nack_total` | counter | Messages negatively acknowledged. |
| `doq_queues_messages` | gauge | Total messages in the queue. |
| `doq_queues_ready_messages` | gauge | Messages ready to be dequeued. |
| `doq_queues_unacked_messages` | gauge | In-flight (dequeued but unacked) messages. |

gRPC request metrics are exposed under the `doq_grpc_` prefix.

A ready-to-run Prometheus + Grafana stack (with a datasource and dashboard) is
provided under `testing/prometheus/`:

```bash
docker compose -f testing/prometheus/docker-compose-metrics.yaml up
```

- Prometheus config: `testing/prometheus/prometheus/prometheus.yml`
- Grafana datasource: `testing/prometheus/grafana/datasources.yaml`
- Grafana dashboard: `testing/prometheus/grafana/dashboard.json`

## Benchmarking

A [k6](https://k6.io/open-source/) load script lives at
`testing/load/queue.js`. With a DOQ server running:

```bash
k6 run -u 100 -d 10s testing/load/queue.js
```

- `-u 100` — 100 virtual users.
- `-d 10s` — 10-second duration.

Adjust `-u`/`-d` to simulate different loads, and watch CPU, memory, and disk I/O
on the server to find bottlenecks.

### Example output

```
     ✓ enqueued
     ✓ dequeued

     checks.........................: 100.00% 258996 out of 258996
     http_req_duration..............: avg=3.56ms  min=62µs  med=3.18ms  max=120.02ms  p(90)=5.4ms  p(95)=7.76ms
     http_req_failed................: 0.00%   0 out of 258998
     http_reqs......................: 258998  25830.306144/s
     iterations.....................: 129498  12915.05334/s
     vus............................: 100     min=100  max=100
```

## Security

DOQ does not implement authentication, authorization, or transport encryption:

- The **HTTP REST API**, **admin UI**, **Swagger docs**, and **`/metrics`**
  endpoint are served without authentication on the HTTP port.
- The **gRPC API** accepts unauthenticated requests, and the leader proxy dials
  peers over an insecure (plaintext) connection.
- **Raft** replication between nodes is unencrypted.

Because any client that can reach a node can read and mutate every queue, treat
network reachability as the security boundary:

- Run the cluster on a **private / trusted network**; never expose node ports
  directly to the public internet.
- Put **authentication and TLS termination** in front of DOQ — for example an
  API gateway, reverse proxy, or service mesh (mTLS) — and restrict who can
  reach the HTTP, gRPC, and Raft ports.
- In Kubernetes, keep the `doq` and `doq-internal` services cluster-internal and
  apply NetworkPolicies to limit access.

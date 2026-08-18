# Configuration & CLI

DOQ is configured through `pkg/config/config.go`. Settings can be supplied three
ways, and are resolved in this order of precedence:

1. **Environment variables** (via Viper `AutomaticEnv`)
2. **Command-line flags**
3. **YAML config file** (`config.yaml` in the working directory, or a path given
   with `--config`)

### Environment variable convention

A dotted flag name maps to an upper-case, underscore-separated environment
variable: `--raft.address` → `RAFT_ADDRESS`, `--storage.data_dir` →
`STORAGE_DATA_DIR`, `--queue.stats.window_side` → `QUEUE_STATS_WINDOW_SIDE`.

## Flags

### Storage (BadgerDB)

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--storage.data_dir` | `STORAGE_DATA_DIR` | `data` | Root directory for node data. Data lives under `<data_dir>/<node_id>/`. |
| `--storage.gc_interval` | `STORAGE_GC_INTERVAL` | `300` | Value-log garbage collection interval (seconds). |
| `--storage.gc_discard_ratio` | `STORAGE_GC_DISCARD_RATIO` | `0.7` | GC discard ratio (0.0–1.0). |
| `--storage.compression` | `STORAGE_COMPRESSION` | `` (none) | Compression: `none`, `snappy`, or `zstd`. |
| `--storage.zstd_compression_level` | `STORAGE_ZSTD_COMPRESSION_LEVEL` | `0` | ZSTD level (applied when > 0). |
| `--storage.block_cache_size` | `STORAGE_BLOCK_CACHE_SIZE` | `0` | Block cache size in bytes (applied when > 0). |
| `--storage.index_cache_size` | `STORAGE_INDEX_CACHE_SIZE` | `0` | Index cache size in bytes (applied when > 0). |
| `--storage.base_table_size` | `STORAGE_BASE_TABLE_SIZE` | `0` | Base table size in bytes (applied when > 0). |
| `--storage.base_level_size` | `STORAGE_BASE_LEVEL_SIZE` | `0` | Max size target for the base level (applied when > 0). |
| `--storage.num_compactors` | `STORAGE_NUM_COMPACTORS` | `0` | Number of compaction workers (applied when > 0). |
| `--storage.num_level_zero_tables` | `STORAGE_NUM_LEVEL_ZERO_TABLES` | `0` | Number of level-0 tables (applied when > 0). |
| `--storage.num_level_zero_tables_stall` | `STORAGE_NUM_LEVEL_ZERO_TABLES_STALL` | `0` | Level-0 tables that stall compaction (applied when > 0). |
| `--storage.num_memtables` | `STORAGE_NUM_MEMTABLES` | `0` | Number of memtables (applied when > 0). |
| `--storage.value_log_file_size` | `STORAGE_VALUE_LOG_FILE_SIZE` | `0` | Value-log file size in bytes (applied when > 0). |

### Cluster

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--cluster.node_id` | `CLUSTER_NODE_ID` | `node-0` | Unique node identifier. Also determines the on-disk data subdirectory. |
| `--cluster.namespace` | `CLUSTER_NAMESPACE` | `default` | Kubernetes namespace used for SRV service discovery. |
| `--cluster.service_name` | `CLUSTER_SERVICE_NAME` | `` | Kubernetes headless service name used to discover peers. |
| `--cluster.join_addr` | `CLUSTER_JOIN_ADDR` | `` | HTTP address of an existing node to join. |
| `--cluster.bootstrap` | `CLUSTER_BOOTSTRAP` | `auto` | Which node seeds the Raft cluster: `auto`, `true`, or `false` (see below). |

**Bootstrap semantics** — exactly one node in a fresh cluster must bootstrap;
all others start empty and join it.

- `auto` (default): derives the role — in Kubernetes the ordinal-0 pod seeds;
  outside Kubernetes a node without a `join_addr` seeds.
- `true`: forces this node to be the seed.
- `false`: forces this node to join an existing cluster.

### Raft

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--raft.address` | `RAFT_ADDRESS` | `localhost:9000` | Raft bind address. |
| `--raft.apply_timeout` | `RAFT_APPLY_TIMEOUT` | `5` | Raft apply timeout (seconds). |

### HTTP & gRPC

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--http.port` | `HTTP_PORT` | `8000` | Port for the REST API, admin UI, health and metrics endpoints. |
| `--grpc.address` | `GRPC_ADDRESS` | `` | gRPC bind address, e.g. `localhost:10000`. |

### Queue

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--queue.acknowledgement_check_interval` | `QUEUE_ACKNOWLEDGEMENT_CHECK_INTERVAL` | `1` | Interval (seconds) at which unacked messages are checked for timeout. |
| `--queue.default_acknowledgement_timeout` | `QUEUE_DEFAULT_ACKNOWLEDGEMENT_TIMEOUT` | `1800` | Default ack timeout (seconds) before a dequeued message is redelivered. |
| `--queue.stats.window_side` | `QUEUE_STATS_WINDOW_SIDE` | `10` | Sliding window (seconds) used to compute per-queue RPS stats. |

### Observability

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--prometheus.enabled` | `PROMETHEUS_ENABLED` | `false` | Expose Prometheus metrics at `/metrics`. |
| `--profiling.enabled` | `PROFILING_ENABLED` | `false` | Enable `pprof` profiling. |
| `--profiling.port` | `PROFILING_PORT` | `6060` | `pprof` HTTP port. |
| `--logging.mode` | `LOGGING_MODE` | `console` | Log output mode: `console` or `stackdriver`. |
| `--logging.level` | `LOGGING_LEVEL` | `warning` | Log level: `debug`, `info`, `warn`, `error`, `fatal`, `panic`. |

### Global

| Flag | Description |
|------|-------------|
| `--config` | Path to a YAML config file (defaults to `config.yaml` in the working directory). |

## Config file

Any settings you want to override can go in a YAML file. You only need to include
the keys you want to change — anything omitted falls back to its default (and can
still be overridden by a flag or environment variable). Each YAML key matches the
corresponding flag name, e.g. `--storage.data_dir` → `storage.data_dir`.

The following reference lists **every** configurable key with its default value:

```yaml
profiling:
  enabled: false            # enable pprof profiling
  port: 6060                # pprof HTTP port

prometheus:
  enabled: false            # expose Prometheus metrics at /metrics

logging:
  mode: "console"           # console | stackdriver
  level: "warning"          # debug | info | warn | error | fatal | panic

http:
  port: "8000"              # REST API, admin UI, /metrics, /livez, /readyz

grpc:
  address: ""               # gRPC bind address, e.g. localhost:10000

raft:
  address: "localhost:9000" # Raft bind address
  apply_timeout: 5          # seconds

storage:
  data_dir: "data"          # data lives under <data_dir>/<node_id>/
  gc_interval: 300          # value-log GC interval (seconds)
  gc_discard_ratio: 0.7     # value-log GC discard ratio (0.0-1.0)
  # BadgerDB tuning — 0 / "" means "use BadgerDB's default"
  compression: ""           # none | snappy | zstd
  zstd_compression_level: 0
  block_cache_size: 0       # bytes
  index_cache_size: 0       # bytes
  base_table_size: 0        # bytes
  base_level_size: 0        # bytes
  num_compactors: 0
  num_level_zero_tables: 0
  num_level_zero_tables_stall: 0
  num_memtables: 0
  value_log_file_size: 0    # bytes

queue:
  acknowledgement_check_interval: 1     # how often unacked messages are checked (seconds)
  default_acknowledgement_timeout: 1800 # redelivery timeout when a queue sets none (seconds)
  stats:
    window_side: 10                     # sliding window for per-queue RPS stats (seconds)

cluster:
  namespace: "default"      # Kubernetes namespace for SRV discovery
  node_id: "node-0"         # unique node id; also the on-disk data subdirectory
  service_name: ""          # Kubernetes headless service used to discover peers
  join_addr: ""             # HTTP address of an existing node to join
  bootstrap: "auto"         # auto | true | false (see the Cluster section above)
```

## Makefile targets

| Target | Description |
|--------|-------------|
| `make build_dev` | Build the admin UI, embed it, and produce `cmd/server/doq`. |
| `make build_web` | Build the Vue/React admin UI and copy it into `cmd/server`. |
| `make build_linux_amd64` | Cross-compile a Linux amd64 binary to `dist/doq`. |
| `make build_darwin_arm64` | Cross-compile a macOS arm64 binary to `dist/doq`. |
| `make proto_compile` | Compile `pkg/proto/*.proto`. |
| `make test` | `go test ./... -race -cover`. |
| `make bench` | `go test ./... -bench=. -benchmem`. |
| `make run_web` | Start the admin UI dev server. |
| `make run_node_0` / `run_node_1` / `run_node_2` | Run a local 3-node cluster (HTTP 8000/8001/8002, Raft 9000/9001/9002, gRPC 10000/10001/10002). |

## The `doq` binary

Running `doq` with no subcommand starts the server. It also provides utility
subcommands that operate directly on a node's on-disk BadgerDB store (run them
against a stopped node's `--storage.data_dir` / `--cluster.node_id`).

### `doq restore`

Restore a database from backup files: a full backup first, then any incremental
backups in order.

```bash
doq restore -f backup-0.bak -i backup-1.bak -i backup-2.bak
```

| Flag | Description |
|------|-------------|
| `-f`, `--full` | Full backup file. |
| `-i`, `--incremental` | Incremental backup file (repeatable). |

### `doq get queues`

List all queues in the store (name, type, and strategy for fair queues).

```bash
doq get queues
```

### `doq get messages`

Read messages from a queue's store, either a page of messages or specific IDs
passed as positional arguments.

```bash
doq get messages -q user_indexing_queue -n 20 -o json
doq get messages -q user_indexing_queue 123 456
```

| Flag | Default | Description |
|------|---------|-------------|
| `-q`, `--queue` | (required) | Queue name. |
| `-n`, `--limit` | `10` | Number of messages to retrieve. |
| `-l`, `--last_id` | `0` | Pagination cursor (last message ID seen). |
| `-o`, `--output` | `text` | Output format: `text`, `json`, or `jsonl`. |

# Development & Contributing

## Prerequisites

- **Go** (see `go.mod` for the required version)
- **Node.js + npm** — to build the admin UI
- **protoc** with the `protoc-gen-go` and `protoc-gen-go-grpc` plugins — only if
  you change the protobuf definitions

## Building from source

```bash
make build_dev          # build the admin UI, embed it, and produce cmd/server/doq
```

Cross-compilation:

```bash
make build_linux_amd64  # → dist/doq (linux/amd64)
make build_darwin_arm64 # → dist/doq (darwin/arm64)
```

See [configuration.md](configuration.md#makefile-targets) for the full list of
Makefile targets.

## Running a local cluster

```bash
make run_node_0   # HTTP 8000, Raft 9000, gRPC 10000
make run_node_1   # HTTP 8001, Raft 9001, gRPC 10001 (joins node-0)
make run_node_2   # HTTP 8002, Raft 9002, gRPC 10002 (joins node-0)
```

Run each in its own terminal. See [deployment.md](deployment.md) for details.

## Admin UI

The admin UI is a React + TypeScript app (built with Vite) under `admin_ui/`.

```bash
cd admin_ui
npm install
npm run dev      # start the Vite dev server with hot reload (also: make run_web)
npm run build    # type-check and produce a production build in dist/
npm run lint     # run ESLint
```

The dev server proxies `/API` requests to `http://127.0.0.1:8000`, so run a DOQ
node on the default HTTP port alongside it. To embed a fresh build into the
server binary, run `make build_web` (or `make build_dev`), which builds the UI
and copies `dist/` into `cmd/server`.

## Protobuf

The gRPC service and all Raft command messages are defined in
[`pkg/proto/doq.proto`](../pkg/proto/doq.proto). After editing it, regenerate the
Go code:

```bash
make proto_compile
```

## Tests

Go tests (with the race detector and coverage):

```bash
make test                                   # go test ./... -race -cover
go test ./pkg/queue/... -run TestName -race # run a single test
make bench                                  # benchmarks
```

### Python integration tests

An end-to-end test suite that exercises the HTTP API lives under `python/`. It
runs against a live DOQ server, using the `BASE_URL` environment variable
(default `http://localhost:8000`).

```bash
# Start a DOQ node first (e.g. make run_node_0), then:
cd python
pip install -r requirements.txt
BASE_URL=http://localhost:8000 pytest
```

## Project layout

| Path | Contents |
|------|----------|
| `cmd/server/` | The `doq` binary: server entrypoint (`main.go`) and CLI subcommands (`commands.go`). |
| `pkg/` | Library packages — see the [key packages table](architecture.md#key-packages). |
| `admin_ui/` | React + TypeScript admin UI, embedded into the binary at build time. |
| `deploy/` | Kustomize manifests (`base/` + `dev`/`staging`/`prod` overlays). |
| `testing/` | k6 load script (`testing/load/`) and a Prometheus/Grafana stack (`testing/prometheus/`). |
| `python/` | Python HTTP-API integration tests. |

For a description of how the pieces fit together, start with
[architecture.md](architecture.md).

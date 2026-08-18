# API Reference

DOQ exposes a REST API (Fiber + Huma/OpenAPI) and a gRPC API. Both drive the
same Raft-replicated state, so you can mix and match.

- Interactive Swagger UI: `http://localhost:8000/docs`
- Prometheus metrics: `http://localhost:8000/metrics` (when
  `--prometheus.enabled=true`)
- Health: `http://localhost:8000/livez` (liveness),
  `http://localhost:8000/readyz` (readiness)

## HTTP REST API

Base paths: `/API/v1/` for queue and message operations, `/db/` for
backup/restore.

### Queues

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/API/v1/queues` | Create a queue. Body: `{name, type, settings?}`. |
| `PUT` | `/API/v1/queues/{queue_name}` | Update queue settings. Body: `{settings}`. |
| `DELETE` | `/API/v1/queues/{queue_name}` | Delete a queue and its messages. |
| `GET` | `/API/v1/queues` | List all queues with stats. |
| `GET` | `/API/v1/queues/{queue_name}` | Get one queue with stats. |

Queue stats include `enqueue_rps`, `dequeue_rps`, `ack_rps`, `nack_rps`, and the
counts `ready`, `unacked`, and `total`.

### Messages

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/API/v1/queues/{queue_name}/messages` | Enqueue a message. Body: `{id?, group?, priority, content, metadata?}`. |
| `GET` | `/API/v1/queues/{queue_name}/messages?ack=<bool>` | Dequeue the next message; `ack=true` acknowledges it immediately. |
| `GET` | `/API/v1/queues/{queue_name}/messages/{id}` | Get a message by ID without removing it. |
| `DELETE` | `/API/v1/queues/{queue_name}/messages/{id}` | Delete a message. |
| `POST` | `/API/v1/queues/{queue_name}/messages/{id}/ack` | Acknowledge an in-flight message. |
| `POST` | `/API/v1/queues/{queue_name}/messages/{id}/nack` | Return a message to the queue. Body: `{priority, metadata?}`. |
| `POST` | `/API/v1/queues/{queue_name}/messages/{id}/touch` | Extend the ack deadline of an in-flight message. |
| `PUT` | `/API/v1/queues/{queue_name}/messages/{id}/priority` | Update a message's priority. Body: `{priority}`. |
| `POST` | `/API/v1/ids` | Generate unique message IDs. Body: `{number}` (1–1000). |

### Cluster

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/API/v1/cluster/join` | Join a node to the cluster (used by Raft). |
| `POST` | `/API/v1/cluster/leave` | Remove a node from the cluster. |
| `GET` | `/API/v1/cluster/servers` | List cluster members. |

### Database

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/db/backup` | Stream a backup of entries newer than `since`; returns an `X-Last-Version` header. |
| `POST` | `/db/restore` | Restore from an uploaded backup file (multipart form field `file`). |

See [deployment.md](deployment.md#backups--restore) for the backup workflow.

### Examples

Enqueue a message:

```bash
curl --request POST \
  --url http://localhost:8000/API/v1/queues/user_indexing_queue/messages \
  --header 'Content-Type: application/json' \
  --data '{"content": "{\"user_id\": 1}", "group": "default", "priority": 60}'
```

Dequeue and acknowledge in one call:

```bash
curl --request GET \
  --url 'http://localhost:8000/API/v1/queues/user_indexing_queue/messages?ack=true'
```

Acknowledge a message that was dequeued without `ack`:

```bash
curl --request POST \
  --url http://localhost:8000/API/v1/queues/user_indexing_queue/messages/123/ack
```

Change a message's priority:

```bash
curl --request PUT \
  --url http://localhost:8000/API/v1/queues/user_indexing_queue/messages/123/priority \
  --header 'Content-Type: application/json' \
  --data '{"priority": 12}'
```

## gRPC API

Service `queue.DOQ`, defined in
[`pkg/proto/doq.proto`](../pkg/proto/doq.proto). Bidirectional streaming is
available for enqueue and dequeue and is the recommended path for
high-throughput producers/consumers.

| RPC | Description |
|-----|-------------|
| `GenerateIDs` | Generate unique message IDs. |
| `CreateQueue` / `UpdateQueue` / `DeleteQueue` | Queue lifecycle. |
| `GetQueue` / `GetQueues` | Read queue info and stats (optionally filtered by type). |
| `Enqueue` / `EnqueueStream` | Enqueue a single message, or a stream of them. |
| `Dequeue` / `DequeueStream` | Dequeue a single message, or stream messages as they arrive. |
| `Get` / `Delete` | Read or delete a message by ID. |
| `Ack` / `Nack` / `Touch` | Acknowledge, return-to-queue, or extend the ack deadline. |
| `UpdatePriority` | Change a message's priority. |

Key message fields (`EnqueueRequest` / `DequeueResponse`): `queueName`, `id`,
`group`, `priority`, `content`, `metadata`. `QueueSettings` carries `strategy`
(`ROUND_ROBIN` / `WEIGHTED`), `max_unacked`, and `ack_timeout` (seconds).

### Producer example

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var address = ""
var numberOfMessages = 100
var sleepMillis = 200
var queueName = ""

func main() {
	flag.StringVar(&address, "address", "localhost:10000", "gRPC server address")
	flag.IntVar(&numberOfMessages, "number", 100, "Number of messages to send")
	flag.IntVar(&sleepMillis, "sleep", 200, "Sleep time in milliseconds")
	flag.StringVar(&queueName, "queue", "test-queue", "Queue name")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano

	// Connect to the gRPC server (leader node)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Msgf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	client.CreateQueue(context.Background(), &pb.CreateQueueRequest{
		Name: queueName,
		Type: "fair",
	})

	// Create a stream for sending messages
	stream, err := client.EnqueueStream(context.Background())
	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	// Produce messages in a loop
	for i := 0; i < numberOfMessages; {
		started := time.Now()
		msg := &pb.EnqueueRequest{
			QueueName: queueName,
			Content:   fmt.Sprintf("Message content %d", i),
			Group:     fmt.Sprintf("group_%d", 1),
			Priority:  10,
		}

		// Send the message to the queue
		if err := stream.Send(msg); err != nil {
			log.Fatal().Msgf("Failed to send message: %v", err)
		}

		// Receive the acknowledgment from the server
		ack, err := stream.Recv()
		if err != nil {
			log.Fatal().Msgf("Failed to receive acknowledgment: %v", err)
		}
		log.Info().
			Uint64("id", ack.Id).
			Str("group", ack.Group).
			Str("took", time.Since(started).String()).
			Msgf("Sent a message: %s", ack.Content)

		i++
		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
	}

	// Close the stream
	if err := stream.CloseSend(); err != nil {
		log.Fatal().Msgf("Failed to close stream: %v", err)
	}
}
```

### Consumer example

```go
package main

import (
	"context"
	"flag"
	"os"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var address = ""
var queueName = ""
var immediateAck = false

func main() {
	flag.StringVar(&address, "address", "localhost:10000", "gRPC server address")
	flag.StringVar(&queueName, "queue", "test-queue", "Queue name")
	flag.BoolVar(&immediateAck, "ack", false, "Immediate ack")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano

	// Connect to the gRPC server (leader node)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Msgf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Open a stream to receive messages from the queue
	stream, err := client.DequeueStream(context.Background())
	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	// Subscribe to the queue and start receiving messages
	err = stream.Send(&pb.DequeueRequest{
		QueueName: queueName,
		Ack:       immediateAck,
	})
	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	// Consume messages from the stream
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Debug().Msgf("Failed to receive message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		processTime := time.Now()

		// Process the message
		log.Info().
			Uint64("id", msg.Id).
			Str("group", msg.Group).
			Msgf("Received message: %s", msg.Content)

		time.Sleep(10 * time.Second)

		if !immediateAck {
			client.Ack(context.Background(), &pb.AckRequest{
				QueueName: queueName,
				Id:        msg.Id,
			})
		}

		log.Info().
			Uint64("id", msg.Id).
			Str("group", msg.Group).
			Str("took", time.Since(processTime).String()).
			Msgf("Acknowledged message: %s", msg.Content)

		// Signal to the server that we are ready for the next message
		stream.Send(&pb.DequeueRequest{})
	}
}
```

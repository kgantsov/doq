// leader_aware_consumer is a reference DOQ consumer that survives leader
// failover by using the leader hint the server stamps on every gRPC response.
//
// It demonstrates the client-side half of the contract implemented server-side:
//
//   - Dial with gRPC keepalive so a hard-killed node (one that never sent a TCP
//     FIN) is detected in seconds instead of hanging forever.
//   - Connect to any node from a bootstrap list; the node proxies to the leader
//     transparently, so the consumer keeps working regardless of who it hits.
//   - Read the "x-doq-leader" / "x-doq-is-leader" trailer on every response. If
//     the serving node is a follower, re-pin the connection directly to the
//     advertised leader so subsequent requests skip the proxy hop.
//   - On UNAVAILABLE (the pinned node died), reconnect against the bootstrap
//     list and let the hint re-pin us to the new leader.
//
// This is the pattern the language clients (e.g. doq_client for Celery) follow.
//
// Note on addresses: the leader hint carries "<leader-host>:<grpc-port>", which
// is the same address followers use to proxy. It is routable from inside the
// cluster; run this example where those addresses resolve (in-cluster, or with
// matching port-forwards), otherwise the re-pin dial will fail and the consumer
// falls back to the bootstrap list.
package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// These mirror the metadata keys the server sets in pkg/grpc/interceptor.go.
// They are duplicated here so the example doesn't pull in the whole server
// package for two constants.
const (
	metadataKeyLeader   = "x-doq-leader"
	metadataKeyIsLeader = "x-doq-is-leader"
)

var (
	logLevel     = "info"
	addresses    = "localhost:10000"
	queueName    = "test-queue"
	immediateAck = false
	sleepMillis  = 200
)

// dialOptions configures the channel with keepalive so a dead peer is detected
// quickly. Time must be >= the server's KeepaliveEnforcementPolicy.MinTime
// (10s), or the server will GOAWAY us for pinging too aggressively.
func dialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                15 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}
}

// consumer holds the current connection and knows how to move it between nodes.
type consumer struct {
	bootstrap []string // all node addresses, used to (re)connect on failure
	next      int      // round-robin cursor into bootstrap

	target string // address we are currently connected to
	conn   *grpc.ClientConn
	client pb.DOQClient
}

func newConsumer(bootstrap []string) (*consumer, error) {
	c := &consumer{bootstrap: bootstrap}
	if err := c.reconnectBootstrap(); err != nil {
		return nil, err
	}
	return c, nil
}

// dial replaces the current connection with one to target.
func (c *consumer) dial(target string) error {
	conn, err := grpc.NewClient(target, dialOptions()...)
	if err != nil {
		return err
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.target = target
	c.conn = conn
	c.client = pb.NewDOQClient(conn)
	return nil
}

// reconnectBootstrap connects to the next node in the bootstrap list. Used at
// startup and whenever the pinned node becomes unreachable.
func (c *consumer) reconnectBootstrap() error {
	target := c.bootstrap[c.next%len(c.bootstrap)]
	c.next++
	log.Warn().Msgf("connecting to bootstrap node %s", target)
	return c.dial(target)
}

// repin moves the connection directly to the leader when we learn (from the
// response trailer) that we are talking to a follower.
func (c *consumer) repin(trailer metadata.MD) {
	leader := first(trailer.Get(metadataKeyLeader))
	isLeader := first(trailer.Get(metadataKeyIsLeader))

	if leader == "" || leader == c.target {
		return
	}
	// Only re-pin when the current node told us it is NOT the leader. If it is
	// the leader we are already optimally connected.
	if isLeader != "false" {
		return
	}

	log.Info().Msgf("served by follower %s, re-pinning to leader %s", c.target, leader)
	if err := c.dial(leader); err != nil {
		log.Warn().Msgf("failed to re-pin to leader %s: %v", leader, err)
	}
}

func first(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func main() {
	flag.StringVar(&addresses, "addresses", addresses, "Comma-separated bootstrap gRPC addresses (all nodes)")
	flag.StringVar(&queueName, "queue", queueName, "Queue name")
	flag.BoolVar(&immediateAck, "ack", immediateAck, "Immediate ack")
	flag.IntVar(&sleepMillis, "sleep", sleepMillis, "Per-message processing time in milliseconds")
	flag.StringVar(&logLevel, "log_level", logLevel, "Log level")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	if level, err := zerolog.ParseLevel(logLevel); err != nil {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(level)
	}

	var bootstrap []string
	for _, a := range strings.Split(addresses, ",") {
		if a = strings.TrimSpace(a); a != "" {
			bootstrap = append(bootstrap, a)
		}
	}
	if len(bootstrap) == 0 {
		log.Fatal().Msg("no addresses provided")
	}

	c, err := newConsumer(bootstrap)
	if err != nil {
		log.Fatal().Msgf("failed to connect: %v", err)
	}
	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	// Best-effort queue creation so the example runs standalone. Proxied to the
	// leader if we happen to be on a follower.
	c.client.CreateQueue(context.Background(), &pb.CreateQueueRequest{
		Name:     queueName,
		Type:     "fair",
		Settings: &pb.QueueSettings{Strategy: pb.QueueSettings_ROUND_ROBIN},
	})

	for {
		var trailer metadata.MD
		msg, err := c.client.Dequeue(
			context.Background(),
			&pb.DequeueRequest{QueueName: queueName, Ack: immediateAck},
			grpc.Trailer(&trailer),
		)

		// The leader hint rides on the trailer of every response, including
		// errors (e.g. an empty queue), so an otherwise-idle consumer still
		// learns about leadership changes and re-pins.
		c.repin(trailer)

		if err != nil {
			if status.Code(err) == codes.Unavailable {
				// The node we were pinned to is gone. Reconnect to another node
				// from the bootstrap list; the next response's hint re-pins us
				// to the freshly elected leader.
				log.Warn().Msgf("node %s unavailable, reconnecting: %v", c.target, err)
				if rerr := c.reconnectBootstrap(); rerr != nil {
					log.Warn().Msgf("reconnect failed: %v", rerr)
					time.Sleep(time.Second)
				}
				continue
			}
			// Any other error (commonly an empty queue) - back off and retry.
			log.Debug().Msgf("dequeue: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		log.Info().Uint64("id", msg.Id).Str("group", msg.Group).Msgf("received: %s", msg.Content)

		if sleepMillis > 0 {
			time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
		}

		if !immediateAck {
			var ackTrailer metadata.MD
			if _, err := c.client.Ack(
				context.Background(),
				&pb.AckRequest{QueueName: queueName, Id: msg.Id},
				grpc.Trailer(&ackTrailer),
			); err != nil {
				log.Warn().Msgf("ack failed for id=%d: %v", msg.Id, err)
			}
			c.repin(ackTrailer)
		}
	}
}

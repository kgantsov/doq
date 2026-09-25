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

var logLevel = "info"
var address = ""
var queueName = ""
var immediateAck = false
var sleepMillis = 200

func main() {
	flag.StringVar(&address, "address", "localhost:10000", "gRPC server address")
	flag.StringVar(&queueName, "queue", "delayed-test-queue", "Queue name")
	flag.BoolVar(&immediateAck, "ack", false, "Immediate ack")
	flag.IntVar(&sleepMillis, "sleep", 200, "Sleep time in milliseconds")
	flag.StringVar(&logLevel, "log_level", logLevel, "Log level")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano

	level, err := zerolog.ParseLevel(logLevel)

	if err != nil {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(level)
	}

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

	// Send a request to subscribe to the queue and start receiving messages
	err = stream.Send(&pb.DequeueRequest{
		QueueName: queueName,
		Ack:       immediateAck,
	})

	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	// Consume messages from the stream. Since this is a DELAYED queue, messages
	// come back lowest-priority-number-first, regardless of enqueue order.
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
			Int64("priority", msg.Priority).
			Msgf("Received message: %s", msg.Content)

		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)

		if !immediateAck {
			client.Ack(context.Background(), &pb.AckRequest{
				QueueName: queueName,
				Id:        msg.Id,
			})
		}

		elapsed := time.Since(processTime)
		log.Info().
			Uint64("id", msg.Id).
			Int64("priority", msg.Priority).
			Str("took", elapsed.String()).
			Msgf("Acknowledged message: %s", msg.Content)

		// Signal to the server that we are ready for the next message
		stream.Send(&pb.DequeueRequest{})
	}
}

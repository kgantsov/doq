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
	flag.StringVar(&queueName, "queue", "test-queue", "Queue name")
	flag.BoolVar(&immediateAck, "ack", false, "Immediate ack")
	flag.IntVar(&sleepMillis, "sleep", 200, "Sleep time in milliseconds")
	flag.StringVar(&logLevel, "log_level", "info", "Log level")
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

	client.CreateQueue(context.Background(), &pb.CreateQueueRequest{
		Name: queueName,
		Type: "fair",
		Settings: &pb.QueueSettings{
			Strategy: pb.QueueSettings_ROUND_ROBIN,
			// MaxUnacked: 0,§
		},
	})

	// Consume messages from the stream

	for i := 0; ; i++ {
		msg, err := client.Dequeue(context.Background(), &pb.DequeueRequest{
			QueueName: queueName,
			Ack:       immediateAck,
		})
		if err != nil {
			log.Warn().Msgf("Failed to get a message: %v %d", err, i)
			time.Sleep(time.Second)
			continue
		}

		processTime := time.Now()

		// Process the message
		log.Info().
			Uint64("id", msg.Id).
			Str("group", msg.Group).
			Msgf("Received message: %s", msg.Content)

		// time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
		if sleepMillis > 0 {
			time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
		}

		if !immediateAck {
			client.Ack(context.Background(), &pb.AckRequest{
				QueueName: queueName,
				Id:        msg.Id,
			})
		}

		elapsed := time.Since(processTime)
		log.Info().
			Uint64("id", msg.Id).
			Str("group", msg.Group).
			Str("took", elapsed.String()).
			Msgf("Acknowledged message: %s", msg.Content)

	}
}

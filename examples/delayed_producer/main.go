package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var address = ""
var numberOfMessages = 20
var sleepMillis = 200
var queueName = ""

func main() {
	flag.StringVar(&address, "address", "localhost:10000", "gRPC server address")
	flag.IntVar(&numberOfMessages, "number", 20, "Number of messages to send")
	flag.IntVar(&sleepMillis, "sleep", 200, "Sleep time in milliseconds")
	flag.StringVar(&queueName, "queue", "delayed-test-queue", "Queue name")
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
		Type: "delayed",
	})

	// Create a stream for sending messages
	stream, err := client.EnqueueStream(context.Background())
	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	// Produce messages with random priorities. Lower number = higher priority,
	// so on dequeue they should come back out lowest-priority-first regardless
	// of the order they were sent in.
	for i := 0; i < numberOfMessages; i++ {
		started := time.Now()
		priority := int64(rand.Intn(100))
		msg := &pb.EnqueueRequest{
			QueueName: queueName,
			Content:   fmt.Sprintf("Message content %d", i),
			Priority:  priority,
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
			Int64("priority", priority).
			Str("took", time.Since(started).String()).
			Msgf(
				"Sent a message: %s",
				ack.Content,
			)

		time.Sleep(time.Duration(sleepMillis) * time.Millisecond) // Simulate delay between messages
	}

	// Close the stream
	if err := stream.CloseSend(); err != nil {
		log.Fatal().Msgf("Failed to close stream: %v", err)
	}
}

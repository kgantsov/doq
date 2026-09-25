package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	// Connect to the gRPC server (leader node)
	conn, err := grpc.Dial("localhost:10000", grpc.WithInsecure())
	if err != nil {
		log.Fatal().Msgf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	queueName := "transcode-fair-test"

	client.CreateQueue(context.Background(), &pb.CreateQueueRequest{
		Name: queueName,
		Type: "fair",
		Settings: &pb.QueueSettings{
			Strategy: pb.QueueSettings_WEIGHTED,
			// MaxUnacked: 8,
		},
	})

	// Create a stream for sending messages
	stream, err := client.EnqueueStream(context.Background())
	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	customerMessages := map[string]int{
		"customer-1": 100,
		"customer-2": 50,
		"customer-3": 30,
		"customer-4": 10,
	}

	sent := 0
	for customer, count := range customerMessages {
		for i := 0; i < count; i++ {
			msg := &pb.EnqueueRequest{
				QueueName: queueName,
				Content:   fmt.Sprintf("Message content %d", sent),
				Group:     customer,
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
			log.Info().Msgf("Sent a message %d %s Success=%v", ack.Id, ack.Content, ack.Success)

			sent++
			// time.Sleep(200 * time.Millisecond) // Simulate delay between messages
		}
		time.Sleep(2 * time.Second)
	}

	// Close the stream
	if err := stream.CloseSend(); err != nil {
		log.Fatal().Msgf("Failed to close stream: %v", err)
	}
}

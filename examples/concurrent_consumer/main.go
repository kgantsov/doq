package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var QueueName = ""
var NumberConsumers = 1

func main() {
	flag.StringVar(&QueueName, "queue", "transcode-fair-test", "Queue name")
	flag.IntVar(&NumberConsumers, "consumers", 1, "Number of consumers")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	// Connect to the gRPC server (leader node)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	signal.Notify(signalChan, os.Kill)

	for i := 0; i < NumberConsumers; i++ {
		go func(workerId int) {
			time.Sleep(time.Duration(workerId) * time.Second)

			log.Info().Int("worker", workerId).Msg("Starting consumer worker")

			conn, err := grpc.Dial("localhost:10000", grpc.WithInsecure())
			if err != nil {
				log.Fatal().Int("worker", workerId).Msgf("Failed to connect: %v", err)
			}
			defer conn.Close()

			client := pb.NewDOQClient(conn)

			queue, err := client.GetQueue(context.Background(), &pb.GetQueueRequest{
				Name: QueueName,
			})
			if err != nil {
				log.Fatal().Int("worker", workerId).Msgf("Failed to get queue: %v", err)
			}
			log.Info().
				Int("worker", workerId).
				Str("name", queue.Name).
				Str("type", queue.Type).
				Int("ready", int(queue.Ready)).
				Int("unacked", int(queue.Unacked)).
				Int("total", int(queue.Total)).
				Float64("enqueue_rps", float64(queue.Stats.EnqueueRPS)).
				Float64("dequeue_rps", float64(queue.Stats.DequeueRPS)).
				Float64("ack_rps", float64(queue.Stats.AckRPS)).
				Float64("nack_rps", float64(queue.Stats.NackRPS)).
				Msg("Queue info")

			// Open a stream to receive messages from the queue
			stream, err := client.DequeueStream(context.Background())
			if err != nil {
				log.Fatal().Int("worker", workerId).Msgf("Failed to open stream: %v", err)
			}

			// Send a request to subscribe to the queue and start receiving messages
			err = stream.Send(&pb.DequeueRequest{
				QueueName: QueueName,
				Ack:       false,
			})

			if err != nil {
				log.Fatal().Msgf("Failed to open stream: %v", err)
			}

			// Consume messages from the stream
			for {
				msg, err := stream.Recv()
				if err != nil {
					log.Fatal().Int("worker", workerId).Msgf("Failed to receive message: %v", err)
				}

				processTime := time.Now()

				// Process the message
				log.Info().
					Int("worker", workerId).
					Uint64("id", msg.Id).
					Str("group", msg.Group).
					Int64("priority", msg.Priority).
					Msgf("Received message: %s", msg.Content)

				// time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
				time.Sleep(time.Duration(10) * time.Second)

				client.Ack(context.Background(), &pb.AckRequest{
					QueueName: QueueName,
					Id:        msg.Id,
				})

				elapsed := time.Since(processTime)
				log.Info().
					Int("worker", workerId).
					Uint64("id", msg.Id).
					Str("group", msg.Group).
					Str("took", elapsed.String()).
					Msgf("Acknowledged message: %s", msg.Content)

				// Signal to the server that we are ready for the next message
				stream.Send(&pb.DequeueRequest{})
			}
		}(i)
	}

	// Wait for interrupt signal
	<-signalChan
	log.Info().Msg("Received interrupt signal, shutting down...")
}

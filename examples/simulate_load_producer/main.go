package main

import (
	"context"
	"flag"
	"math/rand/v2"
	"os"
	"time"

	"github.com/google/uuid"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var logLevel = "info"
var address = ""
var numberOfMessages = 1_000_000
var queueName = ""
var messageContent = ""
var numberOfGroups = 10
var sleepMillis = 200

var exampleMessage = `{
	"task": "media.transcode_video",
	"task_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
	"customer_id": "acme-studios",
	"args": {
		"source_url": "s3://media-uploads/acme-studios/raw/episode-12.mov",
		"destination_url": "s3://media-processed/acme-studios/hls/episode-12/",
		"output_profile": "h264_1080p_hls",
		"audio_tracks": ["en", "es"],
		"generate_thumbnails": true
	},
	"metadata": {
		"submitted_at": "2026-01-15T12:34:56Z",
		"retries": 0,
		"max_retries": 3,
		"timeout_seconds": 3600,
		"worker_pool": "transcode-workers-us-east-1"
	}
}`

func main() {
	flag.StringVar(&address, "address", "localhost:10000", "gRPC server address")
	flag.IntVar(&numberOfMessages, "number", 100, "Number of messages to send")
	flag.StringVar(&queueName, "queue", "test-queue", "Queue name")
	flag.StringVar(&messageContent, "content", "", "Message content")
	flag.IntVar(&numberOfGroups, "groups", 10, "Number of groups")
	flag.IntVar(&sleepMillis, "sleep", 200, "Time to seep in milliseconds")
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

	groups := make([]string, numberOfGroups)
	for i := 0; i < numberOfGroups; i++ {
		groups[i] = uuid.New().String()
	}

	pickSkewed := func() int {
		r := rand.Float64()
		// square bias: makes smaller indices more likely
		idx := int(r * r * float64(numberOfGroups))
		if idx >= numberOfGroups {
			idx = numberOfGroups - 1
		}
		return idx
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
			// MaxUnacked: 0,
		},
	})

	// Create a stream for sending messages
	stream, err := client.EnqueueStream(context.Background())
	if err != nil {
		log.Fatal().Msgf("Failed to open stream: %v", err)
	}

	if messageContent == "" {
		messageContent = exampleMessage
	}

	// Produce messages in a loop
	for i := 0; i < numberOfMessages; {
		started := time.Now()

		// randomely choose a group preferably unevenly distributed
		group := groups[pickSkewed()]

		msg := &pb.EnqueueRequest{
			QueueName: queueName,
			Content:   messageContent,
			Group:     group,
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
			Msgf(
				"Sent a message: %d",
				len(ack.Content),
			)

		if sleepMillis > 0 {
			time.Sleep(time.Duration(sleepMillis) * time.Millisecond) // Simulate delay between messages
		}

		i++
	}

	// Close the stream
	if err := stream.CloseSend(); err != nil {
		log.Fatal().Msgf("Failed to close stream: %v", err)
	}
}

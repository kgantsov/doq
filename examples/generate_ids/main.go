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
var number = 1

func main() {
	flag.StringVar(&address, "address", "localhost:10000", "gRPC server address")
	flag.IntVar(&number, "number", 1, "Number of IDs to generate")
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

	// Produce messages in a loop
	for {
		started := time.Now()

		// Create a stream for sending messages
		msg, err := client.GenerateIDs(
			context.Background(),
			&pb.GenerateIDsRequest{
				Number: int32(number),
			},
		)
		if err != nil {
			log.Fatal().Msgf("Failed to open stream: %v", err)
		}

		log.Info().
			Str("took", time.Since(started).String()).
			Msgf("Sent a message: %v", msg.Ids)
	}
}

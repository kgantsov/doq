package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kgantsov/doq/pkg/http"
	"github.com/kgantsov/doq/pkg/raft"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if len(os.Args) < 4 {
		log.Fatal().Msgf("Usage: %s <nodeID> <httpPort> <raftPort> <peer1> <peer2> ...", os.Args[0])
	}

	nodeID := os.Args[1]
	httpPort := os.Args[2]
	raftPort := os.Args[3]
	peers := os.Args[4:]

	opts := badger.DefaultOptions(fmt.Sprintf("store/%s", nodeID))
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	log.Info().Msgf("Starting node %s with HTTP on %s and Raft on %s", nodeID, httpPort, raftPort)
	raftDir := fmt.Sprintf("raft/%s", nodeID)
	node := raft.NewNode(db, raftDir, nodeID, httpPort, raftPort, peers)

	node.Initialize()
	h := http.NewHttpService(httpPort, node)
	if err := h.Start(); err != nil {
		log.Error().Msgf("failed to start HTTP service: %s", err.Error())
	}
}

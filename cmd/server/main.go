package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/kgantsov/doq/pkg/cluster"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/grpc"
	"github.com/kgantsov/doq/pkg/http"
	"github.com/kgantsov/doq/pkg/raft"
)

func Run(cmd *cobra.Command, args []string) {
	// Load the config
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	logLevel, err := zerolog.ParseLevel(config.Logging.LogLevel)
	if err != nil {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(logLevel)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if config.Storage.DataDir == "" {
		log.Info().Msg("No storage directory specified")
	}
	if err := os.MkdirAll(config.Storage.DataDir, 0700); err != nil {
		log.Fatal().Msgf("failed to create path '%s' for a storage: %s", config.Storage.DataDir, err.Error())
	}

	hosts := []string{}

	var cl *cluster.Cluster
	var j *cluster.Joiner

	if config.Cluster.ServiceName != "" {
		namespace := "default"
		serviceDiscovery := cluster.NewServiceDiscoverySRV(namespace, config.Cluster.ServiceName)
		cl = cluster.NewCluster(serviceDiscovery, namespace, config.Cluster.ServiceName, config.Http.Port)

		if err := cl.Init(); err != nil {
			log.Warn().Msgf("Error initialising a cluster: %s", err)
			os.Exit(1)
		}

		config.Cluster.NodeID = cl.NodeID()
		config.Raft.Address = cl.RaftAddr()
		hosts = cl.Hosts()

	} else {
		if config.Cluster.JoinAddr != "" {
			hosts = append(hosts, config.Cluster.JoinAddr)
		}
	}

	opts := badger.DefaultOptions(
		filepath.Join(config.Storage.DataDir, config.Cluster.NodeID, "store"),
	)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	log.Info().Msgf(
		"Starting node (%s) %s with HTTP on %s and Raft on %s %+v",
		config.Cluster.ServiceName,
		config.Cluster.NodeID,
		config.Http.Port,
		config.Raft.Address,
		hosts,
	)
	node := raft.NewNode(
		db,
		filepath.Join(config.Storage.DataDir, config.Cluster.NodeID, "raft"),
		config,
		hosts,
	)

	if config.Cluster.ServiceName != "" {
		node.SetLeaderChangeFunc(cl.LeaderChanged)
	}

	node.Initialize()

	// If join was specified, make the join request.
	j = cluster.NewJoiner(config.Cluster.NodeID, config.Raft.Address, hosts)

	if err := j.Join(); err != nil {
		log.Fatal().Msg(err.Error())
	}

	node.InitIDGenerator()

	if config.Grpc.Address != "" {
		lis, err := net.Listen("tcp", config.Grpc.Address)
		if err != nil {
			log.Fatal().Msgf("failed to listen: %v", err)
		}

		grpcServer, err := grpc.NewGRPCServer(node)
		if err != nil {
			log.Fatal().Msgf("failed to create GRPC server: %v", err)
		}

		go grpcServer.Serve(lis)
	}

	h := http.NewHttpService(config.Http.Port, node)
	if err := h.Start(); err != nil {
		log.Error().Msgf("failed to start HTTP service: %s", err.Error())
	}
}

func main() {

	rootCmd := config.InitCobraCommand(Run)

	if err := rootCmd.Execute(); err != nil {
		log.Warn().Err(err)
	}
}

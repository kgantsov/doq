package main

import (
	"embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	netHttp "net/http"
	_ "net/http/pprof"

	_ "go.uber.org/automaxprocs"

	"github.com/pkg/profile"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/kgantsov/doq/pkg/cluster"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/grpc"
	"github.com/kgantsov/doq/pkg/http"
	"github.com/kgantsov/doq/pkg/logger"
	"github.com/kgantsov/doq/pkg/raft"
)

// Embed a single file
//
//go:embed index.html
var indexHtmlFS embed.FS

// Embed a directory
//
//go:embed assets/*
var frontendFS embed.FS

func Run(cmd *cobra.Command, args []string) {
	// Load the config
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano

	logLevel, err := zerolog.ParseLevel(config.Logging.LogLevel)
	if err != nil {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(logLevel)
	}

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

	if config.Storage.Compression != "" {
		opts = opts.WithCompression(config.Storage.CompressionType())
	}
	if config.Storage.ZSTDCompressionLevel > 0 {
		opts = opts.WithZSTDCompressionLevel(config.Storage.ZSTDCompressionLevel)
	}
	if config.Storage.BlockCacheSize > 0 {
		opts = opts.WithBlockCacheSize(config.Storage.BlockCacheSize)
	}

	if config.Storage.IndexCacheSize > 0 {
		opts = opts.WithIndexCacheSize(config.Storage.IndexCacheSize)
	}
	if config.Storage.BaseTableSize > 0 {
		opts = opts.WithBaseTableSize(config.Storage.BaseTableSize)
	}
	if config.Storage.NumCompactors > 0 {
		opts = opts.WithNumCompactors(config.Storage.NumCompactors)
	}
	if config.Storage.NumLevelZeroTables > 0 {
		opts = opts.WithNumLevelZeroTables(config.Storage.NumLevelZeroTables)
	}
	if config.Storage.NumLevelZeroTablesStall > 0 {
		opts = opts.WithNumLevelZeroTablesStall(config.Storage.NumLevelZeroTablesStall)
	}
	if config.Storage.NumMemtables > 0 {
		opts = opts.WithNumMemtables(config.Storage.NumMemtables)
	}
	if config.Storage.ValueLogFileSize > 0 {
		opts = opts.WithValueLogFileSize(config.Storage.ValueLogFileSize)
	}

	opts.Logger = &logger.BadgerLogger{}

	log.Debug().Msgf("Badger options: %+v", opts)

	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	go func() {
		ticker := time.NewTicker(time.Duration(config.Storage.GCInterval) * time.Second)
		defer ticker.Stop()

		log.Info().Msg("Started running value GC")

		for range ticker.C {
			log.Debug().Msg("Running value GC")
		again:
			err := db.RunValueLogGC(config.Storage.GCDiscardRatio)
			if err == nil {
				goto again
			}
		}
	}()

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

		port := lis.Addr().(*net.TCPAddr).Port

		grpcServer, err := grpc.NewGRPCServer(node, port)
		if err != nil {
			log.Fatal().Msgf("failed to create GRPC server: %v", err)
		}

		go grpcServer.Serve(lis)
	}

	if config.Profiling.Enabled {
		defer profile.Start(profile.MemProfile).Stop()

		go func() {
			netHttp.ListenAndServe(fmt.Sprintf(":%d", config.Profiling.Port), nil)
		}()
	}

	h := http.NewHttpService(config, node, indexHtmlFS, frontendFS)
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

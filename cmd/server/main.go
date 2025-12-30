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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/kgantsov/doq/pkg/cluster"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/grpc"
	"github.com/kgantsov/doq/pkg/http"
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

func RunServer(cmd *cobra.Command, args []string) {
	// Load the config
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	config.ConfigureLogger()

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
		if config.Cluster.Namespace != "" {
			namespace = config.Cluster.Namespace
		}
		serviceDiscovery := cluster.NewServiceDiscoverySRV(namespace, config.Cluster.ServiceName)
		cl = cluster.NewCluster(
			serviceDiscovery,
			namespace,
			config.Cluster.ServiceName,
			config.Raft.Address,
			config.Http.Port,
		)

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

	db, err := badger.Open(config.BadgerOptions("store"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	raftDB, err := badger.Open(config.BadgerOptions("raft_stable_store"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer raftDB.Close()

	go RunValueLogGC(config, db)
	go RunValueLogGC(config, raftDB)

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
		raftDB,
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

		grpcServer, err := grpc.NewGRPCServer(config, node, port)
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

func RunValueLogGC(config *config.Config, db *badger.DB) {
	interval := time.Duration(config.Storage.GCInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Msgf("Started running value GC for '%s' every %s", db.Opts().Dir, interval)

	for range ticker.C {
		log.Info().Msgf("Running value GC for '%s'", db.Opts().Dir)
	again:
		err := db.RunValueLogGC(config.Storage.GCDiscardRatio)
		if err == nil {
			log.Info().Msgf("Running next iteration of value GC for '%s'", db.Opts().Dir)
			goto again
		}
		log.Info().Msgf("Finished value GC for '%s'", db.Opts().Dir)
	}
}

func main() {
	rootCmd := config.InitCobraCommand(RunServer)

	var getCmd = &cobra.Command{
		Use: "get",
	}

	rootCmd.AddCommand(getCmd)

	rootCmd.AddCommand(NewCmdRestore())
	getCmd.AddCommand(NewCmdQueues())
	getCmd.AddCommand(NewCmdMessages())

	if err := rootCmd.Execute(); err != nil {
		log.Warn().Err(err)
	}
}

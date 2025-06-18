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

func Run(cmd *cobra.Command, args []string) {
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

	opts := config.BadgerOptions()

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

func NewCmdRestore() *cobra.Command {
	var fullBackup string
	var incrementalBackups []string

	cmd := &cobra.Command{
		Use:     "restore",
		Short:   "Restore DB from a backup",
		Aliases: []string{"restore"},
		Run: func(cmd *cobra.Command, args []string) {
			config, err := config.LoadConfig()
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				return
			}

			config.ConfigureLogger()

			log.Info().Msgf("Full backup: %s", fullBackup)
			log.Info().Msgf("Incremental backups: %v", incrementalBackups)

			if config.Storage.DataDir == "" {
				log.Info().Msg("No storage directory specified")
			}
			if err := os.MkdirAll(config.Storage.DataDir, 0700); err != nil {
				log.Fatal().Msgf(
					"failed to create path '%s' for a storage: %s", config.Storage.DataDir, err.Error(),
				)
			}

			opts := config.BadgerOptions()

			db, err := badger.Open(opts)
			if err != nil {
				log.Fatal().Msg(err.Error())
			}
			defer db.Close()

			err = restoreFile(db, fullBackup)
			if err != nil {
				log.Error().Msgf("Error restoring from %s: %v", fullBackup, err)
				return
			}
			for _, backup := range incrementalBackups {
				err = restoreFile(db, backup)
				if err != nil {
					log.Error().Msgf("Error restoring from %s: %v", backup, err)
					return
				}
			}
		},
	}

	cmd.Flags().StringVarP(&fullBackup, "full", "f", "", "Full backup")
	cmd.Flags().StringSliceVarP(
		&incrementalBackups, "incremental", "i", []string{}, "List of incremental backups",
	)

	return cmd
}

func restoreFile(db *badger.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Error().Msgf("Error opening %s: %v", filename, err)
		return err
	}
	defer file.Close()

	err = db.Load(file, 256) // Load with concurrency
	if err != nil {
		log.Error().Err(err)
		return err
	}
	log.Info().Msgf("Restored from %s", filename)
	return nil
}

func main() {

	rootCmd := config.InitCobraCommand(Run)
	rootCmd.AddCommand(NewCmdRestore())

	if err := rootCmd.Execute(); err != nil {
		log.Warn().Err(err)
	}
}

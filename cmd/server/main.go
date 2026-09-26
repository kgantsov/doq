package main

import (
	"embed"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	netHttp "net/http"
	_ "net/http/pprof"

	_ "go.uber.org/automaxprocs"

	"github.com/pkg/profile"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	grpcpkg "google.golang.org/grpc"

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

	// Decide whether this node seeds the Raft cluster. Exactly one node may
	// bootstrap; every other node joins it. Otherwise each pod forms its own
	// isolated Raft group that can never be merged.
	node.SetBootstrap(shouldBootstrap(config))

	node.Initialize()

	// Only call Join for nodes that have no prior Raft state. Nodes that are
	// restarting already exist in the persisted cluster configuration; they
	// rejoin automatically through Raft's peer discovery and must not call
	// Join, otherwise they attempt AddVoter on a non-leader and crash-loop.
	if node.IsNewNode() {
		// Re-resolve peers on every join retry so a node that started before its
		// peers (or before the ordinal-0 seed became leader) still finds them,
		// instead of looping over the stale snapshot captured at startup. Only
		// available in Kubernetes mode, where cl performs SRV discovery.
		var resolve func() ([]string, error)
		if cl != nil {
			resolve = cl.DiscoverPeers
		}
		j = cluster.NewJoiner(config.Cluster.NodeID, config.Raft.Address, hosts, resolve)

		if err := j.Join(); err != nil {
			log.Fatal().Msg(err.Error())
		}
	}

	node.InitIDGenerator()

	var grpcServer *grpcpkg.Server
	if config.Grpc.Address != "" {
		lis, err := net.Listen("tcp", config.Grpc.Address)
		if err != nil {
			log.Fatal().Msgf("failed to listen: %v", err)
		}

		port := lis.Addr().(*net.TCPAddr).Port

		grpcServer, err = grpc.NewGRPCServer(config, node, port)
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

	// Graceful shutdown on SIGINT/SIGTERM (e.g. a Kubernetes rolling restart).
	// Ordering matters:
	//   1. Stop the gRPC and HTTP edges concurrently, draining in-flight
	//      requests. GracefulStop sends a GOAWAY so clients reconnect and
	//      re-resolve to a live node; it is time-bounded so a stuck RPC cannot
	//      hold shutdown past the Kubernetes termination grace period.
	//   2. Once no new requests can arrive, shut Raft down - transferring
	//      leadership first so a successor is elected immediately instead of
	//      after an election timeout.
	//   3. Only then does RunServer return and the deferred BadgerDB closes run,
	//      so the stores are flushed strictly after Raft has stopped touching
	//      them.
	shutdownDone := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info().Msgf("Received signal %s, shutting down gracefully", sig)

		var wg sync.WaitGroup
		if grpcServer != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stopGRPCGracefully(grpcServer, 10*time.Second)
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Shutdown(); err != nil {
				log.Error().Msgf("failed to shut down HTTP service: %s", err.Error())
			}
		}()
		wg.Wait()

		if err := node.Shutdown(); err != nil {
			log.Error().Msgf("failed to shut down Raft: %s", err.Error())
		}

		close(shutdownDone)
	}()

	if err := h.Start(); err != nil {
		// A genuine startup failure (e.g. the port is already in use). No
		// graceful sequence is in flight, so return and let the deferred store
		// closes run.
		log.Error().Msgf("failed to start HTTP service: %s", err.Error())
		return
	}

	// Start returned nil, which for Fiber means Shutdown was called - i.e. the
	// graceful sequence above is running. Wait for it (including Raft shutdown)
	// to finish before RunServer returns and the deferred DB closes execute.
	<-shutdownDone
}

// stopGRPCGracefully drains in-flight RPCs, but forces a stop after timeout so a
// stuck RPC cannot hold shutdown past the Kubernetes termination grace period.
func stopGRPCGracefully(s *grpcpkg.Server, timeout time.Duration) {
	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(timeout):
		log.Warn().Msg("gRPC graceful stop timed out; forcing stop")
		s.Stop()
	}
}

// shouldBootstrap decides whether this node seeds the Raft cluster. The
// cluster.bootstrap config value takes precedence: "true"/"false" force the
// decision, while "auto" (the default) derives it. In Kubernetes the ordinal-0
// pod of the StatefulSet is the deterministic seed; outside Kubernetes a node
// with no explicit join address seeds, and a node with one joins.
func shouldBootstrap(config *config.Config) bool {
	switch strings.ToLower(strings.TrimSpace(config.Cluster.Bootstrap)) {
	case "true":
		return true
	case "false":
		return false
	}

	if config.Cluster.ServiceName != "" {
		// Kubernetes StatefulSet: the ordinal-0 pod is the deterministic seed;
		// all other pods start empty and join it.
		hostname, err := os.Hostname()
		return err == nil && strings.HasSuffix(hostname, "-0")
	}
	// An explicit join address means this node is not the seed.
	return config.Cluster.JoinAddr == ""
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

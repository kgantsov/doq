package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kgantsov/doq/pkg/cluster"
	"github.com/kgantsov/doq/pkg/http"
	"github.com/kgantsov/doq/pkg/raft"
)

const (
	DefaultHTTPPort = "8000"
	DefaultRaftPort = "localhost:9000"
)

var httpPort string
var raftPort string
var dataDir string
var nodeID string
var joinAddr string
var ServiceName string

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	flag.StringVar(&httpPort, "httpAddr", DefaultHTTPPort, "Set the HTTP bind address")
	flag.StringVar(&raftPort, "raftAddr", DefaultRaftPort, "Set Raft bind address")
	flag.StringVar(&dataDir, "dataDir", DefaultRaftPort, "Set data directory")
	flag.StringVar(&joinAddr, "join", "", "Set join address, if any")
	flag.StringVar(&nodeID, "id", "", "Node ID. If not set, same as Raft bind address")
	flag.StringVar(&ServiceName, "service-name", "", "Name of the service in Kubernetes")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if dataDir == "" {
		log.Info().Msg("No storage directory specified")
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		log.Fatal().Msgf("failed to create path '%s' for a storage: %s", dataDir, err.Error())
	}

	hosts := []string{}

	var cl *cluster.Cluster
	var j *cluster.Joiner

	if ServiceName != "" {
		namespace := "default"
		serviceDiscovery := cluster.NewServiceDiscoverySRV(namespace, ServiceName)
		cl = cluster.NewCluster(serviceDiscovery, namespace, ServiceName, httpPort)

		if err := cl.Init(); err != nil {
			log.Warn().Msgf("Error initialising a cluster: %s", err)
			os.Exit(1)
		}

		nodeID = cl.NodeID()
		raftPort = cl.RaftAddr()
		hosts = cl.Hosts()

	} else {
		if joinAddr != "" {
			hosts = append(hosts, joinAddr)
		}
	}

	opts := badger.DefaultOptions(filepath.Join(dataDir, nodeID, "store"))
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	log.Info().Msgf(
		"Starting node (%s) %s with HTTP on %s and Raft on %s %+v", ServiceName, nodeID, httpPort, raftPort, hosts,
	)
	node := raft.NewNode(
		db, filepath.Join(dataDir, nodeID, "raft"), nodeID, httpPort, raftPort, hosts,
	)

	if ServiceName != "" {
		node.SetLeaderChangeFunc(cl.LeaderChanged)
	}

	node.Initialize()

	// If join was specified, make the join request.
	j = cluster.NewJoiner(nodeID, raftPort, hosts)

	if err := j.Join(); err != nil {
		log.Fatal().Msg(err.Error())
	}

	h := http.NewHttpService(httpPort, node)
	if err := h.Start(); err != nil {
		log.Error().Msgf("failed to start HTTP service: %s", err.Error())
	}
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kgantsov/doq/pkg/cluster"
	"github.com/kgantsov/doq/pkg/http"
	"github.com/kgantsov/doq/pkg/raft"
)

type Config struct {
	Http struct {
		Port string `mapstructure:"port"`
	}
	Raft struct {
		Address string `mapstructure:"address"`
	}
	Storage struct {
		DataDir string `mapstructure:"data_dir"`
	}
	Cluster struct {
		NodeID      string `mapstructure:"node_id"`
		ServiceName string `mapstructure:"service_name"`
		JoinAddr    string `mapstructure:"join_addr"`
	}
}

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "doq",
	Short: "DOQ is a distributed queue",
	Run: func(cmd *cobra.Command, args []string) {
		// Load the config
		config, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

		zerolog.SetGlobalLevel(zerolog.DebugLevel)
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
			config.Cluster.NodeID,
			config.Http.Port,
			config.Raft.Address,
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

		h := http.NewHttpService(config.Http.Port, node)
		if err := h.Start(); err != nil {
			log.Error().Msgf("failed to start HTTP service: %s", err.Error())
		}
	},
}

func loadConfig() (*Config, error) {
	var config Config

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	return &config, nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// Command-line flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.yaml)")
	rootCmd.Flags().String("http.port", "8000", "Port to run the HTTP server on")
	rootCmd.Flags().String("raft.address", "localhost:9000", "Raft bind address")
	rootCmd.Flags().String("storage.data_dir", "data", "Data directory")
	rootCmd.Flags().String("cluster.node_id", "node-1", "Node ID. If not set, same as Raft bind address")
	rootCmd.Flags().String("cluster.service_name", "", "Name of the service in Kubernetes")
	rootCmd.Flags().String("cluster.join_addr", "", "Set join address, if any")

	// Bind CLI flags to Viper settings
	viper.BindPFlag("http.port", rootCmd.Flags().Lookup("http.port"))
	viper.BindPFlag("raft.address", rootCmd.Flags().Lookup("raft.address"))
	viper.BindPFlag("storage.data_dir", rootCmd.Flags().Lookup("storage.data_dir"))
	viper.BindPFlag("cluster.node_id", rootCmd.Flags().Lookup("cluster.node_id"))
	viper.BindPFlag("cluster.service_name", rootCmd.Flags().Lookup("cluster.service_name"))
	viper.BindPFlag("cluster.join_addr", rootCmd.Flags().Lookup("cluster.join_addr"))
}

func initConfig() {
	log.Info().Msgf("Initializing config %s", cfgFile)

	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config file
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	// Enable environment variable support
	viper.AutomaticEnv()

	// Read the config file if found
	if err := viper.ReadInConfig(); err == nil {
		log.Warn().Msgf("Using config file: %s", viper.ConfigFileUsed())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Warn().Err(err)
	}
}

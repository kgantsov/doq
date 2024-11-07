package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

type ProfilingConfig struct {
	Enabled bool  `mapstructure:"enabled"`
	Port    int32 `mapstructure:"port"`
}

type PrometheusConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type LoggingConfig struct {
	LogLevel string `mapstructure:"level"`
}

type HttpConfig struct {
	Port string `mapstructure:"port"`
}

type GrpcConfig struct {
	Address string `mapstructure:"address"`
}

type RaftConfig struct {
	Address string `mapstructure:"address"`
}

type StorageConfig struct {
	DataDir        string  `mapstructure:"data_dir"`
	GCInterval     int64   `mapstructure:"gc_interval"`
	GCDiscardRatio float64 `mapstructure:"gc_discard_ratio"`
}

type QueueStatsConfig struct {
	WindowSide int `mapstructure:"window_side"`
}

type QueueConfig struct {
	AcknowledgementCheckInterval int64            `mapstructure:"acknowledgement_check_interval"`
	AcknowledgementTimeout       int64            `mapstructure:"acknowledgement_timeout"`
	QueueStats                   QueueStatsConfig `mapstructure:"stats"`
}

type ClusterConfig struct {
	NodeID      string `mapstructure:"node_id"`
	ServiceName string `mapstructure:"service_name"`
	JoinAddr    string `mapstructure:"join_addr"`
}

type Config struct {
	Profiling  ProfilingConfig
	Prometheus PrometheusConfig
	Logging    LoggingConfig
	Http       HttpConfig
	Grpc       GrpcConfig
	Raft       RaftConfig
	Storage    StorageConfig
	Queue      QueueConfig
	Cluster    ClusterConfig
}

func LoadConfig() (*Config, error) {
	var config Config

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	return &config, nil
}

func InitCobraCommand(runFunc func(cmd *cobra.Command, args []string)) *cobra.Command {
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
		// log.Warn().Msgf("Using config file: %s", viper.ConfigFileUsed())
	}

	var rootCmd = &cobra.Command{
		Use:   "doq",
		Short: "DOQ is a distributed queue",
		Run:   runFunc,
	}

	// Command-line flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.yaml)")
	rootCmd.Flags().Bool("profiling.enabled", false, "Enable profiling")
	rootCmd.Flags().Int32("profiling.port", 6060, "Profiling port")
	rootCmd.Flags().String("prometheus.enabled", "false", "Enable Prometheus")
	rootCmd.Flags().String("logging.level", "warning", "Log level")
	rootCmd.Flags().String("http.port", "8000", "Port to run the HTTP server on")
	rootCmd.Flags().String("grpc.address", "", "Address to run the GRPC server on")
	rootCmd.Flags().String("raft.address", "localhost:9000", "Raft bind address")
	rootCmd.Flags().String("storage.data_dir", "data", "Data directory")
	rootCmd.Flags().Int64("storage.gc_interval", 300, "Garbage collection interval in seconds")
	rootCmd.Flags().Float64("storage.gc_discard_ratio", 0.7, "Garbage collection discard ratio")
	rootCmd.Flags().String("cluster.node_id", "node-1", "Node ID. If not set, same as Raft bind address")
	rootCmd.Flags().String("cluster.service_name", "", "Name of the service in Kubernetes")
	rootCmd.Flags().String("cluster.join_addr", "", "Set join address, if any")
	rootCmd.Flags().Int64("queue.acknowledgement_check_interval", 1, "Acknowledgement check interval in seconds")
	rootCmd.Flags().Int64("queue.acknowledgement_timeout", 60, "Acknowledgement timeout in seconds")
	rootCmd.Flags().Int("queue.stats.window_side", 10, "Window side for queue stats in seconds")

	// Bind CLI flags to Viper settings
	viper.BindPFlag("profiling.enabled", rootCmd.Flags().Lookup("profiling.enabled"))
	viper.BindPFlag("profiling.port", rootCmd.Flags().Lookup("profiling.port"))
	viper.BindPFlag("prometheus.enabled", rootCmd.Flags().Lookup("prometheus.enabled"))
	viper.BindPFlag("logging.level", rootCmd.Flags().Lookup("logging.level"))
	viper.BindPFlag("http.port", rootCmd.Flags().Lookup("http.port"))
	viper.BindPFlag("grpc.address", rootCmd.Flags().Lookup("grpc.address"))
	viper.BindPFlag("raft.address", rootCmd.Flags().Lookup("raft.address"))
	viper.BindPFlag("storage.data_dir", rootCmd.Flags().Lookup("storage.data_dir"))
	viper.BindPFlag("storage.gc_interval", rootCmd.Flags().Lookup("storage.gc_interval"))
	viper.BindPFlag("storage.gc_discard_ratio", rootCmd.Flags().Lookup("storage.gc_discard_ratio"))
	viper.BindPFlag("cluster.node_id", rootCmd.Flags().Lookup("cluster.node_id"))
	viper.BindPFlag("cluster.service_name", rootCmd.Flags().Lookup("cluster.service_name"))
	viper.BindPFlag("cluster.join_addr", rootCmd.Flags().Lookup("cluster.join_addr"))
	viper.BindPFlag("queue.acknowledgement_check_interval", rootCmd.Flags().Lookup("queue.acknowledgement_check_interval"))
	viper.BindPFlag("queue.acknowledgement_timeout", rootCmd.Flags().Lookup("queue.acknowledgement_timeout"))
	viper.BindPFlag("queue.stats.window_side", rootCmd.Flags().Lookup("queue.stats.window_side"))

	return rootCmd
}

package config

import (
	"fmt"

	"github.com/dgraph-io/badger/v4/options"
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
	Address      string `mapstructure:"address"`
	ApplyTimeout int64  `mapstructure:"apply_timeout"`
}

type StorageConfig struct {
	DataDir        string  `mapstructure:"data_dir"`
	GCInterval     int64   `mapstructure:"gc_interval"`
	GCDiscardRatio float64 `mapstructure:"gc_discard_ratio"`

	// BadgerDB options
	Compression             string `mapstructure:"compression"`
	ZSTDCompressionLevel    int    `mapstructure:"zstd_compression_level"`
	BlockCacheSize          int64  `mapstructure:"block_cache_size"`
	IndexCacheSize          int64  `mapstructure:"index_cache_size"`
	BaseTableSize           int64  `mapstructure:"base_table_size"`
	NumCompactors           int    `mapstructure:"num_compactors"`
	NumLevelZeroTables      int    `mapstructure:"num_level_zero_tables"`
	NumLevelZeroTablesStall int    `mapstructure:"num_level_zero_tables_stall"`
	NumMemtables            int    `mapstructure:"num_memtables"`
	ValueLogFileSize        int64  `mapstructure:"value_log_file_size"`
}

func (s *StorageConfig) CompressionType() options.CompressionType {
	switch s.Compression {
	case "none":
		return options.None
	case "snappy":
		return options.Snappy
	case "zstd":
		return options.ZSTD
	default:
		return options.None
	}
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
	rootCmd.Flags().Int("raft.apply_timeout", 5, "Raft apply timeout in seconds")
	rootCmd.Flags().String("storage.data_dir", "data", "Data directory")
	rootCmd.Flags().Int64("storage.gc_interval", 300, "Garbage collection interval in seconds")
	rootCmd.Flags().Float64("storage.gc_discard_ratio", 0.7, "Garbage collection discard ratio")
	rootCmd.Flags().String("storage.compression", "", "Compression type. Options: none, snappy, zstd")
	rootCmd.Flags().Int("storage.zstd_compression_level", 0, "ZSTD compression level")
	rootCmd.Flags().Int64("storage.block_cache_size", 0, "Block cache size in MB")
	rootCmd.Flags().Int64("storage.index_cache_size", 0, "Index cache size in MB")
	rootCmd.Flags().Int64("storage.base_table_size", 0, "Base table size in bytes")
	rootCmd.Flags().Int("storage.num_compactors", 0, "Number of compaction workers")
	rootCmd.Flags().Int("storage.num_level_zero_tables", 0, "Number of level zero tables")
	rootCmd.Flags().Int("storage.num_level_zero_tables_stall", 0, "Number of level zero tables to stall compaction")
	rootCmd.Flags().Int("storage.num_memtables", 0, "Number of memtables")
	rootCmd.Flags().Int64("storage.value_log_file_size", 0, "Value log file size in bytes")
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
	viper.BindPFlag("raft.apply_timeout", rootCmd.Flags().Lookup("raft.apply_timeout"))
	viper.BindPFlag("storage.data_dir", rootCmd.Flags().Lookup("storage.data_dir"))
	viper.BindPFlag("storage.gc_interval", rootCmd.Flags().Lookup("storage.gc_interval"))
	viper.BindPFlag("storage.gc_discard_ratio", rootCmd.Flags().Lookup("storage.gc_discard_ratio"))
	viper.BindPFlag("storage.compression", rootCmd.Flags().Lookup("storage.compression"))
	viper.BindPFlag("storage.zstd_compression_level", rootCmd.Flags().Lookup("storage.zstd_compression_level"))
	viper.BindPFlag("storage.block_cache_size", rootCmd.Flags().Lookup("storage.block_cache_size"))
	viper.BindPFlag("storage.index_cache_size", rootCmd.Flags().Lookup("storage.index_cache_size"))
	viper.BindPFlag("storage.base_table_size", rootCmd.Flags().Lookup("storage.base_table_size"))
	viper.BindPFlag("storage.num_compactors", rootCmd.Flags().Lookup("storage.num_compactors"))
	viper.BindPFlag("storage.num_level_zero_tables", rootCmd.Flags().Lookup("storage.num_level_zero_tables"))
	viper.BindPFlag("storage.num_level_zero_tables_stall", rootCmd.Flags().Lookup("storage.num_level_zero_tables_stall"))
	viper.BindPFlag("storage.num_memtables", rootCmd.Flags().Lookup("storage.num_memtables"))
	viper.BindPFlag("storage.value_log_file_size", rootCmd.Flags().Lookup("storage.value_log_file_size"))
	viper.BindPFlag("cluster.node_id", rootCmd.Flags().Lookup("cluster.node_id"))
	viper.BindPFlag("cluster.service_name", rootCmd.Flags().Lookup("cluster.service_name"))
	viper.BindPFlag("cluster.join_addr", rootCmd.Flags().Lookup("cluster.join_addr"))
	viper.BindPFlag("queue.acknowledgement_check_interval", rootCmd.Flags().Lookup("queue.acknowledgement_check_interval"))
	viper.BindPFlag("queue.acknowledgement_timeout", rootCmd.Flags().Lookup("queue.acknowledgement_timeout"))
	viper.BindPFlag("queue.stats.window_side", rootCmd.Flags().Lookup("queue.stats.window_side"))

	return rootCmd
}

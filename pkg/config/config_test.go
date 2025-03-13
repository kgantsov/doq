package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestInitCobraCommand__DefaultValues(t *testing.T) {
	rootCmd := InitCobraCommand(func(cmd *cobra.Command, args []string) {
		assert.NotNil(t, cmd)

		config, err := LoadConfig()
		assert.Nil(t, err)

		assert.Equal(t, false, config.Profiling.Enabled)
		assert.Equal(t, int32(6060), config.Profiling.Port)
		assert.Equal(t, false, config.Prometheus.Enabled)
		assert.Equal(t, "warning", config.Logging.LogLevel)
		assert.Equal(t, "8000", config.Http.Port)
		assert.Equal(t, "", config.Grpc.Address)
		assert.Equal(t, "localhost:9000", config.Raft.Address)
		assert.Equal(t, int64(5), config.Raft.ApplyTimeout)
		assert.Equal(t, "data", config.Storage.DataDir)
		assert.Equal(t, int64(300), config.Storage.GCInterval)
		assert.Equal(t, float64(0.7), config.Storage.GCDiscardRatio)
		assert.Equal(t, "", config.Storage.Compression)
		assert.Equal(t, 0, config.Storage.ZSTDCompressionLevel)
		assert.Equal(t, int64(0), config.Storage.BaseTableSize)
		assert.Equal(t, int64(0), config.Storage.IndexCacheSize)
		assert.Equal(t, int64(0), config.Storage.BaseTableSize)
		assert.Equal(t, 0, config.Storage.NumCompactors)
		assert.Equal(t, 0, config.Storage.NumLevelZeroTables)
		assert.Equal(t, 0, config.Storage.NumLevelZeroTablesStall)
		assert.Equal(t, 0, config.Storage.NumMemtables)
		assert.Equal(t, int64(0), config.Storage.ValueLogFileSize)
		assert.Equal(t, "node-1", config.Cluster.NodeID)
		assert.Equal(t, "", config.Cluster.ServiceName)
		assert.Equal(t, "", config.Cluster.JoinAddr)
		assert.Equal(t, int64(1), config.Queue.AcknowledgementCheckInterval)
		assert.Equal(t, int64(60), config.Queue.AcknowledgementTimeout)
		assert.Equal(t, 10, config.Queue.QueueStats.WindowSide)
	})

	assert.NotNil(t, rootCmd)

	err := rootCmd.Execute()
	assert.Nil(t, err)
}

func TestBadgerOptions(t *testing.T) {
	config := &Config{
		Storage: StorageConfig{
			DataDir:                 "data",
			Compression:             "zstd",
			ZSTDCompressionLevel:    3,
			BaseTableSize:           2000000,
			IndexCacheSize:          5000000,
			NumCompactors:           2,
			NumLevelZeroTables:      2,
			NumLevelZeroTablesStall: 4,
			NumMemtables:            4,
			ValueLogFileSize:        2000000,
		},
	}

	opts := config.BadgerOptions()

	assert.Equal(t, "data/store", opts.Dir)
	assert.Equal(t, 3, opts.ZSTDCompressionLevel)
	assert.Equal(t, int64(2000000), opts.BaseTableSize)
	assert.Equal(t, int64(5000000), opts.IndexCacheSize)
	assert.Equal(t, 2, opts.NumCompactors)
	assert.Equal(t, 2, opts.NumLevelZeroTables)
	assert.Equal(t, 4, opts.NumLevelZeroTablesStall)
	assert.Equal(t, 4, opts.NumMemtables)
	assert.Equal(t, int64(2000000), opts.ValueLogFileSize)
}

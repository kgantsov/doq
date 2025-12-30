package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

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

			opts := config.BadgerOptions("store")

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

func NewCmdQueues() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "queues",
		Short:   "List all queues",
		Aliases: []string{"queues"},
		Run: func(cmd *cobra.Command, args []string) {
			config := cmd.Context().Value("config").(*config.Config)

			// no need to log anything in cmd
			config.Logging.Level = "warn"

			config.ConfigureLogger()

			if config.Storage.DataDir == "" {
				fmt.Println("No storage directory specified")
				return
			}

			opts := config.BadgerOptions("store")

			db, err := badger.Open(opts)
			if err != nil {
				fmt.Printf("Error opening database: %v\n", err)
				return
			}
			defer db.Close()

			store := storage.NewBadgerStore(db)

			queues, err := store.ListQueues()
			if err != nil {
				fmt.Printf("Error listing queues: %v\n", err)
				return
			}

			fmt.Printf("\nFound %d queues:\n", len(queues))
			for _, queue := range queues {
				strategy := queue.Settings.Strategy
				if queue.Type == "delayed" {
					strategy = ""
				}

				fmt.Printf("%s %s %s\n", queue.Name, queue.Type, strategy)
			}
		},
	}

	return cmd
}

func NewCmdMessages() *cobra.Command {
	var queueName string
	var limit int
	var lastID uint64
	var format string

	cmd := &cobra.Command{
		Use:     "messages",
		Short:   "Get messages from a queue",
		Aliases: []string{"messages"},
		Args:    cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if queueName == "" {
				fmt.Println("Queue name is required")
				return
			}

			config := cmd.Context().Value("config").(*config.Config)

			// no need to log anything in cmd
			config.Logging.Level = "warn"

			config.ConfigureLogger()

			if config.Storage.DataDir == "" {
				fmt.Println("No storage directory specified")
				return
			}

			if config.Cluster.NodeID == "" {
				fmt.Println("No node ID specified")
				return
			}

			opts := config.BadgerOptions("store")

			db, err := badger.Open(opts)
			if err != nil {
				fmt.Printf("Error opening database: %v\n", err)
				return
			}
			defer db.Close()

			store := storage.NewBadgerStore(db)

			var messages []*entity.Message
			if len(args) == 0 {
				messages, err = store.GetMessages(queueName, limit, lastID)
				if err != nil {
					fmt.Printf("Error getting message: %v\n", err)
					return
				}
			} else {
				for _, arg := range args {
					messageID, err := strconv.ParseInt(arg, 10, 64)
					if err != nil {
						fmt.Printf("Invalid message ID: %v\n", err)
						return
					}
					message, err := store.Get(queueName, uint64(messageID))
					if err != nil {
						fmt.Printf("Error getting message: %v\n", err)
						return
					}
					messages = append(messages, message)
				}
			}

			switch format {
			case "text":
				for _, message := range messages {
					fmt.Println("Message")
					fmt.Printf("ID: %d\n", message.ID)
					fmt.Printf("Group: %s\n", message.Group)
					fmt.Printf("Priority: %d\n", message.Priority)
					fmt.Printf("Content: %s\n", message.Content)
					fmt.Printf("Metadata: %+v\n", message.Metadata)
					fmt.Println()
				}
			case "json":
				jsonData, err := json.MarshalIndent(messages, "", "  ")
				if err != nil {
					fmt.Printf("Error marshaling message: %v\n", err)
					return
				}
				fmt.Println(string(jsonData))
			case "jsonl":
				for _, message := range messages {
					jsonData, err := json.Marshal(message)
					if err != nil {
						fmt.Printf("Error marshaling message: %v\n", err)
						return
					}
					fmt.Println(string(jsonData))
				}
			default:
				fmt.Printf("Unknown format: %s\n", format)
				return
			}
		},
	}

	cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue name")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of messages to retrieve")
	cmd.Flags().Uint64VarP(&lastID, "last_id", "l", 0, "Last message ID")
	cmd.Flags().StringVarP(&format, "output", "o", "text", "Output format (text, json, jsonl)")
	return cmd
}

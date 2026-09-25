package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	BaseURL          = "http://localhost:8000/API/v1/queues/transfers/messages"
	ConsumerWorkers  = 1
	ProducerWorkers  = 10
	MinSleepMs       = 100
	MaxSleepMs       = 3000
	MinMsgIntervalMs = 500
	MaxMsgIntervalMs = 10000
	PriorityMin      = 1
	PriorityMax      = 100
)

type TransferMessage struct {
	Group    string `json:"group"`
	Priority int    `json:"priority"`
	Content  string `json:"content"`
}

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
}

func main() {
	// read ConsumersWorkers and ProducerWorkers from flag
	flag.IntVar(&ConsumerWorkers, "consumers", 0, "Number of consumer workers")
	flag.IntVar(&ProducerWorkers, "producers", 10, "Number of producer workers")
	flag.Parse()

	log.Info().Msg("Starting transfer queue simulation")

	// Create context that listens for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// ctx, stop := context.WithCancel(context.Background())
	defer stop()

	var wg sync.WaitGroup

	// Start consumer workers
	for i := 0; i < ConsumerWorkers; i++ {
		wg.Add(1)
		go runConsumer(ctx, &wg, i)
	}

	// Start producer workers (customers)

	log.Info().Int("count", ProducerWorkers).Msg("Starting producers")

	for i := 0; i < ProducerWorkers; i++ {
		wg.Add(1)
		go runProducer(ctx, &wg, i)
	}

	log.Info().Msg("Starting main loop")
	<-ctx.Done()
	log.Info().Msg("Shutdown signal received, waiting for workers to finish")
	// stop()

	// // Wait for all goroutines to finish
	wg.Wait()
	log.Info().Msg("All workers have shutdown. Exiting.")
}

func runConsumer(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	logger := log.With().Int("consumer_id", id).Logger()

	logger.Info().Msg("Consumer started")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Consumer shutting down")
			return
		default:
			// Attempt to get a message from the queue
			resp, err := http.Get(BaseURL + "?ack=true")
			if err != nil {
				logger.Error().Err(err).Msg("Failed to get message from queue")
				time.Sleep(time.Second) // Wait before retrying
				continue
			}

			if resp.StatusCode == http.StatusNoContent {
				// Queue is empty
				logger.Debug().Msg("Queue is empty, waiting...")
				time.Sleep(500 * time.Millisecond)
				resp.Body.Close()
				continue
			}

			if resp.StatusCode != http.StatusOK {
				logger.Error().Int("status", resp.StatusCode).Msg("Unexpected response status")
				resp.Body.Close()
				time.Sleep(time.Second)
				continue
			}

			// Parse the message
			var message TransferMessage
			if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
				logger.Error().Err(err).Msg("Failed to decode message")
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			// Process the message (simulate work)
			processTime := time.Duration(rand.Intn(MaxSleepMs-MinSleepMs)+MinSleepMs) * time.Millisecond
			// processTime := time.Duration(50) * time.Millisecond
			logger.Info().
				Str("group", message.Group).
				Int("priority", message.Priority).
				Str("content", message.Content).
				Dur("process_time", processTime).
				Msg("Processing transfer")

			// Sleep to simulate processing time
			time.Sleep(processTime)

			logger.Info().
				Str("group", message.Group).
				Str("content", message.Content).
				Msg("Transfer processed")
		}
	}
}

func runProducer(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()

	groupName := fmt.Sprintf("customer-%d", id)
	messageCount := 0
	interval := time.Duration(rand.Intn(MaxMsgIntervalMs-MinMsgIntervalMs)+MinMsgIntervalMs) * time.Millisecond

	logger := log.With().Str("producer", groupName).Logger()
	logger.Info().Dur("interval", interval).Msg("Producer started")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Int("messages_sent", messageCount).Msg("Producer shutting down")
			return
		case <-ticker.C:
			messageCount++
			// priority := rand.Intn(PriorityMax-PriorityMin) + PriorityMin
			priority := 100

			message := TransferMessage{
				Group:    groupName,
				Priority: priority,
				Content:  fmt.Sprintf("Transfer payload #%d", messageCount),
			}

			// Send message to queue
			if err := sendMessage(message); err != nil {
				logger.Error().Err(err).Msg("Failed to send message")
				continue
			}

			logger.Info().
				Int("priority", priority).
				Int("count", messageCount).
				Msg("Message sent to queue")
		}
	}
}

func sendMessage(message TransferMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	resp, err := http.Post(BaseURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

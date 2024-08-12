package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kgantsov/doq/pkg/raft"
	"github.com/rs/zerolog/log"
)

type (
	Handler struct {
		node       *raft.Node
		httpClient *http.Client
	}
)

func EnqueueProxy(ctx context.Context, client *http.Client, host string, queueName string, body *EnqueueInputBody) (*EnqueueOutputBody, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", host))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse host", err)
	}

	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8000/API/v1/queues/%s", u.Hostname(), queueName), bytes.NewBuffer(bodyB))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy enqueue request", err)
	}
	defer resp.Body.Close()

	log.Info().Msgf("Response status: %d", resp.StatusCode)
	log.Info().Msgf("Response body: %s", resp.Body)
	var data EnqueueOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func DequeueProxy(ctx context.Context, client *http.Client, host string, queueName string) (*DequeueOutputBody, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", host))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse host", err)
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8000/API/v1/queues/%s", u.Hostname(), queueName), nil)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy enqueue request", err)
	}
	defer resp.Body.Close()

	var data DequeueOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (h *Handler) Enqueue(ctx context.Context, input *EnqueueInput) (*EnqueueOutput, error) {
	queueName := input.QueueName
	priority := input.Body.Priority
	content := input.Body.Content

	log.Info().Msgf("Leader is: %s", h.node.Leader)

	if !h.node.IsLeader() {
		respBody, err := EnqueueProxy(ctx, h.httpClient, h.node.Leader, queueName, &input.Body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to proxy enqueue request")
			return nil, huma.Error503ServiceUnavailable("Failed to proxy enqueue request", err)
		}

		res := &EnqueueOutput{
			Status: http.StatusOK,
			Body: EnqueueOutputBody{
				Status:   "ENQUEUED",
				ID:       respBody.ID,
				Priority: respBody.Priority,
				Content:  respBody.Content,
			},
		}
		return res, nil
	}

	msg, err := h.node.Enqueue(queueName, priority, content)

	if err != nil {
		return nil, huma.Error409Conflict("Failed to enqueue a message", err)
	}

	res := &EnqueueOutput{Status: http.StatusOK}
	res.Body.Status = "ENQUEUED"
	res.Body.ID = msg.ID
	res.Body.Priority = msg.Priority
	res.Body.Content = msg.Content
	return res, nil
}

func (h *Handler) Dequeue(ctx context.Context, input *DequeueInput) (*DequeueOutput, error) {
	queueName := input.QueueName

	if !h.node.IsLeader() {
		respBody, err := DequeueProxy(ctx, h.httpClient, h.node.Leader, queueName)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Failed to proxy dequeue request", err)
		}
		res := &DequeueOutput{
			Status: http.StatusOK,
			Body: DequeueOutputBody{
				Status:   "DEQUEUED",
				ID:       respBody.ID,
				Priority: respBody.Priority,
				Content:  respBody.Content,
			},
		}
		return res, nil
	}

	msg, err := h.node.Dequeue(queueName)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to dequeue a message from a queue", err)
	}

	res := &DequeueOutput{Status: http.StatusOK}
	res.Body.Status = "DEQUEUED"
	res.Body.ID = msg.ID
	res.Body.Priority = msg.Priority
	res.Body.Content = msg.Content
	return res, nil
}

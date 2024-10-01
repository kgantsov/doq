package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func (h *Handler) CreateQueue(ctx context.Context, input *CreateQueueInput) (*CreateQueueOutput, error) {
	queueName := input.Body.Name
	queueType := input.Body.Type

	if !h.node.IsLeader() {
		respBody, err := h.proxy.CreateQueue(ctx, h.node.Leader(), &input.Body)
		if err != nil {
			return nil, err
		}

		res := &CreateQueueOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	err := h.node.CreateQueue(queueType, queueName)

	if err != nil {
		return nil, huma.Error409Conflict("Failed to enqueue a message", err)
	}

	res := &CreateQueueOutput{
		Status: http.StatusOK,
		Body: CreateQueueOutputBody{
			Status: "CREATED",
			Name:   queueName,
			Type:   queueType,
		},
	}

	return res, nil
}

func (h *Handler) DeleteQueue(ctx context.Context, input *DeleteQueueInput) (*DeleteQueueOutput, error) {
	queueName := input.QueueName

	if !h.node.IsLeader() {
		respBody, err := h.proxy.DeleteQueue(ctx, h.node.Leader(), queueName)
		if err != nil {
			return nil, err
		}
		res := &DeleteQueueOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	err := h.node.DeleteQueue(queueName)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to dequeue a message from a queue", err)
	}

	res := &DeleteQueueOutput{
		Status: http.StatusOK,
		Body: DeleteQueueOutputBody{
			Status: "DELETED",
		},
	}
	return res, nil
}

func (h *Handler) QueueStats(ctx context.Context, input *QueueStatsInput) (*QueueStatsOutput, error) {
	queueName := input.QueueName

	stats, err := h.node.GetQueueStats(queueName)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to get stats for a queue", err)
	}

	res := &QueueStatsOutput{
		Status: http.StatusOK,
		Body: QueueStatsOutputBody{
			EnqueueRPS: stats.EnqueueRPS,
			DequeueRPS: stats.DequeueRPS,
			AckRPS:     stats.AckRPS,
			NackRPS:    stats.NackRPS,
			// QueueSize:    stats.QueueSize,
		},
	}

	return res, nil
}

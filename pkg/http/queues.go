package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kgantsov/doq/pkg/entity"
)

func (h *Handler) CreateQueue(ctx context.Context, input *CreateQueueInput) (*CreateQueueOutput, error) {
	queueName := input.Body.Name
	queueType := input.Body.Type
	queueSettings := input.Body.Settings

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

	err := h.node.CreateQueue(queueType, queueName, entity.QueueSettings{
		Strategy:   queueSettings.Strategy,
		MaxUnacked: queueSettings.MaxUnacked,
	})

	if err != nil {
		return nil, huma.Error409Conflict("Failed to enqueue a message", err)
	}

	res := &CreateQueueOutput{
		Status: http.StatusOK,
		Body: CreateQueueOutputBody{
			Status: "CREATED",
			Name:   queueName,
			Type:   queueType,
			Settings: QueueSettings{
				Strategy:   queueSettings.Strategy,
				MaxUnacked: queueSettings.MaxUnacked,
			},
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

func (h *Handler) Queues(ctx context.Context, input *QueuesInput) (*QueuesOutput, error) {
	queues := h.node.GetQueues()

	queueOutputs := make([]QueueOutput, 0, len(queues))

	for _, queue := range queues {
		queueOutputs = append(queueOutputs, QueueOutput{
			Name:       queue.Name,
			Type:       queue.Type,
			EnqueueRPS: queue.Stats.EnqueueRPS,
			DequeueRPS: queue.Stats.DequeueRPS,
			AckRPS:     queue.Stats.AckRPS,
			NackRPS:    queue.Stats.NackRPS,
			Ready:      queue.Ready,
			Unacked:    queue.Unacked,
			Total:      queue.Total,
		})
	}

	res := &QueuesOutput{
		Status: http.StatusOK,
		Body: QueuesOutputBody{
			Queues: queueOutputs,
		},
	}

	return res, nil
}

func (h *Handler) QueueInfo(ctx context.Context, input *QueueInfoInput) (*QueueInfoOutput, error) {
	queueName := input.QueueName

	queueInfo, err := h.node.GetQueueInfo(queueName)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to get stats for a queue", err)
	}

	res := &QueueInfoOutput{
		Status: http.StatusOK,
		Body: QueueInfoOutputBody{
			Name: queueName,
			Type: queueInfo.Type,
			Settings: QueueSettings{
				Strategy:   queueInfo.Settings.Strategy,
				MaxUnacked: queueInfo.Settings.MaxUnacked,
			},
			EnqueueRPS: queueInfo.Stats.EnqueueRPS,
			DequeueRPS: queueInfo.Stats.DequeueRPS,
			AckRPS:     queueInfo.Stats.AckRPS,
			NackRPS:    queueInfo.Stats.NackRPS,
			Ready:      queueInfo.Ready,
			Unacked:    queueInfo.Unacked,
			Total:      queueInfo.Total,
		},
	}

	return res, nil
}

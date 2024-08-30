package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type (
	Handler struct {
		node  Node
		proxy *Proxy
	}
)

func (h *Handler) Enqueue(ctx context.Context, input *EnqueueInput) (*EnqueueOutput, error) {
	queueName := input.QueueName
	group := input.Body.Group
	priority := input.Body.Priority
	content := input.Body.Content

	if !h.node.IsLeader() {
		respBody, err := h.proxy.Enqueue(ctx, h.node.Leader(), queueName, &input.Body)
		if err != nil {
			return nil, err
		}

		res := &EnqueueOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	msg, err := h.node.Enqueue(queueName, group, priority, content)

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
		respBody, err := h.proxy.Dequeue(ctx, h.node.Leader(), queueName, input.Ack)
		if err != nil {
			return nil, err
		}
		res := &DequeueOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	msg, err := h.node.Dequeue(queueName, input.Ack)

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

func (h *Handler) Ack(ctx context.Context, input *AckInput) (*AckOutput, error) {
	queueName := input.QueueName

	if !h.node.IsLeader() {
		respBody, err := h.proxy.Ack(ctx, h.node.Leader(), queueName, input.ID)
		if err != nil {
			return nil, err
		}
		res := &AckOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	err := h.node.Ack(queueName, input.ID)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to acknowledge a message from a queue", err)
	}

	res := &AckOutput{
		Status: http.StatusOK,
		Body: AckOutputBody{
			Status: "ACKNOWLEDGED",
			ID:     input.ID,
		},
	}

	return res, nil
}

func (h *Handler) UpdatePriority(ctx context.Context, input *UpdatePriorityInput) (*UpdatePriorityOutput, error) {
	queueName := input.QueueName
	priority := input.Body.Priority

	if !h.node.IsLeader() {
		respBody, err := h.proxy.UpdatePriority(
			ctx, h.node.Leader(), queueName, input.ID, &input.Body,
		)
		if err != nil {
			return nil, err
		}

		res := &UpdatePriorityOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	err := h.node.UpdatePriority(queueName, input.ID, priority)

	if err != nil {
		return nil, huma.Error409Conflict("Failed to update priority a message", err)
	}

	res := &UpdatePriorityOutput{
		Status: http.StatusOK,
		Body: UpdatePriorityOutputBody{
			Status:   "UPDATED",
			ID:       input.ID,
			Priority: priority,
		},
	}
	return res, nil
}

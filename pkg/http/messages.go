package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/rs/zerolog/log"
)

type (
	Handler struct {
		node   Node
		proxy  *Proxy
		config *config.Config
	}
)

func (h *Handler) Enqueue(ctx context.Context, input *EnqueueInput) (*EnqueueOutput, error) {
	var id uint64
	queueName := input.QueueName
	group := input.Body.Group
	priority := input.Body.Priority
	content := input.Body.Content
	metadata := input.Body.Metadata

	if input.Body.ID != "" {
		id, _ = strconv.ParseUint(input.Body.ID, 10, 64)
	}

	log.Info().Msgf(
		"Enqueue request %t: %s %d, %s, %s, %d, %s %v",
		h.node.IsLeader(),
		h.node.Leader(),
		id,
		queueName,
		group,
		priority,
		content,
		metadata,
	)
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

	msg, err := h.node.Enqueue(queueName, id, group, priority, content, metadata)

	if err != nil {
		return nil, huma.Error409Conflict("Failed to enqueue a message", err)
	}

	res := &EnqueueOutput{
		Status: http.StatusOK,
		Body: EnqueueOutputBody{
			Status:   "ENQUEUED",
			ID:       strconv.Itoa(int(msg.ID)),
			Group:    group,
			Priority: priority,
			Content:  content,
			Metadata: metadata,
		},
	}
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

	res := &DequeueOutput{
		Status: http.StatusOK,
		Body: DequeueOutputBody{
			Status:   "DEQUEUED",
			ID:       strconv.Itoa(int(msg.ID)),
			Group:    msg.Group,
			Priority: msg.Priority,
			Content:  msg.Content,
			Metadata: msg.Metadata,
		},
	}
	return res, nil
}

func (h *Handler) Get(ctx context.Context, input *GetInput) (*GetOutput, error) {
	queueName := input.QueueName

	if !h.node.IsLeader() {
		respBody, err := h.proxy.Get(ctx, h.node.Leader(), queueName, input.ID)
		if err != nil {
			return nil, err
		}
		res := &GetOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	msg, err := h.node.Get(queueName, input.ID)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to get a message from a queue", err)
	}

	res := &GetOutput{
		Status: http.StatusOK,
		Body: GetOutputBody{
			Status:   "GOT",
			ID:       strconv.Itoa(int(msg.ID)),
			Group:    msg.Group,
			Priority: msg.Priority,
			Content:  msg.Content,
			Metadata: msg.Metadata,
		},
	}
	return res, nil
}

func (h *Handler) Delete(ctx context.Context, input *DeleteInput) (*DeleteOutput, error) {
	queueName := input.QueueName

	if !h.node.IsLeader() {
		err := h.proxy.Delete(ctx, h.node.Leader(), queueName, input.ID)
		if err != nil {
			return nil, err
		}
		res := &DeleteOutput{
			Status: http.StatusNoContent,
		}
		return res, nil
	}

	err := h.node.Delete(queueName, input.ID)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to delete a message from a queue", err)
	}

	res := &DeleteOutput{
		Status: http.StatusNoContent,
	}
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
		return nil, huma.Error400BadRequest("Failed to ack a message from a queue", err)
	}

	res := &AckOutput{
		Status: http.StatusOK,
		Body: AckOutputBody{
			Status: "ACKNOWLEDGED",
			ID:     strconv.Itoa(int(input.ID)),
		},
	}

	return res, nil
}

func (h *Handler) Nack(ctx context.Context, input *NackInput) (*NackOutput, error) {
	queueName := input.QueueName
	metadata := input.Body.Metadata

	if !h.node.IsLeader() {
		respBody, err := h.proxy.Nack(ctx, h.node.Leader(), queueName, input.ID, &input.Body)
		if err != nil {
			return nil, err
		}
		res := &NackOutput{
			Status: http.StatusOK,
			Body:   *respBody,
		}
		return res, nil
	}

	err := h.node.Nack(queueName, input.ID, input.Body.Priority, metadata)

	if err != nil {
		return nil, huma.Error400BadRequest("Failed to nack a message from a queue", err)
	}

	res := &NackOutput{
		Status: http.StatusOK,
		Body: NackOutputBody{
			Status:   "UNACKNOWLEDGED",
			ID:       strconv.Itoa(int(input.ID)),
			Metadata: metadata,
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
			ID:       strconv.Itoa(int(input.ID)),
			Priority: priority,
		},
	}
	return res, nil
}

func (h *Handler) GenerateID(ctx context.Context, input *GenerateIdInput) (*GenerateIdOutput, error) {

	ids := make([]string, 0, input.Body.Number)
	for i := 0; i < input.Body.Number; i++ {
		id := h.node.GenerateID()
		ids = append(ids, strconv.Itoa(int(id)))
	}

	res := &GenerateIdOutput{
		Status: http.StatusOK,
		Body: GenerateIdOutputBody{
			IDs: ids,
		},
	}
	return res, nil
}

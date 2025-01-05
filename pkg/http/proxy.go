package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

type Proxy struct {
	client *http.Client
	port   string
}

func NewProxy(client *http.Client, port string) *Proxy {
	return &Proxy{
		client: client,
		port:   port,
	}
}

func (p *Proxy) getHost(host string) string {
	if p.port != "" {
		return fmt.Sprintf("http://%s:%s", host, p.port)
	}

	return fmt.Sprintf("http://%s", host)
}

func (p *Proxy) CreateQueue(
	ctx context.Context, host string, body *CreateQueueInputBody,
) (*CreateQueueOutputBody, huma.StatusError) {

	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/API/v1/queues", p.getHost(host)),
		bytes.NewBuffer(bodyB),
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy create queue request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to create a queue", nil)
	}

	log.Info().Msgf("Response status: %d", resp.StatusCode)
	log.Info().Msgf("Response body: %s", resp.Body)
	var data CreateQueueOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) DeleteQueue(
	ctx context.Context, host string, queueName string,
) (*DeleteQueueOutputBody, huma.StatusError) {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/API/v1/queues/%s", p.getHost(host), queueName),
		nil,
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy delete queue request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to delete a queue", nil)
	}

	var data DeleteQueueOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) Enqueue(ctx context.Context, host string, queueName string, body *EnqueueInputBody) (*EnqueueOutputBody, huma.StatusError) {
	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/API/v1/queues/%s/messages", p.getHost(host), queueName),
		bytes.NewBuffer(bodyB),
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy enqueue request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to enqueue message", nil)
	}

	log.Info().Msgf("Response status: %d", resp.StatusCode)
	log.Info().Msgf("Response body: %s", resp.Body)
	var data EnqueueOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) Dequeue(
	ctx context.Context, host string, queueName string, ack bool,
) (*DequeueOutputBody, huma.StatusError) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf(
			"%s/API/v1/queues/%s/messages?ack=%s",
			p.getHost(host),
			queueName,
			fmt.Sprint(ack),
		),
		nil,
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy enqueue request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to dequeue message", nil)
	}

	var data DequeueOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) Get(
	ctx context.Context, host string, queueName string, id uint64,
) (*GetOutputBody, huma.StatusError) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf(
			"%s/API/v1/queues/%s/messages/%d",
			p.getHost(host),
			queueName,
			id,
		),
		nil,
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy get message request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to get message", nil)
	}

	var data GetOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) Delete(
	ctx context.Context, host string, queueName string, id uint64,
) huma.StatusError {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf(
			"%s/API/v1/queues/%s/messages/%d",
			p.getHost(host),
			queueName,
			id,
		),
		nil,
	)
	if err != nil {
		return huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return huma.Error503ServiceUnavailable("Failed to proxy get message request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		if resp.StatusCode == http.StatusNotFound {
			return huma.Error404NotFound("Message not found", nil)
		}

		return huma.Error400BadRequest("Failed to get message", nil)
	}

	return nil
}

func (p *Proxy) Ack(
	ctx context.Context,
	host string,
	queueName string,
	id uint64,
) (*AckOutputBody, huma.StatusError) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf(
			"%s/API/v1/queues/%s/messages/%d/ack", p.getHost(host), queueName, id,
		),
		nil,
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy ack request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to ack message", nil)
	}

	var data AckOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Warn().Msgf("Failed to decode response: %s", err)
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) Nack(
	ctx context.Context,
	host string,
	queueName string,
	id uint64,
	body *NackInputBody,
) (*NackOutputBody, huma.StatusError) {
	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf(
			"%s/API/v1/queues/%s/messages/%d/nack", p.getHost(host), queueName, id,
		),
		bytes.NewBuffer(bodyB),
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy nack request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to nack message", nil)
	}

	var data NackOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Warn().Msgf("Failed to decode response: %s", err)
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

func (p *Proxy) UpdatePriority(
	ctx context.Context,
	host string,
	queueName string,
	id uint64,
	body *UpdatePriorityInputBody,
) (*UpdatePriorityOutputBody, huma.StatusError) {
	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf(
			"%s/API/v1/queues/%s/messages/%d/priority",
			p.getHost(host),
			queueName,
			id,
		),
		bytes.NewBuffer(bodyB),
	)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create a request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("Failed to proxy enqueue request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error400BadRequest("Failed to update priority", nil)
	}

	log.Info().Msgf("Response status: %d", resp.StatusCode)
	log.Info().Msgf("Response body: %s", resp.Body)
	var data UpdatePriorityOutputBody
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, huma.Error400BadRequest("Failed to decode response", err)
	}

	return &data, nil
}

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

type Proxy struct {
	client *http.Client
}

func NewProxy(client *http.Client) *Proxy {
	return &Proxy{
		client: client,
	}
}

func (p *Proxy) Enqueue(ctx context.Context, host string, queueName string, body *EnqueueInputBody) (*EnqueueOutputBody, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", host))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse host", err)
	}

	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("http://%s:8000/API/v1/queues/%s/messages", u.Hostname(), queueName),
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
) (*DequeueOutputBody, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", host))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse host", err)
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://%s:8000/API/v1/queues/%s/messages?ack=%s", u.Hostname(), queueName, fmt.Sprint(ack)),
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

func (p *Proxy) Ack(
	ctx context.Context,
	host string,
	queueName string,
	id uint64,
) (*AckOutputBody, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", host))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse host", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("http://%s:8000/API/v1/queues/%s/messages/%d/ack", u.Hostname(), queueName, id),
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

func (p *Proxy) UpdatePriority(
	ctx context.Context,
	host string,
	queueName string,
	id uint64,
	body *UpdatePriorityInputBody,
) (*UpdatePriorityOutputBody, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", host))
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse host", err)
	}

	bodyB, err := json.Marshal(body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to marshal body", err)
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://%s:8000/API/v1/queues/%s/messages/%d/priority", u.Hostname(), queueName, id),
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

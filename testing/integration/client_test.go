//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a minimal DOQ HTTP client bound to one node. Followers proxy
// writes to the leader, so any node can be used for queue operations, but
// reads like QueueCounts return that node's own replica.
type Client struct {
	base string
	http *http.Client
}

func NewClient(addr string) *Client {
	return &Client{
		base: "http://" + addr,
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

// Server is one entry of the Raft configuration as reported by a node.
type Server struct {
	ID         string `json:"id"`
	Addr       string `json:"addr"`
	LeaderAddr string `json:"leader_addr"`
	IsLeader   bool   `json:"is_leader"`
	Suffrage   string `json:"suffrage"`
}

// Message is a message as returned by the message endpoints.
type Message struct {
	Status   string            `json:"status,omitempty"`
	ID       string            `json:"id,omitempty"`
	Group    string            `json:"group,omitempty"`
	Priority int64             `json:"priority"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// QueueCounts are a queue's message counts on one node.
type QueueCounts struct {
	Ready   int64 `json:"ready"`
	Unacked int64 `json:"unacked"`
	Total   int64 `json:"total"`
}

// StatusError is returned for any non-2xx response.
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Body)
}

// Alive reports whether the node answers its liveness probe.
func (c *Client) Alive() bool {
	resp, err := c.http.Get(c.base + "/livez")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) Servers() ([]Server, error) {
	var out struct {
		Servers []Server `json:"servers"`
	}
	err := c.do(http.MethodGet, "/API/v1/cluster/servers", nil, &out)
	return out.Servers, err
}

// TransferLeadership asks the node, which must be the leader, to hand
// leadership to another voter. Raft picks the most up-to-date follower; it
// can't be told which one.
func (c *Client) TransferLeadership() error {
	return c.do(http.MethodPost, "/API/v1/cluster/transfer-leadership", nil, nil)
}

// CreateQueue creates a queue. settings is the queue's "settings" object,
// e.g. {"ack_timeout": 60} or {"strategy": "weighted", "max_unacked": 10}.
func (c *Client) CreateQueue(name, queueType string, settings map[string]any) error {
	body := map[string]any{"name": name, "type": queueType, "settings": settings}
	return c.do(http.MethodPost, "/API/v1/queues", body, nil)
}

func (c *Client) Enqueue(queue, group string, priority int64, content string) (Message, error) {
	return c.EnqueueMessage(queue, Message{Group: group, Priority: priority, Content: content})
}

// EnqueueMessage enqueues msg's group, priority, content and metadata.
func (c *Client) EnqueueMessage(queue string, msg Message) (Message, error) {
	var out Message
	body := map[string]any{
		"group": msg.Group, "priority": msg.Priority, "content": msg.Content, "metadata": msg.Metadata,
	}
	err := c.do(http.MethodPost, "/API/v1/queues/"+queue+"/messages", body, &out)
	return out, err
}

// Get returns a message by ID without dequeuing it.
func (c *Client) Get(queue, id string) (Message, error) {
	var msg Message
	err := c.do(http.MethodGet, "/API/v1/queues/"+queue+"/messages/"+id, nil, &msg)
	return msg, err
}

func (c *Client) Delete(queue, id string) error {
	return c.do(http.MethodDelete, "/API/v1/queues/"+queue+"/messages/"+id, nil, nil)
}

// Nack returns an in-flight message to the queue with a new priority and
// metadata.
func (c *Client) Nack(queue, id string, priority int64, metadata map[string]string) error {
	body := map[string]any{"priority": priority, "metadata": metadata}
	return c.do(http.MethodPost, "/API/v1/queues/"+queue+"/messages/"+id+"/nack", body, nil)
}

func (c *Client) UpdatePriority(queue, id string, priority int64) error {
	body := map[string]any{"priority": priority}
	return c.do(http.MethodPut, "/API/v1/queues/"+queue+"/messages/"+id+"/priority", body, nil)
}

// Dequeue returns the next message. ok is false when the queue is empty.
func (c *Client) Dequeue(queue string, ack bool) (msg Message, ok bool, err error) {
	path := fmt.Sprintf("/API/v1/queues/%s/messages?ack=%t", queue, ack)
	err = c.do(http.MethodGet, path, nil, &msg)
	if se, isStatus := err.(*StatusError); isStatus && se.Code == http.StatusNotFound &&
		strings.Contains(se.Body, "queue is empty") {
		return Message{}, false, nil
	}
	return msg, err == nil, err
}

func (c *Client) Ack(queue, id string) error {
	return c.do(http.MethodPost, "/API/v1/queues/"+queue+"/messages/"+id+"/ack", nil, nil)
}

func (c *Client) QueueCounts(queue string) (QueueCounts, error) {
	var counts QueueCounts
	err := c.do(http.MethodGet, "/API/v1/queues/"+queue, nil, &counts)
	return counts, err
}

func (c *Client) do(method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &StatusError{Code: resp.StatusCode, Body: string(data)}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

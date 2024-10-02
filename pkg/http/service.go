package http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v2"

	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/prometheus/client_golang/prometheus"
)

// Service provides HTTP service.
type Service struct {
	api    huma.API
	router *fiber.App
	h      *Handler
	addr   string
}

type Node interface {
	Join(nodeID string, addr string) error
	PrometheusRegistry() prometheus.Registerer
	Leader() string
	IsLeader() bool
	GenerateID() uint64
	CreateQueue(queueType, queueName string) error
	DeleteQueue(queueName string) error
	GetQueues() []*queue.QueueInfo
	GetQueueInfo(queueName string) (*queue.QueueInfo, error)
	Enqueue(queueName string, group string, priority int64, content string) (*queue.Message, error)
	Dequeue(QueueName string, ack bool) (*queue.Message, error)
	Ack(QueueName string, id uint64) error
	GetByID(id uint64) (*queue.Message, error)
	UpdatePriority(queueName string, id uint64, priority int64) error
}

// New returns an uninitialized HTTP service.
func NewHttpService(addr string, node Node) *Service {

	router := fiber.New()
	api := humafiber.New(
		router, huma.DefaultConfig("DOQ a distributed priority queue servie", "1.0.0"),
	)

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	var httpClient = &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}

	proxy := NewProxy(httpClient)

	h := &Handler{
		node:  node,
		proxy: proxy,
	}
	h.ConfigureMiddleware(router)
	h.RegisterRoutes(api)

	return &Service{
		api:    api,
		router: router,
		h:      h,
		addr:   addr,
	}
}

func (h *Handler) ConfigureMiddleware(router *fiber.App) {
	router.Use(logger.New(logger.Config{
		TimeFormat: "2006-01-02T15:04:05.999Z0700",
		TimeZone:   "Local",
		Format:     "${time} [INFO] ${locals:requestid} ${method} ${path} ${status} ${latency} ${error}​\n",
	}))

	router.Use(healthcheck.New())
	router.Use(helmet.New())

	router.Use(requestid.New())

	prom := fiberprometheus.NewWithRegistry(
		h.node.PrometheusRegistry(), "doq", "", "", map[string]string{},
	)
	prom.RegisterAt(router, "/metrics")
	router.Use(prom.Middleware)

	router.Get("/service/metrics", monitor.New())
	router.Use(recover.New())
}

func (h *Handler) RegisterRoutes(api huma.API) {
	huma.Register(
		api,
		huma.Operation{
			OperationID: "raft-join",
			Method:      http.MethodPost,
			Path:        "/join",
			Summary:     "Join cluster",
			Description: "An endpoint for joining cluster used that by raft consensus protocol",
			Tags:        []string{"raft"},
		},
		h.Join,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "create-queue",
			Method:      http.MethodPost,
			Path:        "/API/v1/queues",
			Summary:     "Create a queue",
			Description: "Create a new queue",
			Tags:        []string{"Queues"},
		},
		h.CreateQueue,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "delete-queue",
			Method:      http.MethodDelete,
			Path:        "/API/v1/queues/:queue_name",
			Summary:     "Delete a queue",
			Description: "Delete a queue",
			Tags:        []string{"Queues"},
		},
		h.DeleteQueue,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "enqueue",
			Method:      http.MethodPost,
			Path:        "/API/v1/queues/:queue_name/messages",
			Summary:     "Enqueue a message",
			Description: "Put a message in the queue",
			Tags:        []string{"Messages"},
		},
		h.Enqueue,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "dequeue",
			Method:      http.MethodGet,
			Path:        "/API/v1/queues/:queue_name/messages",
			Summary:     "Dequeue a message",
			Description: "Get and remove the most prioritized message from the queue",
			Tags:        []string{"Messages"},
		},
		h.Dequeue,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "ack",
			Method:      http.MethodPost,
			Path:        "/API/v1/queues/:queue_name/messages/:id/ack",
			Summary:     "Acknowledge a message",
			Description: "Acknowledge the message",
			Tags:        []string{"Messages"},
		},
		h.Ack,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "update-priority",
			Method:      http.MethodPut,
			Path:        "/API/v1/queues/:queue_name/messages/:id/priority",
			Summary:     "Update the priority of a message",
			Description: "Update the priority of a message in the queue",
			Tags:        []string{"Messages"},
		},
		h.UpdatePriority,
	)

	huma.Register(
		api,
		huma.Operation{
			OperationID: "queues",
			Method:      http.MethodGet,
			Path:        "/API/v1/queues",
			Summary:     "List of queues",
			Description: "Get the list of queues",
			Tags:        []string{"Queues"},
		},
		h.Queues,
	)

	huma.Register(
		api,
		huma.Operation{
			OperationID: "queue-info",
			Method:      http.MethodGet,
			Path:        "/API/v1/queues/:queue_name/info",
			Summary:     "Info of a queue",
			Description: "Get the information of a queue including stats",
			Tags:        []string{"Queues"},
		},
		h.QueueInfo,
	)
}

// Start starts the service.
func (s *Service) Start() error {
	return s.router.Listen(fmt.Sprintf(":%s", s.addr))
}

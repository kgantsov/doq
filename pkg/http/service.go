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
	"github.com/kgantsov/doq/pkg/raft"
)

// Service provides HTTP service.
type Service struct {
	api    huma.API
	router *fiber.App
	h      *Handler
	addr   string
}

// New returns an uninitialized HTTP service.
func NewHttpService(addr string, node *raft.Node) *Service {

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

	h := &Handler{node: node, httpClient: httpClient}
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

	prometheus := fiberprometheus.New("doq")
	prometheus.RegisterAt(router, "/metrics")
	router.Use(prometheus.Middleware)

	router.Get("/service/metrics", monitor.New())
	router.Use(recover.New())
}

func (h *Handler) RegisterRoutes(api huma.API) {
	huma.Register(
		api,
		huma.Operation{
			OperationID: "enqueue",
			Method:      http.MethodPost,
			Path:        "/API/v1/queues/:queue_name",
			Summary:     "Enqueue a message",
			Description: "Put a message in the queue",
			Tags:        []string{"Queues"},
		},
		h.Enqueue,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "dequeue",
			Method:      http.MethodDelete,
			Path:        "/API/v1/queues/:queue_name",
			Summary:     "Dequeue a message",
			Description: "Get and remove the most prioritized message from the queue",
			Tags:        []string{"Queues"},
		},
		h.Dequeue,
	)
	huma.Register(
		api,
		huma.Operation{
			OperationID: "update-priority",
			Method:      http.MethodPut,
			Path:        "/API/v1/queues/:queue_name/:id/priority",
			Summary:     "Update the priority of a message",
			Description: "Update the priority of a message in the queue",
			Tags:        []string{"Queues"},
		},
		h.UpdatePriority,
	)
}

// Start starts the service.
func (s *Service) Start() error {
	return s.router.Listen(fmt.Sprintf(":%s", s.addr))
}

// Close closes the service.
func (s *Service) Close() {
	// s.e.Shutdown()
}

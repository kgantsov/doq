package logger

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// ZeroHCLLoggerMiddleware logs requests and responses using zerolog
func ZeroHCLLoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Log request details
		start := time.Now()
		// Process the request
		err := c.Next()

		// Log response details
		duration := time.Since(start)

		log.Info().
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Msgf("%s %s %d %s", c.Method(), c.Path(), c.Response().StatusCode(), duration)

		return err
	}
}

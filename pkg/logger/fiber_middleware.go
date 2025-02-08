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
		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("query", c.OriginalURL()).
			Msg("Incoming request")

		// Process the request
		err := c.Next()

		// Log response details
		duration := time.Since(start)
		log.Info().
			Int("status", c.Response().StatusCode()).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Dur("duration", duration).
			Msg("Request processed")

		return err
	}
}

package logger

import (
	"fmt"
	"io"

	standardlog "log"

	"github.com/hashicorp/go-hclog"
	"github.com/rs/zerolog/log"
)

type ZeroHCLLogger struct {
	name  string
	level hclog.Level
}

// NewZeroHCLLogger creates a new ZeroHCLLogger with the provided name and log level.
func NewZeroHCLLogger(name string, level hclog.Level) *ZeroHCLLogger {
	return &ZeroHCLLogger{
		name:  name,
		level: level,
	}
}

// Log logs a message at the given level with any provided key-value pairs.
func (z *ZeroHCLLogger) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		log.Trace().Fields(argsToMap(args...)).Msgf("[%s] %s", z.name, msg)
	case hclog.Debug:
		log.Debug().Fields(argsToMap(args...)).Msgf("[%s] %s", z.name, msg)
	case hclog.Info:
		log.Info().Fields(argsToMap(args...)).Msgf("[%s] %s", z.name, msg)
	case hclog.Warn:
		log.Warn().Fields(argsToMap(args...)).Msgf("[%s] %s", z.name, msg)
	case hclog.Error:
		log.Error().Fields(argsToMap(args...)).Msgf("[%s] %s", z.name, msg)
	}
}

// Trace logs a trace level message.
func (z *ZeroHCLLogger) Trace(msg string, args ...interface{}) {
	z.Log(hclog.Trace, msg, args...)
}

// Debug logs a debug level message.
func (z *ZeroHCLLogger) Debug(msg string, args ...interface{}) {
	z.Log(hclog.Debug, msg, args...)
}

// Info logs an info level message.
func (z *ZeroHCLLogger) Info(msg string, args ...interface{}) {
	z.Log(hclog.Info, msg, args...)
}

// Warn logs a warning level message.
func (z *ZeroHCLLogger) Warn(msg string, args ...interface{}) {
	z.Log(hclog.Warn, msg, args...)
}

// Error logs an error level message.
func (z *ZeroHCLLogger) Error(msg string, args ...interface{}) {
	z.Log(hclog.Error, msg, args...)
}

// IsTrace indicates if the logger is at trace level.
func (z *ZeroHCLLogger) IsTrace() bool {
	return z.level <= hclog.Trace
}

// IsDebug indicates if the logger is at debug level.
func (z *ZeroHCLLogger) IsDebug() bool {
	return z.level <= hclog.Debug
}

// IsInfo indicates if the logger is at info level.
func (z *ZeroHCLLogger) IsInfo() bool {
	return z.level <= hclog.Info
}

// IsWarn indicates if the logger is at warning level.
func (z *ZeroHCLLogger) IsWarn() bool {
	return z.level <= hclog.Warn
}

// IsError indicates if the logger is at error level.
func (z *ZeroHCLLogger) IsError() bool {
	return z.level <= hclog.Error
}

// With returns a new logger that includes the provided key-value pairs in each log line.
func (z *ZeroHCLLogger) With(args ...interface{}) hclog.Logger {
	return NewZeroHCLLogger(z.name, z.level)
}

func (z *ZeroHCLLogger) Name() string {
	return z.name
}

// Named returns a new logger with the provided name.
func (z *ZeroHCLLogger) Named(name string) hclog.Logger {
	return NewZeroHCLLogger(fmt.Sprintf("%s.%s", z.name, name), z.level)
}

// ResetNamed returns a new logger with the provided name, replacing the existing name.
func (z *ZeroHCLLogger) ResetNamed(name string) hclog.Logger {
	return NewZeroHCLLogger(name, z.level)
}

func (z *ZeroHCLLogger) ImpliedArgs() []interface{} {
	return nil
}

// SetLevel sets the logger's log level.
func (z *ZeroHCLLogger) SetLevel(level hclog.Level) {
	z.level = level
}

func (z *ZeroHCLLogger) GetLevel() hclog.Level {
	return z.level
}

// StandardLogger returns a logger compatible with the standard library's log.Logger.
func (z *ZeroHCLLogger) StandardLogger(opts *hclog.StandardLoggerOptions) *standardlog.Logger {
	return nil // Not implemented
}

func (z *ZeroHCLLogger) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return nil // Not implemented
}

// Helper function to convert key-value pairs to a map for zerolog.
func argsToMap(args ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok || i+1 >= len(args) {
			continue
		}
		fields[key] = args[i+1]
	}
	return fields
}

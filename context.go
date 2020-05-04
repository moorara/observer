package observer

import (
	"context"

	"go.uber.org/zap"
)

type contextKey string

const (
	uuidContextKey   = contextKey("UUID")
	loggerContextKey = contextKey("Logger")
)

// ContextWithUUID creates a new context with a uuid.
func ContextWithUUID(ctx context.Context, uuid string) context.Context {
	return context.WithValue(ctx, uuidContextKey, uuid)
}

// UUIDFromContext retrieves a uuid from a context.
func UUIDFromContext(ctx context.Context) (string, bool) {
	uuid, ok := ctx.Value(uuidContextKey).(string)
	return uuid, ok
}

// ContextWithLogger returns a new context that holds a reference to a logger.
func ContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// LoggerFromContext returns a logger set on a context.
// If no logger found on the context, the singleton logger will be returned!
func LoggerFromContext(ctx context.Context) *zap.Logger {
	val := ctx.Value(loggerContextKey)
	if logger, ok := val.(*zap.Logger); ok {
		return logger
	}

	// Return the singleton logger as the default
	return singleton.logger
}

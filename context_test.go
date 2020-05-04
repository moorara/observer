package observer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestContextWithUUID(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		requestID string
	}{
		{
			"OK",
			context.Background(),
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ContextWithUUID(tc.ctx, tc.requestID)

			assert.Equal(t, tc.requestID, ctx.Value(uuidContextKey))
		})
	}
}

func TestUUIDFromContext(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		expectedOK   bool
		expectedUUID string
	}{
		{
			"WithoutUUID",
			context.Background(),
			false,
			"",
		},
		{
			"WithUUID",
			context.WithValue(context.Background(), uuidContextKey, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			true,
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requestID, ok := UUIDFromContext(tc.ctx)

			assert.Equal(t, tc.expectedOK, ok)
			assert.Equal(t, tc.expectedUUID, requestID)
		})
	}
}

func TestContextWithLogger(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		logger *zap.Logger
	}{
		{
			name:   "OK",
			ctx:    context.Background(),
			logger: zap.NewNop(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ContextWithLogger(tc.ctx, tc.logger)

			logger := ctx.Value(loggerContextKey)
			assert.Equal(t, tc.logger, logger)
		})
	}
}

func TestLoggerFromContext(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		ctx            context.Context
		expectedLogger *zap.Logger
	}{
		{
			name:           "WithoutObserver",
			ctx:            context.Background(),
			expectedLogger: singleton.logger,
		},
		{
			name:           "WithObserver",
			ctx:            context.WithValue(context.Background(), loggerContextKey, logger),
			expectedLogger: logger,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := LoggerFromContext(tc.ctx)

			assert.Equal(t, tc.expectedLogger, logger)
		})
	}
}

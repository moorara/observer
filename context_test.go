package observer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestContextWithObserver(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		observer *Observer
	}{
		{
			name:     "OK",
			ctx:      context.Background(),
			observer: &Observer{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ContextWithObserver(tc.ctx, tc.observer)

			observer := ctx.Value(observerContextKey)
			assert.Equal(t, tc.observer, observer)
		})
	}
}

func TestFromContext(t *testing.T) {
	observer := &Observer{}

	tests := []struct {
		name             string
		ctx              context.Context
		expectedObserver *Observer
	}{
		{
			name:             "WithoutObserver",
			ctx:              context.Background(),
			expectedObserver: singleton,
		},
		{
			name:             "WithObserver",
			ctx:              context.WithValue(context.Background(), observerContextKey, observer),
			expectedObserver: observer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			observer := FromContext(tc.ctx)

			assert.Equal(t, tc.expectedObserver, observer)
		})
	}
}

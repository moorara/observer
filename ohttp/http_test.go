package ohttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type mockObserver struct {
	name   string
	logger *zap.Logger
	meter  metric.Meter
	tracer trace.Tracer
}

func newMockObserver() *mockObserver {
	return &mockObserver{
		name:   "test",
		logger: zap.NewNop(),
		meter:  new(metric.NoopMeterProvider).Meter(""),
		tracer: trace.NewNoopTracerProvider().Tracer(""),
	}
}

func (m *mockObserver) End(ctx context.Context) error {
	return nil
}

func (m *mockObserver) Name() string {
	return m.name
}

func (m *mockObserver) Logger() *zap.Logger {
	return m.logger
}

func (m *mockObserver) SetLogLevel(level zapcore.Level) {
	// Noop
}

func (m *mockObserver) GetLogLevel() zapcore.Level {
	return zapcore.Level(99)
}

func (m *mockObserver) Meter() metric.Meter {
	return m.meter
}

func (m *mockObserver) Tracer() trace.Tracer {
	return m.tracer
}

func (m *mockObserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Noop
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		statusClass string
	}{
		{"200", 101, "1xx"},
		{"200", 200, "2xx"},
		{"201", 201, "2xx"},
		{"201", 202, "2xx"},
		{"300", 300, "3xx"},
		{"400", 400, "4xx"},
		{"400", 403, "4xx"},
		{"404", 404, "4xx"},
		{"400", 409, "4xx"},
		{"500", 500, "5xx"},
		{"500", 501, "5xx"},
		{"502", 502, "5xx"},
		{"500", 503, "5xx"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}

			middleware := func(w http.ResponseWriter, r *http.Request) {
				rw := newResponseWriter(w)
				handler(rw, r)

				assert.Equal(t, tc.statusCode, rw.StatusCode)
				assert.Equal(t, tc.statusClass, rw.StatusClass)
			}

			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			middleware(w, r)
		})
	}
}

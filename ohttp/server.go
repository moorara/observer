package ohttp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/moorara/observer"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/unit"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"go.uber.org/zap"
)

// MiddlewareOptions are optional configurations for creating a server middleware.
type MiddlewareOptions struct {
	// Whether or not to log successful requests at debug level (the default is at info level).
	LogAtDebugLevel bool
}

// Server-side instruments for metrics.
type serverInstruments struct {
	reqCounter  metric.Int64Counter
	reqDuration metric.Int64Measure
}

func newServerInstruments(meter metric.Meter) *serverInstruments {
	mustMeter := metric.Must(meter)

	return &serverInstruments{
		reqCounter: mustMeter.NewInt64Counter(
			"incoming_http_requests_total",
			metric.WithDescription("The total number of incoming requests (server-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithLibraryName(libraryName),
		),
		reqDuration: mustMeter.NewInt64Measure(
			"incoming_http_requests_duration",
			metric.WithDescription("The duration of incoming requests in milliseconds (server-side)"),
			metric.WithUnit(unit.Milliseconds),
			metric.WithLibraryName(libraryName),
		),
	}
}

// Middleware creates observable http handlers with logging, metrics, and tracing.
type Middleware struct {
	opts        MiddlewareOptions
	observer    *observer.Observer
	instruments *serverInstruments
}

// NewMiddleware creates a new observable http middleware
func NewMiddleware(observer *observer.Observer, opts MiddlewareOptions) *Middleware {
	instruments := newServerInstruments(observer.Meter())

	return &Middleware{
		opts:        opts,
		observer:    observer,
		instruments: instruments,
	}
}

// GetHandlerFunc wraps an existing http handler function and returns a new observable handler function.
// This can be used for making http handlers observable via logging, metrics, tracing, etc.
func (m *Middleware) GetHandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		kind := "server"
		protocol := r.Proto
		method := r.Method
		url := r.URL.Path
		ctx := r.Context()

		// Make sure the request has a UUID
		requestUUID := r.Header.Get(requestUUIDHeader)
		if requestUUID == "" {
			requestUUID = uuid.New().String()
			r.Header.Set(requestUUIDHeader, requestUUID)
		}

		// Extract correlation context entries and parent span context if any
		// The first return value is a list of http attributes for the request
		_, entries, spanContext := httptrace.Extract(ctx, r)

		// Create a new correlation context with the extracted entries and new ones
		entries = append(entries,
			key.String("req.uuid", requestUUID),
		)
		ctx = correlation.NewContext(ctx, entries...)

		// Start a new span
		ctx = trace.ContextWithRemoteSpanContext(ctx, spanContext)
		ctx, span := m.observer.Tracer().Start(ctx,
			"server-request",
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Create a contextualized logger
		logger := m.observer.Logger().With(
			zap.String("req.uuid", requestUUID),
			zap.String("req.kind", kind),
			zap.String("req.protocol", protocol),
			zap.String("req.method", method),
			zap.String("req.url", url),
		)

		// Augment the request context
		ctx = observer.ContextWithUUID(ctx, requestUUID)
		ctx = observer.ContextWithLogger(ctx, logger)
		req := r.WithContext(ctx)

		// Create a wrapped response writer, so we can know about the response
		rw := newResponseWriter(w)

		// Call the next http handler function
		next(rw, req)

		duration := time.Since(startTime).Milliseconds()
		statusCode := rw.StatusCode
		statusClass := rw.StatusClass

		// Report metrics
		m.observer.Meter().RecordBatch(ctx,
			[]core.KeyValue{
				key.String("protocol", protocol),
				key.String("method", method),
				key.String("url", url),
				key.Int("status_code", statusCode),
				key.String("status_class", statusClass),
			},
			m.instruments.reqCounter.Measurement(1),
			m.instruments.reqDuration.Measurement(duration),
		)

		// Report logs
		message := fmt.Sprintf("%s %s %d %dms", method, url, statusCode, duration)
		fields := []zap.Field{
			zap.Int("resp.statusCode", statusCode),
			zap.String("resp.statusClass", statusClass),
			zap.Int64("resp.duration", duration),
		}

		// Determine the log level based on the result
		switch {
		case statusCode >= 500:
			logger.Error(message, fields...)
		case statusCode >= 400:
			logger.Warn(message, fields...)
		case statusCode >= 100:
			fallthrough
		default:
			if m.opts.LogAtDebugLevel {
				logger.Debug(message, fields...)
			} else {
				logger.Info(message, fields...)
			}
		}

		// Report the span
		span.SetAttributes(
			key.String("protocol", protocol),
			key.String("method", method),
			key.String("url", url),
			key.Int("status_code", statusCode),
		)
	}
}

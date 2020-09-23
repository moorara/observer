package ohttp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/moorara/observer"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/unit"
	"go.opentelemetry.io/otel/label"
	"go.uber.org/zap"
)

// Server-side instruments for metrics.
type serverInstruments struct {
	reqCounter   metric.Int64Counter
	reqGauge     metric.Int64UpDownCounter
	reqDuration  metric.Int64ValueRecorder
	panicCounter metric.Int64Counter
}

func newServerInstruments(meter metric.Meter) *serverInstruments {
	mm := metric.Must(meter)

	return &serverInstruments{
		reqCounter: mm.NewInt64Counter(
			"incoming_http_requests_total",
			metric.WithDescription("The total number of incoming http requests (server-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithInstrumentationName(libraryName),
		),
		reqGauge: mm.NewInt64UpDownCounter(
			"incoming_http_requests_active",
			metric.WithDescription("The number of in-flight incoming http requests (server-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithInstrumentationName(libraryName),
		),
		reqDuration: mm.NewInt64ValueRecorder(
			"incoming_http_requests_duration",
			metric.WithDescription("The duration of incoming http requests in milliseconds (server-side)"),
			metric.WithUnit(unit.Milliseconds),
			metric.WithInstrumentationName(libraryName),
		),
		panicCounter: mm.NewInt64Counter(
			"handler_panics_total",
			metric.WithDescription("The total number of panics that happened in http handlers (server-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithInstrumentationName(libraryName),
		),
	}
}

// Middleware creates observable http handlers with logging, metrics, and tracing.
type Middleware struct {
	opts        Options
	observer    observer.Observer
	instruments *serverInstruments
}

// NewMiddleware creates a new http middleware for observability.
func NewMiddleware(observer observer.Observer, opts Options) *Middleware {
	opts = opts.withDefaults()
	instruments := newServerInstruments(observer.Meter())

	return &Middleware{
		opts:        opts,
		observer:    observer,
		instruments: instruments,
	}
}

func (m *Middleware) callHandlerFunc(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("critical error: %v", r)
			m.observer.Logger().Error("Panic occurred.", zap.Error(err))
			m.instruments.panicCounter.Add(context.Background(), 1)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	handler(w, r)
}

// Wrap wraps an existing http handler function and returns a new observable handler function.
// This can be used for making http handlers observable via logging, metrics, tracing, etc.
// It also observes and recovers panics that happened inside the inner http handler.
func (m *Middleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		ctx := r.Context()
		kind := "server"
		method := r.Method
		url := r.URL.Path
		route := m.opts.IDRegexp.ReplaceAllString(url, ":id")

		// Increase the number of in-flight requests
		m.instruments.reqGauge.Add(ctx, 1,
			label.String("method", method),
			label.String("route", route),
		)

		// Make sure we decrease the number of in-flight requests
		defer m.instruments.reqGauge.Add(ctx, -1,
			label.String("method", method),
			label.String("route", route),
		)

		// Make sure the request has a UUID
		requestUUID := r.Header.Get(requestUUIDHeader)
		if requestUUID == "" {
			requestUUID = uuid.New().String()
			r.Header.Set(requestUUIDHeader, requestUUID)
		}

		// Get the name of client for the request if any
		clientName := r.Header.Get(clientNameHeader)

		// Propagate request metadata by adding them to outgoing http response headers
		w.Header().Set(requestUUIDHeader, requestUUID)
		w.Header().Set(clientNameHeader, clientName)

		// Extract correlation context from the http headers
		ctx = propagation.ExtractHTTP(ctx, global.Propagators(), r.Header)

		// Get span context and propagated key-values
		// spanContext := trace.RemoteSpanContextFromContext(ctx)
		var keyvalues []label.KeyValue
		correlation.MapFromContext(ctx).Foreach(func(kv label.KeyValue) bool {
			keyvalues = append(keyvalues, kv)
			return true
		})

		// Create a new correlation context
		ctx = correlation.NewContext(ctx,
			label.String("req.uuid", requestUUID),
		)

		// Start a new span
		ctx, span := m.observer.Tracer().Start(ctx,
			"http-server-request",
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Create a contextualized logger
		contextFields := []zap.Field{
			zap.String("req.uuid", requestUUID),
			zap.String("req.kind", kind),
			zap.String("req.method", method),
			zap.String("req.url", url),
			zap.String("req.route", route),
			zap.String("traceId", span.SpanContext().TraceID.String()),
			zap.String("spanId", span.SpanContext().SpanID.String()),
		}
		if clientName != "" {
			contextFields = append(contextFields, zap.String("client.name", clientName))
		}
		logger := m.observer.Logger().With(contextFields...)

		// Augment the request context
		ctx = observer.ContextWithUUID(ctx, requestUUID)
		ctx = observer.ContextWithLogger(ctx, logger)
		req := r.WithContext(ctx)

		// Create a wrapped response writer, so we can know about the response
		rw := newResponseWriter(w)

		// Call http handler
		span.AddEvent(ctx, "calling http handler")
		m.callHandlerFunc(next, rw, req)

		duration := time.Since(startTime).Milliseconds()
		statusCode := rw.StatusCode
		statusClass := rw.StatusClass

		// Report metrics
		m.observer.Meter().RecordBatch(ctx,
			[]label.KeyValue{
				label.String("method", method),
				label.String("route", route),
				label.Int("status_code", statusCode),
				label.String("status_class", statusClass),
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
			if m.opts.LogInDebugLevel {
				logger.Debug(message, fields...)
			} else {
				logger.Info(message, fields...)
			}
		}

		// Report the span
		span.SetAttributes(
			label.String("method", method),
			label.String("url", url),
			label.String("route", route),
			label.Int("status_code", statusCode),
		)
	}
}

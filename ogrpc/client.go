package ogrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/moorara/observer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/unit"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Client-side instruments for metrics.
type clientInstruments struct {
	reqCounter  metric.Int64Counter
	reqGauge    metric.Int64UpDownCounter
	reqDuration metric.Int64ValueRecorder
}

func newClientInstruments(meter metric.Meter) *clientInstruments {
	mm := metric.Must(meter)

	return &clientInstruments{
		reqCounter: mm.NewInt64Counter(
			"outgoing_grpc_requests_total",
			metric.WithDescription("The total number of outgoing grpc requests (client-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithInstrumentationName(libraryName),
		),
		reqGauge: mm.NewInt64UpDownCounter(
			"outgoing_grpc_requests_active",
			metric.WithDescription("The number of in-flight outgoing grpc requests (client-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithInstrumentationName(libraryName),
		),
		reqDuration: mm.NewInt64ValueRecorder(
			"outgoing_grpc_requests_duration",
			metric.WithDescription("The duration of outgoing grpc requests in seconds (client-side)"),
			metric.WithUnit(unit.Milliseconds),
			metric.WithInstrumentationName(libraryName),
		),
	}
}

// ClientInterceptor creates interceptors with logging, metrics, and tracing for grpc clients.
type ClientInterceptor struct {
	opts        Options
	observer    observer.Observer
	instruments *clientInstruments
}

// NewClientInterceptor creates a new server interceptor for observability.
func NewClientInterceptor(observer observer.Observer, opts Options) *ClientInterceptor {
	opts = opts.withDefaults()
	instruments := newClientInstruments(observer.Meter())

	return &ClientInterceptor{
		opts:        opts,
		observer:    observer,
		instruments: instruments,
	}
}

// DialOptions return grpc dial options for unary and stream interceptors.
// This can be used for making gRPC method calls observable via logging, metrics, tracing, etc.
func (i *ClientInterceptor) DialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(i.unaryInterceptor),
		grpc.WithStreamInterceptor(i.streamInterceptor),
	}
}

func (i *ClientInterceptor) unaryInterceptor(ctx context.Context, fullMethod string, req, res interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	startTime := time.Now()
	kind := "client"
	stream := false

	// Get the package, service, and method name for the request
	e, ok := parseEndpoint(fullMethod)
	if !ok {
		return invoker(ctx, fullMethod, req, res, cc, opts...)
	}

	// Check excluded methods
	for _, m := range i.opts.ExcludedMethods {
		if e.Method == m {
			return invoker(ctx, fullMethod, req, res, cc, opts...)
		}
	}

	// Increase the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, 1,
		label.String("package", e.Package),
		label.String("service", e.Service),
		label.String("method", e.Method),
		label.Bool("stream", stream),
	)

	// Make sure we decrease the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, -1,
		label.String("package", e.Package),
		label.String("service", e.Service),
		label.String("method", e.Method),
		label.Bool("stream", stream),
	)

	// Make sure the request has a UUID
	requestUUID, ok := observer.UUIDFromContext(ctx)
	if !ok || requestUUID == "" {
		requestUUID = uuid.New().String()
	}

	// Get grpc request metadata
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.New(nil)
	}

	// Propagate request metadata by adding them to outgoing grpc request metadata
	md.Set(requestUUIDKey, requestUUID)
	md.Set(clientNameKey, i.observer.Name())
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Create a new context
	ctx = baggage.ContextWithValues(ctx,
		label.String("req.uuid", requestUUID),
		label.String("client.name", i.observer.Name()),
	)

	// Start a new span
	ctx, span := i.observer.Tracer().Start(ctx,
		fmt.Sprintf("%s (client unary)", e.Method),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// Inject the context and the span context into the grpc metadata
	otel.GetTextMapPropagator().Inject(ctx, &metadataTextMapCarrier{md: &md})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Call gRPC method invoker
	span.AddEvent("invoking grpc method")
	err := invoker(ctx, fullMethod, req, res, cc, opts...)

	duration := time.Since(startTime).Milliseconds()
	success := err == nil

	// Report metrics
	i.observer.Meter().RecordBatch(ctx,
		[]label.KeyValue{
			label.String("package", e.Package),
			label.String("service", e.Service),
			label.String("method", e.Method),
			label.Bool("stream", stream),
			label.Bool("success", success),
		},
		i.instruments.reqCounter.Measurement(1),
		i.instruments.reqDuration.Measurement(duration),
	)

	// Report logs
	logger := i.observer.Logger()
	message := fmt.Sprintf("%s %s %dms", kind, e, duration)
	fields := []zap.Field{
		zap.String("req.uuid", requestUUID),
		zap.String("req.kind", kind),
		zap.String("req.package", e.Package),
		zap.String("req.service", e.Service),
		zap.String("req.method", e.Method),
		zap.Bool("req.stream", stream),
		zap.Bool("resp.success", success),
		zap.Int64("resp.duration", duration),
		zap.String("traceId", span.SpanContext().TraceID.String()),
		zap.String("spanId", span.SpanContext().SpanID.String()),
	}
	if err != nil {
		fields = append(fields, zap.String("grpc.error", err.Error()))
	}

	// Determine the log level based on the result
	if success {
		if i.opts.LogInDebugLevel {
			logger.Debug(message, fields...)
		} else {
			logger.Info(message, fields...)
		}
	} else {
		logger.Error(message, fields...)
	}

	// Report the span
	span.SetAttributes(
		label.String("package", e.Package),
		label.String("service", e.Service),
		label.String("method", e.Method),
		label.Bool("stream", stream),
		label.Bool("success", success),
	)
	if err != nil {
		code := codes.Code(status.Code(err))
		span.SetStatus(code, err.Error())
	}

	return err
}

func (i *ClientInterceptor) streamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, fullMethod string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	startTime := time.Now()
	kind := "client"
	stream := true

	// Get the package, service, and method name for the request
	e, ok := parseEndpoint(fullMethod)
	if !ok {
		return streamer(ctx, desc, cc, fullMethod, opts...)
	}

	// Check excluded methods
	for _, m := range i.opts.ExcludedMethods {
		if e.Method == m {
			return streamer(ctx, desc, cc, fullMethod, opts...)
		}
	}

	// Increase the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, 1,
		label.String("package", e.Package),
		label.String("service", e.Service),
		label.String("method", e.Method),
		label.Bool("stream", stream),
	)

	// Make sure we decrease the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, -1,
		label.String("package", e.Package),
		label.String("service", e.Service),
		label.String("method", e.Method),
		label.Bool("stream", stream),
	)

	// Make sure the request has a UUID
	requestUUID, ok := observer.UUIDFromContext(ctx)
	if !ok || requestUUID == "" {
		requestUUID = uuid.New().String()
	}

	// Get grpc request metadata
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.New(nil)
	}

	// Propagate request metadata by adding them to outgoing grpc request metadata
	md.Set(requestUUIDKey, requestUUID)
	md.Set(clientNameKey, i.observer.Name())
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Create a new context
	ctx = baggage.ContextWithValues(ctx,
		label.String("req.uuid", requestUUID),
		label.String("client.name", i.observer.Name()),
	)

	// Start a new span
	ctx, span := i.observer.Tracer().Start(ctx,
		fmt.Sprintf("%s (client stream)", e.Method),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// Inject the context and the span context into the grpc metadata
	otel.GetTextMapPropagator().Inject(ctx, &metadataTextMapCarrier{md: &md})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Call gRPC method streamer
	span.AddEvent("invoking grpc method")
	cs, err := streamer(ctx, desc, cc, fullMethod, opts...)

	duration := time.Since(startTime).Milliseconds()
	success := err == nil

	// Report metrics
	i.observer.Meter().RecordBatch(ctx,
		[]label.KeyValue{
			label.String("package", e.Package),
			label.String("service", e.Service),
			label.String("method", e.Method),
			label.Bool("stream", stream),
			label.Bool("success", success),
		},
		i.instruments.reqCounter.Measurement(1),
		i.instruments.reqDuration.Measurement(duration),
	)

	// Report logs
	logger := i.observer.Logger()
	message := fmt.Sprintf("%s %s %dms", kind, e, duration)
	fields := []zap.Field{
		zap.String("req.uuid", requestUUID),
		zap.String("req.kind", kind),
		zap.String("req.package", e.Package),
		zap.String("req.service", e.Service),
		zap.String("req.method", e.Method),
		zap.Bool("req.stream", stream),
		zap.Bool("resp.success", success),
		zap.Int64("resp.duration", duration),
		zap.String("traceId", span.SpanContext().TraceID.String()),
		zap.String("spanId", span.SpanContext().SpanID.String()),
	}
	if err != nil {
		fields = append(fields, zap.String("grpc.error", err.Error()))
	}

	// Determine the log level based on the result
	if success {
		if i.opts.LogInDebugLevel {
			logger.Debug(message, fields...)
		} else {
			logger.Info(message, fields...)
		}
	} else {
		logger.Error(message, fields...)
	}

	// Report the span
	span.SetAttributes(
		label.String("package", e.Package),
		label.String("service", e.Service),
		label.String("method", e.Method),
		label.Bool("stream", stream),
		label.Bool("success", success),
	)
	if err != nil {
		code := codes.Code(status.Code(err))
		span.SetStatus(code, err.Error())
	}

	return cs, err
}

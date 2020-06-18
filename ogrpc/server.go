package ogrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/moorara/observer"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/unit"
	"go.opentelemetry.io/otel/plugin/grpctrace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server-side instruments for metrics.
type serverInstruments struct {
	reqCounter  metric.Int64Counter
	reqGauge    metric.Int64UpDownCounter
	reqDuration metric.Int64ValueRecorder
}

func newServerInstruments(meter metric.Meter) *serverInstruments {
	mm := metric.Must(meter)

	return &serverInstruments{
		reqCounter: mm.NewInt64Counter(
			"incoming_grpc_requests_total",
			metric.WithDescription("The total number of incoming grpc requests (server-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithLibraryName(libraryName),
		),
		reqGauge: mm.NewInt64UpDownCounter(
			"incoming_grpc_requests_active",
			metric.WithDescription("The number of in-flight incoming grpc requests (server-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithLibraryName(libraryName),
		),
		reqDuration: mm.NewInt64ValueRecorder(
			"incoming_grpc_requests_duration",
			metric.WithDescription("The duration of incoming grpc requests in milliseconds (server-side)"),
			metric.WithUnit(unit.Milliseconds),
			metric.WithLibraryName(libraryName),
		),
	}
}

// ServerInterceptor creates interceptors with logging, metrics, and tracing for grpc servers.
type ServerInterceptor struct {
	opts        Options
	observer    observer.Observer
	instruments *serverInstruments
}

// NewServerInterceptor creates a new server interceptor for observability.
func NewServerInterceptor(observer observer.Observer, opts Options) *ServerInterceptor {
	opts = opts.withDefaults()
	instruments := newServerInstruments(observer.Meter())

	return &ServerInterceptor{
		opts:        opts,
		observer:    observer,
		instruments: instruments,
	}
}

// ServerOptions return grpc server options for unary and stream interceptors.
func (i *ServerInterceptor) ServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(i.unaryInterceptor),
		grpc.StreamInterceptor(i.streamInterceptor),
	}
}

func (i *ServerInterceptor) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()

	kind := "server"
	stream := false

	// Get the package, service, and method name for the request
	e, ok := parseEndpoint(info.FullMethod)
	if !ok {
		return handler(ctx, req)
	}

	// Check excluded methods
	for _, m := range i.opts.ExcludedMethods {
		if e.Method == m {
			return handler(ctx, req)
		}
	}

	// Increase the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, 1,
		kv.String("package", e.Package),
		kv.String("service", e.Service),
		kv.String("method", e.Method),
		kv.Bool("stream", stream),
	)

	// Get grpc request metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.New(nil)
	}

	// Make sure the request has a UUID
	var requestUUID string
	if vals := md.Get(requestUUIDKey); len(vals) > 0 {
		requestUUID = vals[0]
	}
	if requestUUID == "" {
		requestUUID = uuid.New().String()
		md.Set(requestUUIDKey, requestUUID)
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	// Get the name of client for the request if any
	var clientName string
	if vals := md.Get(clientNameKey); len(vals) > 0 {
		clientName = vals[0]
	}

	// Extract correlation context entries and parent span context if any
	entries, spanContext := grpctrace.Extract(ctx, &md)

	// Create a new correlation context with the extracted entries and new ones
	entries = append(entries,
		kv.String("req.uuid", requestUUID),
	)
	ctx = correlation.NewContext(ctx, entries...)

	// Start a new span
	ctx = trace.ContextWithRemoteSpanContext(ctx, spanContext)
	ctx, span := i.observer.Tracer().Start(ctx,
		fmt.Sprintf("%s (server unary)", e.Method),
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	// Create a contextualized logger
	contextFields := []zap.Field{
		zap.String("req.uuid", requestUUID),
		zap.String("req.kind", kind),
		zap.String("req.package", e.Package),
		zap.String("req.service", e.Service),
		zap.String("req.method", e.Method),
		zap.Bool("req.stream", stream),
	}
	if clientName != "" {
		contextFields = append(contextFields, zap.String("client.name", clientName))
	}
	logger := i.observer.Logger().With(contextFields...)

	// Augment the request context
	ctx = observer.ContextWithUUID(ctx, requestUUID)
	ctx = observer.ContextWithLogger(ctx, logger)

	// Call gRPC method handler
	span.AddEvent(ctx, "calling grpc method handler")
	res, err := handler(ctx, req)

	duration := time.Since(startTime).Milliseconds()
	success := err == nil

	// Report metrics
	i.observer.Meter().RecordBatch(ctx,
		[]kv.KeyValue{
			kv.String("package", e.Package),
			kv.String("service", e.Service),
			kv.String("method", e.Method),
			kv.Bool("stream", stream),
			kv.Bool("success", success),
		},
		i.instruments.reqCounter.Measurement(1),
		i.instruments.reqDuration.Measurement(duration),
	)

	// Report logs
	message := fmt.Sprintf("%s %s %dms", kind, e, duration)
	fields := []zap.Field{
		zap.Bool("resp.success", success),
		zap.Int64("resp.duration", duration),
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

	// Decrease the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, -1,
		kv.String("package", e.Package),
		kv.String("service", e.Service),
		kv.String("method", e.Method),
		kv.Bool("stream", stream),
	)

	// Report the span
	span.SetAttributes(
		kv.String("package", e.Package),
		kv.String("service", e.Service),
		kv.String("method", e.Method),
		kv.Bool("stream", stream),
		kv.Bool("success", success),
	)
	if err != nil {
		span.SetStatus(status.Code(err), err.Error())
	}

	return res, err
}

func (i *ServerInterceptor) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	startTime := time.Now()

	ctx := ss.Context()
	kind := "server"
	stream := true

	// Get the package, service, and method name for the request
	e, ok := parseEndpoint(info.FullMethod)
	if !ok {
		return handler(srv, ss)
	}

	// Check excluded methods
	for _, m := range i.opts.ExcludedMethods {
		if e.Method == m {
			return handler(srv, ss)
		}
	}

	// Increase the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, 1,
		kv.String("package", e.Package),
		kv.String("service", e.Service),
		kv.String("method", e.Method),
		kv.Bool("stream", stream),
	)

	// Get grpc request metadata (an incoming grpc request context is guaranteed to have metadata)
	md, _ := metadata.FromIncomingContext(ctx)
	md = md.Copy()

	// Make sure the request has a UUID
	var requestUUID string
	if vals := md.Get(requestUUIDKey); len(vals) > 0 {
		requestUUID = vals[0]
	}
	if requestUUID == "" {
		requestUUID = uuid.New().String()
		md.Set(requestUUIDKey, requestUUID)
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	// Get the name of client for the request if any
	var clientName string
	if vals := md.Get(clientNameKey); len(vals) > 0 {
		clientName = vals[0]
	}

	// Extract correlation context entries and parent span context if any
	entries, spanContext := grpctrace.Extract(ctx, &md)

	// Create a new correlation context with the extracted entries and new ones
	entries = append(entries,
		kv.String("req.uuid", requestUUID),
	)
	ctx = correlation.NewContext(ctx, entries...)

	// Start a new span
	ctx = trace.ContextWithRemoteSpanContext(ctx, spanContext)
	ctx, span := i.observer.Tracer().Start(ctx,
		fmt.Sprintf("%s (server stream)", e.Method),
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	// Create a contextualized logger
	contextFields := []zap.Field{
		zap.String("req.uuid", requestUUID),
		zap.String("req.kind", kind),
		zap.String("req.package", e.Package),
		zap.String("req.service", e.Service),
		zap.String("req.method", e.Method),
		zap.Bool("req.stream", stream),
	}
	if clientName != "" {
		contextFields = append(contextFields, zap.String("client.name", clientName))
	}
	logger := i.observer.Logger().With(contextFields...)

	// Augment the request context
	ctx = observer.ContextWithUUID(ctx, requestUUID)
	ctx = observer.ContextWithLogger(ctx, logger)
	ss = ServerStreamWithContext(ctx, ss)

	// Call gRPC method handler
	span.AddEvent(ctx, "calling grpc method handler")
	err := handler(srv, ss)

	duration := time.Since(startTime).Milliseconds()
	success := err == nil

	// Report metrics
	i.observer.Meter().RecordBatch(ctx,
		[]kv.KeyValue{
			kv.String("package", e.Package),
			kv.String("service", e.Service),
			kv.String("method", e.Method),
			kv.Bool("stream", stream),
			kv.Bool("success", success),
		},
		i.instruments.reqCounter.Measurement(1),
		i.instruments.reqDuration.Measurement(duration),
	)

	// Report logs
	message := fmt.Sprintf("%s %s %dms", kind, e, duration)
	fields := []zap.Field{
		zap.Bool("resp.success", success),
		zap.Int64("resp.duration", duration),
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

	// Decrease the number of in-flight requests
	i.instruments.reqGauge.Add(ctx, -1,
		kv.String("package", e.Package),
		kv.String("service", e.Service),
		kv.String("method", e.Method),
		kv.Bool("stream", stream),
	)

	// Report the span
	span.SetAttributes(
		kv.String("package", e.Package),
		kv.String("service", e.Service),
		kv.String("method", e.Method),
		kv.Bool("stream", stream),
		kv.Bool("success", success),
	)
	if err != nil {
		span.SetStatus(status.Code(err), err.Error())
	}

	return err
}

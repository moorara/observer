// Package observer can be used for implementing observability using OpenTelemetry API.
// It aims to unify three pillars of observability in one single package that is easy-to-use and hard-to-misuse.
//
// An Observer encompasses a logger, a meter, and a tracer.
// It offers a single unified developer experience for enabling observability.
package observer

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/controller/pull"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	promexporter "go.opentelemetry.io/otel/exporters/metric/prometheus"
	otlpexporter "go.opentelemetry.io/otel/exporters/otlp"
	jaegerexporter "go.opentelemetry.io/otel/exporters/trace/jaeger"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type shutdownFunc func(context.Context) error

// configs is used for configuring and creating an observer.
type configs struct {
	name        string
	version     string
	environment string
	region      string
	tags        map[string]string

	// Logger
	loggerEnabled bool
	loggerLevel   string

	// Prometheus
	prometheusEnabled bool

	// Jaeger
	jaegerEnabled           bool
	jaegerAgentEndpoint     string
	jaegerCollectorEndpoint string
	jaegerCollectorUserName string
	jaegerCollectorPassword string

	// OpenTelemetry
	opentelemetryEnabled              bool
	opentelemetryCollectorAddress     string
	opentelemetryCollectorCredentials credentials.TransportCredentials
}

func configsFromEnv() configs {
	c := configs{}

	c.name = os.Getenv("OBSERVER_NAME")
	c.version = os.Getenv("OBSERVER_VERSION")
	c.environment = os.Getenv("OBSERVER_ENVIRONMENT")
	c.region = os.Getenv("OBSERVER_REGION")

	c.tags = map[string]string{}
	for _, env := range os.Environ() {
		pair := strings.Split(env, "=")
		if strings.HasPrefix(pair[0], "OBSERVER_TAG_") {
			tag := strings.TrimPrefix(pair[0], "OBSERVER_TAG_")
			tag = strings.ToLower(tag)
			c.tags[tag] = pair[1]
		}
	}

	if val := os.Getenv("OBSERVER_LOGGER_ENABLED"); val != "" {
		c.loggerEnabled, _ = strconv.ParseBool(val)
	}

	c.loggerLevel = os.Getenv("OBSERVER_LOGGER_LEVEL")

	// Defaults
	if c.loggerLevel == "" {
		c.loggerLevel = "info"
	}

	if val := os.Getenv("OBSERVER_PROMETHEUS_ENABLED"); val != "" {
		c.prometheusEnabled, _ = strconv.ParseBool(val)
	}

	if val := os.Getenv("OBSERVER_JAEGER_ENABLED"); val != "" {
		c.jaegerEnabled, _ = strconv.ParseBool(val)
	}

	c.jaegerAgentEndpoint = os.Getenv("OBSERVER_JAEGER_AGENT_ENDPOINT")
	c.jaegerCollectorEndpoint = os.Getenv("OBSERVER_JAEGER_COLLECTOR_ENDPOINT")
	c.jaegerCollectorUserName = os.Getenv("OBSERVER_JAEGER_COLLECTOR_USERNAME")
	c.jaegerCollectorPassword = os.Getenv("OBSERVER_JAEGER_COLLECTOR_PASSWORD")

	// Defaults
	if c.jaegerAgentEndpoint == "" && c.jaegerCollectorEndpoint == "" {
		c.jaegerAgentEndpoint = "localhost:6831"
	}

	if val := os.Getenv("OBSERVER_OPENTELEMETRY_ENABLED"); val != "" {
		c.opentelemetryEnabled, _ = strconv.ParseBool(val)
	}

	c.opentelemetryCollectorAddress = os.Getenv("OBSERVER_OPENTELEMETRY_COLLECTOR_ADDRESS")

	// Defaults
	if c.opentelemetryCollectorAddress == "" {
		c.opentelemetryCollectorAddress = "localhost:55680"
	}

	return c
}

// Option is an optional configuration for an observer.
type Option func(*configs)

// WithMetadata is the option for specifying and reporting metadata.
// All arguments are optional.
func WithMetadata(name, version, environment, region string, tags map[string]string) Option {
	return func(c *configs) {
		c.name = name
		c.version = version
		c.environment = environment
		c.region = region
		c.tags = tags
	}
}

// WithLogger is the option for configuring the logger.
// The default log level is info.
func WithLogger(level string) Option {
	if level == "" {
		level = "info"
	}

	return func(c *configs) {
		c.loggerEnabled = true
		c.loggerLevel = level
	}
}

// WithPrometheus is the option for reporting metrics for Prometheus.
func WithPrometheus() Option {
	return func(c *configs) {
		c.prometheusEnabled = true
	}
}

// WithJaeger is the option for reporting traces to Jaeger.
// Only one of agentEndpoint or collectorEndpoint is required.
// collectorUserName and collectorPassword are optional.
// The default agent endpoint is localhost:6831.
func WithJaeger(agentEndpoint, collectorEndpoint, collectorUserName, collectorPassword string) Option {
	if agentEndpoint == "" && collectorEndpoint == "" {
		agentEndpoint = "localhost:6831"
	}

	return func(c *configs) {
		c.jaegerEnabled = true
		c.jaegerAgentEndpoint = agentEndpoint
		c.jaegerCollectorEndpoint = collectorEndpoint
		c.jaegerCollectorUserName = collectorUserName
		c.jaegerCollectorPassword = collectorPassword
	}
}

// WithOpenTelemetry is the option for reporting metrics and traces to OpenTelemetry Collector.
// collectorCredentials is optional. If not specified, the connection will be insecure.
// The default collector address is localhost:55680.
func WithOpenTelemetry(collectorAddress string, collectorCredentials credentials.TransportCredentials) Option {
	if collectorAddress == "" {
		collectorAddress = "localhost:55680"
	}

	return func(c *configs) {
		c.opentelemetryEnabled = true
		c.opentelemetryCollectorAddress = collectorAddress
		c.opentelemetryCollectorCredentials = collectorCredentials
	}
}

// Observer provides logging, metrics, and tracing capabilities for observability.
type Observer interface {
	// Shutdown flushes and closes the logger, meter, and tracer.
	Shutdown(context.Context) error

	// Name is returns the name of the observer.
	Name() string

	// Logger is used for accessing the logger.
	Logger() *zap.Logger

	// SetLogLevel changes the logging level.
	SetLogLevel(level zapcore.Level)

	// GetLogLevel returns the current logging level.
	GetLogLevel() zapcore.Level

	// Meter is used for accessing the meter.
	Meter() metric.Meter

	// Tracer is used for accessing the tracer.
	Tracer() trace.Tracer

	// ServeHTTP implements http.Handler interface. It serves the metrics endpoint for Prometheus metrics.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type observer struct {
	name          string
	logger        *zap.Logger
	loggerConfig  *zap.Config
	meter         metric.Meter
	promHandler   http.Handler
	tracer        trace.Tracer
	shutdownFuncs []shutdownFunc
}

// New creates a new observer.
// If setAsSingleton set to true, the created observer will be set as the singleton observer too.
// So, you can also access it using observer.Get() function.
func New(setAsSingleton bool, opts ...Option) Observer {
	c := configsFromEnv()
	for _, opt := range opts {
		opt(&c)
	}

	o := &observer{
		name: c.name,
	}

	if c.loggerEnabled {
		var shutdown shutdownFunc
		o.logger, o.loggerConfig, shutdown = initLogger(c)
		o.shutdownFuncs = append(o.shutdownFuncs, shutdown)
	}

	if c.prometheusEnabled {
		o.meter, o.promHandler = initPrometheus(c)
	}

	if c.jaegerEnabled {
		var shutdown shutdownFunc
		o.tracer, shutdown = initJaeger(c)
		o.shutdownFuncs = append(o.shutdownFuncs, shutdown)
	}

	if c.opentelemetryEnabled {
		var shutdown shutdownFunc
		o.meter, o.tracer, shutdown = initOpenTelemetry(c)
		o.shutdownFuncs = append(o.shutdownFuncs, shutdown)
	}

	// Create noop logger, meter, and/or tracer if they are not created so far

	if o.logger == nil {
		o.logger = zap.NewNop()
	}

	if o.loggerConfig == nil {
		o.loggerConfig = &zap.Config{}
	}

	if o.meter == (metric.Meter{}) {
		o.meter = new(metric.NoopMeterProvider).Meter("")
	}

	if o.promHandler == nil {
		o.promHandler = http.NotFoundHandler()
	}

	if o.tracer == nil {
		o.tracer = trace.NewNoopTracerProvider().Tracer("")
	}

	// Assign the new observer to the singleton observer
	if setAsSingleton {
		singleton = o
	}

	return o
}

func initLogger(c configs) (*zap.Logger, *zap.Config, shutdownFunc) {
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development: false,
		Sampling:    nil,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			MessageKey:     "message",
			CallerKey:      "caller",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
		InitialFields:    make(map[string]interface{}),
	}

	if c.name != "" {
		config.InitialFields["logger"] = c.name
	}

	if c.version != "" {
		config.InitialFields["version"] = c.version
	}

	if c.environment != "" {
		config.InitialFields["environment"] = c.environment
	}

	if c.region != "" {
		config.InitialFields["region"] = c.region
	}

	for k, v := range c.tags {
		config.InitialFields[k] = v
	}

	switch strings.ToLower(c.loggerLevel) {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "none":
		fallthrough
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.Level(99))
	}

	logger, _ := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(0),
	)

	shutdown := func(context.Context) error {
		return logger.Sync()
	}

	return logger, &config, shutdown
}

func initPrometheus(c configs) (metric.Meter, http.Handler) {
	// Create a new Prometheus registry
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	config := promexporter.Config{
		Registerer: registry,
		Gatherer:   registry,
		// DefaultSummaryQuantiles:    []float64{},
		// DefaultHistogramBoundaries: []float64{},
	}

	pullOpts := []pull.Option{}
	exporter, err := promexporter.NewExportPipeline(config, pullOpts...)
	if err != nil {
		panic(err)
	}

	otel.SetMeterProvider(exporter.MeterProvider())
	meter := exporter.MeterProvider().Meter(c.name)

	return meter, exporter
}

func initJaeger(c configs) (trace.Tracer, shutdownFunc) {
	var endpointOpt jaegerexporter.EndpointOption
	switch {
	case c.jaegerAgentEndpoint != "":
		endpointOpt = jaegerexporter.WithAgentEndpoint(c.jaegerAgentEndpoint)
	case c.jaegerCollectorEndpoint != "":
		endpointOpt = jaegerexporter.WithCollectorEndpoint(
			c.jaegerCollectorEndpoint,
			jaegerexporter.WithUsername(c.jaegerCollectorUserName),
			jaegerexporter.WithPassword(c.jaegerCollectorPassword),
		)
	}

	tags := []label.KeyValue{}
	for k, v := range c.tags {
		tags = append(tags, label.String(k, v))
	}

	processOpt := jaegerexporter.WithProcess(
		jaegerexporter.Process{
			ServiceName: c.name,
			Tags:        tags,
		},
	)

	sdkOpt := jaegerexporter.WithSDK(
		&tracesdk.Config{
			DefaultSampler: tracesdk.AlwaysSample(),
		},
	)

	provider, flush, err := jaegerexporter.NewExportPipeline(endpointOpt, processOpt, sdkOpt)
	if err != nil {
		panic(err)
	}

	otel.SetTracerProvider(provider)
	tracer := otel.Tracer(c.name)

	shutdown := func(context.Context) error {
		flush()
		return nil
	}

	return tracer, shutdown
}

func initOpenTelemetry(c configs) (metric.Meter, trace.Tracer, shutdownFunc) {
	ctx := context.Background()

	// ====================> Exporter <====================

	expOpts := []otlpexporter.ExporterOption{
		otlpexporter.WithAddress(c.opentelemetryCollectorAddress),
	}

	if c.opentelemetryCollectorCredentials == nil {
		expOpts = append(expOpts, otlpexporter.WithInsecure())
	} else {
		expOpts = append(expOpts, otlpexporter.WithTLSCredentials(c.opentelemetryCollectorCredentials))
	}

	exporter, err := otlpexporter.NewExporter(ctx, expOpts...)
	if err != nil {
		panic(err)
	}

	// ====================> Trace Provider <====================

	r, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(c.name),
		),
	)

	if err != nil {
		panic(err)
	}

	tpOpts := []tracesdk.TracerProviderOption{
		tracesdk.WithResource(r),
		tracesdk.WithConfig(tracesdk.Config{
			DefaultSampler: tracesdk.AlwaysSample(),
		}),
		tracesdk.WithSpanProcessor(
			tracesdk.NewBatchSpanProcessor(exporter),
		),
	}

	traceProvider := tracesdk.NewTracerProvider(tpOpts...)

	// ====================> Meter Provider <====================

	aggregator := simple.NewWithExactDistribution()
	checkpointer := basic.New(aggregator, exporter)
	pushOpts := []push.Option{
		push.WithPeriod(2 * time.Second),
	}

	pusher := push.New(checkpointer, exporter, pushOpts...)

	// ====================> Set Globals <====================

	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(pusher.MeterProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})
	pusher.Start()

	meter := otel.Meter(c.name)
	tracer := otel.Tracer(c.name)

	shutdown := func(ctx context.Context) error {
		if err := traceProvider.Shutdown(ctx); err != nil {
			return err
		}
		if err := exporter.Shutdown(ctx); err != nil {
			return err
		}
		// FIXME:
		// pusher.Stop()
		return nil
	}

	return meter, tracer, shutdown
}

func (o *observer) Shutdown(ctx context.Context) error {
	var err error
	for _, endFunc := range o.shutdownFuncs {
		if e := endFunc(ctx); e != nil {
			err = multierror.Append(err, e)
		}
	}

	return err
}

func (o *observer) Name() string {
	return o.name
}

func (o *observer) Logger() *zap.Logger {
	return o.logger
}

func (o *observer) SetLogLevel(level zapcore.Level) {
	o.loggerConfig.Level.SetLevel(level)
}

func (o *observer) GetLogLevel() zapcore.Level {
	return o.loggerConfig.Level.Level()
}

func (o *observer) Meter() metric.Meter {
	return o.meter
}

func (o *observer) Tracer() trace.Tracer {
	return o.tracer
}

func (o *observer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if o.promHandler != nil {
		o.promHandler.ServeHTTP(w, r)
	}
}

var singleton *observer

// Initialize the singleton observer with a no-op observer.
// init function will be only called once in runtime regardless of how many times the package is imported.
func init() {
	singleton = &observer{
		logger:       zap.NewNop(),
		loggerConfig: &zap.Config{},
		meter:        new(metric.NoopMeterProvider).Meter(""),
		promHandler:  http.NotFoundHandler(),
		tracer:       trace.NewNoopTracerProvider().Tracer(""),
	}
}

// Get returns the singleton Observer.
func Get() Observer {
	return singleton
}

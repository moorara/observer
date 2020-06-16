// Package observer can be used for implementing observability using OpenTelemetry API.
// It aims to unify three pillars of observability in one single package that is easy-to-use and hard-to-misuse.
//
// An Observer encompasses a logger, a meter, and a tracer.
// It offers a single unified developer experience for enabling observability.
package observer

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"

	"go.opentelemetry.io/otel/sdk/metric/controller/pull"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	promexporter "go.opentelemetry.io/otel/exporters/metric/prometheus"
	jaegerexporter "go.opentelemetry.io/otel/exporters/trace/jaeger"
)

// Options are optional configurations for creating an observer (logging, metrics, and tracing).
// LogLevel can be "debug", "info", "warn", "error", or "none" (case-insensitive).
type Options struct {
	Name                    string
	Version                 string
	Environment             string
	Region                  string
	Tags                    map[string]string
	LogLevel                string
	JaegerAgentEndpoint     string
	JaegerCollectorEndpoint string
	JaegerCollectorUserName string
	JaegerCollectorPassword string
}

func (opts Options) withDefaults() Options {
	if opts.LogLevel == "" {
		opts.LogLevel = "info"
	}

	if opts.JaegerAgentEndpoint == "" && opts.JaegerCollectorEndpoint == "" {
		opts.JaegerAgentEndpoint = "localhost:6831"
		opts.JaegerCollectorEndpoint = "http://localhost:14268/api/traces"
	}

	return opts
}

// Observer provides a logger, a meter, and a tracer for observability capabilities.
type Observer struct {
	name          string
	logger        *zap.Logger
	loggerConfig  *zap.Config
	meter         metric.Meter
	metricHandler http.Handler
	tracer        trace.Tracer
	tracerFlush   func()
}

// New creates a new observer.
// If setAsSingleton set to true, the created observer will be set as the singleton observer too.
// So, you can also access it using observer.Get() function.
func New(setAsSingleton bool, opts Options) *Observer {
	opts = opts.withDefaults()

	logger, loggerConfig := newLogger(opts)
	meter, metricHandler := newMeter(opts)
	tracer, tracerFlush := newTracer(opts)

	observer := &Observer{
		name:          opts.Name,
		logger:        logger,
		loggerConfig:  loggerConfig,
		meter:         meter,
		metricHandler: metricHandler,
		tracer:        tracer,
		tracerFlush:   tracerFlush,
	}

	if setAsSingleton {
		singleton = observer
	}

	return observer
}

func newLogger(opts Options) (*zap.Logger, *zap.Config) {
	config := zap.NewProductionConfig()
	config.Encoding = "json"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	config.OutputPaths = []string{"stdout"}
	config.InitialFields = make(map[string]interface{})

	if opts.Name != "" {
		config.InitialFields["logger"] = opts.Name
	}

	if opts.Version != "" {
		config.InitialFields["version"] = opts.Version
	}

	if opts.Environment != "" {
		config.InitialFields["environment"] = opts.Environment
	}

	if opts.Region != "" {
		config.InitialFields["region"] = opts.Region
	}

	for k, v := range opts.Tags {
		config.InitialFields[k] = v
	}

	switch strings.ToLower(opts.LogLevel) {
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

	return logger, &config
}

func newMeter(opts Options) (metric.Meter, http.Handler) {
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

	global.SetMeterProvider(exporter.Provider())
	meter := exporter.Provider().Meter(opts.Name)

	return meter, exporter
}

func newTracer(opts Options) (trace.Tracer, func()) {
	var endpointOpt jaegerexporter.EndpointOption
	switch {
	case opts.JaegerAgentEndpoint != "":
		endpointOpt = jaegerexporter.WithAgentEndpoint(opts.JaegerAgentEndpoint)
	case opts.JaegerCollectorEndpoint != "":
		endpointOpt = jaegerexporter.WithCollectorEndpoint(
			opts.JaegerCollectorEndpoint,
			jaegerexporter.WithUsername(opts.JaegerCollectorUserName),
			jaegerexporter.WithPassword(opts.JaegerCollectorPassword),
		)
	}

	tags := []kv.KeyValue{}
	for k, v := range opts.Tags {
		tags = append(tags, kv.String(k, v))
	}

	processOpt := jaegerexporter.WithProcess(
		jaegerexporter.Process{
			ServiceName: opts.Name,
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

	global.SetTraceProvider(provider)
	tracer := global.TraceProvider().Tracer(opts.Name)

	return tracer, flush
}

// Close implements io.Closer interface.
// It flushes the logger, meter, and tracer.
func (o *Observer) Close() error {
	o.tracerFlush()
	return o.logger.Sync()
}

// Name is returns the name of the observer.
func (o *Observer) Name() string {
	return o.name
}

// Logger is used for accessing the logger.
func (o *Observer) Logger() *zap.Logger {
	return o.logger
}

// SetLogLevel changes the logging level.
func (o *Observer) SetLogLevel(level zapcore.Level) {
	o.loggerConfig.Level.SetLevel(level)
}

// GetLogLevel returns the current logging level.
func (o *Observer) GetLogLevel() zapcore.Level {
	return o.loggerConfig.Level.Level()
}

// Meter is used for accessing the meter.
func (o *Observer) Meter() metric.Meter {
	return o.meter
}

// Tracer is used for accessing the tracer.
func (o *Observer) Tracer() trace.Tracer {
	return o.tracer
}

// ServeHTTP implements http.Handler interface.
// It serves the metrics endpoint for Prometheus metrics.
func (o *Observer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.metricHandler.ServeHTTP(w, r)
}

// The singleton observer.
var singleton *Observer

// Initialize the singleton observer with a no-op observer.
// init function will be only called once in runtime regardless of how many times the package is imported.
func init() {
	mp := metric.NoopProvider{}
	tp := trace.NoopProvider{}

	singleton = &Observer{
		logger:       zap.NewNop(),
		loggerConfig: &zap.Config{},
		meter:        mp.Meter("Noop"),
		tracer:       tp.Tracer("Noop"),
	}
}

// Get returns the singleton Observer.
func Get() *Observer {
	return singleton
}

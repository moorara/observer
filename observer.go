// Package observer can be used for implementing observability using OpenTelemetry API.
// It aims to unify three pillars of observability in one single package that is easy-to-use and hard-to-misuse.
//
// An Observer encompasses a logger, a meter, and a tracer.
// It offers a single unified developer experience for enabling observability.
package observer

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	promexporter "go.opentelemetry.io/otel/exporters/metric/prometheus"
	jaegerexporter "go.opentelemetry.io/otel/exporters/trace/jaeger"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultMetricInterval = 5 * time.Second
)

var (
	defaultSummaryQuantiles = []float64{0.1, 0.5, 0.95, 0.99}
	defaultHistogramBuckets = []float64{0.01, 0.10, 0.50, 1.00, 5.00}
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
	SummaryQuantiles        []float64
	HistogramBuckets        []float64
	JaegerAgentEndpoint     string
	JaegerCollectorEndpoint string
	JaegerCollectorUserName string
	JaegerCollectorPassword string
}

func (opts Options) withDefaults() Options {
	if opts.LogLevel == "" {
		opts.LogLevel = "info"
	}

	if len(opts.SummaryQuantiles) == 0 {
		opts.SummaryQuantiles = defaultSummaryQuantiles
	}

	if len(opts.HistogramBuckets) == 0 {
		opts.HistogramBuckets = defaultHistogramBuckets
	}

	if opts.JaegerAgentEndpoint == "" && opts.JaegerCollectorEndpoint == "" {
		opts.JaegerAgentEndpoint = "localhost:6831"
		opts.JaegerCollectorEndpoint = "http://localhost:14268/api/traces"
	}

	return opts
}

// Observer provides a logger, a meter, and a tracer for observability capabilities.
type Observer struct {
	Logger *zap.Logger
	Meter  metric.Meter
	Tracer trace.Tracer

	loggerConfig   *zap.Config
	metricsHandler http.Handler
	meterClose     func()
	tracerClose    func()
}

// New creates a new observer.
// If setAsSingleton set to true, the created observer will be set as the singleton observer too.
// So, you can also access it using observer.Get() function.
func New(setAsSingleton bool, opts Options) *Observer {
	opts = opts.withDefaults()

	logger, loggerConfig := newLogger(opts)
	meter, meterClose, metricsHandler := newMeter(opts)
	tracer, tracerClose := newTracer(opts)

	observer := &Observer{
		Logger: logger,
		Meter:  meter,
		Tracer: tracer,

		loggerConfig:   loggerConfig,
		metricsHandler: metricsHandler,
		meterClose:     meterClose,
		tracerClose:    tracerClose,
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

func newMeter(opts Options) (metric.Meter, func(), http.Handler) {
	// Create a new Prometheus registry
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	config := promexporter.Config{
		Registerer:              registry,
		Gatherer:                registry,
		DefaultSummaryQuantiles: opts.SummaryQuantiles,
		// TODO: opts.HistogramBuckets
	}

	controller, handler, err := promexporter.NewExportPipeline(config, defaultMetricInterval)
	if err != nil {
		panic(err)
	}

	global.SetMeterProvider(controller)
	meter := global.MeterProvider().Meter(opts.Name)

	return meter, controller.Stop, handler
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

	tags := []core.KeyValue{}
	for k, v := range opts.Tags {
		tags = append(tags, key.String(k, v))
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

	provider, close, err := jaegerexporter.NewExportPipeline(endpointOpt, processOpt, sdkOpt)
	if err != nil {
		panic(err)
	}

	global.SetTraceProvider(provider)
	tracer := global.TraceProvider().Tracer(opts.Name)

	return tracer, close
}

// Close implements io.Closer interface.
// It flushes the logger, meter, and tracer.
func (o *Observer) Close() error {
	o.meterClose()
	o.tracerClose()
	return o.Logger.Sync()
}

// SetLogLevel changes the logging level.
func (o *Observer) SetLogLevel(level zapcore.Level) {
	o.loggerConfig.Level.SetLevel(level)
}

// GetLogLevel returns the current logging level.
func (o *Observer) GetLogLevel() zapcore.Level {
	return o.loggerConfig.Level.Level()
}

// ServeHTTP implements http.Handler interface.
// It serves the metrics endpoint for Prometheus metrics.
func (o *Observer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.metricsHandler.ServeHTTP(w, r)
}

// The singleton observer.
var singleton *Observer

// Initialize the singleton observer with a no-op observer.
// init function will be only called once in runtime regardless of how many times the package is imported.
func init() {
	singleton = &Observer{
		Logger: zap.NewNop(),
		Meter:  &metric.NoopMeter{},
		Tracer: &trace.NoopTracer{},

		loggerConfig: &zap.Config{},
	}
}

// Get returns the singleton Observer.
func Get() *Observer {
	return singleton
}

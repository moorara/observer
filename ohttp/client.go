package ohttp

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moorara/observer"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/unit"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"go.uber.org/zap"
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
			"outgoing_http_requests_total",
			metric.WithDescription("The total number of outgoing http requests (client-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithLibraryName(libraryName),
		),
		reqGauge: mm.NewInt64UpDownCounter(
			"outgoing_http_requests_active",
			metric.WithDescription("The number of in-flight outgoing http requests (client-side)"),
			metric.WithUnit(unit.Dimensionless),
			metric.WithLibraryName(libraryName),
		),
		reqDuration: mm.NewInt64ValueRecorder(
			"outgoing_http_requests_duration",
			metric.WithDescription("The duration of outgoing http requests in seconds (client-side)"),
			metric.WithUnit(unit.Milliseconds),
			metric.WithLibraryName(libraryName),
		),
	}
}

// Client is a drop-in replacement for the standard http.Client.
// It is an observable http client with logging, metrics, and tracing.
type Client struct {
	opts        Options
	client      *http.Client
	observer    *observer.Observer
	instruments *clientInstruments
}

// NewClient creates a new observable http client.
func NewClient(client *http.Client, observer *observer.Observer, opts Options) *Client {
	opts = opts.withDefaults()
	instruments := newClientInstruments(observer.Meter())

	return &Client{
		opts:        opts,
		client:      client,
		observer:    observer,
		instruments: instruments,
	}
}

// CloseIdleConnections is the observable counterpart of standard http Client.CloseIdleConnections.
func (c *Client) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

// Get is the observable counterpart of standard http Client.Get.
// Using this method, request context (UUID and trace) will be auto-generated.
// If you have a context for the request, consider using the Do method.
func (c *Client) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req)
}

// Head is the observable counterpart of standard http Client.Head.
// Using this method, request context (UUID and trace) will be auto-generated.
// If you have a context for the request, consider using the Do method.
func (c *Client) Head(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req)
}

// Post is the observable counterpart of standard http Client.Post.
// Using this method, request context (UUID and trace) will be auto-generated.
// If you have a context for the request, consider using the Do method.
func (c *Client) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)

	return c.client.Do(req)
}

// PostForm is the observable counterpart of standard http Client.PostForm.
// Using this method, request context (UUID and trace) will be auto-generated.
// If you have a context for the request, consider using the Do method.
func (c *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	contentType := "application/x-www-form-urlencoded"
	body := strings.NewReader(data.Encode())

	return c.Post(url, contentType, body)
}

// Do is the observable counterpart of standard http Client.Do.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	ctx := req.Context()
	kind := "client"
	method := req.Method
	url := req.URL.Path
	route := c.opts.IDRegexp.ReplaceAllString(url, ":id")

	// Increase the number of in-flight requests
	c.instruments.reqGauge.Add(ctx, 1,
		kv.String("method", method),
		kv.String("route", route),
	)

	// Make sure the request has a UUID
	requestUUID, ok := observer.UUIDFromContext(ctx)
	if !ok || requestUUID == "" {
		requestUUID = uuid.New().String()
	}

	// Propagate http request headers
	req.Header.Set(requestUUIDHeader, requestUUID)
	req.Header.Set(clientNameHeader, c.observer.Name())

	// Create a new correlation context
	ctx = correlation.NewContext(ctx,
		kv.String("req.uuid", requestUUID),
		kv.String("client.name", c.observer.Name()),
	)

	// Start a new span
	ctx, span := c.observer.Tracer().Start(ctx,
		"http-client-request",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// Inject the correlation context and the span context into the http headers
	ctx, req = httptrace.W3C(ctx, req)
	httptrace.Inject(ctx, req)

	// Make the http call
	span.AddEvent(ctx, "making http call")
	resp, err := c.client.Do(req)

	duration := time.Since(startTime).Milliseconds()

	var statusCode int
	var statusClass string

	if err != nil {
		statusCode = -1
		statusClass = ""
	} else {
		statusCode = resp.StatusCode
		statusClass = fmt.Sprintf("%dxx", statusCode/100)
	}

	// Report metrics
	c.observer.Meter().RecordBatch(ctx,
		[]kv.KeyValue{
			kv.String("method", method),
			kv.String("route", route),
			kv.Int("status_code", statusCode),
			kv.String("status_class", statusClass),
		},
		c.instruments.reqCounter.Measurement(1),
		c.instruments.reqDuration.Measurement(duration),
	)

	// Report logs
	logger := c.observer.Logger()
	message := fmt.Sprintf("%s %s %d %dms", method, url, statusCode, duration)
	fields := []zap.Field{
		zap.String("req.uuid", requestUUID),
		zap.String("req.kind", kind),
		zap.String("req.method", method),
		zap.String("req.url", url),
		zap.String("req.route", route),
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
		if c.opts.LogInDebugLevel {
			logger.Debug(message, fields...)
		} else {
			logger.Info(message, fields...)
		}
	}

	// Decrease the number of in-flight requests
	c.instruments.reqGauge.Add(ctx, -1,
		kv.String("method", method),
		kv.String("route", route),
	)

	// Report the span
	span.SetAttributes(
		kv.String("method", method),
		kv.String("url", url),
		kv.String("route", route),
		kv.Int("status_code", statusCode),
	)

	return resp, err
}

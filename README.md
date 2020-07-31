[![Go Doc][godoc-image]][godoc-url]
[![Build Status][workflow-image]][workflow-url]
[![Go Report Card][goreport-image]][goreport-url]
[![Test Coverage][coverage-image]][coverage-url]
[![Maintainability][maintainability-image]][maintainability-url]

# observer

This package can be used for building observable applications in Go.
It aims to unify three pillars of observability in one single package that is _easy-to-use_ and _hard-to-misuse_.
This package leverages the [OpenTelemetry](https://opentelemetry.io) API.

An Observer encompasses a logger, a meter, and a tracer.
It offers a single unified developer experience for enabling observability.

## The Three Pillars of Observability

### Logging

Logs are used for _auditing_ purposes (sometimes for debugging with limited capabilities).
When looking at logs, you need to know what to look for ahead of the time (known unknowns vs. unknown unknowns).
Since log data can have any arbitrary shape and size, they cannot be used for real-time computational purposes.
Logs are hard to track across different and distributed processes. Logs are also very expensive at scale.

### Metrics

Metrics are _regular time-series_ data with _low and fixed cardinality_.
They are aggregated by time. Metrics are used for **real-time** monitoring purposes.
Using metrics with can implement **SLIs** (service-level indicators), **SLOs** (service-level objectives), and automated alerts.
Metrics are very good at taking the distribution of data into account.
Metrics cannot be used with _high-cardinality data_.

### Tracing

Traces are used for _debugging_ and _tracking_ requests across different processes and services.
They can be used for identifying performance bottlenecks.
Due to their very data-heavy nature, traces in real-world applications need to be _sampled_.
Insights extracted from traces cannot be aggregated since they are sampled.
In other words, information captured by one trace does not tell anything about how this trace is compared against other traces, and what is the distribution of data.

## Quick Start

<details>
  <summary>Example</summary>

```go
package main

import (
  "context"
  "net/http"
  "time"

  "github.com/moorara/observer"
  "go.opentelemetry.io/otel/api/correlation"
  "go.opentelemetry.io/otel/api/kv"
  "go.opentelemetry.io/otel/api/metric"
  "go.uber.org/zap"
)

type instruments struct {
  reqCounter  metric.Int64Counter
  reqDuration metric.Float64ValueRecorder
}

func newInstruments(meter metric.Meter) *instruments {
  mm := metric.Must(meter)

  return &instruments{
    reqCounter:  mm.NewInt64Counter("requests_total", metric.WithDescription("the total number of requests")),
    reqDuration: mm.NewFloat64ValueRecorder("request_duration_seconds", metric.WithDescription("the duration of requests in seconds")),
  }
}

type server struct {
  observer    observer.Observer
  instruments *instruments
}

func (s *server) Handle(ctx context.Context) {
  // Tracing
  ctx, span := s.observer.Tracer().Start(ctx, "handle-request")
  defer span.End()

  start := time.Now()
  s.fetch(ctx)
  s.respond(ctx)
  duration := time.Now().Sub(start)

  labels := []kv.KeyValue{
    kv.String("method", "GET"),
    kv.String("endpoint", "/user"),
    kv.Uint("statusCode", 200),
  }

  // Metrics
  s.observer.Meter().RecordBatch(ctx, labels,
    s.instruments.reqCounter.Measurement(1),
    s.instruments.reqDuration.Measurement(duration.Seconds()),
  )

  // Logging
  s.observer.Logger().Info("request handled successfully.",
    zap.String("method", "GET"),
    zap.String("endpoint", "/user"),
    zap.Uint("statusCode", 200),
  )
}

func (s *server) fetch(ctx context.Context) {
  _, span := s.observer.Tracer().Start(ctx, "read-database")
  defer span.End()

  time.Sleep(50 * time.Millisecond)
}

func (s *server) respond(ctx context.Context) {
  _, span := s.observer.Tracer().Start(ctx, "send-response")
  defer span.End()

  time.Sleep(10 * time.Millisecond)
}

func main() {
  // Creating a new Observer and set it as the singleton
  obsv := observer.New(true, observer.Options{
    Name:        "my-service",
    Version:     "0.1.0",
    Environment: "production",
    Region:      "ca-central-1",
    Tags: map[string]string{
      "domain": "auth",
    },
    LogLevel:            "info",
    JaegerAgentEndpoint: "localhost:6831",
  })
  defer obsv.Close()

  srv := &server{
    observer:    obsv,
    instruments: newInstruments(obsv.Meter()),
  }

  // Creating a correlation context
  ctx := context.Background()
  ctx = correlation.NewContext(ctx,
    kv.String("tenant", "1234"),
  )

  srv.Handle(ctx)

  // Serving metrics endpoint
  http.Handle("/metrics", obsv)
  http.ListenAndServe(":8080", nil)
}
```

Here are the logs from stdout:

```json
{"level":"info","timestamp":"2020-07-31T16:17:23.67794-04:00","caller":"example/main.go:57","message":"request handled successfully.","domain":"auth","environment":"production","logger":"my-service","region":"ca-central-1","version":"0.1.0","method":"GET","endpoint":"/user","statusCode":200}
```

And here are the metrics reported at http://localhost:8080/metrics:

```
# HELP request_duration_seconds the duration of requests in seconds
# TYPE request_duration_seconds histogram
request_duration_seconds_bucket{endpoint="/user",method="GET",statusCode="200",le="+Inf"} 1
request_duration_seconds_sum{endpoint="/user",method="GET",statusCode="200"} 0.063812057
request_duration_seconds_count{endpoint="/user",method="GET",statusCode="200"} 1
# HELP requests_total the total number of requests
# TYPE requests_total counter
requests_total{endpoint="/user",method="GET",statusCode="200"} 1
```
</details>

## OpenTelemetry

### Logging

_TBD_

### Metrics

Metric _instruments capture measurements_ at runtime. A Meter is used for creating metric instruments.

There are two kinds of measurements:

  - **Additive**: measurements for which only the sum is considered useful information
  - **Non-Additive**: measurements for which the set of values (a.k.a. population or distribution) has useful information

Non-additive instruments capture more information than additive instruments, but non-additive measurements are more expensive.

_Aggregation_ is the process of combining multiple measurements into exact or estimated statistics during an interval of time.
Each instrument has a default aggregation. Other standard aggregations (histograms, quantile summaries, cardinality estimates, etc.) are also available.

There are six kinds of metric instruments:

| Name              | Synchronous | Additive | Monotonic | Default Aggregation |
|-------------------|-------------|----------|-----------|---------------------|
| Counter           | Yes         | Yes      | Yes       | Sum                 |
| UpDownCounter     | Yes         | Yes      | No        | Sum                 |
| ValueRecorder     | Yes         | No       | No        | MinMaxSumCount      |
| SumObserver       | No          | Yes      | Yes       | Sum                 |
| UpDownSumObserver | No          | Yes      | No        | Sum                 |
| ValueObserver     | No          | No       | No        | MinMaxSumCount      |

The _synchronous_ instruments are useful for measurements that are gathered in a distributed Context.
The _asynchronous_ instruments are useful when measurements are expensive, therefore should be gathered periodically.
Synchronous instruments are used to capture changes in a sum, whereas asynchronous instruments are used to capture sums directly.
Asynchronous (observer) instruments capture measurements about the state of the application periodically.

### Tracing

_TBD_

## Documentation

  - **Logging**
    - [go.uber.org/zap](https://pkg.go.dev/go.uber.org/zap)
  - **Metrics**
    - [Metrics API](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/metrics/api.md)
    - [go.opentelemetry.io/otel/api/metric](https://pkg.go.dev/go.opentelemetry.io/otel/api/metric)
  - **Tracing**
    - [Tracing API](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/api.md)
    - [go.opentelemetry.io/otel/api/trace](https://pkg.go.dev/go.opentelemetry.io/otel/api/trace)


[godoc-url]: https://pkg.go.dev/github.com/moorara/observer
[godoc-image]: https://godoc.org/github.com/moorara/observer?status.svg
[workflow-url]: https://github.com/moorara/observer/actions
[workflow-image]: https://github.com/moorara/observer/workflows/Main/badge.svg
[goreport-url]: https://goreportcard.com/report/github.com/moorara/observer
[goreport-image]: https://goreportcard.com/badge/github.com/moorara/observer
[coverage-url]: https://codeclimate.com/github/moorara/observer/test_coverage
[coverage-image]: https://api.codeclimate.com/v1/badges/727461eda3a578b3ccc2/test_coverage
[maintainability-url]: https://codeclimate.com/github/moorara/observer/maintainability
[maintainability-image]: https://api.codeclimate.com/v1/badges/727461eda3a578b3ccc2/maintainability

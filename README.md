[![Go Doc][godoc-image]][godoc-url]
[![Build Status][workflow-image]][workflow-url]
[![Go Report Card][goreport-image]][goreport-url]
[![Test Coverage][coverage-image]][coverage-url]
[![Maintainability][maintainability-image]][maintainability-url]

# observer

This package can be used for building observable applications in Go.
It aims to unify three pillars of observability in one single package that is _easy-to-use_ and _hard-to-misuse_.
This package leverages the [OpenTelemetry](https://opentelemetry.io) API in an opinionated way.

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
```

```json
```

```
```
</details>

## Documentation

  - **Logging**
    - [go.uber.org/zap](https://pkg.go.dev/go.uber.org/zap)
  - **Metrics**
    - [Metrics API](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/metrics/api.md)
    - [Metric User-Facing API](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/metrics/api-user.md)
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

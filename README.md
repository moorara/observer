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

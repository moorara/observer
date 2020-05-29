module github.com/moorara/observer

go 1.14

require (
	github.com/google/uuid v1.1.1
	github.com/prometheus/client_golang v1.6.0
	github.com/stretchr/testify v1.6.0
	go.opentelemetry.io/otel v0.4.3
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.4.3
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.4.3
	go.uber.org/zap v1.15.0
)

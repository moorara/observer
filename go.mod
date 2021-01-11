module github.com/moorara/observer

go 1.15

require (
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/prometheus/client_golang v1.9.0
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/otel v0.15.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.15.0
	go.opentelemetry.io/otel/exporters/otlp v0.15.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.15.0
	go.opentelemetry.io/otel/sdk v0.15.0
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.34.1
	google.golang.org/protobuf v1.25.0
)

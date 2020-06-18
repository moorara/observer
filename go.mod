module github.com/moorara/observer

go 1.14

require (
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/prometheus/client_golang v1.7.0
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/otel v0.6.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.6.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.6.0
	go.uber.org/zap v1.15.0
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.24.0
)

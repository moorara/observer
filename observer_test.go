package observer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		setAsSingleton bool
		opts           Options
	}{
		{
			name:           "Default",
			setAsSingleton: false,
			opts:           Options{},
		},
		{
			name:           "Production",
			setAsSingleton: true,
			opts: Options{
				Name:        "my-service",
				Version:     "0.1.0",
				Environment: "production",
				Region:      "us-east-1",
				Tags: map[string]string{
					"domain": "auth",
				},
				LogLevel:            "warn",
				JaegerAgentEndpoint: "localhost:6831",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(T *testing.T) {
			observer := New(tc.setAsSingleton, tc.opts)

			assert.NotNil(t, observer)
			assert.Equal(t, tc.opts.Name, observer.name)
			assert.NotNil(t, observer.logger)
			assert.NotNil(t, observer.loggerConfig)
			assert.NotNil(t, observer.meter)
			assert.NotNil(t, observer.metricHandler)
			assert.NotNil(t, observer.tracer)
			assert.NotNil(t, observer.tracerFlush)
		})
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		opts          Options
		expectedLevel zapcore.Level
	}{
		{
			name:          "Default",
			opts:          Options{},
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name: "Production",
			opts: Options{
				Name:        "my-service",
				Version:     "0.1.0",
				Environment: "production",
				Region:      "us-east-1",
				Tags: map[string]string{
					"domain": "auth",
				},
				LogLevel: "warn",
			},
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name: "LogLevelDebug",
			opts: Options{
				Name:     "my-service",
				LogLevel: "debug",
			},
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name: "LogLevelInfo",
			opts: Options{
				Name:     "my-service",
				LogLevel: "info",
			},
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name: "LogLevelWarn",
			opts: Options{
				Name:     "my-service",
				LogLevel: "warn",
			},
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name: "LogLevelError",
			opts: Options{
				Name:     "my-service",
				LogLevel: "error",
			},
			expectedLevel: zapcore.ErrorLevel,
		},
		{
			name: "LogLevelNone",
			opts: Options{
				Name:     "my-service",
				LogLevel: "none",
			},
			expectedLevel: zapcore.Level(99),
		},
		{
			name: "InvalidLogLevel",
			opts: Options{
				Name:     "my-service",
				LogLevel: "invalid",
			},
			expectedLevel: zapcore.Level(99),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(T *testing.T) {
			opts := tc.opts.withDefaults()
			logger, config := newLogger(opts)

			assert.NotNil(t, logger)
			assert.NotNil(t, config)
			assert.Equal(t, tc.expectedLevel, config.Level.Level())
		})
	}
}

func TestNewMeter(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "Default",
			opts: Options{},
		},
		{
			name: "Production",
			opts: Options{
				Name: "my-service",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts.withDefaults()
			meter, handler := newMeter(opts)

			assert.NotNil(t, meter)
			assert.NotNil(t, handler)
		})
	}
}

func TestNewTracer(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "Default",
			opts: Options{},
		},
		{
			name: "Production",
			opts: Options{
				Name:        "my-service",
				Version:     "0.1.0",
				Environment: "production",
				Region:      "us-east-1",
				Tags: map[string]string{
					"domain": "auth",
				},
			},
		},
		{
			name: "WithAgent",
			opts: Options{
				Name:        "my-service",
				Version:     "0.1.0",
				Environment: "production",
				Region:      "us-east-1",
				Tags: map[string]string{
					"domain": "auth",
				},
				JaegerAgentEndpoint: "localhost:6831",
			},
		},
		{
			name: "WithCollector",
			opts: Options{
				Name:        "my-service",
				Version:     "0.1.0",
				Environment: "production",
				Region:      "us-east-1",
				Tags: map[string]string{
					"domain": "auth",
				},
				JaegerCollectorEndpoint: "http://localhost:14268/api/traces",
				JaegerCollectorUserName: "username",
				JaegerCollectorPassword: "password",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts.withDefaults()
			tracer, flush := newTracer(opts)
			defer flush()

			assert.NotNil(t, tracer)
			assert.NotNil(t, flush)
		})
	}
}

func TestObserverClose(t *testing.T) {
	tests := []struct {
		name          string
		observer      *Observer
		expectedError error
	}{
		{
			name: "Success",
			observer: &Observer{
				logger:      zap.NewNop(),
				tracerFlush: func() {},
			},
			expectedError: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.observer.Close()

			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestObserverName(t *testing.T) {
	tests := []struct {
		name     string
		observer *Observer
	}{
		{
			name: "OK",
			observer: &Observer{
				name: "my-service",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.observer.name, tc.observer.Name())
		})
	}
}

func TestObserverLogger(t *testing.T) {
	tests := []struct {
		name     string
		observer *Observer
	}{
		{
			name: "OK",
			observer: &Observer{
				logger: zap.NewNop(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.observer.logger, tc.observer.Logger())
		})
	}
}

func TestObserverSetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		observer *Observer
		level    zapcore.Level
	}{
		{
			name: "Debug",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.DebugLevel,
		},
		{
			name: "Info",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.InfoLevel,
		},
		{
			name: "Warn",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.WarnLevel,
		},
		{
			name: "Error",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.ErrorLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.observer.SetLogLevel(tc.level)

			assert.Equal(t, tc.level, tc.observer.loggerConfig.Level.Level())
		})
	}
}

func TestObserverGetLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		observer      *Observer
		expectedLevel zapcore.Level
	}{
		{
			name: "Debug",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.DebugLevel),
				},
			},
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name: "Info",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.InfoLevel),
				},
			},
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name: "Warn",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.WarnLevel),
				},
			},
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name: "Error",
			observer: &Observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.ErrorLevel),
				},
			},
			expectedLevel: zapcore.ErrorLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			level := tc.observer.GetLogLevel()

			assert.Equal(t, tc.expectedLevel, level)
		})
	}
}

func TestObserverMeter(t *testing.T) {
	tests := []struct {
		name     string
		observer *Observer
	}{
		{
			name: "OK",
			observer: &Observer{
				meter: new(metric.NoopProvider).Meter("Noop"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.observer.meter, tc.observer.Meter())
		})
	}
}

func TestObserverTracer(t *testing.T) {
	tests := []struct {
		name     string
		observer *Observer
	}{
		{
			name: "OK",
			observer: &Observer{
				tracer: &trace.NoopTracer{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.observer.tracer, tc.observer.Tracer())
		})
	}
}

func TestObserverServeHTTP(t *testing.T) {
	tests := []struct {
		name               string
		observer           *Observer
		req                *http.Request
		expectedStatusCode int
	}{
		{
			name: "OK",
			observer: &Observer{
				metricHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			},
			req:                httptest.NewRequest("GET", "/metrics", nil),
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			tc.observer.ServeHTTP(resp, tc.req)

			statusCode := resp.Result().StatusCode
			assert.Equal(t, tc.expectedStatusCode, statusCode)
		})
	}
}

func TestSingleton(t *testing.T) {
	tests := []struct {
		name      string
		singleton *Observer
	}{
		{
			name:      "OK",
			singleton: &Observer{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			singleton = tc.singleton

			assert.Equal(t, tc.singleton, Get())
		})
	}
}

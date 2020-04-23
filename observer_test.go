package observer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
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
				LogLevel:            "warn",
				JaegerAgentEndpoint: "localhost:6831",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(T *testing.T) {
			observer := New(tc.opts)

			assert.NotNil(t, observer)
			assert.NotNil(t, observer.Logger)
			assert.NotNil(t, observer.Meter)
			assert.NotNil(t, observer.Tracer)
			assert.NotNil(t, observer.loggerConfig)
			assert.NotNil(t, observer.metricsHandler)
			assert.NotNil(t, observer.meterClose)
			assert.NotNil(t, observer.tracerClose)
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
		{
			name: "WithSummaryQuantiles",
			opts: Options{
				Name:             "my-service",
				SummaryQuantiles: []float64{0.1, 0.5, 0.90, 0.95, 0.99},
			},
		},
		{
			name: "WithMetricBuckets",
			opts: Options{
				Name:             "my-service",
				HistogramBuckets: []float64{0.01, 0.20, 0.50, 1.00, 5.00, 10.00},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts.withDefaults()
			meter, close, handler := newMeter(opts)
			defer close()

			assert.NotNil(t, meter)
			assert.NotNil(t, close)
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
			tracer, close := newTracer(opts)
			defer close()

			assert.NotNil(t, tracer)
			assert.NotNil(t, close)
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
				Logger:      zap.NewNop(),
				meterClose:  func() {},
				tracerClose: func() {},
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
				metricsHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				LogLevel:            "warn",
				JaegerAgentEndpoint: "localhost:6831",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("Init", func(t *testing.T) {
				Init(tc.opts)

				assert.NotNil(t, singleton)
				assert.NotNil(t, singleton.Logger)
				assert.NotNil(t, singleton.Meter)
				assert.NotNil(t, singleton.Tracer)
				assert.NotNil(t, singleton.loggerConfig)
				assert.NotNil(t, singleton.metricsHandler)
				assert.NotNil(t, singleton.meterClose)
				assert.NotNil(t, singleton.tracerClose)
			})

			t.Run("Get", func(t *testing.T) {
				assert.Equal(t, singleton, Get())
			})
		})
	}
}

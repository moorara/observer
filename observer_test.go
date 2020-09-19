package observer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConfigsFromEnv(t *testing.T) {
	type keyval struct {
		name  string
		value string
	}

	tests := []struct {
		name            string
		envars          []keyval
		expectedConfigs configs
	}{
		{
			name:   "Defaults",
			envars: []keyval{},
			expectedConfigs: configs{
				loggerLevel:                   "info",
				jaegerAgentEndpoint:           "localhost:6831",
				opentelemetryCollectorAddress: "localhost:55680",
				tags:                          map[string]string{},
			},
		},
		{
			name: "All",
			envars: []keyval{
				keyval{"OBSERVER_NAME", "my-service"},
				keyval{"OBSERVER_VERSION", "0.1.0"},
				keyval{"OBSERVER_ENVIRONMENT", "production"},
				keyval{"OBSERVER_REGION", "ca-central-1"},
				keyval{"OBSERVER_TAG_DOMAIN", "auth"},
				keyval{"OBSERVER_LOGGER_ENABLED", "true"},
				keyval{"OBSERVER_LOGGER_LEVEL", "warn"},
				keyval{"OBSERVER_PROMETHEUS_ENABLED", "true"},
				keyval{"OBSERVER_JAEGER_ENABLED", "true"},
				keyval{"OBSERVER_JAEGER_AGENT_ENDPOINT", "localhost:6831"},
				keyval{"OBSERVER_JAEGER_COLLECTOR_ENDPOINT", "http://localhost:14268/api/traces"},
				keyval{"OBSERVER_JAEGER_COLLECTOR_USERNAME", "username"},
				keyval{"OBSERVER_JAEGER_COLLECTOR_PASSWORD", "password"},
				keyval{"OBSERVER_OPENTELEMETRY_ENABLED", "true"},
				keyval{"OBSERVER_OPENTELEMETRY_COLLECTOR_ADDRESS", "localhost:55680"},
			},
			expectedConfigs: configs{
				name:        "my-service",
				version:     "0.1.0",
				environment: "production",
				region:      "ca-central-1",
				tags: map[string]string{
					"domain": "auth",
				},
				loggerEnabled:                     true,
				loggerLevel:                       "warn",
				prometheusEnabled:                 true,
				jaegerEnabled:                     true,
				jaegerAgentEndpoint:               "localhost:6831",
				jaegerCollectorEndpoint:           "http://localhost:14268/api/traces",
				jaegerCollectorUserName:           "username",
				jaegerCollectorPassword:           "password",
				opentelemetryEnabled:              true,
				opentelemetryCollectorAddress:     "localhost:55680",
				opentelemetryCollectorCredentials: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for _, envar := range tc.envars {
				if err := os.Setenv(envar.name, envar.value); err != nil {
					t.Fatalf("Failed to set environment variable %s: %s", envar.name, err)
				}
				defer os.Unsetenv(envar.name)
			}

			configs := configsFromEnv()

			assert.Equal(t, tc.expectedConfigs, configs)
		})
	}
}

func TestOption(t *testing.T) {
	tests := []struct {
		name            string
		configs         *configs
		option          Option
		expectedConfigs *configs
	}{
		{
			name:            "WithMetadataDefaults",
			configs:         &configs{},
			option:          WithMetadata("", "", "", "", nil),
			expectedConfigs: &configs{},
		},
		{
			name:    "WithMetadata",
			configs: &configs{},
			option: WithMetadata("my-service", "0.1.0", "production", "ca-central-1", map[string]string{
				"domain": "auth",
			}),
			expectedConfigs: &configs{
				name:        "my-service",
				version:     "0.1.0",
				environment: "production",
				region:      "ca-central-1",
				tags: map[string]string{
					"domain": "auth",
				},
			},
		},
		{
			name:    "WithLoggerDefaults",
			configs: &configs{},
			option:  WithLogger(""),
			expectedConfigs: &configs{
				loggerEnabled: true,
				loggerLevel:   "info",
			},
		},
		{
			name:    "WithLogger",
			configs: &configs{},
			option:  WithLogger("warn"),
			expectedConfigs: &configs{
				loggerEnabled: true,
				loggerLevel:   "warn",
			},
		},
		{
			name:    "WithPrometheus",
			configs: &configs{},
			option:  WithPrometheus(),
			expectedConfigs: &configs{
				prometheusEnabled: true,
			},
		},
		{
			name:    "WithJaegerDefaults",
			configs: &configs{},
			option:  WithJaeger("", "", "", ""),
			expectedConfigs: &configs{
				jaegerEnabled:       true,
				jaegerAgentEndpoint: "localhost:6831",
			},
		},
		{
			name:    "WithJaeger",
			configs: &configs{},
			option:  WithJaeger("localhost:6831", "http://localhost:14268/api/traces", "username", "password"),
			expectedConfigs: &configs{
				jaegerEnabled:           true,
				jaegerAgentEndpoint:     "localhost:6831",
				jaegerCollectorEndpoint: "http://localhost:14268/api/traces",
				jaegerCollectorUserName: "username",
				jaegerCollectorPassword: "password",
			},
		},
		{
			name:    "WithOpenTelemetryDefaults",
			configs: &configs{},
			option:  WithOpenTelemetry("", nil),
			expectedConfigs: &configs{
				opentelemetryEnabled:          true,
				opentelemetryCollectorAddress: "localhost:55680",
			},
		},
		{
			name:    "WithOpenTelemetry",
			configs: &configs{},
			option:  WithOpenTelemetry("localhost:55680", nil),
			expectedConfigs: &configs{
				opentelemetryEnabled:              true,
				opentelemetryCollectorAddress:     "localhost:55680",
				opentelemetryCollectorCredentials: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(T *testing.T) {
			tc.option(tc.configs)

			assert.Equal(t, tc.expectedConfigs, tc.configs)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		setAsSingleton bool
		opts           []Option
	}{
		{
			name:           "NoOption",
			setAsSingleton: false,
			opts:           []Option{},
		},
		{
			name:           "PrometheusAndJaeger",
			setAsSingleton: true,
			opts: []Option{
				WithMetadata("my-service", "0.1.0", "production", "ca-central-1", map[string]string{
					"domain": "auth",
				}),
				WithLogger("warn"),
				WithPrometheus(),
				WithJaeger("localhost:6831", "", "", ""),
			},
		},
		{
			name:           "OpenTelemetry",
			setAsSingleton: true,
			opts: []Option{
				WithMetadata("my-service", "0.1.0", "production", "ca-central-1", map[string]string{
					"domain": "auth",
				}),
				WithLogger("warn"),
				WithOpenTelemetry("localhost:55680", nil),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(T *testing.T) {
			observer := New(tc.setAsSingleton, tc.opts...)

			assert.NotNil(t, observer)
			assert.NotNil(t, observer.Logger())
			assert.NotNil(t, observer.Meter())
			assert.NotNil(t, observer.Tracer())
		})
	}
}

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name          string
		configs       configs
		expectedLevel zapcore.Level
	}{
		{
			name: "Production",
			configs: configs{
				name:        "my-service",
				version:     "0.1.0",
				environment: "production",
				region:      "ca-central-1",
				tags: map[string]string{
					"domain": "auth",
				},
				loggerLevel: "warn",
			},
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name: "LogLevelDebug",
			configs: configs{
				name:        "my-service",
				loggerLevel: "debug",
			},
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name: "LogLevelInfo",
			configs: configs{
				name:        "my-service",
				loggerLevel: "info",
			},
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name: "LogLevelWarn",
			configs: configs{
				name:        "my-service",
				loggerLevel: "warn",
			},
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name: "LogLevelError",
			configs: configs{
				name:        "my-service",
				loggerLevel: "error",
			},
			expectedLevel: zapcore.ErrorLevel,
		},
		{
			name: "LogLevelNone",
			configs: configs{
				name:        "my-service",
				loggerLevel: "none",
			},
			expectedLevel: zapcore.Level(99),
		},
		{
			name: "InvalidLogLevel",
			configs: configs{
				name:          "my-service",
				loggerEnabled: true,
				loggerLevel:   "invalid",
			},
			expectedLevel: zapcore.Level(99),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(T *testing.T) {
			logger, config := initLogger(tc.configs)

			assert.NotNil(t, logger)
			assert.NotNil(t, config)
			assert.Equal(t, tc.expectedLevel, config.Level.Level())
		})
	}
}

func TestInitPrometheus(t *testing.T) {
	tests := []struct {
		name    string
		configs configs
	}{
		{
			name: "Production",
			configs: configs{
				name:              "my-service",
				prometheusEnabled: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meter, handler := initPrometheus(tc.configs)

			assert.NotNil(t, meter)
			assert.NotNil(t, handler)
		})
	}
}

func TestInitJaeger(t *testing.T) {
	tests := []struct {
		name    string
		configs configs
	}{
		{
			name: "WithAgent",
			configs: configs{
				name: "my-service",
				tags: map[string]string{
					"domain": "auth",
				},
				jaegerEnabled:       true,
				jaegerAgentEndpoint: "localhost:6831",
			},
		},
		{
			name: "WithCollector",
			configs: configs{
				name: "my-service",
				tags: map[string]string{
					"domain": "auth",
				},
				jaegerEnabled:           true,
				jaegerCollectorEndpoint: "http://localhost:14268/api/traces",
				jaegerCollectorUserName: "username",
				jaegerCollectorPassword: "password",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracer, tracerCloser := initJaeger(tc.configs)
			defer tracerCloser()

			assert.NotNil(t, tracer)
			assert.NotNil(t, tracerCloser)
		})
	}
}

func TestInitOpenTelemetry(t *testing.T) {
	tests := []struct {
		name    string
		configs configs
	}{
		{
			name: "Insecure",
			configs: configs{
				name:                          "my-service",
				opentelemetryEnabled:          true,
				opentelemetryCollectorAddress: "localhost:55680",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meter, tracer, meterCloser, tracerCloser := initOpenTelemetry(tc.configs)
			defer meterCloser()
			defer tracerCloser()

			assert.NotNil(t, meter)
			assert.NotNil(t, tracer)
			assert.NotNil(t, meterCloser)
			assert.NotNil(t, tracerCloser)
		})
	}
}

func TestObserverClose(t *testing.T) {
	tests := []struct {
		name          string
		observer      *observer
		expectedError string
	}{
		{
			name: "Success",
			observer: &observer{
				closers: []closer{
					func() error {
						return nil
					},
				},
			},
			expectedError: "",
		},
		{
			name: "Fail",
			observer: &observer{
				closers: []closer{
					func() error {
						return errors.New("error on closing")
					},
				},
			},
			expectedError: "error on closing",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.observer.Close()

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestObserverName(t *testing.T) {
	tests := []struct {
		name     string
		observer *observer
	}{
		{
			name: "OK",
			observer: &observer{
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
		observer *observer
	}{
		{
			name: "OK",
			observer: &observer{
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
		observer *observer
		level    zapcore.Level
	}{
		{
			name: "Debug",
			observer: &observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.DebugLevel,
		},
		{
			name: "Info",
			observer: &observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.InfoLevel,
		},
		{
			name: "Warn",
			observer: &observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevel(),
				},
			},
			level: zapcore.WarnLevel,
		},
		{
			name: "Error",
			observer: &observer{
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
		observer      *observer
		expectedLevel zapcore.Level
	}{
		{
			name: "Debug",
			observer: &observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.DebugLevel),
				},
			},
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name: "Info",
			observer: &observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.InfoLevel),
				},
			},
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name: "Warn",
			observer: &observer{
				loggerConfig: &zap.Config{
					Level: zap.NewAtomicLevelAt(zapcore.WarnLevel),
				},
			},
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name: "Error",
			observer: &observer{
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
		observer *observer
	}{
		{
			name: "OK",
			observer: &observer{
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
		observer *observer
	}{
		{
			name: "OK",
			observer: &observer{
				tracer: new(trace.NoopProvider).Tracer("Noop"),
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
		observer           *observer
		req                *http.Request
		expectedStatusCode int
	}{
		{
			name: "OK",
			observer: &observer{
				promHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		singleton *observer
	}{
		{
			name:      "OK",
			singleton: &observer{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			singleton = tc.singleton

			assert.Equal(t, tc.singleton, Get())
		})
	}
}

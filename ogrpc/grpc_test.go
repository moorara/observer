package ogrpc

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockObserver struct {
	name   string
	logger *zap.Logger
	meter  metric.Meter
	tracer trace.Tracer
}

func newMockObserver() *mockObserver {
	mp := metric.NoopProvider{}
	tp := trace.NoopProvider{}

	return &mockObserver{
		name:   "test",
		logger: zap.NewNop(),
		meter:  mp.Meter("Noop"),
		tracer: tp.Tracer("Noop"),
	}
}

func (m *mockObserver) Close() error {
	return nil
}

func (m *mockObserver) Name() string {
	return m.name
}

func (m *mockObserver) Logger() *zap.Logger {
	return m.logger
}

func (m *mockObserver) SetLogLevel(level zapcore.Level) {
	// Noop
}

func (m *mockObserver) GetLogLevel() zapcore.Level {
	return zapcore.Level(99)
}

func (m *mockObserver) Meter() metric.Meter {
	return m.meter
}

func (m *mockObserver) Tracer() trace.Tracer {
	return m.tracer
}

func (m *mockObserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Noop
}

type mockServerStream struct {
	SetHeaderInMD     metadata.MD
	SetHeaderOutError error

	SendHeaderInMD     metadata.MD
	SendHeaderOutError error

	SetTrailerInMD metadata.MD

	ContextOutContext context.Context

	SendMsgInMsg    interface{}
	SendMsgOutError error

	RecvMsgInMsg    interface{}
	RecvMsgOutError error
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	m.SetHeaderInMD = md
	return m.SetHeaderOutError
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	m.SendHeaderInMD = md
	return m.SendHeaderOutError
}

func (m *mockServerStream) SetTrailer(md metadata.MD) {
	m.SetTrailerInMD = md
}

func (m *mockServerStream) Context() context.Context {
	return m.ContextOutContext
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	m.SendMsgInMsg = msg
	return m.SendMsgOutError
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	m.RecvMsgInMsg = msg
	return m.RecvMsgOutError
}

func TestEndpoint(t *testing.T) {
	tests := []struct {
		name            string
		fullMethod      string
		expectedOK      bool
		expectedPackage string
		expectedService string
		expectedMethod  string
		expectedString  string
	}{
		{
			name:       "Invalid",
			fullMethod: "GetUser",
			expectedOK: false,
		},
		{
			name:            "Valid",
			fullMethod:      "/userPB.UserManager/GetUser",
			expectedOK:      true,
			expectedPackage: "userPB",
			expectedService: "UserManager",
			expectedMethod:  "GetUser",
			expectedString:  "userPB::UserManager::GetUser",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e, ok := parseEndpoint(tc.fullMethod)

			assert.Equal(t, tc.expectedOK, ok)
			assert.Equal(t, tc.expectedPackage, e.Package)
			assert.Equal(t, tc.expectedService, e.Service)
			assert.Equal(t, tc.expectedMethod, e.Method)
			assert.Equal(t, tc.expectedString, e.String())
		})
	}
}

func TestServerStream(t *testing.T) {
	type contextKey string
	baseCtx := context.WithValue(context.Background(), contextKey("key"), "value")
	newCtx := context.WithValue(context.Background(), contextKey("foo"), "bar")

	tests := []struct {
		name        string
		ctx         context.Context
		stream      grpc.ServerStream
		expextedCtx context.Context
	}{
		{
			name:        "NoContext",
			ctx:         nil,
			stream:      &mockServerStream{ContextOutContext: baseCtx},
			expextedCtx: baseCtx,
		},
		{
			name:        "WithContext",
			ctx:         newCtx,
			stream:      &mockServerStream{ContextOutContext: baseCtx},
			expextedCtx: newCtx,
		},
		{
			name:        "AlreadyWrapped",
			ctx:         nil,
			stream:      &serverStream{ServerStream: &mockServerStream{ContextOutContext: baseCtx}},
			expextedCtx: baseCtx,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ss := ServerStreamWithContext(tc.ctx, tc.stream)

			assert.NotNil(t, ss)
			assert.Equal(t, tc.expextedCtx, ss.Context())
		})
	}
}

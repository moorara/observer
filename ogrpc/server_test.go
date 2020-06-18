package ogrpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestServerUnaryInterceptor(t *testing.T) {
	tests := []struct {
		name                string
		opts                Options
		ctx                 context.Context
		req                 interface{}
		info                *grpc.UnaryServerInfo
		mockHandlerResponse interface{}
		mockHandlerError    error
		expectedPackage     string
		expectedService     string
		expectedMethod      string
		expectedStream      bool
		expectedSuccess     bool
	}{
		{
			name:                "InvalidMethod",
			opts:                Options{},
			ctx:                 context.Background(),
			req:                 nil,
			info:                &grpc.UnaryServerInfo{FullMethod: ""},
			mockHandlerResponse: nil,
			mockHandlerError:    nil,
		},
		{
			name: "ExcludedMethods",
			opts: Options{
				ExcludedMethods: []string{"GetItem"},
			},
			ctx:                 context.Background(),
			req:                 nil,
			info:                &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			mockHandlerResponse: nil,
			mockHandlerError:    nil,
		},
		{
			name:                "HandlerFails",
			opts:                Options{},
			ctx:                 context.Background(),
			req:                 nil,
			info:                &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			mockHandlerResponse: nil,
			mockHandlerError:    errors.New("error on grpc method"),
			expectedPackage:     "itemPB",
			expectedService:     "ItemManager",
			expectedMethod:      "GetItem",
			expectedStream:      false,
			expectedSuccess:     false,
		},
		{
			name:                "HandlerSucceeds",
			opts:                Options{},
			ctx:                 context.Background(),
			req:                 nil,
			info:                &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			mockHandlerResponse: nil,
			mockHandlerError:    nil,
			expectedPackage:     "itemPB",
			expectedService:     "ItemManager",
			expectedMethod:      "GetItem",
			expectedStream:      false,
			expectedSuccess:     true,
		},
		{
			name: "LogInDebugLevel",
			opts: Options{
				LogInDebugLevel: true,
			},
			ctx:                 context.Background(),
			req:                 nil,
			info:                &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			mockHandlerResponse: nil,
			mockHandlerError:    nil,
			expectedPackage:     "itemPB",
			expectedService:     "ItemManager",
			expectedMethod:      "GetItem",
			expectedStream:      false,
			expectedSuccess:     true,
		},
		{
			name: "WithRequestMetadata",
			opts: Options{},
			ctx: metadata.NewIncomingContext(context.Background(),
				metadata.New(map[string]string{
					requestUUIDKey: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					clientNameKey:  "test-client",
				}),
			),
			req:                 nil,
			info:                &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			mockHandlerResponse: nil,
			mockHandlerError:    nil,
			expectedPackage:     "itemPB",
			expectedService:     "ItemManager",
			expectedMethod:      "GetItem",
			expectedStream:      false,
			expectedSuccess:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obsv := newMockObserver()
			si := NewServerInterceptor(obsv, tc.opts)
			assert.NotNil(t, si)

			serverOpts := si.ServerOptions()
			assert.Len(t, serverOpts, 2)

			// grpc handler for testing
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return tc.mockHandlerResponse, tc.mockHandlerError
			}

			// Testing
			res, err := si.unaryInterceptor(tc.ctx, tc.req, tc.info, handler)
			assert.Equal(t, tc.mockHandlerResponse, res)
			assert.Equal(t, tc.mockHandlerError, err)

			// TODO: Verify logs
			// TODO: Verify metrics
			// TODO: Verify traces
		})
	}
}

func TestServerStreamInterceptor(t *testing.T) {
	tests := []struct {
		name             string
		opts             Options
		srv              interface{}
		ss               *mockServerStream
		info             *grpc.StreamServerInfo
		mockHandlerError error
		expectedPackage  string
		expectedService  string
		expectedMethod   string
		expectedStream   bool
		expectedSuccess  bool
	}{
		{
			name:             "InvalidMethod",
			opts:             Options{},
			srv:              nil,
			ss:               &mockServerStream{ContextOutContext: context.Background()},
			info:             &grpc.StreamServerInfo{FullMethod: ""},
			mockHandlerError: nil,
		},
		{
			name: "ExcludedMethods",
			opts: Options{
				ExcludedMethods: []string{"GetItems"},
			},
			srv:              nil,
			ss:               &mockServerStream{ContextOutContext: context.Background()},
			info:             &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			mockHandlerError: nil,
		},
		{
			name:             "HandlerFails",
			opts:             Options{},
			srv:              nil,
			ss:               &mockServerStream{ContextOutContext: context.Background()},
			info:             &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			mockHandlerError: errors.New("error on grpc method"),
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItems",
			expectedStream:   true,
			expectedSuccess:  false,
		},
		{
			name:             "HandlerSucceeds",
			opts:             Options{},
			srv:              nil,
			ss:               &mockServerStream{ContextOutContext: context.Background()},
			info:             &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			mockHandlerError: nil,
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItems",
			expectedStream:   true,
			expectedSuccess:  true,
		},
		{
			name: "LogInDebugLevel",
			opts: Options{
				LogInDebugLevel: true,
			},
			srv:              nil,
			ss:               &mockServerStream{ContextOutContext: context.Background()},
			info:             &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			mockHandlerError: nil,
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItems",
			expectedStream:   true,
			expectedSuccess:  true,
		},
		{
			name: "WithRequestMetadata",
			opts: Options{},
			srv:  nil,
			ss: &mockServerStream{
				ContextOutContext: metadata.NewIncomingContext(context.Background(),
					metadata.New(map[string]string{
						requestUUIDKey: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
						clientNameKey:  "test-client",
					}),
				),
			},
			info:             &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			mockHandlerError: nil,
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItems",
			expectedStream:   true,
			expectedSuccess:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obsv := newMockObserver()
			si := NewServerInterceptor(obsv, tc.opts)
			assert.NotNil(t, si)

			serverOpts := si.ServerOptions()
			assert.Len(t, serverOpts, 2)

			// grpc handler for testing
			handler := func(srv interface{}, stream grpc.ServerStream) error {
				time.Sleep(2 * time.Millisecond)
				return tc.mockHandlerError
			}

			// Testing
			err := si.streamInterceptor(tc.srv, tc.ss, tc.info, handler)
			assert.Equal(t, tc.mockHandlerError, err)

			// TODO: Verify logs
			// TODO: Verify metrics
			// TODO: Verify traces
		})
	}
}

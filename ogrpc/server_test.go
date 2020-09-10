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
		name             string
		opts             Options
		ctx              context.Context
		req              interface{}
		info             *grpc.UnaryServerInfo
		handler          grpc.UnaryHandler
		expectedResponse interface{}
		expectedError    error
		expectedPackage  string
		expectedService  string
		expectedMethod   string
		expectedStream   bool
		expectedSuccess  bool
	}{
		{
			name: "InvalidMethod",
			opts: Options{},
			ctx:  context.Background(),
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: ""},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, nil
			},
			expectedResponse: nil,
			expectedError:    nil,
		},
		{
			name: "ExcludedMethods",
			opts: Options{
				ExcludedMethods: []string{"GetItem"},
			},
			ctx:  context.Background(),
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, nil
			},
			expectedResponse: nil,
			expectedError:    nil,
		},
		{
			name: "HandlerPanics",
			opts: Options{},
			ctx:  context.Background(),
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				panic("something went wrong")
			},
			expectedResponse: nil,
			expectedError:    errors.New("panic occurred: something went wrong"),
		},
		{
			name: "HandlerFails",
			opts: Options{},
			ctx:  context.Background(),
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, errors.New("error on grpc method")
			},
			expectedResponse: nil,
			expectedError:    errors.New("error on grpc method"),
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItem",
			expectedStream:   false,
			expectedSuccess:  false,
		},
		{
			name: "HandlerSucceeds",
			opts: Options{},
			ctx:  context.Background(),
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, nil
			},
			expectedResponse: nil,
			expectedError:    nil,
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItem",
			expectedStream:   false,
			expectedSuccess:  true,
		},
		{
			name: "LogInDebugLevel",
			opts: Options{
				LogInDebugLevel: true,
			},
			ctx:  context.Background(),
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, nil
			},
			expectedResponse: nil,
			expectedError:    nil,
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItem",
			expectedStream:   false,
			expectedSuccess:  true,
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
			req:  nil,
			info: &grpc.UnaryServerInfo{FullMethod: "/itemPB.ItemManager/GetItem"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, nil
			},
			expectedResponse: nil,
			expectedError:    nil,
			expectedPackage:  "itemPB",
			expectedService:  "ItemManager",
			expectedMethod:   "GetItem",
			expectedStream:   false,
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

			// Testing
			res, err := si.unaryInterceptor(tc.ctx, tc.req, tc.info, tc.handler)
			assert.Equal(t, tc.expectedResponse, res)
			assert.Equal(t, tc.expectedError, err)

			// TODO: Verify logs
			// TODO: Verify metrics
			// TODO: Verify traces
		})
	}
}

func TestServerStreamInterceptor(t *testing.T) {
	tests := []struct {
		name            string
		opts            Options
		srv             interface{}
		ss              *mockServerStream
		info            *grpc.StreamServerInfo
		handler         grpc.StreamHandler
		expectedError   error
		expectedPackage string
		expectedService string
		expectedMethod  string
		expectedStream  bool
		expectedSuccess bool
	}{
		{
			name: "InvalidMethod",
			opts: Options{},
			srv:  nil,
			ss:   &mockServerStream{ContextOutContext: context.Background()},
			info: &grpc.StreamServerInfo{FullMethod: ""},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				time.Sleep(2 * time.Millisecond)
				return nil
			},
			expectedError: nil,
		},
		{
			name: "ExcludedMethods",
			opts: Options{
				ExcludedMethods: []string{"GetItems"},
			},
			srv:  nil,
			ss:   &mockServerStream{ContextOutContext: context.Background()},
			info: &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				time.Sleep(2 * time.Millisecond)
				return nil
			},
			expectedError: nil,
		},
		{
			name: "HandlerPanics",
			opts: Options{},
			srv:  nil,
			ss:   &mockServerStream{ContextOutContext: context.Background()},
			info: &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				panic("something went wrong")
			},
			expectedError: errors.New("panic occurred: something went wrong"),
		},
		{
			name: "HandlerFails",
			opts: Options{},
			srv:  nil,
			ss:   &mockServerStream{ContextOutContext: context.Background()},
			info: &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				time.Sleep(2 * time.Millisecond)
				return errors.New("error on grpc method")
			},
			expectedError:   errors.New("error on grpc method"),
			expectedPackage: "itemPB",
			expectedService: "ItemManager",
			expectedMethod:  "GetItems",
			expectedStream:  true,
			expectedSuccess: false,
		},
		{
			name: "HandlerSucceeds",
			opts: Options{},
			srv:  nil,
			ss:   &mockServerStream{ContextOutContext: context.Background()},
			info: &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			expectedError:   nil,
			expectedPackage: "itemPB",
			expectedService: "ItemManager",
			expectedMethod:  "GetItems",
			expectedStream:  true,
			expectedSuccess: true,
		},
		{
			name: "LogInDebugLevel",
			opts: Options{
				LogInDebugLevel: true,
			},
			srv:  nil,
			ss:   &mockServerStream{ContextOutContext: context.Background()},
			info: &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			expectedError:   nil,
			expectedPackage: "itemPB",
			expectedService: "ItemManager",
			expectedMethod:  "GetItems",
			expectedStream:  true,
			expectedSuccess: true,
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
			info: &grpc.StreamServerInfo{FullMethod: "/itemPB.ItemManager/GetItems"},
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			expectedError:   nil,
			expectedPackage: "itemPB",
			expectedService: "ItemManager",
			expectedMethod:  "GetItems",
			expectedStream:  true,
			expectedSuccess: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obsv := newMockObserver()
			si := NewServerInterceptor(obsv, tc.opts)
			assert.NotNil(t, si)

			serverOpts := si.ServerOptions()
			assert.Len(t, serverOpts, 2)

			// Testing
			err := si.streamInterceptor(tc.srv, tc.ss, tc.info, tc.handler)
			assert.Equal(t, tc.expectedError, err)

			// TODO: Verify logs
			// TODO: Verify metrics
			// TODO: Verify traces
		})
	}
}

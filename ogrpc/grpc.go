// Package ogrpc is an observable grpc package.
// It can be used for building gRPC servers and clients that automatically report logs, metrics, and traces.
package ogrpc

import (
	"context"
	"fmt"
	"regexp"

	"google.golang.org/grpc"
)

const (
	libraryName    = "observer/ogrpc"
	requestUUIDKey = "request-uuid"
	clientNameKey  = "client-name"
)

var (
	fullMethodRegex = regexp.MustCompile(`/|\.`)
)

// Options are optional configurations for creating interceptors.
type Options struct {
	LogInDebugLevel bool
	ExcludedMethods []string
}

func (opts Options) withDefaults() Options {
	return opts
}

// endpoint is a grpc endpoint.
type endpoint struct {
	Package string
	Service string
	Method  string
}

// fullMethod is in the form of /package.service/method
func parseEndpoint(fullMethod string) (endpoint, bool) {
	subs := fullMethodRegex.Split(fullMethod, 4)
	if len(subs) != 4 {
		return endpoint{}, false
	}

	return endpoint{
		Package: subs[1],
		Service: subs[2],
		Method:  subs[3],
	}, true
}

// String implements the fmt.Stringer interface.
func (e endpoint) String() string {
	var s string
	if e.Package != "" && e.Service != "" && e.Method != "" {
		s = fmt.Sprintf("%s::%s::%s", e.Package, e.Service, e.Method)
	}
	return s
}

type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStream) Context() context.Context {
	if s.ctx == nil {
		return s.ServerStream.Context()
	}

	return s.ctx
}

// ServerStreamWithContext returns a new grpc.ServerStream with a new context.
func ServerStreamWithContext(ctx context.Context, s grpc.ServerStream) grpc.ServerStream {
	if ss, ok := s.(*serverStream); ok {
		ss.ctx = ctx
		return ss
	}

	return &serverStream{
		ServerStream: s,
		ctx:          ctx,
	}
}

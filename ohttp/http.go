// Package ohttp is an observable http package.
// It can be used for building HTTP servers and clients that automatically report logs, metrics, and traces.
package ohttp

import (
	"fmt"
	"net/http"
	"regexp"
)

const (
	libraryName       = "observer/ohttp"
	requestUUIDHeader = "Request-UUID"
	clientNameHeader  = "Client-Name"
)

// Options are optional configurations for creating middleware and clients.
type Options struct {
	LogInDebugLevel bool
	IDRegexp        *regexp.Regexp
}

func (opts Options) withDefaults() Options {
	if opts.IDRegexp == nil {
		opts.IDRegexp = regexp.MustCompile("[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}")
	}

	return opts
}

// responseWriter extends the standard http.ResponseWriter.
type responseWriter struct {
	http.ResponseWriter
	StatusCode  int
	StatusClass string
}

// NewResponseWriter creates a new response writer.
func newResponseWriter(rw http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: rw,
	}
}

// WriteHeader overrides the implementation of http.WriteHeader.
func (r *responseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)

	// Only capture the first value
	if r.StatusCode == 0 {
		r.StatusCode = statusCode
		r.StatusClass = fmt.Sprintf("%dxx", statusCode/100)
	}
}

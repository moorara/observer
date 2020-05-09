package ohttp

import (
	"fmt"
	"net/http"
)

const (
	libraryName       = "observer"
	requestUUIDHeader = "Request-UUID"
)

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

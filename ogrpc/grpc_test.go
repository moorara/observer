package ogrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndpoint(t *testing.T) {
	tests := []struct {
		name            string
		fullMethod      string
		expectedOK      bool
		expectedPackage string
		expectedService string
		expectedMethod  string
		expectedString  string
	}{}

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

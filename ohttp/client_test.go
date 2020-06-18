package ohttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/moorara/observer"
	"github.com/stretchr/testify/assert"
)

func TestClientDo(t *testing.T) {
	tests := []struct {
		name                string
		opts                Options
		method              string
		url                 string
		ctx                 context.Context
		mockStatusCode      int
		expectedMethod      string
		expectedURL         string
		expectedRoute       string
		expectedStatusCode  int
		expectedStatusClass string
	}{
		{
			name:                "Success",
			opts:                Options{},
			method:              "GET",
			url:                 "/v1/items/00000000-0000-0000-0000-000000000000",
			ctx:                 context.Background(),
			mockStatusCode:      200,
			expectedMethod:      "GET",
			expectedURL:         "/v1/items/00000000-0000-0000-0000-000000000000",
			expectedRoute:       "/v1/items/:id",
			expectedStatusCode:  200,
			expectedStatusClass: "2xx",
		},
		{
			name:                "BadRequest",
			opts:                Options{},
			method:              "GET",
			url:                 "/v1/items/00000000-0000-0000-0000-000000000000",
			ctx:                 context.Background(),
			mockStatusCode:      400,
			expectedMethod:      "GET",
			expectedURL:         "/v1/items/00000000-0000-0000-0000-000000000000",
			expectedRoute:       "/v1/items/:id",
			expectedStatusCode:  400,
			expectedStatusClass: "4xx",
		},
		{
			name:                "InternalServerError",
			opts:                Options{},
			method:              "GET",
			url:                 "/v1/items/00000000-0000-0000-0000-000000000000",
			ctx:                 context.Background(),
			mockStatusCode:      500,
			expectedMethod:      "GET",
			expectedURL:         "/v1/items/00000000-0000-0000-0000-000000000000",
			expectedRoute:       "/v1/items/:id",
			expectedStatusCode:  500,
			expectedStatusClass: "5xx",
		},
		{
			name: "LogInDebugLevel",
			opts: Options{
				LogInDebugLevel: true,
			},
			method:              "GET",
			url:                 "/v1/items/00000000-0000-0000-0000-000000000000",
			ctx:                 context.Background(),
			mockStatusCode:      200,
			expectedMethod:      "GET",
			expectedURL:         "/v1/items/00000000-0000-0000-0000-000000000000",
			expectedRoute:       "/v1/items/:id",
			expectedStatusCode:  200,
			expectedStatusClass: "2xx",
		},
		{
			name: "WithRequestUUID",
			opts: Options{
				LogInDebugLevel: true,
			},
			method:              "GET",
			url:                 "/v1/items/00000000-0000-0000-0000-000000000000",
			ctx:                 observer.ContextWithUUID(context.Background(), "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			mockStatusCode:      200,
			expectedMethod:      "GET",
			expectedURL:         "/v1/items/00000000-0000-0000-0000-000000000000",
			expectedRoute:       "/v1/items/:id",
			expectedStatusCode:  200,
			expectedStatusClass: "2xx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &http.Client{}
			obsv := newMockObserver()
			client := NewClient(c, obsv, tc.opts)
			assert.NotNil(t, client)

			// http server for testing
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Millisecond)
				w.WriteHeader(tc.mockStatusCode)
			}))
			defer ts.Close()

			// Create an http request
			url := ts.URL + tc.url
			request, _ := http.NewRequest(tc.method, url, nil)
			request = request.WithContext(tc.ctx)

			// Testing
			resp, err := client.Do(request)

			assert.NoError(t, err)
			assert.Equal(t, tc.mockStatusCode, resp.StatusCode)

			// TODO: Verify logs
			// TODO: Verify metrics
			// TODO: Verify traces
		})
	}
}

func TestClientMisc(t *testing.T) {
	tests := []struct {
		name                string
		opts                Options
		url                 string
		mockStatusCode      int
		expectError         bool
		expectedURL         string
		expectedRoute       string
		expectedStatusCode  int
		expectedStatusClass string
	}{
		{
			name:        "InvalidURL",
			opts:        Options{},
			url:         " ",
			expectError: true,
		},
		{
			name:                "Success",
			opts:                Options{},
			url:                 "/v1/items",
			mockStatusCode:      200,
			expectedURL:         "/v1/items",
			expectedRoute:       "/v1/items",
			expectedStatusCode:  200,
			expectedStatusClass: "2xx",
		},
		{
			name:                "BadRequest",
			opts:                Options{},
			url:                 "/v1/items",
			mockStatusCode:      400,
			expectedURL:         "/v1/items",
			expectedRoute:       "/v1/items",
			expectedStatusCode:  400,
			expectedStatusClass: "4xx",
		},
		{
			name:                "InternalServerError",
			opts:                Options{},
			url:                 "/v1/items",
			mockStatusCode:      500,
			expectedURL:         "/v1/items",
			expectedRoute:       "/v1/items",
			expectedStatusCode:  500,
			expectedStatusClass: "5xx",
		},
		{
			name: "LogInDebugLevel",
			opts: Options{
				LogInDebugLevel: true,
			},
			url:                 "/v1/items",
			mockStatusCode:      200,
			expectedURL:         "/v1/items",
			expectedRoute:       "/v1/items",
			expectedStatusCode:  200,
			expectedStatusClass: "2xx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &http.Client{}
			obsv := newMockObserver()
			client := NewClient(c, obsv, tc.opts)
			assert.NotNil(t, client)

			// http server for testing
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Millisecond)
				w.WriteHeader(tc.mockStatusCode)
			}))
			defer ts.Close()

			t.Run("CloseIdleConnections", func(t *testing.T) {
				client.CloseIdleConnections()
			})

			t.Run("Get", func(t *testing.T) {
				// Testing
				resp, err := client.Get(ts.URL + tc.url)

				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.mockStatusCode, resp.StatusCode)
				}

				// TODO: Verify logs
				// TODO: Verify metrics
				// TODO: Verify traces
			})

			t.Run("Head", func(t *testing.T) {
				// Testing
				resp, err := client.Head(ts.URL + tc.url)

				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.mockStatusCode, resp.StatusCode)
				}

				// TODO: Verify logs
				// TODO: Verify metrics
				// TODO: Verify traces
			})

			t.Run("Post", func(t *testing.T) {
				// Testing
				resp, err := client.Post(ts.URL+tc.url, "application/json", nil)

				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.mockStatusCode, resp.StatusCode)
				}

				// TODO: Verify logs
				// TODO: Verify metrics
				// TODO: Verify traces
			})

			t.Run("PostForm", func(t *testing.T) {
				// Testing
				resp, err := client.PostForm(ts.URL+tc.url, nil)

				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.mockStatusCode, resp.StatusCode)
				}

				// TODO: Verify logs
				// TODO: Verify metrics
				// TODO: Verify traces
			})
		})
	}
}
